package service

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	auditdomain "github.com/example/adnova/internal/audit/domain"
	auditservice "github.com/example/adnova/internal/audit/service"
	ingestiondomain "github.com/example/adnova/internal/ingestion/domain"
	ingestionservice "github.com/example/adnova/internal/ingestion/service"
	mmpdomain "github.com/example/adnova/internal/mmp/domain"
	mmpprovider "github.com/example/adnova/internal/mmp/provider"
	"github.com/example/adnova/internal/mmp/repository"
	"github.com/example/adnova/pkg/provider"
	"github.com/stretchr/testify/require"
)

type memoryRepo struct {
	connections map[string]*mmpdomain.Connection
	runs        map[string]*mmpdomain.SyncRun
}

type losingClaimRepo struct {
	*memoryRepo
	competingStartedAt time.Time
}

type recordingRenewRepo struct {
	*memoryRepo
	renewCalls atomic.Int32
	loseClaim  bool
}

func (r *recordingRenewRepo) RenewSyncRunClaim(ctx context.Context, tenant, id, token string, lockedUntil time.Time) (bool, error) {
	r.renewCalls.Add(1)
	if r.loseClaim {
		return false, nil
	}
	return r.memoryRepo.RenewSyncRunClaim(ctx, tenant, id, token, lockedUntil)
}

func (r *losingClaimRepo) ClaimSyncRun(_ context.Context, tenant, id, _ string, _ time.Time, _ time.Time, _ time.Time, lockedUntil time.Time, _ string) (string, bool, error) {
	row := r.runs[id]
	if row != nil && row.TenantID == tenant {
		row.Status = mmpdomain.SyncProcessing
		row.StartedAt = r.competingStartedAt
		row.LockedUntil = &lockedUntil
		row.ClaimToken = "competing-claim"
		row.FinishedAt = nil
	}
	return "", false, nil
}

