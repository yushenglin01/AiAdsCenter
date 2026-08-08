package dto

type UpsertChannelRequest struct {
	Code     string `json:"code" binding:"required,max=40"`
	Name     string `json:"name" binding:"required,max=80"`
	Provider string `json:"provider" binding:"required,max=40"`
	Status   string `json:"status" binding:"required,oneof=ACTIVE INACTIVE"`
}

type UpsertCampaignRequest struct {
	GameID      string `json:"game_id" binding:"required"`
	ChannelID   string `json:"channel_id" binding:"required"`
	ExternalID  string `json:"external_id" binding:"required,max=120"`
	Name        string `json:"name" binding:"required,max=160"`
	Country     string `json:"country" binding:"required,len=2"`
	DailyBudget string `json:"daily_budget" binding:"required"`
	Currency    string `json:"currency" binding:"required,len=3"`
	Status      string `json:"status" binding:"required,oneof=ACTIVE PAUSED ARCHIVED"`
}
