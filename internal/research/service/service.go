package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"time"

	auditdomain "github.com/example/adnova/internal/audit/domain"
	auditservice "github.com/example/adnova/internal/audit/service"
	"github.com/example/adnova/internal/common/apperror"
	researchdomain "github.com/example/adnova/internal/research/domain"
	"github.com/google/uuid"
)

type Repository interface {
	ScopeExists(context.Context, string, string, string) (bool, error)
	FindByHash(context.Context, string, string) (*researchdomain.Source, error)
	Create(context.Context, *researchdomain.Source) error
	Get(context.Context, string, string) (*researchdomain.Source, error)
	List(context.Context, string, researchdomain.Filter) ([]researchdomain.Source, error)
	Decide(context.Context, string, string, string, string, string) (*researchdomain.Source, error)
	SearchVerified(context.Context, string, string, string, time.Time, int) ([]researchdomain.Source, error)
}

type CreateInput struct {
	GameID      string `json:"game_id"`
	CampaignID  string `json:"campaign_id"`
	Category    string `json:"category"`
	Title       string `json:"title"`
	Summary     string `json:"summary"`
	SourceURL   string `json:"source_url"`
	Publisher   string `json:"publisher"`
	PublishedAt string `json:"published_at"`
}

type DecisionInput struct {
	Status  string `json:"status"`
	Comment string `json:"comment"`
}

type Actor struct {
	TenantID  string
	UserID    string
	TraceID   string
	RequestID string
}

type Service struct {
	repo    Repository
	auditor auditservice.Recorder
}

func New(repo Repository, auditors ...auditservice.Recorder) *Service {
	result := &Service{repo: repo}
	if len(auditors) > 0 {
		result.auditor = auditors[0]
	}
	return result
}

func (s *Service) Create(ctx context.Context, actor Actor, input CreateInput) (*researchdomain.Source, error) {
	input.Category = strings.ToUpper(strings.TrimSpace(input.Category))
	input.Title, input.Summary = strings.TrimSpace(input.Title), strings.TrimSpace(input.Summary)
	input.Publisher = strings.TrimSpace(input.Publisher)
	if !validCategory(input.Category) || input.Title == "" || input.Summary == "" || input.Publisher == "" {
		return nil, apperror.Validation("category, title, summary and publisher are required")
	}
	if input.CampaignID != "" && input.GameID == "" {
		return nil, apperror.Validation("game_id is required when campaign_id is provided")
	}
	if input.GameID != "" {
		exists, scopeErr := s.repo.ScopeExists(ctx, actor.TenantID, input.GameID, input.CampaignID)
		if scopeErr != nil {
			return nil, scopeErr
		}
		if !exists {
			return nil, apperror.Validation("research source scope does not belong to the current tenant")
		}
	}
	normalizedURL, err := normalizeSourceURL(input.SourceURL)
	if err != nil {
		return nil, apperror.Validation(err.Error())
	}
	publishedAt, err := time.Parse("2006-01-02", input.PublishedAt)
	if err != nil {
		return nil, apperror.Validation("published_at must be YYYY-MM-DD")
	}
	if publishedAt.After(time.Now().UTC().AddDate(0, 0, 1)) {
		return nil, apperror.Validation("published_at cannot be in the future")
	}
	digest := sha256.Sum256([]byte(strings.Join([]string{normalizedURL, input.Title, input.Summary}, "\n")))
	hash := hex.EncodeToString(digest[:])
	existing, err := s.repo.FindByHash(ctx, actor.TenantID, hash)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}
	row := &researchdomain.Source{ID: uuid.NewString(), TenantID: actor.TenantID, GameID: input.GameID, CampaignID: input.CampaignID, Category: input.Category, Title: input.Title, Summary: input.Summary, SourceURL: normalizedURL, Publisher: input.Publisher, PublishedAt: publishedAt, ContentHash: hash, Status: researchdomain.StatusPending, CreatedBy: actor.UserID}
	if err := s.repo.Create(ctx, row); err != nil {
		return nil, err
	}
	if err := s.record(ctx, actor, "RESEARCH_SOURCE_REGISTERED", row); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *Service) Decide(ctx context.Context, actor Actor, id string, input DecisionInput) (*researchdomain.Source, error) {
	input.Status = strings.ToUpper(strings.TrimSpace(input.Status))
	input.Comment = strings.TrimSpace(input.Comment)
	if input.Status != researchdomain.StatusVerified && input.Status != researchdomain.StatusRejected {
		return nil, apperror.Validation("status must be VERIFIED or REJECTED")
	}
	if input.Status == researchdomain.StatusRejected && input.Comment == "" {
		return nil, apperror.Validation("rejection comment is required")
	}
	row, err := s.repo.Decide(ctx, actor.TenantID, id, input.Status, input.Comment, actor.UserID)
	if err != nil {
		return nil, err
	}
	if err := s.record(ctx, actor, "RESEARCH_SOURCE_"+input.Status, row); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *Service) List(ctx context.Context, tenantID string, filter researchdomain.Filter) ([]researchdomain.Source, error) {
	return s.repo.List(ctx, tenantID, filter)
}

func (s *Service) SearchVerified(ctx context.Context, tenantID, gameID, campaignID, analysisDate string) ([]researchdomain.Evidence, error) {
	through, err := time.Parse("2006-01-02", analysisDate)
	if err != nil {
		return nil, fmt.Errorf("analysis_date must be YYYY-MM-DD")
	}
	rows, err := s.repo.SearchVerified(ctx, tenantID, gameID, campaignID, through.Add(24*time.Hour-time.Nanosecond), 20)
	if err != nil {
		return nil, err
	}
	result := make([]researchdomain.Evidence, 0, len(rows))
	for _, row := range rows {
		if row.ReviewedAt == nil {
			continue
		}
		result = append(result, researchdomain.Evidence{ID: row.ID, Category: row.Category, Title: row.Title, Summary: row.Summary, SourceURL: row.SourceURL, Publisher: row.Publisher, PublishedAt: row.PublishedAt, VerifiedAt: *row.ReviewedAt})
	}
	return result, nil
}

func validCategory(value string) bool {
	return value == researchdomain.CategoryPolicy || value == researchdomain.CategoryCompetitor || value == researchdomain.CategoryMarket
}

func normalizeSourceURL(value string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return "", fmt.Errorf("source_url must be an absolute HTTPS URL without credentials")
	}
	parsed.Fragment = ""
	parsed.Host = strings.ToLower(parsed.Host)
	return parsed.String(), nil
}

func (s *Service) record(ctx context.Context, actor Actor, action string, row *researchdomain.Source) error {
	if s.auditor == nil {
		return nil
	}
	return s.auditor.Record(ctx, auditdomain.RecordInput{TenantID: actor.TenantID, ActorID: actor.UserID, ActorType: "USER", Action: action, ResourceType: "RESEARCH_SOURCE", ResourceID: row.ID, After: row, RequestID: actor.RequestID, TraceID: actor.TraceID})
}