func newMemoryRepo() *memoryRepo {
	return &memoryRepo{connections: map[string]*mmpdomain.Connection{}, runs: map[string]*mmpdomain.SyncRun{}}
}
func (r *memoryRepo) GameExists(context.Context, string, string) (bool, error) { return true, nil }
func (r *memoryRepo) ListConnections(_ context.Context, tenant string) ([]mmpdomain.Connection, error) {
	var rows []mmpdomain.Connection
	for _, row := range r.connections {
		if row.TenantID == tenant {
			rows = append(rows, *row)
		}
	}
	return rows, nil
}
func (r *memoryRepo) ListAllConnections(context.Context) ([]mmpdomain.Connection, error) {
	rows := make([]mmpdomain.Connection, 0, len(r.connections))
	for _, row := range r.connections {
		rows = append(rows, *row)
	}
	return rows, nil
}
func (r *memoryRepo) GetConnection(_ context.Context, tenant, id string) (*mmpdomain.Connection, error) {
	row := r.connections[id]
	if row == nil || row.TenantID != tenant {
		return nil, repository.ErrNotFound
	}
	copy := *row
	return &copy, nil
}
func (r *memoryRepo) FindConnection(_ context.Context, tenant, game, providerName string) (*mmpdomain.Connection, error) {
	for _, row := range r.connections {
		if row.TenantID == tenant && row.GameID == game && row.Provider == providerName {
			copy := *row
			return &copy, nil
		}
	}
	return nil, repository.ErrNotFound
}
func (r *memoryRepo) UpsertConnection(_ context.Context, row *mmpdomain.Connection) error {
	for id, existing := range r.connections {
		if existing.TenantID == row.TenantID && existing.GameID == row.GameID && existing.Provider == row.Provider {
			row.ID = id
		}
	}
	copy := *row
	r.connections[row.ID] = &copy
	return nil
}
func (r *memoryRepo) TouchConnection(_ context.Context, tenant, id string, at time.Time) error {
	if row := r.connections[id]; row != nil && row.TenantID == tenant {
		row.LastSyncAt = &at
		return nil
	}
	return repository.ErrNotFound
}
func (r *memoryRepo) ListSyncRuns(_ context.Context, tenant string) ([]mmpdomain.SyncRun, error) {
	var rows []mmpdomain.SyncRun
	for _, row := range r.runs {
		if row.TenantID == tenant {
			rows = append(rows, *row)
		}
	}
	return rows, nil
}
func (r *memoryRepo) FindSyncRunByKey(_ context.Context, tenant, key string) (*mmpdomain.SyncRun, error) {
	for _, row := range r.runs {
		if row.TenantID == tenant && row.IdempotencyKey == key {
			copy := *row
			return &copy, nil
		}
	}
	return nil, repository.ErrNotFound
}
func (r *memoryRepo) CreateSyncRun(_ context.Context, row *mmpdomain.SyncRun) error {
	copy := *row
	r.runs[row.ID] = &copy
	return nil
}
func (r *memoryRepo) ClaimSyncRun(_ context.Context, tenant, id, expectedStatus string, expectedStartedAt, startedAt, now, lockedUntil time.Time, requestedBy string) (string, bool, error) {
	row := r.runs[id]
	if row == nil || row.TenantID != tenant || row.Status != expectedStatus || !row.StartedAt.Equal(expectedStartedAt) {
		return "", false, nil
	}
	if expectedStatus == mmpdomain.SyncProcessing && row.LockedUntil != nil && !row.LockedUntil.Before(now) {
		return "", false, nil
	}
	token := requestedBy + ":" + startedAt.Format(time.RFC3339Nano)
	row.Status, row.RequestedBy, row.StartedAt, row.FinishedAt = mmpdomain.SyncProcessing, requestedBy, startedAt, nil
	row.LockedUntil, row.ClaimToken = &lockedUntil, token
	row.ImportJobID, row.SourceRows, row.NormalizedRows, row.SkippedRows, row.WarningMessage = "", 0, 0, 0, ""
	row.ErrorCode, row.ErrorMessage = "", ""
	return token, true, nil
}
func (r *memoryRepo) RenewSyncRunClaim(_ context.Context, tenant, id, token string, lockedUntil time.Time) (bool, error) {
	row := r.runs[id]
	if row == nil || row.TenantID != tenant || row.Status != mmpdomain.SyncProcessing || row.ClaimToken != token {
		return false, nil
	}
	row.LockedUntil = &lockedUntil
	return true, nil
}
func (r *memoryRepo) UpdateClaimedSyncRun(_ context.Context, row *mmpdomain.SyncRun) (bool, error) {
	current := r.runs[row.ID]
	if current == nil || current.TenantID != row.TenantID || current.Status != mmpdomain.SyncProcessing || current.ClaimToken != row.ClaimToken {
		return false, nil
	}
	copy := *row
	copy.LockedUntil, copy.ClaimToken = nil, ""
	r.runs[row.ID] = &copy
	return true, nil
}

type fakeFetcher struct {
	configured bool
	result     *mmpprovider.FetchResult
	err        error
	calls      int
	input      mmpprovider.FetchInput
}

func (f *fakeFetcher) Configured() bool { return f.configured }
func (f *fakeFetcher) Fetch(_ context.Context, input mmpprovider.FetchInput) (*mmpprovider.FetchResult, error) {
	f.calls++
	f.input = input
	return f.result, f.err
}

func newService(repo Repository, fetcher mmpprovider.Fetcher, importer Importer, auditor auditservice.Recorder) *Service {
	return New(repo, map[string]mmpprovider.Fetcher{mmpdomain.ProviderAppsFlyer: fetcher}, importer, auditor, map[string]int{mmpdomain.ProviderAppsFlyer: 7})
}

type fakeImporter struct {
	calls int
	input ingestionservice.ImportInput
}

func (f *fakeImporter) Import(_ context.Context, input ingestionservice.ImportInput) (*ingestiondomain.ImportJob, error) {
	f.calls++
	f.input = input
	return &ingestiondomain.ImportJob{ID: "job-1"}, nil
}

type slowImporter struct {
	delay time.Duration
	calls int
}

func (f *slowImporter) Import(ctx context.Context, _ ingestionservice.ImportInput) (*ingestiondomain.ImportJob, error) {
	f.calls++
	timer := time.NewTimer(f.delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
		return &ingestiondomain.ImportJob{ID: "slow-job"}, nil
	}
}

