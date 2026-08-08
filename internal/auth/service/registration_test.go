package service

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/example/adnova/internal/auth/dto"
	"github.com/example/adnova/internal/auth/repository"
	"github.com/example/adnova/internal/common/model"
	"github.com/example/adnova/internal/config"
	"github.com/stretchr/testify/require"
)

type memoryRegistrations struct {
	user  *model.User
	token *model.EmailVerificationToken
	roles []model.Role
}

func (m *memoryRegistrations) FindByTenantIDAndUsername(_ context.Context, tenantID, identifier string) (*model.User, error) {
	if m.user != nil && m.user.TenantID == tenantID && (m.user.Username == identifier || m.user.Email == identifier) {
		return m.user, nil
	}
	return nil, repository.ErrNotFound
}

func (m *memoryRegistrations) FindByID(_ context.Context, tenantID, userID string) (*model.User, error) {
	if m.user != nil && m.user.TenantID == tenantID && m.user.ID == userID {
		return m.user, nil
	}
	return nil, repository.ErrNotFound
}

func (m *memoryRegistrations) FindByEmailOrUsername(_ context.Context, tenantID, email, username string) (*model.User, error) {
	if m.user != nil && m.user.TenantID == tenantID && (m.user.Email == email || (username != "" && m.user.Username == username)) {
		return m.user, nil
	}
	return nil, repository.ErrNotFound
}

func (m *memoryRegistrations) CreatePendingRegistration(_ context.Context, user *model.User, token *model.EmailVerificationToken) error {
	m.user = user
	m.token = token
	return nil
}

func (m *memoryRegistrations) ReplaceVerificationToken(_ context.Context, _, _ string, token *model.EmailVerificationToken, _ time.Duration) (bool, error) {
	m.token = token
	return true, nil
}

func (m *memoryRegistrations) InvalidateVerificationToken(_ context.Context, _, tokenHash string, now time.Time) error {
	if m.token != nil && m.token.TokenHash == tokenHash {
		m.token.ConsumedAt = &now
	}
	return nil
}

func (m *memoryRegistrations) VerifyEmail(_ context.Context, tenantID, tokenHash string, now time.Time) (*model.User, error) {
	if m.token == nil || m.user == nil || m.token.TenantID != tenantID || m.token.TokenHash != tokenHash || m.token.ConsumedAt != nil {
		return nil, repository.ErrNotFound
	}
	if !m.token.ExpiresAt.After(now) {
		return nil, repository.ErrVerificationExpired
	}
	if m.user.Status != StatusPendingEmail {
		return nil, repository.ErrInvalidTransition
	}
	m.token.ConsumedAt = &now
	m.user.Status = StatusPendingApproval
	m.user.EmailVerifiedAt = &now
	return m.user, nil
}

func (m *memoryRegistrations) ListRegistrationApplications(_ context.Context, _ string, _ repository.RegistrationFilter) ([]model.User, error) {
	if m.user == nil {
		return nil, nil
	}
	return []model.User{*m.user}, nil
}

func (m *memoryRegistrations) ListRoles(_ context.Context) ([]model.Role, error) { return m.roles, nil }

func (m *memoryRegistrations) ApproveRegistration(_ context.Context, tenantID, userID, adminID string, roleCodes []string, now time.Time) (*model.User, error) {
	if m.user == nil || m.user.TenantID != tenantID || m.user.ID != userID {
		return nil, repository.ErrNotFound
	}
	if m.user.Status != StatusPendingApproval {
		return nil, repository.ErrInvalidTransition
	}
	m.user.Status = StatusActive
	m.user.ApprovedAt = &now
	m.user.ApprovedBy = adminID
	m.user.Roles = nil
	for _, code := range roleCodes {
		for _, role := range m.roles {
			if role.Code == code {
				m.user.Roles = append(m.user.Roles, role)
			}
		}
	}
	return m.user, nil
}

func (m *memoryRegistrations) RejectRegistration(_ context.Context, tenantID, userID, adminID, reason string, now time.Time) (*model.User, error) {
	if m.user == nil || m.user.TenantID != tenantID || m.user.ID != userID {
		return nil, repository.ErrNotFound
	}
	if m.user.Status != StatusPendingApproval {
		return nil, repository.ErrInvalidTransition
	}
	m.user.Status = StatusRejected
	m.user.RejectedAt = &now
	m.user.RejectedBy = adminID
	m.user.RejectionReason = reason
	return m.user, nil
}

type captureSender struct {
	recipient string
	link      string
	err       error
}

func (s *captureSender) SendVerification(_ context.Context, recipient, _ string, link string) error {
	s.recipient = recipient
	s.link = link
	return s.err
}

