package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	researchdomain "github.com/example/adnova/internal/research/domain"
	"github.com/example/adnova/internal/research/websearch"
	"github.com/stretchr/testify/require"
)

type fakeWebSearch struct {
	capability websearch.Capability
	response   websearch.Response
	err        error
}

func (f fakeWebSearch) Capability() websearch.Capability { return f.capability }
func (f fakeWebSearch) Search(context.Context, websearch.Query) (websearch.Response, error) {
	return f.response, f.err
}

type fakeRepository struct {
	rows      map[string]*researchdomain.Source
	schedules map[string]*researchdomain.Schedule
	runs      []researchdomain.ScheduleRun
}

func (f *fakeRepository) ScopeExists(context.Context, string, string, string) (bool, error) {
	return true, nil
}

func (f *fakeRepository) FindByHash(_ context.Context, tenantID, hash string) (*researchdomain.Source, error) {
	for _, row := range f.rows {
		if row.TenantID == tenantID && row.ContentHash == hash {
			return row, nil
		}
	}
	return nil, nil
}
func (f *fakeRepository) Create(_ context.Context, row *researchdomain.Source) error {
	f.rows[row.ID] = row
	return nil
}
func (f *fakeRepository) Get(_ context.Context, tenantID, id string) (*researchdomain.Source, error) {
	return f.rows[id], nil
}
func (f *fakeRepository) List(context.Context, string, researchdomain.Filter) ([]researchdomain.Source, error) {
	return nil, nil
}
func (f *fakeRepository) Decide(_ context.Context, _, id, status, comment, actorID string) (*researchdomain.Source, error) {
	row := f.rows[id]
	now := time.Now().UTC()
	row.Status, row.ReviewComment, row.ReviewedBy, row.ReviewedAt = status, comment, actorID, &now
	return row, nil
}
func (f *fakeRepository) SearchVerified(_ context.Context, tenantID, _, _ string, _ time.Time, _ int) ([]researchdomain.Source, error) {
	result := []researchdomain.Source{}
	for _, row := range f.rows {
		if row.TenantID == tenantID && row.Status == researchdomain.StatusVerified {
			result = append(result, *row)
		}
	}
	return result, nil
}

func (f *fakeRepository) CountEnabledSchedules(_ context.Context, tenantID string) (int64, error) {
	var count int64
	for _, row := range f.schedules {
		if row.TenantID == tenantID && row.Enabled {
			count++
		}
	}
	return count, nil
}

func (f *fakeRepository) CreateSchedule(_ context.Context, row *researchdomain.Schedule) error {
	if f.schedules == nil {
		f.schedules = map[string]*researchdomain.Schedule{}
	}
	f.schedules[row.ID] = row
	return nil
}

func (f *fakeRepository) GetSchedule(_ context.Context, tenantID, id string) (*researchdomain.Schedule, error) {
	row := f.schedules[id]
	if row == nil || row.TenantID != tenantID {
		return nil, fmt.Errorf("not found")
	}
	copy := *row
	return &copy, nil
}

func (f *fakeRepository) FindScheduleByKey(_ context.Context, tenantID, key string) (*researchdomain.Schedule, error) {
	for _, row := range f.schedules {
		if row.TenantID == tenantID && row.IdempotencyKey == key {
			copy := *row
			return &copy, nil
		}
	}
	return nil, nil
}

func (f *fakeRepository) ListSchedules(_ context.Context, tenantID string) ([]researchdomain.Schedule, error) {
	rows := []researchdomain.Schedule{}
	for _, row := range f.schedules {
		if row.TenantID == tenantID {
			rows = append(rows, *row)
		}
	}
	return rows, nil
}

func (f *fakeRepository) UpdateSchedule(_ context.Context, row *researchdomain.Schedule, _ time.Time) (bool, error) {
	copy := *row
	f.schedules[row.ID] = &copy
	return true, nil
}

func (f *fakeRepository) ListScheduleRuns(_ context.Context, tenantID, scheduleID string, _ int) ([]researchdomain.ScheduleRun, error) {
	rows := []researchdomain.ScheduleRun{}
	for _, row := range f.runs {
		if row.TenantID == tenantID && (scheduleID == "" || row.ScheduleID == scheduleID) {
			rows = append(rows, row)
		}
	}
	return rows, nil
}

