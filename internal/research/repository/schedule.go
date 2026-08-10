package repository

import (
	"context"
	"errors"
	"time"

	"github.com/example/adnova/internal/common/apperror"
	researchdomain "github.com/example/adnova/internal/research/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *Repository) CountEnabledSchedules(ctx context.Context, tenantID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&researchdomain.Schedule{}).Where("tenant_id = ? AND enabled = ?", tenantID, true).Count(&count).Error
	return count, err
}

func (r *Repository) CreateSchedule(ctx context.Context, row *researchdomain.Schedule) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *Repository) GetSchedule(ctx context.Context, tenantID, id string) (*researchdomain.Schedule, error) {
	var row researchdomain.Schedule
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperror.NotFound
	}
	return &row, err
}

func (r *Repository) FindScheduleByKey(ctx context.Context, tenantID, key string) (*researchdomain.Schedule, error) {
	var row researchdomain.Schedule
	result := r.db.WithContext(ctx).Where("tenant_id = ? AND idempotency_key = ?", tenantID, key).Limit(1).Find(&row)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &row, nil
}

func (r *Repository) ListSchedules(ctx context.Context, tenantID string) ([]researchdomain.Schedule, error) {
	var rows []researchdomain.Schedule
	err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("enabled DESC, created_at DESC").Find(&rows).Error
	return rows, err
}

func (r *Repository) UpdateSchedule(ctx context.Context, row *researchdomain.Schedule, now time.Time) (bool, error) {
	result := r.db.WithContext(ctx).Model(&researchdomain.Schedule{}).
		Where("tenant_id = ? AND id = ? AND (locked_until IS NULL OR locked_until < ?)", row.TenantID, row.ID, now).
		Updates(map[string]any{
			"name": row.Name, "game_id": nullableString(row.GameID), "campaign_id": nullableString(row.CampaignID),
			"category": row.Category, "query": row.Query, "query_hash": row.QueryHash, "idempotency_key": row.IdempotencyKey,
			"country": row.Country, "search_lang": row.SearchLang, "freshness": row.Freshness,
			"result_count": row.ResultCount, "interval_minutes": row.IntervalMinutes,
			"enabled": row.Enabled, "next_run_at": row.NextRunAt, "updated_by": row.UpdatedBy,
		})
	return result.RowsAffected == 1, result.Error
}

func (r *Repository) ListScheduleRuns(ctx context.Context, tenantID, scheduleID string, limit int) ([]researchdomain.ScheduleRun, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	query := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)
	if scheduleID != "" {
		query = query.Where("schedule_id = ?", scheduleID)
	}
	var rows []researchdomain.ScheduleRun
	err := query.Order("scheduled_for DESC, created_at DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *Repository) ListDueScheduleIDs(ctx context.Context, now time.Time, limit int) ([]string, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var ids []string
	err := r.db.WithContext(ctx).Model(&researchdomain.Schedule{}).
		Where("enabled = ? AND next_run_at IS NOT NULL AND next_run_at <= ? AND (locked_until IS NULL OR locked_until < ?)", true, now, now).
		Order("next_run_at ASC").Limit(limit).Pluck("id", &ids).Error
	return ids, err
}

func (r *Repository) ClaimSchedule(ctx context.Context, id string, now, lockedUntil time.Time, requestedBy, provider string) (*researchdomain.Schedule, *researchdomain.ScheduleRun, bool, error) {
	var schedule researchdomain.Schedule
	var run researchdomain.ScheduleRun
	claimed := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND enabled = ? AND next_run_at IS NOT NULL AND next_run_at <= ? AND (locked_until IS NULL OR locked_until < ?)", id, true, now, now).
			First(&schedule).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		token := uuid.NewString()
		if err := tx.Model(&researchdomain.Schedule{}).Where("id = ?", schedule.ID).
			Updates(map[string]any{"lock_token": token, "locked_until": lockedUntil}).Error; err != nil {
			return err
		}
		scheduledFor := *schedule.NextRunAt
		err = tx.Where("tenant_id = ? AND schedule_id = ? AND scheduled_for = ?", schedule.TenantID, schedule.ID, scheduledFor).First(&run).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			run = researchdomain.ScheduleRun{
				ID: uuid.NewString(), TenantID: schedule.TenantID, ScheduleID: schedule.ID,
				ScheduledFor: scheduledFor, Status: researchdomain.ScheduleRunProcessing,
				Provider: provider, QueryHash: schedule.QueryHash, RequestedBy: requestedBy,
				ClaimToken: token, StartedAt: now,
			}
			if err := tx.Create(&run).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			if run.Status != researchdomain.ScheduleRunProcessing {
				return nil
			}
			if err := tx.Model(&researchdomain.ScheduleRun{}).Where("id = ? AND status = ?", run.ID, researchdomain.ScheduleRunProcessing).
				Updates(map[string]any{"claim_token": token, "started_at": now, "finished_at": nil, "error_code": "", "error_message": ""}).Error; err != nil {
				return err
			}
			run.ClaimToken, run.StartedAt, run.FinishedAt = token, now, nil
		}
		schedule.LockToken, schedule.LockedUntil = token, &lockedUntil
		claimed = true
		return nil
	})
	return &schedule, &run, claimed, err
}

func (r *Repository) CompleteScheduleRun(ctx context.Context, schedule *researchdomain.Schedule, run *researchdomain.ScheduleRun, nextRunAt, finishedAt time.Time) (bool, error) {
	completed := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&researchdomain.ScheduleRun{}).
			Where("tenant_id = ? AND id = ? AND status = ? AND claim_token = ?", run.TenantID, run.ID, researchdomain.ScheduleRunProcessing, run.ClaimToken).
			Updates(map[string]any{
				"status": run.Status, "result_count": run.ResultCount, "imported_count": run.ImportedCount,
				"duplicate_count": run.DuplicateCount, "skipped_count": run.SkippedCount,
				"error_code": run.ErrorCode, "error_message": run.ErrorMessage,
				"finished_at": finishedAt, "claim_token": "",
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return nil
		}
		result = tx.Model(&researchdomain.Schedule{}).
			Where("tenant_id = ? AND id = ? AND lock_token = ?", schedule.TenantID, schedule.ID, run.ClaimToken).
			Updates(map[string]any{
				"next_run_at": nextRunAt, "last_run_at": finishedAt, "last_status": run.Status,
				"last_error_code": run.ErrorCode, "last_error_message": run.ErrorMessage,
				"lock_token": "", "locked_until": nil,
			})
		if result.Error != nil {
			return result.Error
		}
		completed = result.RowsAffected == 1
		return nil
	})
	return completed, err
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
