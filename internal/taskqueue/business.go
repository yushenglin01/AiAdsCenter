package taskqueue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/example/adnova/internal/config"
	"github.com/hibiken/asynq"
)

const TypeBusinessAnalysis = "analysis:business"

type BusinessAnalysisPayload struct {
	TaskID       string `json:"task_id"`
	TenantID     string `json:"tenant_id"`
	GameID       string `json:"game_id"`
	CampaignID   string `json:"campaign_id"`
	CreatedBy    string `json:"created_by"`
	AnalysisDate string `json:"analysis_date"`
	TraceID      string `json:"trace_id"`
}

type Enqueuer interface {
	EnqueueBusinessAnalysis(context.Context, BusinessAnalysisPayload) error
}

type AsynqEnqueuer struct {
	client *asynq.Client
	config config.QueueConfig
}

func RedisOption(cfg config.RedisConfig) asynq.RedisClientOpt {
	return asynq.RedisClientOpt{Addr: cfg.Address, Password: cfg.Password, DB: cfg.DB}
}

func NewAsynqEnqueuer(redisCfg config.RedisConfig, queueCfg config.QueueConfig) *AsynqEnqueuer {
	return &AsynqEnqueuer{client: asynq.NewClient(RedisOption(redisCfg)), config: queueCfg}
}

func (e *AsynqEnqueuer) Close() error { return e.client.Close() }

func (e *AsynqEnqueuer) EnqueueBusinessAnalysis(ctx context.Context, payload BusinessAnalysisPayload) error {
	if payload.TaskID == "" || payload.TenantID == "" || payload.GameID == "" || payload.CampaignID == "" || payload.CreatedBy == "" {
		return fmt.Errorf("business analysis queue payload is incomplete")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode business analysis task: %w", err)
	}
	task := asynq.NewTask(TypeBusinessAnalysis, body)
	_, err = e.client.EnqueueContext(ctx, task,
		asynq.TaskID(payload.TaskID),
		asynq.Queue(e.config.Name),
		asynq.MaxRetry(e.config.MaxRetry),
		asynq.Timeout(e.config.TaskTimeout),
		asynq.Retention(e.config.Retention),
	)
	if errors.Is(err, asynq.ErrTaskIDConflict) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("enqueue business analysis: %w", err)
	}
	return nil
}

func DecodeBusinessAnalysisPayload(task *asynq.Task) (BusinessAnalysisPayload, error) {
	var payload BusinessAnalysisPayload
	if task.Type() != TypeBusinessAnalysis {
		return payload, fmt.Errorf("unexpected task type %s", task.Type())
	}
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return payload, fmt.Errorf("decode business analysis task: %w", err)
	}
	if payload.TaskID == "" || payload.TenantID == "" || payload.GameID == "" || payload.CampaignID == "" || payload.CreatedBy == "" {
		return payload, fmt.Errorf("business analysis queue payload is incomplete")
	}
	return payload, nil
}
