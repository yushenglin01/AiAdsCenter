package domain

import (
	agentdomain "github.com/example/adnova/internal/agent/domain"
	businessdomain "github.com/example/adnova/internal/business/domain"
)

const (
	StatusPending  = "PENDING"
	StatusApproved = "APPROVED"
	StatusRejected = "REJECTED"
)

type Detail struct {
	Approval       businessdomain.ApprovalRequest `json:"approval"`
	Recommendation businessdomain.Recommendation  `json:"recommendation"`
	Task           agentdomain.AgentTask          `json:"task"`
	Report         *businessdomain.AnalysisReport `json:"report,omitempty"`
	CampaignName   string                         `json:"campaign_name"`
}

type Filter struct {
	Status string
	Action string
	Limit  int
}

type Decision struct {
	TenantID   string
	ApprovalID string
	TaskID     string
	ActorID    string
	ActorType  string
	Status     string
	Comment    string
	RequestID  string
	TraceID    string
	IPAddress  string
}
