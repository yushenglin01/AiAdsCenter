package handler

import (
	"github.com/example/adnova/internal/common/apperror"
	"github.com/example/adnova/internal/common/identity"
	"github.com/example/adnova/internal/common/response"
	"github.com/example/adnova/internal/game/dto"
	"github.com/example/adnova/internal/game/service"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *service.Service }

func New(service *service.Service) *Handler { return &Handler{service: service} }

func (h *Handler) Create(c *gin.Context) {
	var req dto.UpsertGameRequest
	if c.ShouldBindJSON(&req) != nil {
		response.Fail(c, apperror.InvalidArgument)
		return
	}
	game, err := h.service.Create(c, identity.TenantID(c), req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, game)
}
func (h *Handler) List(c *gin.Context) {
	games, err := h.service.List(c, identity.TenantID(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, games)
}
func (h *Handler) Get(c *gin.Context) {
	game, err := h.service.Get(c, identity.TenantID(c), c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, game)
}
func (h *Handler) Update(c *gin.Context) {
	var req dto.UpsertGameRequest
	if c.ShouldBindJSON(&req) != nil {
		response.Fail(c, apperror.InvalidArgument)
		return
	}
	game, err := h.service.Update(c, identity.TenantID(c), c.Param("id"), req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, game)
}
