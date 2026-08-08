package handler

import (
	"context"

	analysisservice "github.com/example/adnova/internal/analysis/service"
	"github.com/example/adnova/internal/common/apperror"
	"github.com/example/adnova/internal/common/identity"
	"github.com/example/adnova/internal/common/response"
	"github.com/example/adnova/internal/metrics/service"
	"github.com/gin-gonic/gin"
)

type Pipeline interface {
	Run(ctx context.Context, tenantID, gameID string) (*analysisservice.Result, error)
}

type Handler struct {
	service  *service.Service
	pipeline Pipeline
}

func New(service *service.Service, pipelines ...Pipeline) *Handler {
	h := &Handler{service: service}
	if len(pipelines) > 0 {
		h.pipeline = pipelines[0]
	}
	return h
}

func (h *Handler) Recalculate(c *gin.Context) {
	gameID := c.Query("game_id")
	if gameID == "" {
		response.Fail(c, apperror.Validation("game_id 不能为空"))
		return
	}
	if h.pipeline != nil {
		result, err := h.pipeline.Run(c, identity.TenantID(c), gameID)
		if err != nil {
			response.Fail(c, err)
			return
		}
		response.OK(c, result)
		return
	}
	count, err := h.service.Recalculate(c, identity.TenantID(c), gameID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"calculated_rows": count})
}
func (h *Handler) Campaigns(c *gin.Context) {
	rows, err := h.service.Campaigns(c, identity.TenantID(c), c.Query("game_id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, rows)
}
func (h *Handler) Campaign(c *gin.Context) {
	row, err := h.service.Campaign(c, identity.TenantID(c), c.Param("id"))
	if err != nil {
		response.Fail(c, apperror.NotFound)
		return
	}
	response.OK(c, row)
}
func (h *Handler) Overview(c *gin.Context) {
	row, err := h.service.Overview(c, identity.TenantID(c), c.Query("game_id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, row)
}
func (h *Handler) Trends(c *gin.Context) {
	rows, err := h.service.Trends(c, identity.TenantID(c), c.Query("game_id"), c.Query("campaign_id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, rows)
}
