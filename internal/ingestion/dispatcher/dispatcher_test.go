package dispatcher

import (
	"context"
	"errors"
	"testing"
	"time"

	analysisservice "github.com/example/adnova/internal/analysis/service"
	"github.com/example/adnova/internal/ingestion/domain"
	"github.com/stretchr/testify/require"
)

type fakeWindowRepository struct {
	windows        []domain.AnalysisWindow
	claimToken     string
	claimLease     time.Duration
	completedToken string
	retriedToken   string
	retryAttempts  int
}

func (r *fakeWindowRepository) ListReadyAnalysisWindows(context.Context, int) ([]domain.AnalysisWindow, error) {
	return r.windows, nil
}

func (r *fakeWindowRepository) ClaimAnalysisWindow(_ context.Context, _ string, _ int, lease time.Duration) (string, bool, error) {
	r.claimLease = lease
	return r.claimToken, true, nil
}

func (r *fakeWindowRepository) CompleteAnalysisWindow(_ context.Context, _ string, _ int, token string) error {
	r.completedToken = token
	return nil
}

func (r *fakeWindowRepository) RetryAnalysisWindow(_ context.Context, _ string, _ int, token string, attempts int, _ time.Time, _ string) error {
	r.retriedToken = token
	r.retryAttempts = attempts
	return nil
}

type fakePipeline struct{ err error }

func (p fakePipeline) Run(context.Context, string, string) (*analysisservice.Result, error) {
	return &analysisservice.Result{}, p.err
}

func TestDispatcherCompletesWithClaimToken(t *testing.T) {
	repo := &fakeWindowRepository{
		windows:    []domain.AnalysisWindow{{ID: "window-1", TenantID: "tenant-1", GameID: "game-1", Version: 3}},
		claimToken: "claim-1",
	}
	lease := 15 * time.Minute

	err := New(repo, fakePipeline{}, lease).RunOnce(context.Background(), 20)

	require.NoError(t, err)
	require.Equal(t, lease, repo.claimLease)
	require.Equal(t, "claim-1", repo.completedToken)
	require.Empty(t, repo.retriedToken)
}

func TestDispatcherRetriesWithClaimToken(t *testing.T) {
	repo := &fakeWindowRepository{
		windows:    []domain.AnalysisWindow{{ID: "window-1", TenantID: "tenant-1", GameID: "game-1", Version: 3, AttemptCount: 2}},
		claimToken: "claim-2",
	}

	err := New(repo, fakePipeline{err: errors.New("analysis failed")}, 15*time.Minute).RunOnce(context.Background(), 20)

	require.NoError(t, err)
	require.Equal(t, "claim-2", repo.retriedToken)
	require.Equal(t, 3, repo.retryAttempts)
	require.Empty(t, repo.completedToken)
}