type fakeAuditor struct{ actions []string }

func (a *fakeAuditor) Record(_ context.Context, input auditdomain.RecordInput) error {
	a.actions = append(a.actions, input.Action)
	return nil
}

func TestConfigureAppsFlyerReportsCredentialHealthWithoutExposingToken(t *testing.T) {
	repo := newMemoryRepo()
	fetcher := &fakeFetcher{configured: false}
	auditor := &fakeAuditor{}
	svc := newService(repo, fetcher, &fakeImporter{}, auditor)
	view, err := svc.ConfigureAppsFlyer(context.Background(), ConfigureInput{TenantID: "tenant", UserID: "user", GameID: "game", ExternalAppID: "com.example.game"})
	require.NoError(t, err)
	require.False(t, view.CredentialConfigured)
	require.Equal(t, "NOT_CONFIGURED", view.Health)
	require.Equal(t, []string{"MMP_CONNECTION_CONFIGURED"}, auditor.actions)
}

func TestConfigureAdjustUsesIndependentProviderConnection(t *testing.T) {
	repo := newMemoryRepo()
	adjustFetcher := &fakeFetcher{configured: true}
	svc := New(repo, map[string]mmpprovider.Fetcher{mmpdomain.ProviderAdjust: adjustFetcher}, &fakeImporter{}, nil, map[string]int{mmpdomain.ProviderAdjust: 31})
	view, err := svc.ConfigureAdjust(context.Background(), ConfigureInput{TenantID: "tenant", UserID: "user", GameID: "game", ExternalAppID: "adjust-app-token"})
	require.NoError(t, err)
	require.Equal(t, mmpdomain.ProviderAdjust, view.Provider)
	require.Equal(t, "READY", view.Health)
	require.True(t, view.CredentialConfigured)
}

func TestConfigureAppsFlyerRejectsAppIDChangeAfterSuccessfulSync(t *testing.T) {
	repo := newMemoryRepo()
	lastSync := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	repo.connections["connection"] = &mmpdomain.Connection{ID: "connection", TenantID: "tenant", GameID: "game", Provider: mmpdomain.ProviderAppsFlyer, ExternalAppID: "old.app", Status: mmpdomain.ConnectionActive, LastSyncAt: &lastSync}
	svc := newService(repo, &fakeFetcher{configured: true}, &fakeImporter{}, nil)
	_, err := svc.ConfigureAppsFlyer(context.Background(), ConfigureInput{TenantID: "tenant", UserID: "user", GameID: "game", ExternalAppID: "new.app"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "不能直接更换")
}

func TestSyncImportsDeterministicRecordsAndIsIdempotent(t *testing.T) {
	repo := newMemoryRepo()
	repo.connections["connection"] = &mmpdomain.Connection{ID: "connection", TenantID: "tenant", GameID: "game", Provider: mmpdomain.ProviderAppsFlyer, ExternalAppID: "com.example.game", Status: mmpdomain.ConnectionActive}
	fetcher := &fakeFetcher{configured: true, result: &mmpprovider.FetchResult{Records: []provider.Record{{"date": "2026-08-01", "campaign_external_id": "cmp", "country": "US", "currency": "USD", "installs": "2", "activations": "2", "payers": "1", "revenue": "3.000000"}}, SourceRows: 3}}
	importer := &fakeImporter{}
	auditor := &fakeAuditor{}
	svc := newService(repo, fetcher, importer, auditor)
	svc.now = func() time.Time { return time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC) }
	input := SyncInput{TenantID: "tenant", UserID: "user", ConnectionID: "connection", From: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC)}
	first, err := svc.Sync(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, mmpdomain.SyncSucceeded, first.Status)
	require.Equal(t, "job-1", first.ImportJobID)
	require.Equal(t, 1, fetcher.calls)
	require.Equal(t, 1, importer.calls)
	require.True(t, importer.input.Authoritative)
	require.Equal(t, input.From, importer.input.PeriodStart)
	require.Equal(t, input.To, importer.input.PeriodEnd)
	require.NotContains(t, string(importer.input.Data), "secret")
	second, err := svc.Sync(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
	require.Equal(t, 1, fetcher.calls)
}

