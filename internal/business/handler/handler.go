package handler

import (
	"time"

	"github.com/example/adnova/internal/business/service"
	"github.com/example/adnova/internal/common/apperror"
	"github.com/example/adnova/internal/common/identity"
	"github.com/example/adnova/internal/common/response"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service       *service.Service
	eventInterval time.Duration
	eventDuration time.Duration
}

func New(service *service.Service) *Handler {
	return &Handler{service: service, eventInterval: 500 * time.Millisecond, eventDuration: 25 * time.Second}
}

func (h *Handler) Analyze(c *gin.Context) {
	var input service.AnalyzeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Fail(c, apperror.Validation("请求参数格式错误"))
		return
	}
	traceID, _ := c.Get("trace_id")
	result, err := h.service.Submit(c, identity.TenantID(c), identity.UserID(c), stringValue(traceID), input)
	if err != nil {
		response.Fail(c, apperror.Validation(err.Error()))
		return
	}
	response.Accepted(c, result)
}

func (h *Handler) Report(c *gin.Context) {
	report, err := h.service.GetReport(c, identity.TenantID(c), c.Param("id"))
	if err != nil {
		response.Fail(c, apperror.NotFound)
		return
	}
	response.OK(c, report)
}

func (h *Handler) Events(c *gin.Context) {
	tenantID, taskID := identity.TenantID(c), c.Param("id")
	if _, err := h.service.Get(c, tenantID, taskID); err != nil {
		response.Fail(c, apperror.NotFound)
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache, no-transform")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	ticker := time.NewTicker(h.eventInterval)
	defer ticker.Stop()
	timeout := time.NewTimer(h.eventDuration)
	defer timeout.Stop()
	lastVersion := ""
	for {
		details, err := h.service.Get(c, tenantID, taskID)
		if err != nil {
			return
		}
		version := details.Task.UpdatedAt.UTC().Format(time.RFC3339Nano) + ":" + details.Task.Status + ":" + details.Task.CurrentStep
		if version != lastVersion {
			c.SSEvent("task", details)
			c.Writer.Flush()
			lastVersion = version
		}
		if terminal(details.Task.Status) {
			return
		}
		select {
		case <-c.Request.Context().Done():
			return
		case <-timeout.C:
			c.SSEvent("reconnect", gin.H{"task_id": taskID})
			c.Writer.Flush()
			return
		case <-ticker.C:
		}
	}
}

func terminal(status string) bool {
	return status == "WAITING_APPROVAL" || status == "SUCCEEDED" || status == "FAILED" || status == "MANUAL_REVIEW" || status == "CANCELLED"
}

func (h *Handler) List(c *gin.Context) {
	rows, err := h.service.List(c, identity.TenantID(c))
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

func (h *Handler) Health(c *gin.Context) {
	row, err := h.service.Health(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"health": row, "capabilities": h.service.Capabilities(c)})
}

func stringValue(value any) string { result, _ := value.(string); return result }