func TestMemberRegistrationEmailApprovalAndLogin(t *testing.T) {
	fixedNow := time.Date(2026, 8, 9, 5, 0, 0, 0, time.UTC)
	repo := &memoryRegistrations{roles: []model.Role{{ID: "role-viewer", Code: "VIEWER", Name: "只读用户"}}}
	sender := &captureSender{}
	svc := New(repo, config.JWTConfig{Secret: "test-secret", Issuer: "test", AccessTTL: time.Minute, RefreshTTL: time.Hour}, "tenant-1",
		WithRegistration(repo, sender, config.RegistrationConfig{
			Enabled: true, PublicBaseURL: "https://portal.example.com", VerificationTTL: time.Hour,
			ResendCooldown: time.Minute, AllowedEmailDomains: []string{"example.com"},
		}, nil),
	)
	svc.now = func() time.Time { return fixedNow }

	accepted, err := svc.Register(context.Background(), dto.RegisterRequest{
		Username: "alice.chen", Email: "Alice.Chen@example.com", DisplayName: "陈晓", Department: "市场部",
		JobTitle: "投放分析师", Password: "SecurePass!2026",
	}, RequestMetadata{})
	require.NoError(t, err)
	require.Equal(t, StatusPendingEmail, accepted.Status)
	require.Equal(t, "alice.chen@example.com", sender.recipient)
	require.Equal(t, StatusPendingEmail, repo.user.Status)

	verificationURL, err := url.Parse(sender.link)
	require.NoError(t, err)
	rawToken := verificationURL.Query().Get("token")
	require.NotEmpty(t, rawToken)
	require.NotEqual(t, rawToken, repo.token.TokenHash, "only the token hash may be persisted")

	verified, err := svc.VerifyEmail(context.Background(), rawToken, RequestMetadata{})
	require.NoError(t, err)
	require.Equal(t, StatusPendingApproval, verified.Status)
	_, err = svc.Login(context.Background(), dto.LoginRequest{Username: "alice.chen@example.com", Password: "SecurePass!2026"})
	require.ErrorContains(t, err, "等待管理员授权")

	approved, err := svc.ApproveRegistration(context.Background(), AdminActor{TenantID: "tenant-1", UserID: "admin-1"}, repo.user.ID, []string{"VIEWER"})
	require.NoError(t, err)
	require.Equal(t, StatusActive, approved.Status)
	require.Equal(t, []string{"VIEWER"}, approved.Roles)

	pair, err := svc.Login(context.Background(), dto.LoginRequest{Username: "alice.chen@example.com", Password: "SecurePass!2026"})
	require.NoError(t, err)
	require.NotEmpty(t, pair.AccessToken)
}

func TestMemberRegistrationRejectsExternalDomain(t *testing.T) {
	repo := &memoryRegistrations{}
	svc := New(repo, config.JWTConfig{}, "tenant-1", WithRegistration(repo, &captureSender{}, config.RegistrationConfig{
		Enabled: true, PublicBaseURL: "https://portal.example.com", VerificationTTL: time.Hour,
		ResendCooldown: time.Minute, AllowedEmailDomains: []string{"example.com"},
	}, nil))

	_, err := svc.Register(context.Background(), dto.RegisterRequest{
		Username: "outsider", Email: "outsider@public.test", DisplayName: "外部用户", Department: "外部",
		Password: "SecurePass!2026",
	}, RequestMetadata{})
	require.ErrorContains(t, err, "公司邮箱")
	require.Nil(t, repo.user)
}

func TestMemberRegistrationRejectsBlankOrUnsafeIdentityFields(t *testing.T) {
	repo := &memoryRegistrations{}
	svc := New(repo, config.JWTConfig{}, "tenant-1", WithRegistration(repo, &captureSender{}, config.RegistrationConfig{
		Enabled: true, PublicBaseURL: "https://portal.example.com", VerificationTTL: time.Hour,
		ResendCooldown: time.Minute, AllowedEmailDomains: []string{"example.com"},
	}, nil))

	for name, request := range map[string]dto.RegisterRequest{
		"blank department": {Username: "alice", Email: "alice@example.com", DisplayName: "陈晓", Department: "  ", Password: "SecurePass!2026"},
		"header break":     {Username: "alice", Email: "alice@example.com", DisplayName: "陈晓\nBcc", Department: "市场部", Password: "SecurePass!2026"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := svc.Register(context.Background(), request, RequestMetadata{})
			require.Error(t, err)
			require.Nil(t, repo.user)
		})
	}
}

func TestMemberRegistrationInvalidTokenDoesNotRevealUser(t *testing.T) {
	repo := &memoryRegistrations{}
	svc := New(repo, config.JWTConfig{}, "tenant-1", WithRegistration(repo, &captureSender{}, config.RegistrationConfig{
		Enabled: true, PublicBaseURL: "https://portal.example.com", VerificationTTL: time.Hour, ResendCooldown: time.Minute,
	}, nil))
	_, err := svc.VerifyEmail(context.Background(), "not-a-real-token-not-a-real-token", RequestMetadata{})
	require.Error(t, err)
	require.True(t, errors.Is(err, verificationInvalid) || err.Error() == verificationInvalid.Error())
}

func TestRegistrationConfigUsesEmptyArrayWithoutDomainAllowlist(t *testing.T) {
	repo := &memoryRegistrations{}
	svc := New(repo, config.JWTConfig{}, "tenant-1", WithRegistration(repo, &captureSender{}, config.RegistrationConfig{Enabled: true}, nil))

	result := svc.RegistrationConfig()
	require.True(t, result.Enabled)
	require.NotNil(t, result.AllowedEmailDomains)
	require.Empty(t, result.AllowedEmailDomains)
}
