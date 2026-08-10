package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	auditdomain "github.com/example/adnova/internal/audit/domain"
	auditservice "github.com/example/adnova/internal/audit/service"
	"github.com/example/adnova/internal/common/apperror"
	"github.com/example/adnova/internal/common/identity"
	researchdomain "github.com/example/adnova/internal/research/domain"
	"github.com/example/adnova/internal/research/websearch"
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

type ScheduleRepository interface {
	CountEnabledSchedules(context.Context, string) (int64, error)
	CreateSchedule(context.Context, *researchdomain.Schedule) error
	GetSchedule(context.Context, string, string) (*researchdomain.Schedule, error)
	FindScheduleByKey(context.Context, string, string) (*researchdomain.Schedule, error)
	ListSchedules(context.Context, string) ([]researchdomain.Schedule, error)
	UpdateSchedule(context.Context, *researchdomain.Schedule, time.Time) (bool, error)
	ListScheduleRuns(context.Context, string, string, int) ([]researchdomain.ScheduleRun, error)
	ListDueScheduleIDs(context.Context, time.Time, int) ([]string, error)
	ClaimSchedule(context.Context, string, time.Time, time.Time, string, string) (*researchdomain.Schedule, *researchdomain.ScheduleRun, bool, error)
	CompleteScheduleRun(context.Context, *researchdomain.Schedule, *researchdomain.ScheduleRun, time.Time, time.Time) (bool, error)
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

type WebSearchInput struct {
	Query      string `json:"query"`
	GameID     string `json:"game_id"`
	CampaignID string `json:"campaign_id"`
	Category   string `json:"category"`
	Count      int    `json:"count"`
	Country    string `json:"country"`
	SearchLang string `json:"search_lang"`
	Freshness  string `json:"freshness"`
}

type WebSearchResponse struct {
	Capability websearch.Capability `json:"capability"`
	QueryHash  string               `json:"query_hash"`
	Results    []websearch.Result   `json:"results"`
	SearchedAt time.Time            `json:"searched_at"`
}

type WebImportInput struct {
	Query      string           `json:"query"`
	GameID     string           `json:"game_id"`
	CampaignID string           `json:"campaign_id"`
	Category   string           `json:"category"`
	Result     websearch.Result `json:"result"`
}

type discovery struct {
	method, provider, queryHash, scheduleID string
	discoveredAt                            *time.Time
}

type Service struct {
	repo         Repository
	scheduleRepo ScheduleRepository
	auditor      auditservice.Recorder
	web          websearch.Provider
	now          func() time.Time
}

func New(repo Repository, auditors ...auditservice.Recorder) *Service {
	result := &Service{repo: repo, now: func() time.Time { return time.Now().UTC() }}
	if scheduleRepo, ok := repo.(ScheduleRepository); ok {
		result.scheduleRepo = scheduleRepo
	}
	if len(auditors) > 0 {
		result.auditor = auditors[0]
	}
	return result
}

func NewWithWebSearch(repo Repository, provider websearch.Provider, auditors ...auditservice.Recorder) *Service {
	result := New(repo, auditors...)
	result.web = provider
	return result
}

func (s *Service) Create(ctx context.Context, actor Actor, input CreateInput) (*researchdomain.Source, error) {
	return s.create(ctx, actor, input, discovery{method: "MANUAL"})
}

func (s *Service) create(ctx context.Context, actor Actor, input CreateInput, origin discovery) (*researchdomain.Source, error) {
	row, _, err := s.createWithStatus(ctx, actor, input, origin)
	return row, err
}

func (s *Service) createWithStatus(ctx context.Context, actor Actor, input CreateInput, origin discovery) (*researchdomain.Source, bool, error) {
	input.Category = strings.ToUpper(strings.TrimSpace(input.Category))
	input.Title, input.Summary = strings.TrimSpace(input.Title), strings.TrimSpace(input.Summary)
	input.Publisher = strings.TrimSpace(input.Publisher)
	if !validCategory(input.Category) || input.Title == "" || input.Summary == "" || input.Publisher == "" {
		return nil, false, apperror.Validation("category, title, summary and publisher are required")
	}
	if input.CampaignID != "" && input.GameID == "" {
		return nil, false, apperror.Validation("game_id is required when campaign_id is provided")
	}
	if input.GameID != "" {
		exists, scopeErr := s.repo.ScopeExists(ctx, actor.TenantID, input.GameID, input.CampaignID)
		if scopeErr != nil {
			return nil, false, scopeErr
		}
		if !exists {
			return nil, false, apperror.Validation("research source scope does not belong to the current tenant")
		}
	}
	normalizedURL, err := normalizeSourceURL(input.SourceURL)
	if err != nil {
		return nil, false, apperror.Validation(err.Error())
	}
	publishedAt, err := time.Parse("2006-01-02", input.PublishedAt)
	if err != nil {
		return nil, false, apperror.Validation("published_at must be YYYY-MM-DD")
	}
	if publishedAt.After(time.Now().UTC().AddDate(0, 0, 1)) {
		return nil, false, apperror.Validation("published_at cannot be in the future")
	}
	digest := sha256.Sum256([]byte(strings.Join([]string{normalizedURL, input.Title, input.Summary}, "\n")))
	hash := hex.EncodeToString(digest[:])
	existing, err := s.repo.FindByHash(ctx, actor.TenantID, hash)
	if err != nil {
		return nil, false, err
	}
	if existing != nil {
		return existing, false, nil
	}
	if origin.method == "" {
		origin.method = "MANUAL"
	}
	row := &researchdomain.Source{ID: uuid.NewString(), TenantID: actor.TenantID, GameID: input.GameID, CampaignID: input.CampaignID, Category: input.Category, Title: input.Title, Summary: input.Summary, SourceURL: normalizedURL, DiscoveryMethod: origin.method, DiscoveryProvider: origin.provider, DiscoveryQueryHash: origin.queryHash, DiscoveryScheduleID: origin.scheduleID, DiscoveredAt: origin.discoveredAt, Publisher: input.Publisher, PublishedAt: publishedAt, ContentHash: hash, Status: researchdomain.StatusPending, CreatedBy: actor.UserID}
	if err := s.repo.Create(ctx, row); err != nil {
		return nil, false, err
	}
	if err := s.record(ctx, actor, "RESEARCH_SOURCE_REGISTERED", row); err != nil {
		return nil, false, err
	}
	return row, true, nil
}

func (s *Service) WebCapability() websearch.Capability {
	if s.web == nil {
		return websearch.Capability{Provider: "disabled", Details: []string{"实时联网连接器未配置"}}
	}
	return s.web.Capability()
}

func (s *Service) SearchWeb(ctx context.Context, actor Actor, input WebSearchInput) (*WebSearchResponse, error) {
	if s.web == nil || !s.web.Capability().Configured {
		return nil, apperror.New(10020, 503, "实时联网连接器尚未配置")
	}
	input.Query = strings.TrimSpace(input.Query)
	input.Category = strings.ToUpper(strings.TrimSpace(input.Category))
	input.Country = strings.ToUpper(strings.TrimSpace(input.Country))
	input.SearchLang = strings.ToLower(strings.TrimSpace(input.SearchLang))
	input.Freshness = strings.ToLower(strings.TrimSpace(input.Freshness))
	if input.Query == "" || utf8.RuneCountInString(input.Query) > 400 || len(strings.Fields(input.Query)) > 50 {
		return nil, apperror.Validation("query is required and must be at most 400 characters and 50 words")
	}
	if !validCategory(input.Category) {
		return nil, apperror.Validation("category must be POLICY, COMPETITOR or MARKET")
	}
	if input.Count < 1 || input.Count > s.web.Capability().MaxResults {
		return nil, apperror.Validation(fmt.Sprintf("count must be between 1 and %d", s.web.Capability().MaxResults))
	}
	if !validCountry(input.Country) || !validSearchLang(input.SearchLang) || !validFreshness(input.Freshness) {
		return nil, apperror.Validation("country, search_lang or freshness is invalid")
	}
	if err := s.validateScope(ctx, actor.TenantID, input.GameID, input.CampaignID); err != nil {
		return nil, err
	}
	response, err := s.web.Search(ctx, websearch.Query{Text: input.Query, Count: input.Count, Country: input.Country, SearchLang: input.SearchLang, Freshness: input.Freshness})
	if err != nil {
		return nil, webError(err)
	}
	queryHash := hashText(input.Query)
	if s.auditor != nil {
		auditID := uuid.NewString()
		if err := s.auditor.Record(ctx, auditdomain.RecordInput{TenantID: actor.TenantID, ActorID: actor.UserID, ActorType: actorType(actor.UserID), Action: "RESEARCH_WEB_SEARCHED", ResourceType: "RESEARCH_WEB_QUERY", ResourceID: auditID, After: map[string]any{"query_hash": queryHash, "provider": response.Provider, "result_count": len(response.Results), "game_id": input.GameID, "campaign_id": input.CampaignID}, RequestID: actor.RequestID, TraceID: actor.TraceID}); err != nil {
			return nil, err
		}
	}
	return &WebSearchResponse{Capability: s.web.Capability(), QueryHash: queryHash, Results: response.Results, SearchedAt: response.SearchedAt}, nil
}

func (s *Service) ImportWebResult(ctx context.Context, actor Actor, input WebImportInput) (*researchdomain.Source, error) {
	capability := s.WebCapability()
	if !capability.Configured {
		return nil, apperror.New(10020, 503, "实时联网连接器尚未配置")
	}
	if !capability.ImportEnabled {
		return nil, apperror.Validation("搜索结果入库未启用；请先确认供应商计划包含结果存储权")
	}
	input.Query = strings.TrimSpace(input.Query)
	input.Result.Title = strings.TrimSpace(input.Result.Title)
	input.Result.Description = strings.TrimSpace(input.Result.Description)
	input.Result.Publisher = strings.TrimSpace(input.Result.Publisher)
	if input.Query == "" || input.Result.Title == "" || input.Result.Description == "" || input.Result.Publisher == "" {
		return nil, apperror.Validation("query and a complete web result are required")
	}
	publishedAt := webPublishedDate(input.Result.PublishedAt)
	now := time.Now().UTC()
	return s.create(ctx, actor, CreateInput{GameID: input.GameID, CampaignID: input.CampaignID, Category: input.Category, Title: input.Result.Title, Summary: input.Result.Description, SourceURL: input.Result.URL, Publisher: input.Result.Publisher, PublishedAt: publishedAt}, discovery{method: "WEB_SEARCH", provider: capability.Provider, queryHash: hashText(input.Query), discoveredAt: &now})
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

func validCountry(value string) bool {
	if value == "" {
		return true
	}
	return len(value) == 2 && value[0] >= 'A' && value[0] <= 'Z' && value[1] >= 'A' && value[1] <= 'Z'
}

func validSearchLang(value string) bool {
	if value == "" {
		return true
	}
	if len(value) < 2 || len(value) > 12 {
		return false
	}
	for _, char := range value {
		if (char < 'a' || char > 'z') && char != '-' {
			return false
		}
	}
	return true
}

func validFreshness(value string) bool {
	return value == "" || value == "pd" || value == "pw" || value == "pm" || value == "py"
}

func (s *Service) validateScope(ctx context.Context, tenantID, gameID, campaignID string) error {
	if campaignID != "" && gameID == "" {
		return apperror.Validation("game_id is required when campaign_id is provided")
	}
	if gameID == "" {
		return nil
	}
	exists, err := s.repo.ScopeExists(ctx, tenantID, gameID, campaignID)
	if err != nil {
		return err
	}
	if !exists {
		return apperror.Validation("research source scope does not belong to the current tenant")
	}
	return nil
}

func hashText(value string) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(digest[:])
}

func webPublishedDate(value string) string {
	value = strings.TrimSpace(value)
	for _, layout := range []string{time.RFC3339, "2006-01-02", "January 2, 2006", "Jan 2, 2006"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC().Format("2006-01-02")
		}
	}
	return time.Now().UTC().Format("2006-01-02")
}

func webError(err error) error {
	var providerErr *websearch.Error
	if !errors.As(err, &providerErr) {
		return err
	}
	switch providerErr.Code {
	case "RATE_LIMITED":
		return apperror.New(10021, 429, providerErr.Message)
	case "UPSTREAM_UNAVAILABLE":
		return apperror.New(10022, 503, providerErr.Message)
	case "UNAUTHORIZED", "NOT_CONFIGURED":
		return apperror.New(10020, 503, providerErr.Message)
	default:
		return apperror.New(10023, 502, providerErr.Message)
	}
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
	return s.auditor.Record(ctx, auditdomain.RecordInput{TenantID: actor.TenantID, ActorID: actor.UserID, ActorType: actorType(actor.UserID), Action: action, ResourceType: "RESEARCH_SOURCE", ResourceID: row.ID, After: row, RequestID: actor.RequestID, TraceID: actor.TraceID})
}

func actorType(userID string) string {
	if userID == identity.SystemAgentUserID {
		return "SYSTEM_AGENT"
	}
	return "USER"
}
