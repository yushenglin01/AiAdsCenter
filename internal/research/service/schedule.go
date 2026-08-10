package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	auditdomain "github.com/example/adnova/internal/audit/domain"
	"github.com/example/adnova/internal/common/apperror"
	"github.com/example/adnova/internal/common/identity"
	researchdomain "github.com/example/adnova/internal/research/domain"
	"github.com/example/adnova/internal/research/websearch"
	"github.com/google/uuid"
)

const (
	minimumScheduleInterval = 60
	maximumScheduleInterval = 7 * 24 * 60
	maximumEnabledSchedules = 20
)

type ScheduleInput struct {
	Name            string `json:"name"`
	GameID          string `json:"game_id"`
	CampaignID      string `json:"campaign_id"`
	Category        string `json:"category"`
	Query           string `json:"query"`
	Country         string `json:"country"`
	SearchLang      string `json:"search_lang"`
	Freshness       string `json:"freshness"`
	ResultCount     int    `json:"result_count"`
	IntervalMinutes int    `json:"interval_minutes"`
	Enabled         bool   `json:"enabled"`
}

func (s *Service) CreateSchedule(ctx context.Context, actor Actor, input ScheduleInput) (*researchdomain.Schedule, error) {
	if s.scheduleRepo == nil {
		return nil, errors.New("research schedule repository is not configured")
	}
	input, err := s.normalizeScheduleInput(ctx, actor.TenantID, input)
	if err != nil {
		return nil, err
	}
	idempotencyKey := scheduleIdempotencyKey(input)
	existing, err := s.scheduleRepo.FindScheduleByKey(ctx, actor.TenantID, idempotencyKey)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}
	if input.Enabled {
		if err := s.validateScheduleCapability(); err != nil {
			return nil, err
		}
		if err := s.validateScheduleQuota(ctx, actor.TenantID); err != nil {
			return nil, err
		}
	}
	now := s.now().UTC().Truncate(time.Millisecond)
	row := &researchdomain.Schedule{
		ID: uuid.NewString(), TenantID: actor.TenantID, Name: input.Name,
		GameID: input.GameID, CampaignID: input.CampaignID, Category: input.Category,
		Query: input.Query, QueryHash: hashText(input.Query), IdempotencyKey: idempotencyKey, Country: input.Country,
		SearchLang: input.SearchLang, Freshness: input.Freshness, ResultCount: input.ResultCount,
		IntervalMinutes: input.IntervalMinutes, Enabled: input.Enabled,
		CreatedBy: actor.UserID, UpdatedBy: actor.UserID,
	}
	if row.Enabled {
		next := now.Add(time.Duration(row.IntervalMinutes) * time.Minute)
		row.NextRunAt = &next
	}
	if err := s.scheduleRepo.CreateSchedule(ctx, row); err != nil {
		if existing, lookupErr := s.scheduleRepo.FindScheduleByKey(ctx, actor.TenantID, idempotencyKey); lookupErr == nil && existing != nil {
			return existing, nil
		}
		return nil, err
	}
	if err := s.recordSchedule(ctx, actor, "RESEARCH_SCHEDULE_CREATED", nil, row); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *Service) UpdateSchedule(ctx context.Context, actor Actor, id string, input ScheduleInput) (*researchdomain.Schedule, error) {
	if s.scheduleRepo == nil {
		return nil, errors.New("research schedule repository is not configured")
	}
	before, err := s.scheduleRepo.GetSchedule(ctx, actor.TenantID, id)
	if err != nil {
		return nil, err
	}
	input, err = s.normalizeScheduleInput(ctx, actor.TenantID, input)
	if err != nil {
		return nil, err
	}
	if input.Enabled {
		if err := s.validateScheduleCapability(); err != nil {
			return nil, err
		}
		if !before.Enabled {
			if err := s.validateScheduleQuota(ctx, actor.TenantID); err != nil {
				return nil, err
			}
		}
	}
	now := s.now().UTC().Truncate(time.Millisecond)
	row := *before
	executionChanged := before.Query != input.Query || before.GameID != input.GameID || before.CampaignID != input.CampaignID || before.Category != input.Category || before.Country != input.Country || before.SearchLang != input.SearchLang || before.Freshness != input.Freshness || before.ResultCount != input.ResultCount || before.IntervalMinutes != input.IntervalMinutes
	row.Name, row.GameID, row.CampaignID, row.Category = input.Name, input.GameID, input.CampaignID, input.Category
	row.Query, row.QueryHash, row.Country, row.SearchLang = input.Query, hashText(input.Query), input.Country, input.SearchLang
	row.IdempotencyKey = scheduleIdempotencyKey(input)
	row.Freshness, row.ResultCount, row.IntervalMinutes, row.Enabled = input.Freshness, input.ResultCount, input.IntervalMinutes, input.Enabled
	row.UpdatedBy = actor.UserID
	if existing, lookupErr := s.scheduleRepo.FindScheduleByKey(ctx, actor.TenantID, row.IdempotencyKey); lookupErr != nil {
		return nil, lookupErr
	} else if existing != nil && existing.ID != row.ID {
		return nil, apperror.Conflict
	}
	switch {
	case !row.Enabled:
		row.NextRunAt = nil
	case !before.Enabled || executionChanged || row.NextRunAt == nil:
		next := now.Add(time.Duration(row.IntervalMinutes) * time.Minute)
		row.NextRunAt = &next
	}
	updated, err := s.scheduleRepo.UpdateSchedule(ctx, &row, now)
	if err != nil {
		return nil, err
	}
	if !updated {
		return nil, apperror.Conflict
	}
	saved, err := s.scheduleRepo.GetSchedule(ctx, actor.TenantID, id)
	if err != nil {
		return nil, err
	}
	if err := s.recordSchedule(ctx, actor, "RESEARCH_SCHEDULE_UPDATED", before, saved); err != nil {
		return nil, err
	}
	return saved, nil
}

