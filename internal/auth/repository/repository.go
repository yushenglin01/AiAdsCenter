package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/example/adnova/internal/common/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrNotFound            = errors.New("user not found")
	ErrInvalidTransition   = errors.New("invalid registration transition")
	ErrVerificationExpired = errors.New("verification token expired")
)

type RegistrationFilter struct {
	Status string
	Limit  int
}

type UserRepository interface {
	FindByTenantIDAndUsername(ctx context.Context, tenantID, username string) (*model.User, error)
	FindByID(ctx context.Context, tenantID, userID string) (*model.User, error)
}

type RegistrationRepository interface {
	FindByEmailOrUsername(ctx context.Context, tenantID, email, username string) (*model.User, error)
	CreatePendingRegistration(ctx context.Context, user *model.User, token *model.EmailVerificationToken) error
	ReplaceVerificationToken(ctx context.Context, tenantID, userID string, token *model.EmailVerificationToken, cooldown time.Duration) (bool, error)
	InvalidateVerificationToken(ctx context.Context, tenantID, tokenHash string, now time.Time) error
	VerifyEmail(ctx context.Context, tenantID, tokenHash string, now time.Time) (*model.User, error)
	ListRegistrationApplications(ctx context.Context, tenantID string, filter RegistrationFilter) ([]model.User, error)
	ListRoles(ctx context.Context) ([]model.Role, error)
	ApproveRegistration(ctx context.Context, tenantID, userID, adminID string, roleCodes []string, now time.Time) (*model.User, error)
	RejectRegistration(ctx context.Context, tenantID, userID, adminID, reason string, now time.Time) (*model.User, error)
}

type GormUserRepository struct{ db *gorm.DB }

func New(db *gorm.DB) *GormUserRepository { return &GormUserRepository{db: db} }

func (r *GormUserRepository) FindByTenantIDAndUsername(ctx context.Context, tenantID, identifier string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).
		Preload("Roles").
		Where("tenant_id = ? AND (username = ? OR email = ?)", tenantID, identifier, identifier).
		First(&user).Error
	return userResult(&user, err)
}

func (r *GormUserRepository) FindByID(ctx context.Context, tenantID, userID string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Preload("Roles").
		Where("tenant_id = ? AND id = ?", tenantID, userID).
		First(&user).Error
	return userResult(&user, err)
}

func (r *GormUserRepository) FindByEmailOrUsername(ctx context.Context, tenantID, email, username string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND (email = ? OR username = ?)", tenantID, email, username).
		First(&user).Error
	return userResult(&user, err)
}

func (r *GormUserRepository) CreatePendingRegistration(ctx context.Context, user *model.User, token *model.EmailVerificationToken) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return fmt.Errorf("create pending user: %w", err)
		}
		if err := tx.Create(token).Error; err != nil {
			return fmt.Errorf("create verification token: %w", err)
		}
		return nil
	})
}

func (r *GormUserRepository) ReplaceVerificationToken(ctx context.Context, tenantID, userID string, token *model.EmailVerificationToken, cooldown time.Duration) (bool, error) {
	created := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND id = ?", tenantID, userID).First(&user).Error; err != nil {
			return mapNotFound(err)
		}
		if user.Status != "PENDING_EMAIL" {
			return nil
		}
		var latest model.EmailVerificationToken
		result := tx.Where("tenant_id = ? AND user_id = ? AND consumed_at IS NULL", tenantID, userID).Order("created_at DESC").Limit(1).Find(&latest)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 1 && token.CreatedAt.Sub(latest.CreatedAt) < cooldown {
			return nil
		}
		now := token.CreatedAt
		if err := tx.Model(&model.EmailVerificationToken{}).
			Where("tenant_id = ? AND user_id = ? AND consumed_at IS NULL", tenantID, userID).
			Update("consumed_at", now).Error; err != nil {
			return err
		}
		if err := tx.Create(token).Error; err != nil {
			return err
		}
		created = true
		return nil
	})
	return created, err
}

func (r *GormUserRepository) InvalidateVerificationToken(ctx context.Context, tenantID, tokenHash string, now time.Time) error {
	return r.db.WithContext(ctx).Model(&model.EmailVerificationToken{}).
		Where("tenant_id = ? AND token_hash = ? AND consumed_at IS NULL", tenantID, tokenHash).
		Update("consumed_at", now).Error
}

