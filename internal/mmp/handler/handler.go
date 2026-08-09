package handler

import (
	"time"

	"github.com/example/adnova/internal/common/apperror"
	"github.com/example/adnova/internal/common/identity"
	"github.com/example/adnova/internal/common/response"
	"github.com/example/adnova/internal/mmp/dto"
	"github.com/example/adnova/internal/mmp/service"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *service.Service }

func New(service *service.Service) *Handler { return &Handler{service: service} }

func (h *Handler) ListConnections(c *gin.Context) {
	rows, err := h.service.ListConnections(c.Request.Context(), identity.TenantID(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, rows)
}

func (h *Handler) ConfigureAppsFlyer(c *gin.Context) {
	var req dto.ConfigureConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, apperror.Validation("game_id 和 external_app_id 不能为空"))
		return
	}
	row, err := h.service.ConfigureAppsFlyer(c.Request.Context(), service.ConfigureInput{TenantID: identity.TenantID(c), UserID: identity.UserID(c), GameID: req.GameID, ExternalAppID: req.ExternalAppID, Status: req.Status, RequestID: contextValue(c, "request_id"), TraceID: contextValue(c, "trace_id"), IPAddress: c.ClientIP()})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, row)
}

func (h *Handler) ConfigureAdjust(c *gin.Context) {
	var req dto.ConfigureConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, apperror.Validation("game_id 和 external_app_id 不能为空"))
		return
	}
	row, err := h.service.ConfigureAdjust(c.Request.Context(), service.ConfigureInput{TenantID: identity.TenantID(c), UserID: identity.UserID(c), GameID: req.GameID, ExternalAppID: req.ExternalAppID, Status: req.Status, RequestID: contextValue(c, "request_id"), TraceID: contextValue(c, "trace_id"), IPAddress: c.ClientIP()})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, row)
}

func (h *Handler) ListSyncRuns(c *gin.Context) {
	rows, err := h.service.ListSyncRuns(c.Request.Context(), identity.TenantID(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, rows)
}

func (h *Handler) Sync(c *gin.Context) {
	var req dto.SyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, apperror.Validation("from 和 to 不能为空"))
		return
	}
	from, fromErr := time.Parse("2006-01-02", req.From)
	to, toErr := time.Parse("2006-01-02", req.To)
	if fromErr != nil || toErr != nil {
		response.Fail(c, apperror.Validation("from 和 to 必须为 YYYY-MM-DD"))
		return
	}
	run, err := h.service.Sync(c.Request.Context(), service.SyncInput{TenantID: identity.TenantID(c), UserID: identity.UserID(c), ConnectionID: c.Param("id"), From: from, To: to, RequestID: contextValue(c, "request_id"), TraceID: contextValue(c, "trace_id"), IPAddress: c.ClientIP()})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, run)
}

func contextValue(c *gin.Context, key string) string {
	value, _ := c.Get(key)
	text, _ := value.(string)
	return text
}
