package handler

import (
	"strconv"

	"github.com/example/adnova/internal/common/identity"
	"github.com/example/adnova/internal/common/response"
	recommendationdomain "github.com/example/adnova/internal/recommendation/domain"
	"github.com/example/adnova/internal/recommendation/service"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *service.Service }

func New(service *service.Service) *Handler { return &Handler{service: service} }
func (h *Handler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	rows, err := h.service.List(c.Request.Context(), identity.TenantID(c), recommendationdomain.Filter{Status: c.Query("status"), Action: c.Query("action"), RiskLevel: c.Query("risk_level"), Limit: limit})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, rows)
}
func (h *Handler) Get(c *gin.Context) {
	row, err := h.service.Get(c.Request.Context(), identity.TenantID(c), c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, row)
}