func (r *GormUserRepository) VerifyEmail(ctx context.Context, tenantID, tokenHash string, now time.Time) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var token model.EmailVerificationToken
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("tenant_id = ? AND token_hash = ? AND consumed_at IS NULL", tenantID, tokenHash).
			First(&token).Error; err != nil {
			return mapNotFound(err)
		}
		if !token.ExpiresAt.After(now) {
			return ErrVerificationExpired
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND id = ?", tenantID, token.UserID).First(&user).Error; err != nil {
			return mapNotFound(err)
		}
		if user.Status != "PENDING_EMAIL" {
			return ErrInvalidTransition
		}
		if err := tx.Model(&user).Updates(map[string]any{"status": "PENDING_APPROVAL", "email_verified_at": now}).Error; err != nil {
			return err
		}
		if err := tx.Model(&token).Update("consumed_at", now).Error; err != nil {
			return err
		}
		user.Status = "PENDING_APPROVAL"
		user.EmailVerifiedAt = &now
		return nil
	})
	return &user, err
}

func (r *GormUserRepository) ListRegistrationApplications(ctx context.Context, tenantID string, filter RegistrationFilter) ([]model.User, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	query := r.db.WithContext(ctx).Preload("Roles").Where("tenant_id = ?", tenantID)
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	} else {
		query = query.Where("status IN ?", []string{"PENDING_EMAIL", "PENDING_APPROVAL", "REJECTED", "ACTIVE", "DISABLED"})
	}
	var users []model.User
	err := query.Order("created_at DESC").Limit(limit).Find(&users).Error
	return users, err
}

func (r *GormUserRepository) ListRoles(ctx context.Context) ([]model.Role, error) {
	var roles []model.Role
	err := r.db.WithContext(ctx).Where("code <> ?", "SYSTEM_AGENT").Order("code ASC").Find(&roles).Error
	return roles, err
}

func (r *GormUserRepository) ApproveRegistration(ctx context.Context, tenantID, userID, adminID string, roleCodes []string, now time.Time) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND id = ?", tenantID, userID).First(&user).Error; err != nil {
			return mapNotFound(err)
		}
		if user.Status != "PENDING_APPROVAL" || user.EmailVerifiedAt == nil {
			return ErrInvalidTransition
		}
		var roles []model.Role
		if err := tx.Where("code IN ? AND code <> ?", roleCodes, "SYSTEM_AGENT").Find(&roles).Error; err != nil {
			return err
		}
		if len(roles) != len(roleCodes) {
			return fmt.Errorf("one or more roles do not exist")
		}
		if err := tx.Where("tenant_id = ? AND user_id = ?", tenantID, userID).Delete(&model.UserRole{}).Error; err != nil {
			return err
		}
		assignments := make([]model.UserRole, 0, len(roles))
		for _, role := range roles {
			assignments = append(assignments, model.UserRole{TenantID: tenantID, UserID: userID, RoleID: role.ID})
		}
		if err := tx.Create(&assignments).Error; err != nil {
			return err
		}
		if err := tx.Model(&user).Updates(map[string]any{
			"status": "ACTIVE", "approved_at": now, "approved_by": adminID,
			"rejected_at": nil, "rejected_by": "", "rejection_reason": "",
		}).Error; err != nil {
			return err
		}
		user.Status = "ACTIVE"
		user.ApprovedAt = &now
		user.ApprovedBy = adminID
		user.Roles = roles
		return nil
	})
	return &user, err
}

func (r *GormUserRepository) RejectRegistration(ctx context.Context, tenantID, userID, adminID, reason string, now time.Time) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND id = ?", tenantID, userID).First(&user).Error; err != nil {
			return mapNotFound(err)
		}
		if user.Status != "PENDING_APPROVAL" {
			return ErrInvalidTransition
		}
		if err := tx.Model(&user).Updates(map[string]any{
			"status": "REJECTED", "rejected_at": now, "rejected_by": adminID, "rejection_reason": reason,
		}).Error; err != nil {
			return err
		}
		user.Status = "REJECTED"
		user.RejectedAt = &now
		user.RejectedBy = adminID
		user.RejectionReason = reason
		return nil
	})
	return &user, err
}

func userResult(user *model.User, err error) (*model.User, error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return user, err
}

func mapNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}
