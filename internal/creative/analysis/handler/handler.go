package handler

import (
	"github.com/example/adnova/internal/common/apperror"
	"github.com/example/adnova/internal/common/identity"
	"github.com/example/adnova/internal/common/response"
	creativeservice "github.com/example/adnova/internal/creative/analysis/service"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *creativeservice.Service }

func New(service *creativeservice.Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *gin.Context) {
	rows, err := h.service.List(c, identity.TenantID(c), c.Query("game_id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, rows)
}

func (h *Handler) Get(c *gin.Context) {
	row, err := h.service.Get(c, identity.TenantID(c), c.Param("id"))
	if err != nil {
		response.Fail(c, apperror.NotFound)
		return
	}
	response.OK(c, row)
}
