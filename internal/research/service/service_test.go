package service

import (
	"context"
	"testing"
	"time"

	researchdomain "github.com/example/adnova/internal/research/domain"
	"github.com/stretchr/testify/require"
)

type fakeRepository struct {
	rows map[string]*researchdomain.Source
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
