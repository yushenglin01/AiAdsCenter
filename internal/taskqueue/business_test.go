package taskqueue

import (
	"testing"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/require"
)

func TestDecodeBusinessAnalysisPayload(t *testing.T) {
	task := asynq.NewTask(TypeBusinessAnalysis, []byte(`{"task_id":"task-1","tenant_id":"tenant-1","game_id":"game-1","campaign_id":"campaign-1","created_by":"user-1","analysis_date":"2026-08-04","trace_id":"trace-1"}`))
	payload, err := DecodeBusinessAnalysisPayload(task)
	require.NoError(t, err)
	require.Equal(t, "task-1", payload.TaskID)
	require.Equal(t, "trace-1", payload.TraceID)
}

func TestDecodeBusinessAnalysisPayloadRejectsIncompleteTask(t *testing.T) {
	task := asynq.NewTask(TypeBusinessAnalysis, []byte(`{"task_id":"task-1"}`))
	_, err := DecodeBusinessAnalysisPayload(task)
	require.Error(t, err)
}
