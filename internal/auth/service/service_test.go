package service

import (
	"context"
	"testing"
	"time"

	"github.com/example/adnova/internal/auth/dto"
	authrepo "github.com/example/adnova/internal/auth/repository"
	"github.com/example/adnova/internal/common/model"
	"github.com/example/adnova/internal/config"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

type fakeUsers struct{ users []model.User }

func (f *fakeUsers) FindByTenantIDAndUsername(_ context.Context, tenantID, username string) (*model.User, error) {
	for i := range f.users {
		if f.users[i].TenantID == tenantID && f.users[i].Username == username {
			return &f.users[i], nil
		}
	}
	return nil, authrepo.ErrNotFound
}

func (f *fakeUsers) FindByID(_ context.Context, tenantID, userID string) (*model.User, error) {
	for i := range f.users {
		if f.users[i].TenantID == tenantID && f.users[i].ID == userID {
			return &f.users[i], nil
		}
	}
	return nil, authrepo.ErrNotFound
}

func TestLoginAndRefresh(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("Demo@123456"), bcrypt.MinCost)
	require.NoError(t, err)
	repo := &fakeUsers{users: []model.User{{
		ID: "user-1", TenantID: "tenant-1", Username: "admin", PasswordHash: string(hash), Status: "ACTIVE",
		Roles: []model.Role{{Code: "ADMIN"}},
	}}}
	svc := New(repo, config.JWTConfig{Secret: "test-secret", Issuer: "test", AccessTTL: time.Minute, RefreshTTL: time.Hour}, "tenant-1")

	pair, err := svc.Login(context.Background(), dto.LoginRequest{Username: "admin", Password: "Demo@123456"})
	require.NoError(t, err)
	claims, err := svc.Parse(pair.AccessToken, "access")
	require.NoError(t, err)
	require.Equal(t, "tenant-1", claims.TenantID)
	require.Equal(t, []string{"ADMIN"}, claims.Roles)

	refreshed, err := svc.Refresh(context.Background(), pair.RefreshToken)
	require.NoError(t, err)
	require.NotEmpty(t, refreshed.AccessToken)
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("Demo@123456"), bcrypt.MinCost)
	require.NoError(t, err)
	svc := New(&fakeUsers{users: []model.User{{ID: "user-1", TenantID: "tenant-1", Username: "admin", PasswordHash: string(hash), Status: "ACTIVE"}}}, config.JWTConfig{Secret: "test", Issuer: "test", AccessTTL: time.Minute, RefreshTTL: time.Hour}, "tenant-1")
	_, err = svc.Login(context.Background(), dto.LoginRequest{Username: "admin", Password: "incorrect"})
	require.Error(t, err)
}

func TestCurrentUserIsTenantIsolated(t *testing.T) {
	repo := &fakeUsers{users: []model.User{{ID: "same-resource-id", TenantID: "tenant-a", Username: "admin", Status: "ACTIVE"}}}
	svc := New(repo, config.JWTConfig{}, "tenant-a")
	_, err := svc.CurrentUser(context.Background(), "tenant-b", "same-resource-id")
	require.Error(t, err, "a user id from another tenant must remain invisible")
}