func (s *Service) ListSchedules(ctx context.Context, tenantID string) ([]researchdomain.Schedule, error) {
	if s.scheduleRepo == nil {
		return nil, errors.New("research schedule repository is not configured")
	}
	return s.scheduleRepo.ListSchedules(ctx, tenantID)
}

func (s *Service) ListScheduleRuns(ctx context.Context, tenantID, scheduleID string, limit int) ([]researchdomain.ScheduleRun, error) {
	if s.scheduleRepo == nil {
		return nil, errors.New("research schedule repository is not configured")
	}
	if scheduleID != "" {
		if _, err := s.scheduleRepo.GetSchedule(ctx, tenantID, scheduleID); err != nil {
			return nil, err
		}
	}
	return s.scheduleRepo.ListScheduleRuns(ctx, tenantID, scheduleID, limit)
}

func (s *Service) RunDueSchedules(ctx context.Context, limit int, lease time.Duration) (int, error) {
	if s.scheduleRepo == nil {
		return 0, errors.New("research schedule repository is not configured")
	}
	if lease < time.Minute {
		return 0, fmt.Errorf("research schedule lease must be at least one minute")
	}
	now := s.now().UTC().Truncate(time.Millisecond)
	ids, err := s.scheduleRepo.ListDueScheduleIDs(ctx, now, limit)
	if err != nil {
		return 0, err
	}
	processed := 0
	var failures []error
	for _, id := range ids {
		if ctx.Err() != nil {
			return processed, errors.Join(append(failures, ctx.Err())...)
		}
		claimed, runErr := s.runSchedule(ctx, id, lease)
		if claimed {
			processed++
		}
		if runErr != nil {
			failures = append(failures, fmt.Errorf("schedule %s: %w", id, runErr))
		}
	}
	return processed, errors.Join(failures...)
}