func (f *fakeRepository) ListDueScheduleIDs(_ context.Context, now time.Time, _ int) ([]string, error) {
	ids := []string{}
	for id, row := range f.schedules {
		if row.Enabled && row.NextRunAt != nil && !row.NextRunAt.After(now) {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (f *fakeRepository) ClaimSchedule(_ context.Context, id string, now, lockedUntil time.Time, requestedBy, provider string) (*researchdomain.Schedule, *researchdomain.ScheduleRun, bool, error) {
	row := f.schedules[id]
	if row == nil || !row.Enabled || row.NextRunAt == nil || row.NextRunAt.After(now) {
		return nil, nil, false, nil
	}
	token := "claim-" + id
	row.LockToken, row.LockedUntil = token, &lockedUntil
	run := researchdomain.ScheduleRun{ID: "run-" + id, TenantID: row.TenantID, ScheduleID: id, ScheduledFor: *row.NextRunAt, Status: researchdomain.ScheduleRunProcessing, Provider: provider, QueryHash: row.QueryHash, RequestedBy: requestedBy, ClaimToken: token, StartedAt: now}
	f.runs = append(f.runs, run)
	return row, &f.runs[len(f.runs)-1], true, nil
}

func (f *fakeRepository) CompleteScheduleRun(_ context.Context, schedule *researchdomain.Schedule, run *researchdomain.ScheduleRun, nextRunAt, finishedAt time.Time) (bool, error) {
	for index := range f.runs {
		if f.runs[index].ID == run.ID {
			f.runs[index] = *run
			f.runs[index].FinishedAt = &finishedAt
		}
	}
	schedule.NextRunAt, schedule.LastRunAt, schedule.LastStatus = &nextRunAt, &finishedAt, run.Status
	schedule.LastErrorCode, schedule.LastErrorMessage = run.ErrorCode, run.ErrorMessage
	schedule.LockToken, schedule.LockedUntil = "", nil
	return true, nil
}

func TestSourceMustBeVerifiedBeforeAgentCanReadIt(t *testing.T) {
	repo := &fakeRepository{rows: map[string]*researchdomain.Source{}}
	service := New(repo)
	actor := Actor{TenantID: "tenant-1", UserID: "user-1"}
	row, err := service.Create(context.Background(), actor, CreateInput{Category: "market", Title: "Market update", Summary: "Verified market context", SourceURL: "https://Example.com/news#fragment", Publisher: "Example", PublishedAt: "2026-08-04"})
	require.NoError(t, err)
	require.Equal(t, researchdomain.StatusPending, row.Status)
	items, err := service.SearchVerified(context.Background(), "tenant-1", "game-1", "campaign-1", "2026-08-05")
	require.NoError(t, err)
	require.Empty(t, items)

	_, err = service.Decide(context.Background(), actor, row.ID, DecisionInput{Status: researchdomain.StatusVerified})
	require.NoError(t, err)
	items, err = service.SearchVerified(context.Background(), "tenant-1", "game-1", "campaign-1", "2026-08-05")
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "https://example.com/news", items[0].SourceURL)
}

func TestCreateRejectsUntrustedSourceURL(t *testing.T) {
	service := New(&fakeRepository{rows: map[string]*researchdomain.Source{}})
	_, err := service.Create(context.Background(), Actor{TenantID: "tenant-1", UserID: "user-1"}, CreateInput{Category: "POLICY", Title: "Policy", Summary: "Summary", SourceURL: "http://example.com", Publisher: "Example", PublishedAt: "2026-08-04"})
	require.Error(t, err)
}

func TestWebSearchReturnsEphemeralResultsWithQueryHash(t *testing.T) {
	repo := &fakeRepository{rows: map[string]*researchdomain.Source{}}
	provider := fakeWebSearch{capability: websearch.Capability{Configured: true, Provider: "brave", MaxResults: 8}, response: websearch.Response{Provider: "brave", SearchedAt: time.Now().UTC(), Results: []websearch.Result{{Title: "Fresh report", URL: "https://example.com/report", Description: "Current signal", Publisher: "Example"}}}}
	service := NewWithWebSearch(repo, provider)
	result, err := service.SearchWeb(context.Background(), Actor{TenantID: "tenant-1", UserID: "user-1"}, WebSearchInput{Query: "game market signal", Category: "MARKET", Count: 5, Country: "US", SearchLang: "en", Freshness: "pw"})
	require.NoError(t, err)
	require.Len(t, result.Results, 1)
	require.Len(t, result.QueryHash, 64)
	require.Empty(t, repo.rows, "search results must remain ephemeral before explicit import")
}

func TestWebSearchRejectsOverlongQuery(t *testing.T) {
	service := NewWithWebSearch(&fakeRepository{rows: map[string]*researchdomain.Source{}}, fakeWebSearch{capability: websearch.Capability{Configured: true, Provider: "brave", MaxResults: 8}})
	_, err := service.SearchWeb(context.Background(), Actor{TenantID: "tenant-1", UserID: "user-1"}, WebSearchInput{Query: string(make([]byte, 401)), Category: "MARKET", Count: 5})
	require.Error(t, err)
}

func TestImportWebResultCreatesPendingSourceWithProvenance(t *testing.T) {
	repo := &fakeRepository{rows: map[string]*researchdomain.Source{}}
	provider := fakeWebSearch{capability: websearch.Capability{Configured: true, Provider: "brave", ImportEnabled: true, MaxResults: 8}}
	service := NewWithWebSearch(repo, provider)
	row, err := service.ImportWebResult(context.Background(), Actor{TenantID: "tenant-1", UserID: "user-1"}, WebImportInput{Query: "game market", Category: "MARKET", Result: websearch.Result{Title: "Fresh report", URL: "https://example.com/report", Description: "Current signal", Publisher: "Example", PublishedAt: "2026-08-09T02:00:00Z"}})
	require.NoError(t, err)
	require.Equal(t, researchdomain.StatusPending, row.Status)
	require.Equal(t, "WEB_SEARCH", row.DiscoveryMethod)
	require.Equal(t, "brave", row.DiscoveryProvider)
	require.NotNil(t, row.DiscoveredAt)
	require.Len(t, row.DiscoveryQueryHash, 64)
}

func TestImportWebResultRequiresStoragePermission(t *testing.T) {
	service := NewWithWebSearch(&fakeRepository{rows: map[string]*researchdomain.Source{}}, fakeWebSearch{capability: websearch.Capability{Configured: true, Provider: "brave", ImportEnabled: false, MaxResults: 8}})
	_, err := service.ImportWebResult(context.Background(), Actor{TenantID: "tenant-1", UserID: "user-1"}, WebImportInput{Query: "game market", Category: "MARKET", Result: websearch.Result{Title: "Fresh report", URL: "https://example.com/report", Description: "Current signal", Publisher: "Example"}})
	require.ErrorContains(t, err, "存储权")
}

func TestCreateResearchScheduleRequiresImportPermission(t *testing.T) {
	repo := &fakeRepository{rows: map[string]*researchdomain.Source{}, schedules: map[string]*researchdomain.Schedule{}}
	provider := fakeWebSearch{capability: websearch.Capability{Configured: true, Provider: "brave", ImportEnabled: false, MaxResults: 8}}
	service := NewWithWebSearch(repo, provider)
	_, err := service.CreateSchedule(context.Background(), Actor{TenantID: "tenant-1", UserID: "user-1"}, ScheduleInput{Name: "Weekly market", Category: "MARKET", Query: "mobile game market", ResultCount: 5, IntervalMinutes: 1440, Enabled: true})
	require.ErrorContains(t, err, "自动发现")
}

func TestDuplicateResearchScheduleSubmissionReturnsExistingTask(t *testing.T) {
	repo := &fakeRepository{rows: map[string]*researchdomain.Source{}, schedules: map[string]*researchdomain.Schedule{}}
	provider := fakeWebSearch{capability: websearch.Capability{Configured: true, Provider: "brave", ImportEnabled: true, MaxResults: 8}}
	service := NewWithWebSearch(repo, provider)
	input := ScheduleInput{Name: "Weekly market", Category: "MARKET", Query: "mobile game market", ResultCount: 5, IntervalMinutes: 1440, Enabled: true}
	first, err := service.CreateSchedule(context.Background(), Actor{TenantID: "tenant-1", UserID: "user-1"}, input)
	require.NoError(t, err)
	second, err := service.CreateSchedule(context.Background(), Actor{TenantID: "tenant-1", UserID: "user-1"}, input)
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
	require.Len(t, repo.schedules, 1)
}

func TestScheduledResearchCreatesPendingSourcesAndRunProvenance(t *testing.T) {
	repo := &fakeRepository{rows: map[string]*researchdomain.Source{}, schedules: map[string]*researchdomain.Schedule{}}
	provider := fakeWebSearch{
		capability: websearch.Capability{Configured: true, Provider: "brave", ImportEnabled: true, MaxResults: 8},
		response: websearch.Response{Provider: "brave", Results: []websearch.Result{
			{Title: "Market report", URL: "https://example.com/report", Description: "A current market signal", Publisher: "Example", PublishedAt: "2026-08-09"},
			{Title: "Incomplete", URL: "https://example.com/incomplete", Publisher: "Example"},
		}},
	}
	service := NewWithWebSearch(repo, provider)
	now := time.Date(2026, 8, 10, 4, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	schedule, err := service.CreateSchedule(context.Background(), Actor{TenantID: "tenant-1", UserID: "user-1"}, ScheduleInput{Name: "Daily market", Category: "MARKET", Query: "mobile game market", ResultCount: 5, IntervalMinutes: 60, Enabled: true})
	require.NoError(t, err)
	due := now.Add(-time.Minute)
	schedule.NextRunAt = &due

	processed, err := service.RunDueSchedules(context.Background(), 20, 2*time.Minute)
	require.NoError(t, err)
	require.Equal(t, 1, processed)
	require.Len(t, repo.rows, 1)
	for _, source := range repo.rows {
		require.Equal(t, researchdomain.StatusPending, source.Status)
		require.Equal(t, "SCHEDULED_WEB_SEARCH", source.DiscoveryMethod)
		require.Equal(t, schedule.ID, source.DiscoveryScheduleID)
		require.Equal(t, "brave", source.DiscoveryProvider)
	}
	require.Len(t, repo.runs, 1)
	require.Equal(t, researchdomain.ScheduleRunSucceeded, repo.runs[0].Status)
	require.Equal(t, 2, repo.runs[0].ResultCount)
	require.Equal(t, 1, repo.runs[0].ImportedCount)
	require.Equal(t, 1, repo.runs[0].SkippedCount)
	require.Equal(t, researchdomain.ScheduleRunSucceeded, schedule.LastStatus)
	require.True(t, schedule.NextRunAt.After(now))
}

func TestScheduledResearchRecordsSafeProviderFailureAndAdvances(t *testing.T) {
	repo := &fakeRepository{rows: map[string]*researchdomain.Source{}, schedules: map[string]*researchdomain.Schedule{}}
	provider := fakeWebSearch{capability: websearch.Capability{Configured: true, Provider: "brave", ImportEnabled: true, MaxResults: 8}, err: &websearch.Error{Code: "RATE_LIMITED", Message: "provider rate limited", Retryable: true}}
	service := NewWithWebSearch(repo, provider)
	now := time.Date(2026, 8, 10, 4, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	schedule, err := service.CreateSchedule(context.Background(), Actor{TenantID: "tenant-1", UserID: "user-1"}, ScheduleInput{Name: "Daily market", Category: "MARKET", Query: "mobile game market", ResultCount: 5, IntervalMinutes: 60, Enabled: true})
	require.NoError(t, err)
	due := now.Add(-time.Minute)
	schedule.NextRunAt = &due

	processed, err := service.RunDueSchedules(context.Background(), 20, 2*time.Minute)
	require.Equal(t, 1, processed)
	require.Error(t, err)
	require.Len(t, repo.runs, 1)
	require.Equal(t, researchdomain.ScheduleRunFailed, repo.runs[0].Status)
	require.Equal(t, "RATE_LIMITED", repo.runs[0].ErrorCode)
	require.Equal(t, researchdomain.ScheduleRunFailed, schedule.LastStatus)
	require.True(t, schedule.NextRunAt.After(now))
}
