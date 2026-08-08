package taskqueue

import (
	"context"
	"fmt"

	"github.com/hibiken/asynq"
)

type BusinessProcessor interface {
	ProcessQueued(context.Context, BusinessAnalysisPayload, int) error
	MarkRetrying(context.Context, BusinessAnalysisPayload, error, int) error
	MarkFailed(context.Context, BusinessAnalysisPayload, error, int) error
}

type BusinessHandler struct{ processor BusinessProcessor }

func NewBusinessHandler(processor BusinessProcessor) *BusinessHandler {
	return &BusinessHandler{processor: processor}
}

func (h *BusinessHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	payload, err := DecodeBusinessAnalysisPayload(task)
	if err != nil {
		return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
	}
	retryCount, _ := asynq.GetRetryCount(ctx)
	maxRetry, _ := asynq.GetMaxRetry(ctx)
	return h.process(ctx, payload, retryCount, maxRetry)
}

func (h *BusinessHandler) process(ctx context.Context, payload BusinessAnalysisPayload, retryCount, maxRetry int) error {
	if err := h.processor.ProcessQueued(ctx, payload, retryCount); err != nil {
		if retryCount >= maxRetry {
			if markErr := h.processor.MarkFailed(ctx, payload, err, retryCount); markErr != nil {
				return fmt.Errorf("%w: retry exhausted: %v; persist failure: %v", asynq.SkipRetry, err, markErr)
			}
			return fmt.Errorf("%w: retry exhausted: %v", asynq.SkipRetry, err)
		}
		if markErr := h.processor.MarkRetrying(ctx, payload, err, retryCount+1); markErr != nil {
			return fmt.Errorf("process failed: %v; persist retry: %w", err, markErr)
		}
		return err
	}
	return nil
}
