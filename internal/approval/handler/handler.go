package handler

import (
	"strconv"

	approvaldomain "github.com/example/adnova/internal/approval/domain"
	"github.com/example/adnova/internal/approval/dto"
	"github.com/example/adnova/internal/approval/service"
	"github.com/example/adnova/internal/common/apperror"
	"github.com/example/adnova/internal/common/identity"
	"github.com/example/adnova/internal/common/response"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *service.Service }

func New(service *service.Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	rows, err := h.service.List(c.Request.Context(), identity.TenantID(c), approvaldomain.Filter{Status: c.Query("status"), Action: c.Query("action"), Limit: limit})
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

func (h *Handler) Approve(c *gin.Context) { h.decide(c, approvaldomain.StatusApproved) }
func (h *Handler) Reject(c *gin.Context)  { h.decide(c, approvaldomain.StatusRejected) }

func (h *Handler) decide(c *gin.Context, status string) {
	var request dto.DecisionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Fail(c, apperror.Validation("invalid approval decision payload"))
		return
	}
	requestID, _ := c.Get("request_id")
	traceID, _ := c.Get("trace_id")
	row, err := h.service.Decide(c.Request.Context(), approvaldomain.Decision{
		TenantID: identity.TenantID(c), ApprovalID: c.Param("id"), ActorID: identity.UserID(c), ActorType: "USER", Status: status, Comment: request.Comment,
		RequestID: stringValue(requestID), TraceID: stringValue(traceID), IPAddress: c.ClientIP(),
	}, identity.Roles(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, row)
}

func stringValue(value any) string {
	result, _ := value.(string)
	return result
}