func TestSyncPersistsSafeProviderFailure(t *testing.T) {
	repo := newMemoryRepo()
	repo.connections["connection"] = &mmpdomain.Connection{ID: "connection", TenantID: "tenant", GameID: "game", Provider: mmpdomain.ProviderAppsFlyer, ExternalAppID: "app", Status: mmpdomain.ConnectionActive}
	fetcher := &fakeFetcher{configured: true, err: &mmpprovider.Error{Code: "UNAUTHORIZED", Message: "AppsFlyer Token 无效"}}
	auditor := &fakeAuditor{}
	svc := newService(repo, fetcher, &fakeImporter{}, auditor)
	svc.now = func() time.Time { return time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC) }
	_, err := svc.Sync(context.Background(), SyncInput{TenantID: "tenant", UserID: "user", ConnectionID: "connection", From: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)})
	require.Error(t, err)
	runs, _ := repo.ListSyncRuns(context.Background(), "tenant")
	require.Len(t, runs, 1)
	require.Equal(t, mmpdomain.SyncFailed, runs[0].Status)
	require.Equal(t, "UNAUTHORIZED", runs[0].ErrorCode)
	require.Equal(t, []string{"MMP_SYNC_FAILED"}, auditor.actions)
	require.False(t, errors.Is(err, repository.ErrNotFound))
}

func TestSyncDoesNotFetchWhenAnotherExecutorWinsRetryClaim(t *testing.T) {
	base := newMemoryRepo()
	base.connections["connection"] = &mmpdomain.Connection{ID: "connection", TenantID: "tenant", GameID: "game", Provider: mmpdomain.ProviderAppsFlyer, ExternalAppID: "app", Status: mmpdomain.ConnectionActive}
	initialFetcher := &fakeFetcher{configured: true, err: &mmpprovider.Error{Code: "UPSTREAM_UNAVAILABLE", Message: "temporarily unavailable"}}
	initialService := newService(base, initialFetcher, &fakeImporter{}, nil)
	initialService.now = func() time.Time { return time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC) }
	input := SyncInput{TenantID: "tenant", UserID: "user", ConnectionID: "connection", From: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)}
	_, err := initialService.Sync(context.Background(), input)
	require.Error(t, err)

	competingStartedAt := time.Date(2026, 8, 4, 1, 0, 0, 0, time.UTC)
	repo := &losingClaimRepo{memoryRepo: base, competingStartedAt: competingStartedAt}
	fetcher := &fakeFetcher{configured: true, result: &mmpprovider.FetchResult{}}
	importer := &fakeImporter{}
	svc := newService(repo, fetcher, importer, nil)
	svc.now = func() time.Time { return competingStartedAt }

	run, err := svc.Sync(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, mmpdomain.SyncProcessing, run.Status)
	require.Equal(t, competingStartedAt, run.StartedAt)
	require.Zero(t, fetcher.calls)
	require.Zero(t, importer.calls)
}

