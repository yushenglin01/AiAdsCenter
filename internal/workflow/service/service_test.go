package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewStepsDefinesSevenAgentWorkflow(t *testing.T) {
	steps := newSteps("workflow-1", "tenant-1")
	require.Len(t, steps, 7)
	names := make([]string, 0, len(steps))
	for _, step := range steps {
		names = append(names, step.AgentName)
	}
	require.Equal(t, []string{"openclaw-agent", "data-agent", "attribution-agent", "creative-agent", "research-agent", "business-agent", "report-agent"}, names)
	require.Equal(t, "ASYNCHRONOUS_LLM", steps[5].ExecutionMode)
}

func TestWorkflowTerminalStates(t *testing.T) {
	require.True(t, workflowTerminal("COMPLETED"))
	require.True(t, workflowTerminal("FAILED"))
	require.False(t, workflowTerminal("WAITING_APPROVAL"))
}

func TestNotificationContentOnlyForActionableStates(t *testing.T) {
	title, message := notificationContent("WAITING_APPROVAL")
	require.NotEmpty(t, title)
	require.Contains(t, message, "尚未执行")
	title, message = notificationContent("RUNNING")
	require.Empty(t, title)
	require.Empty(t, message)
}
