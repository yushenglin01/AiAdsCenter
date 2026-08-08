package taskqueue

import (
	"context"
	"errors"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/require"
)

type processorStub struct {
	processErr   error
	retryingCall int
	failedCall   int
	retryValue   int
}

func (s *processorStub) ProcessQueued(context.Context, BusinessAnalysisPayload, int) error {
	return s.processErr
}
func (s *processorStub) MarkRetrying(_ context.Context, _ BusinessAnalysisPayload, _ error, retry int) error {
	s.retryingCall++
	s.retryValue = retry
	return nil
}
func (s *processorStub) MarkFailed(context.Context, BusinessAnalysisPayload, error, int) error {
	s.failedCall++
	return nil
}

func TestBusinessHandlerMarksTransientFailureForRetry(t *testing.T) {
	stub := &processorStub{processErr: errors.New("temporary")}
	err := NewBusinessHandler(stub).process(context.Background(), BusinessAnalysisPayload{TaskID: "task"}, 1, 3)
	require.EqualError(t, err, "temporary")
	require.Equal(t, 1, stub.retryingCall)
	require.Equal(t, 2, stub.retryValue)
	require.Zero(t, stub.failedCall)
}

func TestBusinessHandlerMarksRetryExhaustedAsFailed(t *testing.T) {
	stub := &processorStub{processErr: errors.New("permanent")}
	err := NewBusinessHandler(stub).process(context.Background(), BusinessAnalysisPayload{TaskID: "task"}, 3, 3)
	require.ErrorIs(t, err, asynq.SkipRetry)
	require.Equal(t, 1, stub.failedCall)
	require.Zero(t, stub.retryingCall)
}

func TestBusinessHandlerReturnsNilOnSuccess(t *testing.T) {
	stub := &processorStub{}
	require.NoError(t, NewBusinessHandler(stub).process(context.Background(), BusinessAnalysisPayload{TaskID: "task"}, 0, 3))
}
