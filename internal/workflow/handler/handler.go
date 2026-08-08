package handler

import (
	"time"

	"github.com/example/adnova/internal/common/apperror"
	"github.com/example/adnova/internal/common/identity"
	"github.com/example/adnova/internal/common/response"
	openclawservice "github.com/example/adnova/internal/openclaw/service"
	workflowservice "github.com/example/adnova/internal/workflow/service"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service       *workflowservice.Service
	commands      *openclawservice.Service
	eventInterval time.Duration
	eventDuration time.Duration
}

func New(service *workflowservice.Service, commands ...*openclawservice.Service) *Handler {
	result := &Handler{service: service, eventInterval: 500 * time.Millisecond, eventDuration: 25 * time.Second}
	if len(commands) > 0 {
		result.commands = commands[0]
	}
	return result
}

func (h *Handler) Start(c *gin.Context) {
	var input workflowservice.StartInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Fail(c, apperror.Validation("请求参数格式错误"))
		return
	}
	traceID, _ := c.Get("trace_id")
	result, err := h.service.Start(c, identity.TenantID(c), identity.UserID(c), stringValue(traceID), input)
	if err != nil {
		response.Fail(c, apperror.Validation(err.Error()))
		return
	}
	response.Accepted(c, result)
}

func (h *Handler) OpenClawCommand(c *gin.Context) {
	var request openclawservice.Command
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Fail(c, apperror.Validation("请求参数格式错误"))
		return
	}
	if h.commands == nil {
		response.Fail(c, apperror.Internal)
		return
	}
	traceID, _ := c.Get("trace_id")
	result, err := h.commands.Execute(c, openclawservice.Actor{TenantID: identity.TenantID(c), UserID: identity.UserID(c), TraceID: stringValue(traceID)}, request)
	if err != nil {
		response.Fail(c, apperror.Validation(err.Error()))
		return
	}
	if result.Accepted {
		response.Accepted(c, result)
		return
	}
	response.OK(c, result)
}

func (h *Handler) Get(c *gin.Context) {
	result, err := h.service.Get(c, identity.TenantID(c), c.Param("id"))
	if err != nil {
		response.Fail(c, apperror.NotFound)
		return
	}
	response.OK(c, result)
}

func (h *Handler) Events(c *gin.Context) {
	tenantID, workflowID := identity.TenantID(c), c.Param("id")
	if _, err := h.service.Get(c, tenantID, workflowID); err != nil {
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
		details, err := h.service.Get(c, tenantID, workflowID)
		if err != nil {
			return
		}
		version := details.Run.UpdatedAt.UTC().Format(time.RFC3339Nano) + ":" + details.Run.Status + ":" + details.Run.CurrentStep
		if version != lastVersion {
			c.SSEvent("workflow", details)
			c.Writer.Flush()
			lastVersion = version
		}
		if streamTerminal(details.Run.Status) {
			return
		}
		select {
		case <-c.Request.Context().Done():
			return
		case <-timeout.C:
			c.SSEvent("reconnect", gin.H{"workflow_id": workflowID})
			c.Writer.Flush()
			return
		case <-ticker.C:
		}
	}
}

func (h *Handler) List(c *gin.Context) {
	rows, err := h.service.List(c, identity.TenantID(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, rows)
}

func stringValue(value any) string { result, _ := value.(string); return result }

func streamTerminal(status string) bool {
	return status == "WAITING_APPROVAL" || status == "COMPLETED" || status == "FAILED" || status == "MANUAL_REVIEW" || status == "CANCELLED"
}
