package dto

import "time"

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required,min=8,max=128"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

type CurrentUser struct {
	ID          string   `json:"id"`
	TenantID    string   `json:"tenant_id"`
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	DisplayName string   `json:"display_name"`
	Roles       []string `json:"roles"`
}

type RegisterRequest struct {
	Username    string `json:"username" binding:"required,min=3,max=80"`
	Email       string `json:"email" binding:"required,email,max=254"`
	DisplayName string `json:"display_name" binding:"required,min=2,max=120"`
	Department  string `json:"department" binding:"required,min=2,max=120"`
	JobTitle    string `json:"job_title" binding:"max=120"`
	Password    string `json:"password" binding:"required,min=12,max=128"`
}

type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required,min=32,max=256"`
}

type ResendVerificationRequest struct {
	Email string `json:"email" binding:"required,email,max=254"`
}

type RegistrationAccepted struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type RegistrationConfig struct {
	Enabled             bool     `json:"enabled"`
	AllowedEmailDomains []string `json:"allowed_email_domains"`
}

type RegistrationApplication struct {
	ID              string     `json:"id"`
	Username        string     `json:"username"`
	Email           string     `json:"email"`
	DisplayName     string     `json:"display_name"`
	Department      string     `json:"department"`
	JobTitle        string     `json:"job_title"`
	Status          string     `json:"status"`
	Roles           []string   `json:"roles"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
	ApprovedAt      *time.Time `json:"approved_at,omitempty"`
	ApprovedBy      string     `json:"approved_by,omitempty"`
	RejectedAt      *time.Time `json:"rejected_at,omitempty"`
	RejectedBy      string     `json:"rejected_by,omitempty"`
	RejectionReason string     `json:"rejection_reason,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

type RoleOption struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ApproveRegistrationRequest struct {
	Roles []string `json:"roles" binding:"required,min=1,max=5,dive,required"`
}

type RejectRegistrationRequest struct {
	Reason string `json:"reason" binding:"required,min=2,max=500"`
}
