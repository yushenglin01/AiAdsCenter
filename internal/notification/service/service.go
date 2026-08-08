package service

import (
	"context"
	"encoding/json"

	"github.com/example/adnova/internal/notification/domain"
	"github.com/example/adnova/internal/notification/repository"
	"github.com/google/uuid"
)

type PublishInput struct {
	TenantID   string
	WorkflowID string
	EventType  string
	Title      string
	Message    string
	Metadata   map[string]any
}

type Service struct{ repo *repository.Repository }

func New(repo *repository.Repository) *Service { return &Service{repo: repo} }

func (s *Service) Publish(ctx context.Context, input PublishInput) error {
	metadata, _ := json.Marshal(input.Metadata)
	return s.repo.CreateOnce(ctx, &domain.Notification{ID: uuid.NewString(), TenantID: input.TenantID, WorkflowID: input.WorkflowID, EventType: input.EventType, Channel: "INTERNAL", Title: input.Title, Message: input.Message, Status: domain.StatusUnread, MetadataJSON: metadata})
}

func (s *Service) List(ctx context.Context, tenantID, status string) ([]domain.Notification, error) {
	if status != "" && status != domain.StatusUnread && status != domain.StatusRead {
		status = ""
	}
	return s.repo.List(ctx, tenantID, status, 50)
}

func (s *Service) MarkRead(ctx context.Context, tenantID, id, userID string) (*domain.Notification, error) {
	return s.repo.MarkRead(ctx, tenantID, id, userID)
}
