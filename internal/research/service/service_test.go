package service

import (
	"context"
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
