package repository

import (
	"context"
	"errors"
	"time"

	"github.com/example/adnova/internal/mmp/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrNotFound = errors.New("mmp resource not found")

type Repository struct{ db *gorm.DB }

func New(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) GameExists(ctx context.Context, tenantID, gameID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("games").Where("tenant_id = ? AND id = ? AND deleted_at IS NULL", tenantID, gameID).Count(&count).Error
	return count == 1, err
}

func (r *Repository) ListConnections(ctx context.Context, tenantID string) ([]domain.Connection, error) {
	var rows []domain.Connection
	err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("created_at DESC").Find(&rows).Error
	return rows, err
}

func (r *Repository) ListAllConnections(ctx context.Context) ([]domain.Connection, error) {
	var rows []domain.Connection
	err := r.db.WithContext(ctx).Order("tenant_id ASC, updated_at DESC").Find(&rows).Error
	return rows, err
}

func (r *Repository) GetConnection(ctx context.Context, tenantID, id string) (*domain.Connection, error) {
	var row domain.Connection
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &row, err
}

func (r *Repository) FindConnection(ctx context.Context, tenantID, gameID, provider string) (*domain.Connection, error) {
	var row domain.Connection
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND game_id = ? AND provider = ?", tenantID, gameID, provider).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &row, err
}

func (r *Repository) UpsertConnection(ctx context.Context, row *domain.Connection) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "game_id"}, {Name: "provider"}},
		DoUpdates: clause.AssignmentColumns([]string{"external_app_id", "status", "updated_by", "updated_at"}),
	}).Create(row).Error
}

func (r *Repository) TouchConnection(ctx context.Context, tenantID, id string, at time.Time) error {
	return r.db.WithContext(ctx).Model(&domain.Connection{}).Where("tenant_id = ? AND id = ?", tenantID, id).Update("last_sync_at", at).Error
}

func (r *Repository) ListSyncRuns(ctx context.Context, tenantID string) ([]domain.SyncRun, error) {
	var rows []domain.SyncRun
	err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("created_at DESC").Limit(200).Find(&rows).Error
	return rows, err
}

func (r *Repository) FindSyncRunByKey(ctx context.Context, tenantID, key string) (*domain.SyncRun, error) {
	var row domain.SyncRun
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND idempotency_key = ?", tenantID, key).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &row, err
}

func (r *Repository) CreateSyncRun(ctx context.Context, row *domain.SyncRun) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *Repository) ClaimSyncRun(ctx context.Context, tenantID, id, expectedStatus string, expectedStartedAt, startedAt, now, lockedUntil time.Time, requestedBy string) (string, bool, error) {
	token := uuid.NewString()
	query := r.db.WithContext(ctx).Model(&domain.SyncRun{}).
		Where("tenant_id = ? AND id = ? AND status = ? AND started_at = ?", tenantID, id, expectedStatus, expectedStartedAt)
	if expectedStatus == domain.SyncProcessing {
		query = query.Where("locked_until IS NULL OR locked_until < ?", now)
	}
	result := query.
		Updates(map[string]any{
			"status": domain.SyncProcessing, "requested_by": requestedBy, "started_at": startedAt, "finished_at": nil,
			"locked_until": lockedUntil, "claim_token": token,
			"import_job_id": "", "source_rows": 0, "normalized_rows": 0, "skipped_rows": 0,
			"warning_message": "", "error_code": "", "error_message": "",
		})
	return token, result.RowsAffected == 1, result.Error
}

func (r *Repository) RenewSyncRunClaim(ctx context.Context, tenantID, id, token string, lockedUntil time.Time) (bool, error) {
	result := r.db.WithContext(ctx).Model(&domain.SyncRun{}).
		Where("tenant_id = ? AND id = ? AND status = ? AND claim_token = ?", tenantID, id, domain.SyncProcessing, token).
		Update("locked_until", lockedUntil)
	return result.RowsAffected == 1, result.Error
}

func (r *Repository) UpdateClaimedSyncRun(ctx context.Context, row *domain.SyncRun) (bool, error) {
	result := r.db.WithContext(ctx).Model(&domain.SyncRun{}).
		Where("tenant_id = ? AND id = ? AND status = ? AND claim_token = ?", row.TenantID, row.ID, domain.SyncProcessing, row.ClaimToken).
		Updates(map[string]any{
			"status": row.Status, "import_job_id": row.ImportJobID, "source_rows": row.SourceRows,
			"normalized_rows": row.NormalizedRows, "skipped_rows": row.SkippedRows,
			"warning_message": row.WarningMessage, "error_code": row.ErrorCode,
			"error_message": row.ErrorMessage, "finished_at": row.FinishedAt,
			"locked_until": nil, "claim_token": "",
		})
	return result.RowsAffected == 1, result.Error
}
