package bootstrap

import (
	auditrepo "github.com/example/adnova/internal/audit/repository"
	auditservice "github.com/example/adnova/internal/audit/service"
	"github.com/example/adnova/internal/config"
	researchrepo "github.com/example/adnova/internal/research/repository"
	researchservice "github.com/example/adnova/internal/research/service"
	"github.com/example/adnova/internal/research/websearch"
	"gorm.io/gorm"
)

func NewResearchService(cfg config.Config, db *gorm.DB) *researchservice.Service {
	repository := researchrepo.New(db)
	auditor := auditservice.New(auditrepo.New(db))
	if cfg.WebSearch.Provider == "disabled" {
		return researchservice.New(repository, auditor)
	}
	provider := websearch.NewBrave(websearch.BraveConfig{
		BaseURL: cfg.WebSearch.BaseURL, APIKey: cfg.WebSearch.APIKey,
		Timeout: cfg.WebSearch.Timeout, MaxResults: cfg.WebSearch.MaxResults,
		SafeSearch: cfg.WebSearch.SafeSearch, ImportEnabled: cfg.WebSearch.ImportEnabled,
	})
	return researchservice.NewWithWebSearch(repository, provider, auditor)
}
