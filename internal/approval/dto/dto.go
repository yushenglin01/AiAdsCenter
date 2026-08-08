package dto

type DecisionRequest struct {
	Comment string `json:"comment" binding:"max=1000"`
}
