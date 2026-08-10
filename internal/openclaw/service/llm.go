package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/example/adnova/internal/agent/intelligence"
)

type Parser interface {
	Enabled() bool
	Parse(context.Context, Actor, string) (*ParsedCommand, error)
}

type ParsedCommand struct {
	Intent               string   `json:"intent"`
	Input                Input    `json:"input"`
	RequiresConfirmation bool     `json:"requires_confirmation"`
	Confidence           float64  `json:"confidence"`
	MissingFields        []string `json:"missing_fields"`
	Provider             string   `json:"provider"`
	Model                string   `json:"model"`
	PromptVersion        string   `json:"prompt_version"`
	SchemaVersion        string   `json:"schema_version"`
}

type LLMParser struct {
	runtime  *intelligence.Runtime
	contract intelligence.Contract
}

func NewLLMParser(runtime *intelligence.Runtime, contract intelligence.Contract) *LLMParser {
	return &LLMParser{runtime: runtime, contract: contract}
}

func (p *LLMParser) Enabled() bool {
	return p != nil && p.runtime.Enabled("openclaw-agent")
}

func (p *LLMParser) Parse(ctx context.Context, actor Actor, message string) (*ParsedCommand, error) {
	message = strings.TrimSpace(message)
	if message == "" || len([]rune(message)) > 2000 {
		return nil, fmt.Errorf("OpenClaw message length is invalid")
	}
	input, _ := json.Marshal(map[string]any{"message": message, "allowed_intents": allowedIntentNames()})
	generated, err := p.runtime.Generate(ctx, intelligence.Request{
		AgentName: "openclaw-agent", TenantID: actor.TenantID, TraceID: actor.TraceID,
		Contract: p.contract, UserPrompt: "把用户消息解析为一个白名单命令；不要执行命令。",
		InputJSON: input, Temperature: 0, MaxTokens: 800,
	})
	if err != nil {
		return nil, fmt.Errorf("parse OpenClaw command: %s", intelligence.FailureCategory(err))
	}
	var parsed ParsedCommand
	decoder := json.NewDecoder(bytes.NewReader(generated.Content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&parsed); err != nil {
		p.runtime.MarkValidationFailed(ctx, generated)
		return nil, fmt.Errorf("OpenClaw command output failed validation")
	}
	if err := validateParsedCommand(&parsed); err != nil {
		p.runtime.MarkValidationFailed(ctx, generated)
		return nil, err
	}
	parsed.Provider, parsed.Model = generated.Provider, generated.Model
	parsed.PromptVersion, parsed.SchemaVersion = generated.PromptVersion, generated.SchemaVersion
	return &parsed, nil
}

func validateParsedCommand(command *ParsedCommand) error {
	allowed := map[string]bool{
		IntentRunFullAnalysis: true, IntentGetWorkflowStatus: true, IntentListPendingApprovals: true,
		IntentListNotifications: true, IntentMarkNotificationRead: true,
	}
	if !allowed[command.Intent] || command.Confidence < 0.8 || command.Confidence > 1 || len(command.MissingFields) > 0 {
		return fmt.Errorf("OpenClaw could not resolve a safe command")
	}
	if command.Input.AnalysisDate != "" {
		if _, err := time.Parse("2006-01-02", command.Input.AnalysisDate); err != nil {
			return fmt.Errorf("OpenClaw analysis_date must be YYYY-MM-DD")
		}
	}
	switch command.Intent {
	case IntentRunFullAnalysis:
		if command.Input.GameID == "" || command.Input.CampaignID == "" || !command.RequiresConfirmation {
			return fmt.Errorf("OpenClaw analysis command requires game_id, campaign_id and confirmation")
		}
	case IntentGetWorkflowStatus:
		if command.Input.WorkflowID == "" || command.RequiresConfirmation {
			return fmt.Errorf("OpenClaw workflow query is invalid")
		}
	case IntentMarkNotificationRead:
		if command.Input.NotificationID == "" || command.RequiresConfirmation {
			return fmt.Errorf("OpenClaw notification command is invalid")
		}
	default:
		if command.RequiresConfirmation {
			return fmt.Errorf("OpenClaw read-only command must not require confirmation")
		}
	}
	return nil
}

func allowedIntentNames() []string {
	return []string{IntentRunFullAnalysis, IntentGetWorkflowStatus, IntentListPendingApprovals, IntentListNotifications, IntentMarkNotificationRead}
}
