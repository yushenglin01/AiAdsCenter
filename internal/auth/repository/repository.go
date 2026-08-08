package repository

import (
	"context"
	"errors"

	"github.com/example/adnova/internal/common/model"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("user not found")

type UserRepository interface {
	FindByTenantIDAndUsername(ctx context.Context, tenantID, username string) (*model.User, error)
	FindByID(ctx context.Context, tenantID, userID string) (*model.User, error)
}

type GormUserRepository struct{ db *gorm.DB }

func New(db *gorm.DB) *GormUserRepository { return &GormUserRepository{db: db} }

func (r *GormUserRepository) FindByTenantIDAndUsername(ctx context.Context, tenantID, username string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).
		Preload("Roles").
		Where("tenant_id = ? AND username = ?", tenantID, username).
		First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &user, err
}

func (r *GormUserRepository) FindByID(ctx context.Context, tenantID, userID string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Preload("Roles").
		Where("tenant_id = ? AND id = ?", tenantID, userID).
		First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &user, err
}
