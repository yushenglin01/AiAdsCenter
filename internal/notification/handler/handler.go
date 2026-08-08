package handler

import (
	"github.com/example/adnova/internal/common/identity"
	"github.com/example/adnova/internal/common/response"
	"github.com/example/adnova/internal/notification/service"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *service.Service }

func New(service *service.Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *gin.Context) {
	rows, err := h.service.List(c, identity.TenantID(c), c.Query("status"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, rows)
}

func (h *Handler) MarkRead(c *gin.Context) {
	row, err := h.service.MarkRead(c, identity.TenantID(c), c.Param("id"), identity.UserID(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, row)
}
