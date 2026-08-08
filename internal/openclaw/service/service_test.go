package service

import (
	"context"
	"testing"

	approvaldomain "github.com/example/adnova/internal/approval/domain"
	notificationdomain "github.com/example/adnova/internal/notification/domain"
	workflowdomain "github.com/example/adnova/internal/workflow/domain"
	workflowservice "github.com/example/adnova/internal/workflow/service"
	"github.com/stretchr/testify/require"
)

type fakeWorkflow struct{}

func (fakeWorkflow) Start(_ context.Context, tenantID, _, _ string, input workflowservice.StartInput) (*workflowdomain.Details, error) {
	return &workflowdomain.Details{Run: workflowdomain.Run{ID: "workflow-1", TenantID: tenantID, GameID: input.GameID}}, nil
}
func (fakeWorkflow) Get(_ context.Context, tenantID, id string) (*workflowdomain.Details, error) {
	return &workflowdomain.Details{Run: workflowdomain.Run{ID: id, TenantID: tenantID}}, nil
}

type fakeApprovals struct{}

func (fakeApprovals) List(context.Context, string, approvaldomain.Filter) ([]approvaldomain.Detail, error) {
	return []approvaldomain.Detail{}, nil
}

type fakeNotifications struct{}

func (fakeNotifications) List(context.Context, string, string) ([]notificationdomain.Notification, error) {
	return []notificationdomain.Notification{}, nil
}
func (fakeNotifications) MarkRead(_ context.Context, tenantID, id, userID string) (*notificationdomain.Notification, error) {
	return &notificationdomain.Notification{ID: id, TenantID: tenantID, ReadBy: userID}, nil
}

func TestExecuteSupportsWorkflowAndInboxCommands(t *testing.T) {
	service := New(fakeWorkflow{}, fakeApprovals{}, fakeNotifications{})
	actor := Actor{TenantID: "tenant-1", UserID: "user-1"}
	started, err := service.Execute(context.Background(), actor, Command{Intent: IntentRunFullAnalysis, Input: Input{GameID: "game-1", CampaignID: "campaign-1"}})
	require.NoError(t, err)
	require.True(t, started.Accepted)

	marked, err := service.Execute(context.Background(), actor, Command{Intent: IntentMarkNotificationRead, Input: Input{NotificationID: "notification-1"}})
	require.NoError(t, err)
	require.Equal(t, "SUCCEEDED", marked.Status)
}

func TestExecuteRejectsUnknownIntent(t *testing.T) {
	service := New(fakeWorkflow{}, fakeApprovals{}, fakeNotifications{})
	_, err := service.Execute(context.Background(), Actor{}, Command{Intent: "EXECUTE_AD_CHANGE"})
	require.Error(t, err)
}
