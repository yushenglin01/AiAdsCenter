package runtime

import (
	"context"
	"fmt"
	"sort"
	"sync"

	agentdomain "github.com/example/adnova/internal/agent/domain"
)

type Availability string

const (
	AvailabilityReady      Availability = "READY"
	AvailabilityDegraded   Availability = "DEGRADED"
	AvailabilityConfigured Availability = "CONFIGURED"
)

type Definition struct {
	Spec         agentdomain.AgentSpec     `json:"spec"`
	Availability Availability              `json:"availability"`
	Details      []string                  `json:"details,omitempty"`
	Executor     agentdomain.AgentExecutor `json:"-"`
}

type Registry struct {
	mu     sync.RWMutex
	agents map[string]Definition
}

func NewRegistry() *Registry {
	return &Registry{agents: map[string]Definition{}}
}

func (r *Registry) Register(definition Definition) error {
	if definition.Spec.Name == "" {
		return fmt.Errorf("agent name is required")
	}
	if definition.Executor == nil && definition.Availability == AvailabilityReady {
		return fmt.Errorf("ready agent %s requires an executor", definition.Spec.Name)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.agents[definition.Spec.Name]; exists {
		return fmt.Errorf("agent %s is already registered", definition.Spec.Name)
	}
	r.agents[definition.Spec.Name] = definition
	return nil
}

func (r *Registry) Get(name string) (Definition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	definition, exists := r.agents[name]
	return definition, exists
}

func (r *Registry) List() []Definition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Definition, 0, len(r.agents))
	for _, definition := range r.agents {
		result = append(result, definition)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Spec.Name < result[j].Spec.Name })
	return result
}

func (r *Registry) Execute(ctx context.Context, name string, input agentdomain.AgentInput) (*agentdomain.AgentResult, error) {
	definition, exists := r.Get(name)
	if !exists {
		return nil, fmt.Errorf("agent %s is not registered", name)
	}
	if definition.Executor == nil {
		return nil, fmt.Errorf("agent %s is not executable: %s", name, definition.Availability)
	}
	return definition.Executor.Execute(ctx, input)
}
