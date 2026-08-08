package service

import (
	"encoding/json"
	"testing"
)

func TestValidatorAcceptsGroundedRecommendation(t *testing.T) {
	input := json.RawMessage(`{"campaign":{"campaign_id":"meta","roas_d7":"1.0"},"benchmark":[{"value":"1.3"}],"rule_findings":[{"evidence":{"roas_d7":"1.0","threshold":"1.3"}}]}`)
	output := json.RawMessage(`{"status":"CRITICAL","summary":"risk","findings":[{"type":"roas","rule_code":"ROAS_BELOW_TARGET","severity":"HIGH","conclusion":"low","description":"low","evidence":[{"metric":"roas_d7","actual":"1.0","target":"1.3"}],"possible_causes":[],"impact":"risk","confidence":0.9}],"recommendations":[{"action":"REDUCE_BUDGET","description":"reduce","priority":1,"risk_level":"HIGH","requires_approval":true}]}`)
	result, errors := NewValidator().Validate(output, input)
	if result == nil || len(errors) != 0 {
		t.Fatalf("result=%#v errors=%v", result, errors)
	}
}

func TestValidatorRejectsHallucinationAndDirectAction(t *testing.T) {
	input := json.RawMessage(`{"campaign":{"campaign_id":"meta","roas_d7":"1.0"}}`)
	output := json.RawMessage(`{"status":"CRITICAL","summary":"risk","findings":[{"type":"roas","rule_code":"X","severity":"HIGH","conclusion":"low","description":"low","evidence":[{"metric":"roas_d7","actual":"9.9"}],"confidence":1.2}],"recommendations":[{"action":"update_campaign_budget","description":"execute","priority":1,"risk_level":"HIGH","requires_approval":false}]}`)
	result, errors := NewValidator().Validate(output, input)
	if result != nil || len(errors) < 3 {
		t.Fatalf("result=%#v errors=%v", result, errors)
	}
}
