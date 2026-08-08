package handler

import (
	"strconv"

	"github.com/example/adnova/internal/audit/domain"
	"github.com/example/adnova/internal/audit/service"
	"github.com/example/adnova/internal/common/identity"
	"github.com/example/adnova/internal/common/response"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *service.Service }

func New(service *service.Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	rows, err := h.service.List(c.Request.Context(), identity.TenantID(c), domain.Filter{
		Action: c.Query("action"), ResourceType: c.Query("resource_type"), ActorID: c.Query("actor_id"), TaskID: c.Query("task_id"), Limit: limit,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, rows)
}
