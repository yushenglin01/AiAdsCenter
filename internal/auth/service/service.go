package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/example/adnova/internal/auth/dto"
	authrepo "github.com/example/adnova/internal/auth/repository"
	"github.com/example/adnova/internal/common/apperror"
	"github.com/example/adnova/internal/common/model"
	"github.com/example/adnova/internal/config"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Claims struct {
	TenantID string   `json:"tenant_id"`
	Roles    []string `json:"roles"`
	Type     string   `json:"type"`
	jwt.RegisteredClaims
}

type Service struct {
	users           authrepo.UserRepository
	cfg             config.JWTConfig
	defaultTenantID string
}

func New(users authrepo.UserRepository, cfg config.JWTConfig, defaultTenantID string) *Service {
	return &Service{users: users, cfg: cfg, defaultTenantID: defaultTenantID}
}

func (s *Service) Login(ctx context.Context, req dto.LoginRequest) (*dto.TokenPair, error) {
	user, err := s.users.FindByTenantIDAndUsername(ctx, s.defaultTenantID, req.Username)
	if err != nil || user == nil || user.Status != "ACTIVE" || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		return nil, apperror.Unauthorized
	}
	return s.issuePair(user)
}

func (s *Service) Refresh(ctx context.Context, raw string) (*dto.TokenPair, error) {
	claims, err := s.Parse(raw, "refresh")
	if err != nil {
		return nil, apperror.Unauthorized
	}
	user, err := s.users.FindByID(ctx, claims.TenantID, claims.Subject)
	if err != nil || user.Status != "ACTIVE" {
		return nil, apperror.Unauthorized
	}
	return s.issuePair(user)
}

func (s *Service) CurrentUser(ctx context.Context, tenantID, userID string) (*dto.CurrentUser, error) {
	user, err := s.users.FindByID(ctx, tenantID, userID)
	if errors.Is(err, authrepo.ErrNotFound) {
		return nil, apperror.NotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find current user: %w", err)
	}
	return toCurrentUser(user), nil
}

func (s *Service) Parse(raw, expectedType string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(raw, &Claims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.cfg.Secret), nil
	}, jwt.WithIssuer(s.cfg.Issuer), jwt.WithExpirationRequired())
	if err != nil || !token.Valid {
		return nil, apperror.Unauthorized
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || claims.Type != expectedType || claims.Subject == "" || claims.TenantID == "" {
		return nil, apperror.Unauthorized
	}
	return claims, nil
}

func (s *Service) issuePair(user *model.User) (*dto.TokenPair, error) {
	roles := make([]string, 0, len(user.Roles))
	for _, role := range user.Roles {
		roles = append(roles, role.Code)
	}
	access, err := s.sign(user, roles, "access", s.cfg.AccessTTL)
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}
	refresh, err := s.sign(user, roles, "refresh", s.cfg.RefreshTTL)
	if err != nil {
		return nil, fmt.Errorf("sign refresh token: %w", err)
	}
	return &dto.TokenPair{AccessToken: access, RefreshToken: refresh, TokenType: "Bearer", ExpiresIn: int64(s.cfg.AccessTTL.Seconds())}, nil
}

func (s *Service) sign(user *model.User, roles []string, tokenType string, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		TenantID: user.TenantID,
		Roles:    roles,
		Type:     tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: s.cfg.Issuer, Subject: user.ID,
			IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.cfg.Secret))
}

func toCurrentUser(user *model.User) *dto.CurrentUser {
	roles := make([]string, 0, len(user.Roles))
	for _, role := range user.Roles {
		roles = append(roles, role.Code)
	}
	return &dto.CurrentUser{ID: user.ID, TenantID: user.TenantID, Username: user.Username, DisplayName: user.DisplayName, Roles: roles}
}
