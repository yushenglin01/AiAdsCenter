package service

import (
	"context"
	"errors"
	"testing"
	"time"

	auditdomain "github.com/example/adnova/internal/audit/domain"
	ingestiondomain "github.com/example/adnova/internal/ingestion/domain"
	ingestionservice "github.com/example/adnova/internal/ingestion/service"
	"github.com/example/adnova/internal/mmp/appsflyer"
	mmpdomain "github.com/example/adnova/internal/mmp/domain"
	"github.com/example/adnova/internal/mmp/repository"
	"github.com/example/adnova/pkg/provider"
	"github.com/stretchr/testify/require"
)

type memoryRepo struct {
	connections map[string]*mmpdomain.Connection
	runs        map[string]*mmpdomain.SyncRun
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
func (r *memoryRepo) UpdateSyncRun(_ context.Context, row *mmpdomain.SyncRun) error {
	copy := *row
	r.runs[row.ID] = &copy
	return nil
}

type fakeFetcher struct {
	configured bool
	result     *appsflyer.FetchResult
	err        error
	calls      int
}

func (f *fakeFetcher) Configured() bool { return f.configured }
func (f *fakeFetcher) Fetch(context.Context, appsflyer.FetchInput) (*appsflyer.FetchResult, error) {
	f.calls++
	return f.result, f.err
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

type fakeAuditor struct{ actions []string }

func (a *fakeAuditor) Record(_ context.Context, input auditdomain.RecordInput) error {
	a.actions = append(a.actions, input.Action)
	return nil
}

func TestConfigureAppsFlyerReportsCredentialHealthWithoutExposingToken(t *testing.T) {
	repo := newMemoryRepo()
	fetcher := &fakeFetcher{configured: false}
	auditor := &fakeAuditor{}
	svc := New(repo, fetcher, &fakeImporter{}, auditor, 7)
	view, err := svc.ConfigureAppsFlyer(context.Background(), ConfigureInput{TenantID: "tenant", UserID: "user", GameID: "game", ExternalAppID: "com.example.game"})
	require.NoError(t, err)
	require.False(t, view.CredentialConfigured)
	require.Equal(t, "NOT_CONFIGURED", view.Health)
	require.Equal(t, []string{"MMP_CONNECTION_CONFIGURED"}, auditor.actions)
}

func TestConfigureAppsFlyerRejectsAppIDChangeAfterSuccessfulSync(t *testing.T) {
	repo := newMemoryRepo()
	lastSync := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	repo.connections["connection"] = &mmpdomain.Connection{ID: "connection", TenantID: "tenant", GameID: "game", Provider: mmpdomain.ProviderAppsFlyer, ExternalAppID: "old.app", Status: mmpdomain.ConnectionActive, LastSyncAt: &lastSync}
	svc := New(repo, &fakeFetcher{configured: true}, &fakeImporter{}, nil, 7)
	_, err := svc.ConfigureAppsFlyer(context.Background(), ConfigureInput{TenantID: "tenant", UserID: "user", GameID: "game", ExternalAppID: "new.app"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "不能直接更换")
}

func TestSyncImportsDeterministicRecordsAndIsIdempotent(t *testing.T) {
	repo := newMemoryRepo()
	repo.connections["connection"] = &mmpdomain.Connection{ID: "connection", TenantID: "tenant", GameID: "game", Provider: mmpdomain.ProviderAppsFlyer, ExternalAppID: "com.example.game", Status: mmpdomain.ConnectionActive}
	fetcher := &fakeFetcher{configured: true, result: &appsflyer.FetchResult{Records: []provider.Record{{"date": "2026-08-01", "campaign_external_id": "cmp", "country": "US", "currency": "USD", "installs": "2", "activations": "2", "payers": "1", "revenue": "3.000000"}}, SourceRows: 3}}
	importer := &fakeImporter{}
	auditor := &fakeAuditor{}
	svc := New(repo, fetcher, importer, auditor, 7)
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
	fetcher := &fakeFetcher{configured: true, err: &appsflyer.ProviderError{Code: "UNAUTHORIZED", Message: "AppsFlyer Token 无效"}}
	auditor := &fakeAuditor{}
	svc := New(repo, fetcher, &fakeImporter{}, auditor, 7)
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
