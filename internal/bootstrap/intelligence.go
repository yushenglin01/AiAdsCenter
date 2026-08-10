package bootstrap

import (
	"fmt"

	"github.com/example/adnova/internal/agent/intelligence"
	"github.com/example/adnova/internal/config"
	"github.com/example/adnova/internal/llm"
	modelusagerepo "github.com/example/adnova/internal/modelusage/repository"
	"gorm.io/gorm"
)

type AgentIntelligence struct {
	Client    llm.Client
	Runtime   *intelligence.Runtime
	Contracts map[string]intelligence.Contract
}

func NewAgentIntelligence(cfg config.Config, db *gorm.DB) (*AgentIntelligence, error) {
	client, err := newLLMClient(cfg)
	if err != nil {
		return nil, err
	}
	definitions := []struct {
		agentName, promptName, schemaName string
	}{
		{"research-agent", "research_agent_llm", "research-agent-llm-output"},
		{"creative-agent", "creative_agent_llm", "creative-agent-llm-output"},
		{"openclaw-agent", "openclaw_agent_llm", "openclaw-agent-llm-output"},
		{"report-agent", "report_agent_llm", "report-agent-llm-output"},
	}
	contracts := make(map[string]intelligence.Contract, len(definitions))
	for _, definition := range definitions {
		contract, loadErr := intelligence.LoadContract(
			cfg.LLM.PromptDir, cfg.LLM.SchemaDir, definition.agentName,
			definition.promptName, "1.0.0", definition.schemaName, "1.0.0",
		)
		if loadErr != nil {
			return nil, fmt.Errorf("load %s intelligence: %w", definition.agentName, loadErr)
		}
		contracts[definition.agentName] = contract
	}
	runtime := intelligence.New(client, cfg.LLM.Model, map[string]bool{
		"research-agent": cfg.LLM.ResearchEnabled,
		"creative-agent": cfg.LLM.CreativeEnabled,
		"openclaw-agent": cfg.LLM.OpenClawEnabled,
		"report-agent":   cfg.LLM.ReportEnabled,
	}, modelusagerepo.New(db))
	return &AgentIntelligence{Client: client, Runtime: runtime, Contracts: contracts}, nil
}

func newLLMClient(cfg config.Config) (llm.Client, error) {
	if cfg.LLM.Provider == config.LLMProviderMock {
		return llm.NewMock(cfg.LLM.Model), nil
	}
	return llm.NewOpenAICompatibleForProvider(cfg.LLM.Provider, cfg.LLM.BaseURL, cfg.LLM.APIKey, cfg.LLM.Model, cfg.LLM.Timeout)
}
