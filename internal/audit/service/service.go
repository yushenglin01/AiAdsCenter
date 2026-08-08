package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/example/adnova/internal/audit/domain"
	"github.com/example/adnova/internal/audit/repository"
	"github.com/google/uuid"
)

type Recorder interface {
	Record(ctx context.Context, input domain.RecordInput) error
}

type Service struct{ repo *repository.Repository }

func New(repo *repository.Repository) *Service { return &Service{repo: repo} }

func (s *Service) Record(ctx context.Context, input domain.RecordInput) error {
	if input.TenantID == "" || input.ActorID == "" || input.Action == "" || input.ResourceType == "" || input.ResourceID == "" {
		return fmt.Errorf("audit tenant, actor, action and resource are required")
	}
	row := &domain.AuditLog{
		ID: uuid.NewString(), TenantID: input.TenantID, ActorID: input.ActorID, ActorType: input.ActorType,
		Action: input.Action, ResourceType: input.ResourceType, ResourceID: input.ResourceID, TaskID: input.TaskID,
		BeforeJSON: marshal(input.Before), AfterJSON: marshal(input.After), MetadataJSON: marshal(input.Metadata),
		RequestID: input.RequestID, TraceID: input.TraceID, IPAddress: input.IPAddress,
	}
	return s.repo.Create(ctx, row)
}

func (s *Service) List(ctx context.Context, tenantID string, filter domain.Filter) ([]domain.AuditLog, error) {
	return s.repo.List(ctx, tenantID, filter)
}

func marshal(value any) json.RawMessage {
	if value == nil {
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(`{"serialization_error":true}`)
	}
	return raw
}
