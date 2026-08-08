package service

import (
	"context"
	"errors"
	"testing"

	approvaldomain "github.com/example/adnova/internal/approval/domain"
	"github.com/example/adnova/internal/common/apperror"
	"github.com/stretchr/testify/require"
)

type fakeRepository struct {
	decision approvaldomain.Decision
	error    error
}

func (f *fakeRepository) List(context.Context, string, approvaldomain.Filter) ([]approvaldomain.Detail, error) {
	return nil, nil
}
func (f *fakeRepository) Get(context.Context, string, string) (*approvaldomain.Detail, error) {
	return &approvaldomain.Detail{}, nil
}
func (f *fakeRepository) Decide(_ context.Context, decision approvaldomain.Decision) (*approvaldomain.Detail, error) {
	f.decision = decision
	return &approvaldomain.Detail{}, f.error
}

func TestDecisionPolicy(t *testing.T) {
	tests := []struct {
		name     string
		roles    []string
		status   string
		comment  string
		expected error
	}{
		{name: "manager approves", roles: []string{"MANAGER"}, status: "APPROVED"},
		{name: "admin rejects with reason", roles: []string{"ADMIN"}, status: "REJECTED", comment: "数据证据不足"},
		{name: "operator forbidden", roles: []string{"OPERATOR"}, status: "APPROVED", expected: apperror.Forbidden},
		{name: "analyst forbidden", roles: []string{"ANALYST"}, status: "APPROVED", expected: apperror.Forbidden},
		{name: "system agent forbidden", roles: []string{"SYSTEM_AGENT", "MANAGER"}, status: "APPROVED", expected: apperror.Forbidden},
		{name: "reject requires reason", roles: []string{"MANAGER"}, status: "REJECTED", expected: apperror.Validation("rejection comment is required")},
		{name: "invalid state", roles: []string{"MANAGER"}, status: "CANCELLED", expected: apperror.Validation("approval decision must be APPROVED or REJECTED")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := &fakeRepository{}
			_, err := New(repo).Decide(context.Background(), approvaldomain.Decision{Status: test.status, Comment: test.comment}, test.roles)
			if test.expected == nil {
				require.NoError(t, err)
				require.Equal(t, test.status, repo.decision.Status)
				return
			}
			var actual *apperror.Error
			var expected *apperror.Error
			require.True(t, errors.As(err, &actual))
			require.True(t, errors.As(test.expected, &expected))
			require.Equal(t, expected.Code, actual.Code)
			require.Equal(t, expected.Message, actual.Message)
		})
	}
}
