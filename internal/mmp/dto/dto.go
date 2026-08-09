package dto

type ConfigureConnectionRequest struct {
	GameID        string `json:"game_id" binding:"required"`
	ExternalAppID string `json:"external_app_id" binding:"required"`
	Status        string `json:"status"`
}

type SyncRequest struct {
	From string `json:"from" binding:"required"`
	To   string `json:"to" binding:"required"`
}
