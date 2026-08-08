package handler

import (
	"github.com/example/adnova/internal/campaign/dto"
	"github.com/example/adnova/internal/campaign/service"
	"github.com/example/adnova/internal/common/apperror"
	"github.com/example/adnova/internal/common/identity"
	"github.com/example/adnova/internal/common/response"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *service.Service }

func New(service *service.Service) *Handler { return &Handler{service: service} }
func (h *Handler) CreateChannel(c *gin.Context) {
	var req dto.UpsertChannelRequest
	if c.ShouldBindJSON(&req) != nil {
		response.Fail(c, apperror.InvalidArgument)
		return
	}
	row, err := h.service.CreateChannel(c, identity.TenantID(c), req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, row)
}
func (h *Handler) ListChannels(c *gin.Context) {
	rows, err := h.service.ListChannels(c, identity.TenantID(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, rows)
}
func (h *Handler) GetChannel(c *gin.Context) {
	row, err := h.service.GetChannel(c, identity.TenantID(c), c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, row)
}
func (h *Handler) UpdateChannel(c *gin.Context) {
	var req dto.UpsertChannelRequest
	if c.ShouldBindJSON(&req) != nil {
		response.Fail(c, apperror.InvalidArgument)
		return
	}
	row, err := h.service.UpdateChannel(c, identity.TenantID(c), c.Param("id"), req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, row)
}
func (h *Handler) CreateCampaign(c *gin.Context) {
	var req dto.UpsertCampaignRequest
	if c.ShouldBindJSON(&req) != nil {
		response.Fail(c, apperror.InvalidArgument)
		return
	}
	row, err := h.service.CreateCampaign(c, identity.TenantID(c), req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, row)
}
func (h *Handler) ListCampaigns(c *gin.Context) {
	rows, err := h.service.ListCampaigns(c, identity.TenantID(c), c.Query("game_id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, rows)
}
func (h *Handler) GetCampaign(c *gin.Context) {
	row, err := h.service.GetCampaign(c, identity.TenantID(c), c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, row)
}
func (h *Handler) UpdateCampaign(c *gin.Context) {
	var req dto.UpsertCampaignRequest
	if c.ShouldBindJSON(&req) != nil {
		response.Fail(c, apperror.InvalidArgument)
		return
	}
	row, err := h.service.UpdateCampaign(c, identity.TenantID(c), c.Param("id"), req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, row)
}