func (s *Service) runSchedule(ctx context.Context, id string, lease time.Duration) (bool, error) {
	now := s.now().UTC().Truncate(time.Millisecond)
	capability := s.WebCapability()
	schedule, run, claimed, err := s.scheduleRepo.ClaimSchedule(ctx, id, now, now.Add(lease), identity.SystemAgentUserID, capability.Provider)
	if err != nil || !claimed {
		return claimed, err
	}
	actor := Actor{TenantID: schedule.TenantID, UserID: identity.SystemAgentUserID}
	search, searchErr := s.SearchWeb(ctx, actor, WebSearchInput{
		Query: schedule.Query, GameID: schedule.GameID, CampaignID: schedule.CampaignID,
		Category: schedule.Category, Count: schedule.ResultCount, Country: schedule.Country,
		SearchLang: schedule.SearchLang, Freshness: schedule.Freshness,
	})
	if searchErr == nil && !capability.ImportEnabled {
		searchErr = apperror.Validation("搜索结果入库未启用；自动发现需要供应商存储权")
	}
	if searchErr == nil {
		run.Provider = search.Capability.Provider
		run.ResultCount = len(search.Results)
		for _, result := range search.Results {
			if strings.TrimSpace(result.Title) == "" || strings.TrimSpace(result.Description) == "" || strings.TrimSpace(result.Publisher) == "" {
				run.SkippedCount++
				continue
			}
			publishedAt := webPublishedDate(result.PublishedAt)
			discoveredAt := s.now().UTC()
			_, created, importErr := s.createWithStatus(ctx, actor, CreateInput{
				GameID: schedule.GameID, CampaignID: schedule.CampaignID, Category: schedule.Category,
				Title: result.Title, Summary: result.Description, SourceURL: result.URL,
				Publisher: result.Publisher, PublishedAt: publishedAt,
			}, discovery{method: "SCHEDULED_WEB_SEARCH", provider: search.Capability.Provider, queryHash: schedule.QueryHash, scheduleID: schedule.ID, discoveredAt: &discoveredAt})
			if importErr != nil {
				searchErr = importErr
				break
			}
			if created {
				run.ImportedCount++
			} else {
				run.DuplicateCount++
			}
		}
	}
	finished := s.now().UTC().Truncate(time.Millisecond)
	if searchErr != nil {
		run.Status = researchdomain.ScheduleRunFailed
		run.ErrorCode, run.ErrorMessage = scheduleFailure(searchErr)
	} else {
		run.Status = researchdomain.ScheduleRunSucceeded
	}
	next := nextScheduleTime(run.ScheduledFor, schedule.IntervalMinutes, finished)
	completed, completeErr := s.scheduleRepo.CompleteScheduleRun(ctx, schedule, run, next, finished)
	if completeErr != nil {
		return true, completeErr
	}
	if !completed {
		return true, apperror.Conflict
	}
	if err := s.recordScheduleRun(ctx, schedule, run); err != nil {
		return true, err
	}
	return true, searchErr
}

func (s *Service) normalizeScheduleInput(ctx context.Context, tenantID string, input ScheduleInput) (ScheduleInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Query = strings.TrimSpace(input.Query)
	input.Category = strings.ToUpper(strings.TrimSpace(input.Category))
	input.Country = strings.ToUpper(strings.TrimSpace(input.Country))
	input.SearchLang = strings.ToLower(strings.TrimSpace(input.SearchLang))
	input.Freshness = strings.ToLower(strings.TrimSpace(input.Freshness))
	if utf8.RuneCountInString(input.Name) < 2 || utf8.RuneCountInString(input.Name) > 120 {
		return input, apperror.Validation("name must be between 2 and 120 characters")
	}
	if input.Query == "" || utf8.RuneCountInString(input.Query) > 400 || len(strings.Fields(input.Query)) > 50 {
		return input, apperror.Validation("query is required and must be at most 400 characters and 50 words")
	}
	if !validCategory(input.Category) {
		return input, apperror.Validation("category must be POLICY, COMPETITOR or MARKET")
	}
	if !validCountry(input.Country) || !validSearchLang(input.SearchLang) || !validFreshness(input.Freshness) {
		return input, apperror.Validation("country, search_lang or freshness is invalid")
	}
	capability := s.WebCapability()
	if input.ResultCount < 1 || input.ResultCount > capability.MaxResults {
		return input, apperror.Validation(fmt.Sprintf("result_count must be between 1 and %d", capability.MaxResults))
	}
	if input.IntervalMinutes < minimumScheduleInterval || input.IntervalMinutes > maximumScheduleInterval {
		return input, apperror.Validation("interval_minutes must be between 60 and 10080")
	}
	if err := s.validateScope(ctx, tenantID, input.GameID, input.CampaignID); err != nil {
		return input, err
	}
	return input, nil
}

