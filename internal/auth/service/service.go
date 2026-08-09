package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	netmail "net/mail"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	auditdomain "github.com/example/adnova/internal/audit/domain"
	auditservice "github.com/example/adnova/internal/audit/service"
	"github.com/example/adnova/internal/auth/dto"
	"github.com/example/adnova/internal/auth/email"
	authrepo "github.com/example/adnova/internal/auth/repository"
	"github.com/example/adnova/internal/common/apperror"
	"github.com/example/adnova/internal/common/model"
	"github.com/example/adnova/internal/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	StatusPendingEmail    = "PENDING_EMAIL"
	StatusPendingApproval = "PENDING_APPROVAL"
	StatusActive          = "ACTIVE"
	StatusRejected        = "REJECTED"
	StatusDisabled        = "DISABLED"
)

var (
	usernamePattern           = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{2,79}$`)
	registrationUnavailable   = apperror.New(10110, 403, "member registration is not available")
	emailVerificationRequired = apperror.New(10111, 403, "请先完成公司邮箱验证")
	approvalPending           = apperror.New(10112, 403, "邮箱已验证，正在等待管理员授权")
	registrationRejected      = apperror.New(10113, 403, "成员申请未获批准，请联系管理员")
	accountDisabled           = apperror.New(10114, 403, "账号已停用，请联系管理员")
	verificationInvalid       = apperror.New(10115, 400, "验证链接无效或已使用")
	verificationExpired       = apperror.New(10116, 400, "验证链接已过期，请重新发送")
	registrationTransition    = apperror.New(10117, 409, "registration status has changed")
)

type Claims struct {
	TenantID string   `json:"tenant_id"`
	Roles    []string `json:"roles"`
	Type     string   `json:"type"`
	jwt.RegisteredClaims
}

type RequestMetadata struct {
	RequestID string
	TraceID   string
	IPAddress string
}

type AdminActor struct {
	TenantID string
	UserID   string
	RequestMetadata
}

type Option func(*Service)

func WithRegistration(repo authrepo.RegistrationRepository, sender email.Sender, cfg config.RegistrationConfig, auditor auditservice.Recorder) Option {
	return func(service *Service) {
		service.registrations = repo
		service.sender = sender
		service.registrationCfg = cfg
		service.auditor = auditor
	}
}

type Service struct {
	users           authrepo.UserRepository
	cfg             config.JWTConfig
	defaultTenantID string
	registrations   authrepo.RegistrationRepository
	sender          email.Sender
	registrationCfg config.RegistrationConfig
	auditor         auditservice.Recorder
	now             func() time.Time
}

func New(users authrepo.UserRepository, cfg config.JWTConfig, defaultTenantID string, options ...Option) *Service {
	service := &Service{users: users, cfg: cfg, defaultTenantID: defaultTenantID, now: func() time.Time { return time.Now().UTC() }}
	for _, option := range options {
		option(service)
	}
	return service
}

func (s *Service) Login(ctx context.Context, req dto.LoginRequest) (*dto.TokenPair, error) {
	identifier := strings.ToLower(strings.TrimSpace(req.Username))
	user, err := s.users.FindByTenantIDAndUsername(ctx, s.defaultTenantID, identifier)
	if err != nil || user == nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		return nil, apperror.Unauthorized
	}
	switch user.Status {
	case StatusActive:
		return s.issuePair(user)
	case StatusPendingEmail:
		return nil, emailVerificationRequired
	case StatusPendingApproval:
		return nil, approvalPending
	case StatusRejected:
		return nil, registrationRejected
	default:
		return nil, accountDisabled
	}
}

func (s *Service) RegistrationConfig() dto.RegistrationConfig {
	return dto.RegistrationConfig{Enabled: s.registrationCfg.Enabled, AllowedEmailDomains: append([]string{}, s.registrationCfg.AllowedEmailDomains...)}
}

func (s *Service) Register(ctx context.Context, req dto.RegisterRequest, metadata RequestMetadata) (*dto.RegistrationAccepted, error) {
	if err := s.ensureRegistrationAvailable(); err != nil {
		return nil, err
	}
	username := strings.ToLower(strings.TrimSpace(req.Username))
	emailAddress := strings.ToLower(strings.TrimSpace(req.Email))
	displayName := strings.TrimSpace(req.DisplayName)
	department := strings.TrimSpace(req.Department)
	jobTitle := strings.TrimSpace(req.JobTitle)
	if !usernamePattern.MatchString(username) {
		return nil, apperror.Validation("用户名只能包含小写字母、数字、点、下划线或连字符，且必须以字母或数字开头")
	}
	if !s.emailDomainAllowed(emailAddress) {
		return nil, apperror.Validation("请使用公司邮箱申请")
	}
	parsedEmail, err := netmail.ParseAddress(emailAddress)
	if err != nil || parsedEmail.Address != emailAddress || len([]rune(emailAddress)) > 254 {
		return nil, apperror.Validation("请输入有效的公司邮箱")
	}
	if length := len([]rune(displayName)); length < 2 || length > 120 {
		return nil, apperror.Validation("姓名长度应为 2 到 120 个字符")
	}
	if length := len([]rune(department)); length < 2 || length > 120 {
		return nil, apperror.Validation("所属部门长度应为 2 到 120 个字符")
	}
	if len([]rune(jobTitle)) > 120 {
		return nil, apperror.Validation("职位不能超过 120 个字符")
	}
	if containsControl(displayName) || containsControl(department) || containsControl(jobTitle) {
		return nil, apperror.Validation("成员资料不能包含控制字符")
	}
	if err := validatePassword(req.Password); err != nil {
		return nil, err
	}

	existing, err := s.registrations.FindByEmailOrUsername(ctx, s.defaultTenantID, emailAddress, username)
	if err == nil && existing != nil {
		if existing.Email == emailAddress && existing.Status == StatusPendingEmail {
			if err := s.rotateAndSend(ctx, existing); err != nil {
				return nil, err
			}
		}
		return genericRegistrationAccepted(), nil
	}
	if err != nil && !errors.Is(err, authrepo.ErrNotFound) {
		return nil, fmt.Errorf("check registration uniqueness: %w", err)
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash registration password: %w", err)
	}
	rawToken, tokenHash, err := newVerificationToken()
	if err != nil {
		return nil, fmt.Errorf("generate verification token: %w", err)
	}
	now := s.now()
	user := &model.User{
		ID: uuid.NewString(), TenantID: s.defaultTenantID, Username: username, Email: emailAddress,
		DisplayName: displayName, Department: department, JobTitle: jobTitle,
		PasswordHash: string(passwordHash), Status: StatusPendingEmail,
	}
	token := &model.EmailVerificationToken{
		ID: uuid.NewString(), TenantID: s.defaultTenantID, UserID: user.ID, TokenHash: tokenHash,
		ExpiresAt: now.Add(s.registrationCfg.VerificationTTL), CreatedAt: now,
	}
	if err := s.registrations.CreatePendingRegistration(ctx, user, token); err != nil {
		// A concurrent request may win the unique username/email race after the
		// initial lookup. Return the same non-enumerating response in that case.
		winner, lookupErr := s.registrations.FindByEmailOrUsername(ctx, s.defaultTenantID, emailAddress, username)
		if lookupErr == nil && winner != nil {
			if winner.Email == emailAddress && winner.Status == StatusPendingEmail {
				if resendErr := s.rotateAndSend(ctx, winner); resendErr != nil {
					return nil, resendErr
				}
			}
			return genericRegistrationAccepted(), nil
		}
		return nil, fmt.Errorf("create member registration: %w", err)
	}
	if err := s.sendVerification(ctx, user, rawToken); err != nil {
		_ = s.registrations.InvalidateVerificationToken(ctx, s.defaultTenantID, tokenHash, s.now())
		return nil, err
	}
	if err := s.record(ctx, auditdomain.RecordInput{
		TenantID: user.TenantID, ActorID: user.ID, ActorType: "REGISTRATION", Action: "MEMBER_REGISTERED",
		ResourceType: "USER", ResourceID: user.ID, After: safeRegistration(user),
		Metadata: map[string]any{"email_domain": emailDomain(user.Email)}, RequestID: metadata.RequestID, TraceID: metadata.TraceID, IPAddress: metadata.IPAddress,
	}); err != nil {
		return nil, err
	}
	return genericRegistrationAccepted(), nil
}

func (s *Service) ResendVerification(ctx context.Context, emailAddress string) (*dto.RegistrationAccepted, error) {
	if err := s.ensureRegistrationAvailable(); err != nil {
		return nil, err
	}
	emailAddress = strings.ToLower(strings.TrimSpace(emailAddress))
	user, err := s.registrations.FindByEmailOrUsername(ctx, s.defaultTenantID, emailAddress, "")
	if errors.Is(err, authrepo.ErrNotFound) || user == nil {
		return genericRegistrationAccepted(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("find pending registration: %w", err)
	}
	if user.Email == emailAddress && user.Status == StatusPendingEmail {
		if err := s.rotateAndSend(ctx, user); err != nil {
			return nil, err
		}
	}
	return genericRegistrationAccepted(), nil
}

func (s *Service) VerifyEmail(ctx context.Context, rawToken string, metadata RequestMetadata) (*dto.RegistrationAccepted, error) {
	if err := s.ensureRegistrationAvailable(); err != nil {
		return nil, err
	}
	tokenHash := hashToken(strings.TrimSpace(rawToken))
	user, err := s.registrations.VerifyEmail(ctx, s.defaultTenantID, tokenHash, s.now())
	switch {
	case errors.Is(err, authrepo.ErrVerificationExpired):
		return nil, verificationExpired
	case errors.Is(err, authrepo.ErrNotFound), errors.Is(err, authrepo.ErrInvalidTransition):
		return nil, verificationInvalid
	case err != nil:
		return nil, fmt.Errorf("verify member email: %w", err)
	}
	if err := s.record(ctx, auditdomain.RecordInput{
		TenantID: user.TenantID, ActorID: user.ID, ActorType: "REGISTRATION", Action: "MEMBER_EMAIL_VERIFIED",
		ResourceType: "USER", ResourceID: user.ID, After: safeRegistration(user),
		RequestID: metadata.RequestID, TraceID: metadata.TraceID, IPAddress: metadata.IPAddress,
	}); err != nil {
		return nil, err
	}
	return &dto.RegistrationAccepted{Status: StatusPendingApproval, Message: "公司邮箱已确认，请等待管理员授权"}, nil
}

func (s *Service) ListRegistrationApplications(ctx context.Context, tenantID, status string) ([]dto.RegistrationApplication, error) {
	if s.registrations == nil {
		return nil, registrationUnavailable
	}
	status = strings.ToUpper(strings.TrimSpace(status))
	if status != "" && !validRegistrationStatus(status) {
		return nil, apperror.InvalidArgument
	}
	users, err := s.registrations.ListRegistrationApplications(ctx, tenantID, authrepo.RegistrationFilter{Status: status, Limit: 200})
	if err != nil {
		return nil, fmt.Errorf("list registration applications: %w", err)
	}
	result := make([]dto.RegistrationApplication, 0, len(users))
	for index := range users {
		result = append(result, toRegistrationApplication(&users[index]))
	}
	return result, nil
}

func (s *Service) ListRoles(ctx context.Context) ([]dto.RoleOption, error) {
	roles, err := s.registrations.ListRoles(ctx)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	result := make([]dto.RoleOption, 0, len(roles))
	for _, role := range roles {
		result = append(result, dto.RoleOption{Code: role.Code, Name: role.Name, Description: role.Description})
	}
	return result, nil
}

func (s *Service) ApproveRegistration(ctx context.Context, actor AdminActor, userID string, roleCodes []string) (*dto.RegistrationApplication, error) {
	roleCodes, err := normalizeRoles(roleCodes)
	if err != nil {
		return nil, err
	}
	user, err := s.registrations.ApproveRegistration(ctx, actor.TenantID, userID, actor.UserID, roleCodes, s.now())
	if errors.Is(err, authrepo.ErrNotFound) {
		return nil, apperror.NotFound
	}
	if errors.Is(err, authrepo.ErrInvalidTransition) {
		return nil, registrationTransition
	}
	if err != nil {
		return nil, fmt.Errorf("approve member registration: %w", err)
	}
	if err := s.record(ctx, auditdomain.RecordInput{
		TenantID: actor.TenantID, ActorID: actor.UserID, ActorType: "USER", Action: "MEMBER_REGISTRATION_APPROVED",
		ResourceType: "USER", ResourceID: user.ID, After: safeRegistration(user), Metadata: map[string]any{"roles": roleCodes},
		RequestID: actor.RequestID, TraceID: actor.TraceID, IPAddress: actor.IPAddress,
	}); err != nil {
		return nil, err
	}
	result := toRegistrationApplication(user)
	return &result, nil
}

func (s *Service) RejectRegistration(ctx context.Context, actor AdminActor, userID, reason string) (*dto.RegistrationApplication, error) {
	reason = strings.TrimSpace(reason)
	if len([]rune(reason)) < 2 || len([]rune(reason)) > 500 {
		return nil, apperror.InvalidArgument
	}
	user, err := s.registrations.RejectRegistration(ctx, actor.TenantID, userID, actor.UserID, reason, s.now())
	if errors.Is(err, authrepo.ErrNotFound) {
		return nil, apperror.NotFound
	}
	if errors.Is(err, authrepo.ErrInvalidTransition) {
		return nil, registrationTransition
	}
	if err != nil {
		return nil, fmt.Errorf("reject member registration: %w", err)
	}
	if err := s.record(ctx, auditdomain.RecordInput{
		TenantID: actor.TenantID, ActorID: actor.UserID, ActorType: "USER", Action: "MEMBER_REGISTRATION_REJECTED",
		ResourceType: "USER", ResourceID: user.ID, After: safeRegistration(user), Metadata: map[string]any{"reason": reason},
		RequestID: actor.RequestID, TraceID: actor.TraceID, IPAddress: actor.IPAddress,
	}); err != nil {
		return nil, err
	}
	result := toRegistrationApplication(user)
	return &result, nil
}

func (s *Service) Refresh(ctx context.Context, raw string) (*dto.TokenPair, error) {
	claims, err := s.Parse(raw, "refresh")
	if err != nil {
		return nil, apperror.Unauthorized
	}
	user, err := s.users.FindByID(ctx, claims.TenantID, claims.Subject)
	if err != nil || user.Status != StatusActive {
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
	now := s.now()
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

func (s *Service) rotateAndSend(ctx context.Context, user *model.User) error {
	rawToken, tokenHash, err := newVerificationToken()
	if err != nil {
		return fmt.Errorf("generate verification token: %w", err)
	}
	now := s.now()
	token := &model.EmailVerificationToken{ID: uuid.NewString(), TenantID: user.TenantID, UserID: user.ID, TokenHash: tokenHash, ExpiresAt: now.Add(s.registrationCfg.VerificationTTL), CreatedAt: now}
	created, err := s.registrations.ReplaceVerificationToken(ctx, user.TenantID, user.ID, token, s.registrationCfg.ResendCooldown)
	if err != nil {
		return fmt.Errorf("rotate verification token: %w", err)
	}
	if !created {
		return nil
	}
	if err := s.sendVerification(ctx, user, rawToken); err != nil {
		_ = s.registrations.InvalidateVerificationToken(ctx, user.TenantID, tokenHash, s.now())
		return err
	}
	return nil
}

func (s *Service) sendVerification(ctx context.Context, user *model.User, rawToken string) error {
	verificationURL := s.registrationCfg.PublicBaseURL + "/verify-email?token=" + url.QueryEscape(rawToken)
	if err := s.sender.SendVerification(ctx, user.Email, user.DisplayName, verificationURL); err != nil {
		return fmt.Errorf("send verification email: %w", err)
	}
	return nil
}

func (s *Service) ensureRegistrationAvailable() error {
	if !s.registrationCfg.Enabled || s.registrations == nil || s.sender == nil {
		return registrationUnavailable
	}
	return nil
}

func (s *Service) emailDomainAllowed(emailAddress string) bool {
	domain := emailDomain(emailAddress)
	if domain == "" {
		return false
	}
	if len(s.registrationCfg.AllowedEmailDomains) == 0 {
		return true
	}
	for _, allowed := range s.registrationCfg.AllowedEmailDomains {
		if domain == allowed {
			return true
		}
	}
	return false
}

func (s *Service) record(ctx context.Context, input auditdomain.RecordInput) error {
	if s.auditor == nil {
		return nil
	}
	if err := s.auditor.Record(ctx, input); err != nil {
		return fmt.Errorf("record registration audit: %w", err)
	}
	return nil
}

func newVerificationToken() (string, string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", err
	}
	raw := base64.RawURLEncoding.EncodeToString(bytes)
	return raw, hashToken(raw), nil
}

func hashToken(raw string) string {
	digest := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(digest[:])
}

func validatePassword(password string) error {
	var upper, lower, digit, symbol bool
	for _, character := range password {
		switch {
		case unicode.IsUpper(character):
			upper = true
		case unicode.IsLower(character):
			lower = true
		case unicode.IsDigit(character):
			digit = true
		default:
			symbol = true
		}
	}
	if length := len([]rune(password)); length < 12 || length > 128 || !upper || !lower || !digit || !symbol {
		return apperror.Validation("密码至少 12 位，并包含大写字母、小写字母、数字和特殊字符")
	}
	return nil
}

func containsControl(value string) bool {
	for _, character := range value {
		if unicode.IsControl(character) {
			return true
		}
	}
	return false
}

func normalizeRoles(values []string) ([]string, error) {
	allowed := map[string]bool{"ADMIN": true, "MANAGER": true, "OPERATOR": true, "ANALYST": true, "VIEWER": true}
	seen := make(map[string]bool)
	result := make([]string, 0, len(values))
	for _, value := range values {
		role := strings.ToUpper(strings.TrimSpace(value))
		if !allowed[role] || seen[role] {
			return nil, apperror.InvalidArgument
		}
		seen[role] = true
		result = append(result, role)
	}
	if len(result) == 0 {
		return nil, apperror.InvalidArgument
	}
	sort.Strings(result)
	return result, nil
}

func genericRegistrationAccepted() *dto.RegistrationAccepted {
	return &dto.RegistrationAccepted{Status: StatusPendingEmail, Message: "如果申请信息有效，确认邮件将发送到公司邮箱"}
}

func validRegistrationStatus(status string) bool {
	switch status {
	case StatusPendingEmail, StatusPendingApproval, StatusActive, StatusRejected, StatusDisabled:
		return true
	default:
		return false
	}
}

func emailDomain(emailAddress string) string {
	index := strings.LastIndex(emailAddress, "@")
	if index <= 0 || index == len(emailAddress)-1 {
		return ""
	}
	return strings.ToLower(emailAddress[index+1:])
}

func safeRegistration(user *model.User) map[string]any {
	return map[string]any{
		"id": user.ID, "username": user.Username, "email": user.Email, "display_name": user.DisplayName,
		"department": user.Department, "job_title": user.JobTitle, "status": user.Status,
		"email_verified_at": user.EmailVerifiedAt, "approved_at": user.ApprovedAt, "approved_by": user.ApprovedBy,
		"rejected_at": user.RejectedAt, "rejected_by": user.RejectedBy,
	}
}

func toRegistrationApplication(user *model.User) dto.RegistrationApplication {
	roles := make([]string, 0, len(user.Roles))
	for _, role := range user.Roles {
		roles = append(roles, role.Code)
	}
	return dto.RegistrationApplication{
		ID: user.ID, Username: user.Username, Email: user.Email, DisplayName: user.DisplayName,
		Department: user.Department, JobTitle: user.JobTitle, Status: user.Status, Roles: roles,
		EmailVerifiedAt: user.EmailVerifiedAt, ApprovedAt: user.ApprovedAt, ApprovedBy: user.ApprovedBy,
		RejectedAt: user.RejectedAt, RejectedBy: user.RejectedBy, RejectionReason: user.RejectionReason, CreatedAt: user.CreatedAt,
	}
}

func toCurrentUser(user *model.User) *dto.CurrentUser {
	roles := make([]string, 0, len(user.Roles))
	for _, role := range user.Roles {
		roles = append(roles, role.Code)
	}
	return &dto.CurrentUser{ID: user.ID, TenantID: user.TenantID, Username: user.Username, Email: user.Email, DisplayName: user.DisplayName, Roles: roles}
}
