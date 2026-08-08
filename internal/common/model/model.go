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
	ID           string    `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID     string    `gorm:"type:char(36);not null;uniqueIndex:uidx_user_tenant_username" json:"tenant_id"`
	Username     string    `gorm:"size:80;not null;uniqueIndex:uidx_user_tenant_username" json:"username"`
	DisplayName  string    `gorm:"size:120;not null" json:"display_name"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	Status       string    `gorm:"size:20;not null" json:"status"`
	Roles        []Role    `gorm:"many2many:user_roles" json:"roles,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
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

func (Tenant) TableName() string   { return "tenants" }
func (User) TableName() string     { return "users" }
func (Role) TableName() string     { return "roles" }
func (UserRole) TableName() string { return "user_roles" }
