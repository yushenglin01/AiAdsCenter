package repository

import (
	"context"
	"errors"

	"github.com/example/adnova/internal/game/domain"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("game not found")

type Repository struct{ db *gorm.DB }

func New(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Create(ctx context.Context, game *domain.Game) error {
	return r.db.WithContext(ctx).Create(game).Error
}

func (r *Repository) List(ctx context.Context, tenantID string) ([]domain.Game, error) {
	var games []domain.Game
	err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("created_at DESC").Find(&games).Error
	return games, err
}

func (r *Repository) Get(ctx context.Context, tenantID, id string) (*domain.Game, error) {
	var game domain.Game
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&game).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &game, err
}

func (r *Repository) Update(ctx context.Context, game *domain.Game) error {
	return r.db.WithContext(ctx).Save(game).Error
}
