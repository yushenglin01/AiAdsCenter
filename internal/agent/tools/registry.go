package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
)

type Context struct {
	TenantID string
	UserID   string
	TaskID   string
}

type Handler func(ctx context.Context, toolContext Context, input json.RawMessage) (json.RawMessage, error)

type Definition struct {
	Name        string
	Description string
	Permission  string
	Handler     Handler
}

type Registry struct {
	mu    sync.RWMutex
	tools map[string]Definition
}

func NewRegistry() *Registry { return &Registry{tools: map[string]Definition{}} }

func (r *Registry) Register(tool Definition) error {
	if tool.Name == "" || tool.Handler == nil {
		return fmt.Errorf("tool name and handler are required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.tools[tool.Name]; exists {
		return fmt.Errorf("tool %s already registered", tool.Name)
	}
	r.tools[tool.Name] = tool
	return nil
}

func (r *Registry) Execute(ctx context.Context, allowed []string, name string, toolContext Context, input json.RawMessage) (json.RawMessage, error) {
	if !contains(allowed, name) {
		return nil, fmt.Errorf("tool %s is not allowed for this agent", name)
	}
	r.mu.RLock()
	tool, exists := r.tools[name]
	r.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("tool %s is not registered", name)
	}
	if toolContext.TenantID == "" {
		return nil, fmt.Errorf("tenant context is required")
	}
	return tool.Handler(ctx, toolContext, input)
}

func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]string, 0, len(r.tools))
	for name := range r.tools {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

func contains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}
