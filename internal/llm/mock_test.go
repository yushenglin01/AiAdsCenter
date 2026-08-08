package llm

import (
	"context"
	"encoding/json"
	"testing"
)

func TestMockClientIsDeterministicStructuredJSON(t *testing.T) {
	input := json.RawMessage(`{"campaign":{"campaign_id":"meta","campaign_name":"Meta US Growth"},"rule_findings":[{"rule_code":"ROAS_BELOW_TARGET","severity":"HIGH","title":"ROAS low","description":"low","evidence":{"actual":"1.0","threshold":"1.3"}}],"attribution_findings":[],"creative_findings":[]}`)
	client := NewMock("")
	first, err := client.GenerateStructured(context.Background(), GenerateRequest{InputJSON: input})
	if err != nil {
		t.Fatal(err)
	}
	second, err := client.GenerateStructured(context.Background(), GenerateRequest{InputJSON: input})
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(first.Content) || string(first.Content) != string(second.Content) {
		t.Fatalf("mock output must be stable JSON: %s / %s", first.Content, second.Content)
	}
	if first.Usage.InputTokens == 0 || first.Usage.OutputTokens == 0 {
		t.Fatal("mock must report deterministic token estimates")
	}
}
