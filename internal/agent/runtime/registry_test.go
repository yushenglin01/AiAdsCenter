package runtime

import (
	"context"
	"testing"

	agentdomain "github.com/example/adnova/internal/agent/domain"
	"github.com/stretchr/testify/require"
)

type fakeExecutor struct{}

func (fakeExecutor) Execute(_ context.Context, input agentdomain.AgentInput) (*agentdomain.AgentResult, error) {
	return &agentdomain.AgentResult{TaskID: input.TaskID, AgentName: "data-agent", Status: "SUCCEEDED"}, nil
}
func (fakeExecutor) Health(context.Context) (*agentdomain.HealthStatus, error) {
	return &agentdomain.HealthStatus{Status: "UP"}, nil
}
func (fakeExecutor) Capabilities(context.Context) []string { return []string{"recalculate_metrics"} }

func TestRegistryRegisterListAndExecute(t *testing.T) {
	registry := NewRegistry()
	require.NoError(t, registry.Register(Definition{Spec: agentdomain.AgentSpec{Name: "data-agent"}, Availability: AvailabilityReady, Executor: fakeExecutor{}}))
	require.Error(t, registry.Register(Definition{Spec: agentdomain.AgentSpec{Name: "data-agent"}, Availability: AvailabilityReady, Executor: fakeExecutor{}}))
	require.Len(t, registry.List(), 1)
	result, err := registry.Execute(context.Background(), "data-agent", agentdomain.AgentInput{TaskID: "task-1"})
	require.NoError(t, err)
	require.Equal(t, "SUCCEEDED", result.Status)
}

func TestRegistryRejectsUnavailableExecution(t *testing.T) {
	registry := NewRegistry()
	require.NoError(t, registry.Register(Definition{Spec: agentdomain.AgentSpec{Name: "research-agent"}, Availability: AvailabilityDegraded}))
	_, err := registry.Execute(context.Background(), "research-agent", agentdomain.AgentInput{})
	require.ErrorContains(t, err, "not executable")
}