func (s *Service) validateScheduleCapability() error {
	capability := s.WebCapability()
	if !capability.Configured {
		return apperror.New(10020, 503, "实时联网连接器尚未配置")
	}
	if !capability.ImportEnabled {
		return apperror.Validation("自动发现需要启用搜索结果入库并确认供应商存储权")
	}
	return nil
}

func (s *Service) validateScheduleQuota(ctx context.Context, tenantID string) error {
	count, err := s.scheduleRepo.CountEnabledSchedules(ctx, tenantID)
	if err != nil {
		return err
	}
	if count >= maximumEnabledSchedules {
		return apperror.Validation("each tenant can enable at most 20 research schedules")
	}
	return nil
}

func nextScheduleTime(scheduledFor time.Time, intervalMinutes int, now time.Time) time.Time {
	interval := time.Duration(intervalMinutes) * time.Minute
	next := scheduledFor.Add(interval)
	if next.After(now) {
		return next
	}
	missed := now.Sub(next)/interval + 1
	return next.Add(missed * interval)
}

func scheduleIdempotencyKey(input ScheduleInput) string {
	return hashText(strings.Join([]string{
		input.Name, input.GameID, input.CampaignID, input.Category, input.Query,
		input.Country, input.SearchLang, input.Freshness, strconv.Itoa(input.ResultCount),
		strconv.Itoa(input.IntervalMinutes), strconv.FormatBool(input.Enabled),
	}, "\n"))
}

func scheduleFailure(err error) (string, string) {
	var providerErr *websearch.Error
	if errors.As(err, &providerErr) {
		return providerErr.Code, providerErr.Message
	}
	var appErr *apperror.Error
	if errors.As(err, &appErr) {
		switch appErr.HTTPStatus {
		case 429:
			return "RATE_LIMITED", appErr.Message
		case 502, 503:
			return "UPSTREAM_UNAVAILABLE", appErr.Message
		default:
			return "VALIDATION_FAILED", appErr.Message
		}
	}
	return "INTERNAL_ERROR", "自动研究发现执行失败"
}

func (s *Service) recordSchedule(ctx context.Context, actor Actor, action string, before, after *researchdomain.Schedule) error {
	if s.auditor == nil {
		return nil
	}
	resource := after
	if resource == nil {
		resource = before
	}
	return s.auditor.Record(ctx, auditdomain.RecordInput{
		TenantID: actor.TenantID, ActorID: actor.UserID, ActorType: actorType(actor.UserID),
		Action: action, ResourceType: "RESEARCH_SCHEDULE", ResourceID: resource.ID,
		Before: safeSchedule(before), After: safeSchedule(after), RequestID: actor.RequestID, TraceID: actor.TraceID,
	})
}

func (s *Service) recordScheduleRun(ctx context.Context, schedule *researchdomain.Schedule, run *researchdomain.ScheduleRun) error {
	if s.auditor == nil {
		return nil
	}
	action := "RESEARCH_SCHEDULE_" + run.Status
	return s.auditor.Record(ctx, auditdomain.RecordInput{
		TenantID: schedule.TenantID, ActorID: identity.SystemAgentUserID, ActorType: "SYSTEM_AGENT",
		Action: action, ResourceType: "RESEARCH_SCHEDULE_RUN", ResourceID: run.ID,
		After: map[string]any{"schedule_id": schedule.ID, "status": run.Status, "query_hash": run.QueryHash, "provider": run.Provider, "result_count": run.ResultCount, "imported_count": run.ImportedCount, "duplicate_count": run.DuplicateCount, "skipped_count": run.SkippedCount, "error_code": run.ErrorCode},
	})
}

func safeSchedule(row *researchdomain.Schedule) any {
	if row == nil {
		return nil
	}
	return map[string]any{
		"id": row.ID, "name": row.Name, "game_id": row.GameID, "campaign_id": row.CampaignID,
		"category": row.Category, "query_hash": row.QueryHash, "country": row.Country,
		"search_lang": row.SearchLang, "freshness": row.Freshness, "result_count": row.ResultCount,
		"interval_minutes": row.IntervalMinutes, "enabled": row.Enabled, "next_run_at": row.NextRunAt,
	}
}
