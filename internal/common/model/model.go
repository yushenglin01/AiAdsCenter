package model

import "time"

type Tenant struct {
	ID        string    `gorm:"type:char(36);primaryKey" json:"id"`
	Slug      string    `gorm:"size:80;uniqueIndex;not null" json:"slug"`
	Name      string    `gorm:"size:160;not null" json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type User struct {
	ID              string     `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID        string     `gorm:"type:char(36);not null;uniqueIndex:uidx_user_tenant_username;uniqueIndex:uidx_user_tenant_email" json:"tenant_id"`
	Username        string     `gorm:"size:80;not null;uniqueIndex:uidx_user_tenant_username" json:"username"`
	Email           string     `gorm:"size:254;not null;uniqueIndex:uidx_user_tenant_email" json:"email"`
	DisplayName     string     `gorm:"size:120;not null" json:"display_name"`
	Department      string     `gorm:"size:120;not null" json:"department"`
	JobTitle        string     `gorm:"size:120;not null" json:"job_title"`
	PasswordHash    string     `gorm:"size:255;not null" json:"-"`
	Status          string     `gorm:"size:20;not null" json:"status"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
	ApprovedAt      *time.Time `json:"approved_at,omitempty"`
	ApprovedBy      string     `gorm:"type:char(36)" json:"approved_by,omitempty"`
	RejectedAt      *time.Time `json:"rejected_at,omitempty"`
	RejectedBy      string     `gorm:"type:char(36)" json:"rejected_by,omitempty"`
	RejectionReason string     `gorm:"size:500;not null" json:"rejection_reason,omitempty"`
	Roles           []Role     `gorm:"many2many:user_roles" json:"roles,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type EmailVerificationToken struct {
	ID         string     `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID   string     `gorm:"type:char(36);not null;index" json:"tenant_id"`
	UserID     string     `gorm:"type:char(36);not null;index" json:"user_id"`
	TokenHash  string     `gorm:"type:char(64);not null;uniqueIndex" json:"-"`
	ExpiresAt  time.Time  `gorm:"not null;index" json:"expires_at"`
	ConsumedAt *time.Time `json:"consumed_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

type Role struct {
	ID          string    `gorm:"type:char(36);primaryKey" json:"id"`
	Code        string    `gorm:"size:40;uniqueIndex;not null" json:"code"`
	Name        string    `gorm:"size:80;not null" json:"name"`
	Description string    `gorm:"size:255" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type UserRole struct {
	TenantID string `gorm:"type:char(36);primaryKey"`
	UserID   string `gorm:"type:char(36);primaryKey"`
	RoleID   string `gorm:"type:char(36);primaryKey"`
}

func (Tenant) TableName() string                 { return "tenants" }
func (User) TableName() string                   { return "users" }
func (Role) TableName() string                   { return "roles" }
func (UserRole) TableName() string               { return "user_roles" }
func (EmailVerificationToken) TableName() string { return "email_verification_tokens" }
