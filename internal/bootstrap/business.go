package bootstrap

import (
	"fmt"

	attributionrepo "github.com/example/adnova/internal/attribution/repository"
	auditrepo "github.com/example/adnova/internal/audit/repository"
	auditservice "github.com/example/adnova/internal/audit/service"
	businessrepo "github.com/example/adnova/internal/business/repository"
	businessservice "github.com/example/adnova/internal/business/service"
	"github.com/example/adnova/internal/config"
	creativeanalysisrepo "github.com/example/adnova/internal/creative/analysis/repository"
	"github.com/example/adnova/internal/llm"
	metricsrepo "github.com/example/adnova/internal/metrics/repository"
	metricsservice "github.com/example/adnova/internal/metrics/service"
	researchrepo "github.com/example/adnova/internal/research/repository"
	rulesrepo "github.com/example/adnova/internal/rules/repository"
	"github.com/example/adnova/internal/taskqueue"
	"gorm.io/gorm"
)

func NewBusinessService(cfg config.Config, db *gorm.DB, enqueuer taskqueue.Enqueuer) (*businessservice.Service, error) {
	prompts, err := businessservice.LoadPrompts(cfg.LLM.PromptDir, cfg.LLM.SchemaDir)
	if err != nil {
		return nil, err
	}
	var llmClient llm.Client = llm.NewMock(cfg.LLM.Model)
	if cfg.LLM.Provider == "openai-compatible" {
		llmClient, err = llm.NewOpenAICompatible(cfg.LLM.BaseURL, cfg.LLM.APIKey, cfg.LLM.Model, cfg.LLM.Timeout)
		if err != nil {
			return nil, err
		}
	}
	service, err := businessservice.New(
		businessrepo.New(db),
		metricsservice.New(metricsrepo.New(db)),
		rulesrepo.New(db),
		attributionrepo.New(db),
		creativeanalysisrepo.New(db),
		researchrepo.New(db),
		llmClient,
		prompts,
		cfg.LLM.Model,
		enqueuer,
		cfg.Queue.TaskTimeout,
		auditservice.New(auditrepo.New(db)),
	)
	if err != nil {
		return nil, fmt.Errorf("initialize business service: %w", err)
	}
	return service, nil
}
