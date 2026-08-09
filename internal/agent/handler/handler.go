package handler

import (
	"context"

	agentdomain "github.com/example/adnova/internal/agent/domain"
	agentruntime "github.com/example/adnova/internal/agent/runtime"
	"github.com/example/adnova/internal/common/apperror"
	"github.com/example/adnova/internal/common/identity"
	"github.com/example/adnova/internal/common/response"
	workflowdomain "github.com/example/adnova/internal/workflow/domain"
	"github.com/gin-gonic/gin"
)

type RuntimeReader interface {
	AgentRuntime(context.Context, string) ([]workflowdomain.AgentRuntime, error)
}

type Handler struct {
	registry *agentruntime.Registry
	runtime  RuntimeReader
}

type CatalogItem struct {
	Definition agentruntime.Definition   `json:"definition"`
	Health     *agentdomain.HealthStatus `json:"health,omitempty"`
}

func New(registry *agentruntime.Registry, readers ...RuntimeReader) *Handler {
	handler := &Handler{registry: registry}
	if len(readers) > 0 {
		handler.runtime = readers[0]
	}
	return handler
}

func (h *Handler) List(c *gin.Context) {
	definitions := h.registry.List()
	result := make([]CatalogItem, 0, len(definitions))
	for _, definition := range definitions {
		result = append(result, CatalogItem{Definition: definition, Health: health(c, definition)})
	}
	response.OK(c, result)
}

func (h *Handler) Get(c *gin.Context) {
	definition, exists := h.registry.Get(c.Param("name"))
	if !exists {
		response.Fail(c, apperror.NotFound)
		return
	}
	response.OK(c, CatalogItem{Definition: definition, Health: health(c, definition)})
}

func (h *Handler) Runtime(c *gin.Context) {
	if h.runtime == nil {
		response.OK(c, []workflowdomain.AgentRuntime{})
		return
	}
	rows, err := h.runtime.AgentRuntime(c, identity.TenantID(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, rows)
}

func health(ctx context.Context, definition agentruntime.Definition) *agentdomain.HealthStatus {
	if definition.Executor == nil {
		return &agentdomain.HealthStatus{Status: string(definition.Availability), Provider: "internal", Details: append([]string(nil), definition.Details...)}
	}
	status, err := definition.Executor.Health(ctx)
	if err != nil {
		return &agentdomain.HealthStatus{Status: "DOWN", Details: []string{err.Error()}}
	}
	return status
}
