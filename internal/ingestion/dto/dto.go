package dto

import "encoding/json"

type JSONImportRequest struct {
	GameID   string          `json:"game_id" binding:"required"`
	Source   string          `json:"source" binding:"required"`
	FileName string          `json:"file_name" binding:"required"`
	Records  json.RawMessage `json:"records" binding:"required"`
}
