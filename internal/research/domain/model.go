package domain

import "time"

const (
	CategoryPolicy     = "POLICY"
	CategoryCompetitor = "COMPETITOR"
	CategoryMarket     = "MARKET"

	StatusPending  = "PENDING"
	StatusVerified = "VERIFIED"
	StatusRejected = "REJECTED"
)

type Source struct {
	ID                 string     `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID           string     `gorm:"type:char(36);not null;uniqueIndex:uidx_research_source_hash" json:"tenant_id"`
	GameID             string     `gorm:"type:char(36);index" json:"game_id,omitempty"`
	CampaignID         string     `gorm:"type:char(36);index" json:"campaign_id,omitempty"`
	Category           string     `gorm:"size:30;not null;index" json:"category"`
	Title              string     `gorm:"size:300;not null" json:"title"`
	Summary            string     `gorm:"type:text;not null" json:"summary"`
	SourceURL          string     `gorm:"size:1000;not null" json:"source_url"`
	DiscoveryMethod    string     `gorm:"size:30;not null;default:MANUAL" json:"discovery_method"`
	DiscoveryProvider  string     `gorm:"size:50" json:"discovery_provider,omitempty"`
	DiscoveryQueryHash string     `gorm:"type:char(64)" json:"discovery_query_hash,omitempty"`
	DiscoveredAt       *time.Time `json:"discovered_at,omitempty"`
	Publisher          string     `gorm:"size:200;not null" json:"publisher"`
	PublishedAt        time.Time  `gorm:"not null;index" json:"published_at"`
	ContentHash        string     `gorm:"size:64;not null;uniqueIndex:uidx_research_source_hash" json:"content_hash"`
	Status             string     `gorm:"size:20;not null;index" json:"status"`
	ReviewComment      string     `gorm:"type:text" json:"review_comment,omitempty"`
	CreatedBy          string     `gorm:"type:char(36);not null" json:"created_by"`
	ReviewedBy         string     `gorm:"type:char(36)" json:"reviewed_by,omitempty"`
	ReviewedAt         *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

func (Source) TableName() string { return "research_sources" }

type Evidence struct {
	ID          string    `json:"id"`
	Category    string    `json:"category"`
	Title       string    `json:"title"`
	Summary     string    `json:"summary"`
	SourceURL   string    `json:"source_url"`
	Publisher   string    `json:"publisher"`
	PublishedAt time.Time `json:"published_at"`
	VerifiedAt  time.Time `json:"verified_at"`
}

type Filter struct {
	GameID     string
	CampaignID string
	Category   string
	Status     string
	Limit      int
}
