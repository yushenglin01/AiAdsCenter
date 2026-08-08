package domain

import (
	"time"

	"gorm.io/gorm"
)

type Game struct {
	ID          string         `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID    string         `gorm:"type:char(36);not null;uniqueIndex:uidx_game_tenant_code" json:"tenant_id"`
	Code        string         `gorm:"size:80;not null;uniqueIndex:uidx_game_tenant_code" json:"code"`
	Name        string         `gorm:"size:160;not null" json:"name"`
	PackageName string         `gorm:"size:200" json:"package_name"`
	Timezone    string         `gorm:"size:64;not null" json:"timezone"`
	Currency    string         `gorm:"size:3;not null" json:"currency"`
	Status      string         `gorm:"size:20;not null" json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Game) TableName() string { return "games" }
