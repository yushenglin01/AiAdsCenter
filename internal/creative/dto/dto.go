package dto

type UpsertCreativeRequest struct {
	CampaignID string `json:"campaign_id" binding:"required"`
	ExternalID string `json:"external_id" binding:"required,max=120"`
	Name       string `json:"name" binding:"required,max=160"`
	Type       string `json:"type" binding:"required,oneof=IMAGE VIDEO PLAYABLE TEXT"`
	Status     string `json:"status" binding:"required,oneof=ACTIVE INACTIVE ARCHIVED"`
}
