package domain

import businessdomain "github.com/example/adnova/internal/business/domain"

type Detail struct {
	Recommendation businessdomain.Recommendation   `json:"recommendation"`
	Approval       *businessdomain.ApprovalRequest `json:"approval,omitempty"`
	CampaignName   string                          `json:"campaign_name"`
	TaskStatus     string                          `json:"task_status"`
	ReportID       string                          `json:"report_id,omitempty"`
}

type Filter struct {
	Status    string
	Action    string
	RiskLevel string
	Limit     int
}
