package repository

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/example/adnova/internal/ingestion/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrMessageBusy     = errors.New("ingestion message is being processed")
	ErrMessageConflict = errors.New("ingestion event identity or payload conflict")
)

type MessageClaim struct {
	TenantID       string
	EventID        string
	ProducerSystem string
	SchemaVersion  string
	Topic          string
	Partition      int32
	Offset         int64
	PayloadHash    string
	Lease          time.Duration
}

func (r *Repository) ClaimMessage(ctx context.Context, input MessageClaim) (*domain.IngestionMessage, bool, error) {
	now := time.Now().UTC()
	lockedUntil := now.Add(input.Lease)
	row := &domain.IngestionMessage{
		ID: uuid.NewString(), TenantID: input.TenantID, EventID: input.EventID,
		ProducerSystem: input.ProducerSystem, SchemaVersion: input.SchemaVersion,
		Topic: input.Topic, Partition: input.Partition, Offset: input.Offset,
		PayloadHash: input.PayloadHash, Status: "PROCESSING", AttemptCount: 1,
		LockedUntil: &lockedUntil,
	}
	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(row)
	if result.Error != nil {
		return nil, false, result.Error
	}
	if result.RowsAffected == 1 {
		return row, true, nil
	}
	if err := r.db.WithContext(ctx).Where("tenant_id = ? AND producer_system = ? AND event_id = ?", input.TenantID, input.ProducerSystem, input.EventID).First(row).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, err
		}
		if positionErr := r.db.WithContext(ctx).Where("topic = ? AND partition_number = ? AND kafka_offset = ?", input.Topic, input.Partition, input.Offset).First(row).Error; positionErr != nil {
			return nil, false, positionErr
		}
		return nil, false, ErrMessageConflict
	}
	if row.PayloadHash != input.PayloadHash {
		return nil, false, ErrMessageConflict
	}
	if row.Status == "SUCCEEDED" {
		return row, false, nil
	}
	claim := r.db.WithContext(ctx).Model(&domain.IngestionMessage{}).
		Where("id = ? AND (status <> ? OR locked_until IS NULL OR locked_until < ?)", row.ID, "PROCESSING", now).
		Updates(map[string]any{
			"status": "PROCESSING", "attempt_count": gorm.Expr("attempt_count + 1"),
			"locked_until": lockedUntil, "error_code": "", "error_message": "",
		})
	if claim.Error != nil {
		return nil, false, claim.Error
	}
	if claim.RowsAffected == 0 {
		return nil, false, ErrMessageBusy
	}
	row.Status, row.LockedUntil = "PROCESSING", &lockedUntil
	row.AttemptCount++
	return row, true, nil
}

func (r *Repository) CompleteMessage(ctx context.Context, id string) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Model(&domain.IngestionMessage{}).Where("id = ?", id).Updates(map[string]any{
		"status": "SUCCEEDED", "processed_at": now, "locked_until": nil, "error_code": "", "error_message": "",
	}).Error
}

func (r *Repository) FailMessage(ctx context.Context, id, code, message string) error {
	return r.db.WithContext(ctx).Model(&domain.IngestionMessage{}).Where("id = ?", id).Updates(map[string]any{
		"status": "FAILED", "locked_until": nil, "error_code": code, "error_message": message,
	}).Error
}

func (r *Repository) TouchAnalysisWindow(ctx context.Context, tenantID, gameID string, start, end time.Time, dataset string, debounce time.Duration) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row domain.AnalysisWindow
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where(
			"tenant_id = ? AND game_id = ? AND period_start = ? AND period_end = ?", tenantID, gameID, start, end,
		).First(&row).Error
		now := time.Now().UTC()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			datasets, _ := json.Marshal([]string{dataset})
			row = domain.AnalysisWindow{ID: uuid.NewString(), TenantID: tenantID, GameID: gameID, PeriodStart: start, PeriodEnd: end, ReceivedDatasets: datasets, Status: "PENDING", Version: 1, NextRunAt: now.Add(debounce)}
			return tx.Create(&row).Error
		}
		if err != nil {
			return err
		}
		set := map[string]bool{}
		var values []string
		_ = json.Unmarshal(row.ReceivedDatasets, &values)
		for _, value := range values {
			set[value] = true
		}
		set[dataset] = true
		values = values[:0]
		for value := range set {
			values = append(values, value)
		}
		sort.Strings(values)
		datasets, _ := json.Marshal(values)
		return tx.Model(&row).Updates(map[string]any{
			"received_datasets": datasets, "status": "PENDING", "version": gorm.Expr("version + 1"),
			"attempt_count": 0, "next_run_at": now.Add(debounce), "locked_until": nil, "claim_token": "",
			"last_error": "", "processed_at": nil,
		}).Error
	})
}

func (r *Repository) ListReadyAnalysisWindows(ctx context.Context, limit int) ([]domain.AnalysisWindow, error) {
	var rows []domain.AnalysisWindow
	now := time.Now().UTC()
	err := r.db.WithContext(ctx).Where(
		"(status = ? AND next_run_at <= ?) OR (status = ? AND (locked_until IS NULL OR locked_until < ?))",
		"PENDING", now, "PROCESSING", now,
	).Order("next_run_at").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *Repository) ClaimAnalysisWindow(ctx context.Context, id string, version int, lease time.Duration) (string, bool, error) {
	now := time.Now().UTC()
	token := uuid.NewString()
	result := r.db.WithContext(ctx).Model(&domain.AnalysisWindow{}).Where(
		"id = ? AND version = ? AND ((status = ? AND next_run_at <= ?) OR (status = ? AND (locked_until IS NULL OR locked_until < ?)))",
		id, version, "PENDING", now, "PROCESSING", now,
	).Updates(map[string]any{"status": "PROCESSING", "locked_until": now.Add(lease), "claim_token": token})
	return token, result.RowsAffected == 1, result.Error
}

func (r *Repository) CompleteAnalysisWindow(ctx context.Context, id string, version int, token string) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Model(&domain.AnalysisWindow{}).Where("id = ? AND version = ? AND status = ? AND claim_token = ?", id, version, "PROCESSING", token).Updates(map[string]any{
		"status": "SUCCEEDED", "processed_at": now, "locked_until": nil, "claim_token": "", "last_error": "",
	}).Error
}

func (r *Repository) RetryAnalysisWindow(ctx context.Context, id string, version int, token string, attempts int, next time.Time, message string) error {
	return r.db.WithContext(ctx).Model(&domain.AnalysisWindow{}).Where("id = ? AND version = ? AND status = ? AND claim_token = ?", id, version, "PROCESSING", token).Updates(map[string]any{
		"status": "PENDING", "attempt_count": attempts, "next_run_at": next, "locked_until": nil, "claim_token": "", "last_error": message,
	}).Error
}
