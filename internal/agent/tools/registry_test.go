package tools

import (
	"context"
	"encoding/json"
	"testing"
)

func TestRegistryEnforcesAllowlistAndTenant(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Definition{Name: "safe_read", Handler: func(_ context.Context, tc Context, _ json.RawMessage) (json.RawMessage, error) {
		return json.Marshal(map[string]string{"tenant_id": tc.TenantID})
	}}); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Execute(context.Background(), nil, "safe_read", Context{TenantID: "tenant"}, nil); err == nil {
		t.Fatal("tool outside agent allowlist must be rejected")
	}
	if _, err := registry.Execute(context.Background(), []string{"safe_read"}, "safe_read", Context{}, nil); err == nil {
		t.Fatal("missing tenant context must be rejected")
	}
	result, err := registry.Execute(context.Background(), []string{"safe_read"}, "safe_read", Context{TenantID: "tenant"}, nil)
	if err != nil || string(result) != `{"tenant_id":"tenant"}` {
		t.Fatalf("result=%s err=%v", result, err)
	}
}
