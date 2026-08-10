package service

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/example/adnova/internal/business/domain"
)

var allowedActions = map[string]bool{"REDUCE_BUDGET": true, "INCREASE_BUDGET": true, "PAUSE_RECOMMENDED": true, "REPLACE_CREATIVE": true, "VERIFY_ATTRIBUTION": true, "CHECK_DATA_DELAY": true, "ADJUST_AUDIENCE": true, "OBSERVE": true}

type Validator struct{}

func NewValidator() *Validator { return &Validator{} }

func (v *Validator) Validate(raw json.RawMessage, input json.RawMessage) (*domain.Result, []string) {
	errors := []string{}
	var result domain.Result
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, []string{"输出不是合法 JSON 结构"}
	}
	if result.Status == "" || result.Summary == "" {
		errors = append(errors, "status 和 summary 不能为空")
	}
	allowedEvidence := collectEvidence(input)
	for index, finding := range result.Findings {
		prefix := fmt.Sprintf("findings[%d]", index)
		if finding.Confidence < 0 || finding.Confidence > 1 || math.IsNaN(finding.Confidence) {
			errors = append(errors, prefix+" confidence 必须在 0 到 1")
		}
		if len(finding.Evidence) == 0 && finding.Confidence >= 0.8 {
			errors = append(errors, prefix+" 高置信度结论必须提供证据")
		}
		for _, evidence := range finding.Evidence {
			if evidence.Metric == "requires_verification" {
				continue
			}
			values, exists := allowedEvidence[evidence.Metric]
			if !exists || !values[evidence.Actual] {
				errors = append(errors, fmt.Sprintf("%s 证据 %s=%s 不存在于输入", prefix, evidence.Metric, evidence.Actual))
			}
			if evidence.Target != "" && !allowedTarget(input, evidence.Target) {
				errors = append(errors, fmt.Sprintf("%s target=%s 与输入不一致", prefix, evidence.Target))
			}
		}
	}
	for index, recommendation := range result.Recommendations {
		prefix := fmt.Sprintf("recommendations[%d]", index)
		if !allowedActions[recommendation.Action] {
			errors = append(errors, prefix+" action 不在允许列表")
		}
		if recommendation.RiskLevel == "HIGH" || recommendation.RiskLevel == "CRITICAL" {
			if !recommendation.RequiresApproval {
				errors = append(errors, prefix+" 高风险建议必须 requires_approval=true")
			}
		}
	}
	if len(errors) > 0 {
		sort.Strings(errors)
		return nil, errors
	}
	return &result, nil
}

func collectEvidence(input json.RawMessage) map[string]map[string]bool {
	var value any
	_ = json.Unmarshal(input, &value)
	result := map[string]map[string]bool{}
	var walk func(any)
	walk = func(current any) {
		switch typed := current.(type) {
		case map[string]any:
			for key, item := range typed {
				if key == "agent_context" {
					continue
				}
				if scalar(item) {
					if result[key] == nil {
						result[key] = map[string]bool{}
					}
					result[key][fmt.Sprint(item)] = true
				}
				walk(item)
			}
		case []any:
			for _, item := range typed {
				walk(item)
			}
		}
	}
	walk(value)
	return result
}

func allowedTarget(input json.RawMessage, target string) bool {
	return strings.Contains(string(input), `"`+target+`"`) || strings.Contains(string(input), ":"+target)
}

func scalar(value any) bool {
	switch value.(type) {
	case string, float64, bool, nil:
		return true
	default:
		return false
	}
}