func TestSyncRenewsLeaseWhileImporterIsRunning(t *testing.T) {
	base := newMemoryRepo()
	base.connections["connection"] = &mmpdomain.Connection{ID: "connection", TenantID: "tenant", GameID: "game", Provider: mmpdomain.ProviderAppsFlyer, ExternalAppID: "app", Status: mmpdomain.ConnectionActive}
	repo := &recordingRenewRepo{memoryRepo: base}
	fetcher := &fakeFetcher{configured: true, result: &mmpprovider.FetchResult{Records: []provider.Record{}}}
	importer := &slowImporter{delay: 35 * time.Millisecond}
	svc := newService(repo, fetcher, importer, nil)
	svc.syncLease = 30 * time.Millisecond
	svc.heartbeat = 5 * time.Millisecond
	svc.now = func() time.Time { return time.Now().UTC() }

	run, err := svc.Sync(context.Background(), SyncInput{TenantID: "tenant", UserID: "user", ConnectionID: "connection", From: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	require.Equal(t, mmpdomain.SyncSucceeded, run.Status)
	require.GreaterOrEqual(t, repo.renewCalls.Load(), int32(2))
}

func TestSyncStopsBeforeCompletionWhenHeartbeatLosesClaim(t *testing.T) {
	base := newMemoryRepo()
	base.connections["connection"] = &mmpdomain.Connection{ID: "connection", TenantID: "tenant", GameID: "game", Provider: mmpdomain.ProviderAppsFlyer, ExternalAppID: "app", Status: mmpdomain.ConnectionActive}
	repo := &recordingRenewRepo{memoryRepo: base, loseClaim: true}
	fetcher := &fakeFetcher{configured: true, result: &mmpprovider.FetchResult{Records: []provider.Record{}}}
	importer := &slowImporter{delay: time.Second}
	svc := newService(repo, fetcher, importer, nil)
	svc.syncLease = 30 * time.Millisecond
	svc.heartbeat = 5 * time.Millisecond
	svc.now = func() time.Time { return time.Now().UTC() }

	run, err := svc.Sync(context.Background(), SyncInput{TenantID: "tenant", UserID: "user", ConnectionID: "connection", From: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)})
	require.Error(t, err)
	require.Contains(t, err.Error(), "已被其他执行接管")
	require.Equal(t, mmpdomain.SyncProcessing, run.Status)
	require.Equal(t, 1, importer.calls)
	require.Equal(t, int32(1), repo.renewCalls.Load())
	stored, findErr := repo.FindSyncRunByKey(context.Background(), "tenant", run.IdempotencyKey)
	require.NoError(t, findErr)
	require.Equal(t, mmpdomain.SyncProcessing, stored.Status)
}

func TestAutoSyncAllCoversTenantsAndHonorsProviderRanges(t *testing.T) {
	repo := newMemoryRepo()
	repo.connections["af"] = &mmpdomain.Connection{ID: "af", TenantID: "tenant-a", GameID: "game-a", Provider: mmpdomain.ProviderAppsFlyer, ExternalAppID: "af-app", Status: mmpdomain.ConnectionActive}
	repo.connections["adjust"] = &mmpdomain.Connection{ID: "adjust", TenantID: "tenant-b", GameID: "game-b", Provider: mmpdomain.ProviderAdjust, ExternalAppID: "adjust-app", Status: mmpdomain.ConnectionActive}
	result := &mmpprovider.FetchResult{Records: []provider.Record{{"date": "2026-08-08", "campaign_external_id": "cmp", "country": "US", "currency": "USD", "installs": "1", "activations": "1", "payers": "0", "revenue": "0.000000"}}, SourceRows: 1}
	afFetcher, adjustFetcher := &fakeFetcher{configured: true, result: result}, &fakeFetcher{configured: true, result: result}
	svc := New(repo, map[string]mmpprovider.Fetcher{mmpdomain.ProviderAppsFlyer: afFetcher, mmpdomain.ProviderAdjust: adjustFetcher}, &fakeImporter{}, nil, map[string]int{mmpdomain.ProviderAppsFlyer: 7, mmpdomain.ProviderAdjust: 31})
	svc.now = func() time.Time { return time.Date(2026, 8, 9, 8, 0, 0, 0, time.UTC) }

	synced, err := svc.AutoSyncAll(context.Background(), 14)
	require.NoError(t, err)
	require.Equal(t, 2, synced)
	require.Equal(t, "2026-08-02", afFetcher.input.From.Format("2006-01-02"))
	require.Equal(t, "2026-07-26", adjustFetcher.input.From.Format("2006-01-02"))
	require.Equal(t, "2026-08-08", afFetcher.input.To.Format("2006-01-02"))
	require.NotNil(t, repo.connections["af"].LastSyncAt)
	require.NotNil(t, repo.connections["adjust"].LastSyncAt)

	synced, err = svc.AutoSyncAll(context.Background(), 14)
	require.NoError(t, err)
	require.Equal(t, 2, synced)
	require.Equal(t, 1, afFetcher.calls)
	require.Equal(t, 1, adjustFetcher.calls)
}
