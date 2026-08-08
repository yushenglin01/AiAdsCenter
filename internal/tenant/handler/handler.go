package handler

import (
	"github.com/example/adnova/internal/common/identity"
	"github.com/example/adnova/internal/common/response"
	"github.com/example/adnova/internal/tenant/service"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *service.Service }

func New(service *service.Service) *Handler { return &Handler{service: service} }

func (h *Handler) Current(c *gin.Context) {
	tenant, err := h.service.Current(c.Request.Context(), identity.TenantID(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, tenant)
}
