package bootstrap

import (
	auditrepo "github.com/example/adnova/internal/audit/repository"
	auditservice "github.com/example/adnova/internal/audit/service"
	"github.com/example/adnova/internal/config"
	ingestionrepo "github.com/example/adnova/internal/ingestion/repository"
	ingestionservice "github.com/example/adnova/internal/ingestion/service"
	adjustprovider "github.com/example/adnova/internal/mmp/adjust"
	appsflyerprovider "github.com/example/adnova/internal/mmp/appsflyer"
	mmpdomain "github.com/example/adnova/internal/mmp/domain"
	mmpprovider "github.com/example/adnova/internal/mmp/provider"
	mmprepo "github.com/example/adnova/internal/mmp/repository"
	mmpservice "github.com/example/adnova/internal/mmp/service"
	"gorm.io/gorm"
)

func NewMMPService(cfg config.Config, db *gorm.DB) *mmpservice.Service {
	ingestion := ingestionservice.New(ingestionrepo.New(db))
	fetchers := map[string]mmpprovider.Fetcher{
		mmpdomain.ProviderAppsFlyer: appsflyerprovider.New(appsflyerprovider.Config{
			BaseURL: cfg.AppsFlyer.BaseURL, Token: cfg.AppsFlyer.APIToken, Timeout: cfg.AppsFlyer.Timeout,
			MaxRetries: cfg.AppsFlyer.MaxRetries, PurchaseEvents: cfg.AppsFlyer.PurchaseEvents,
		}),
		mmpdomain.ProviderAdjust: adjustprovider.New(adjustprovider.Config{
			BaseURL: cfg.Adjust.BaseURL, Token: cfg.Adjust.APIToken, ActivationMetric: cfg.Adjust.ActivationMetric,
			PayerMetric: cfg.Adjust.PayerMetric, RevenueMetric: cfg.Adjust.RevenueMetric,
			Timeout: cfg.Adjust.Timeout, MaxRetries: cfg.Adjust.MaxRetries,
		}),
	}
	maxRanges := map[string]int{mmpdomain.ProviderAppsFlyer: cfg.AppsFlyer.MaxRangeDays, mmpdomain.ProviderAdjust: cfg.Adjust.MaxRangeDays}
	return mmpservice.New(mmprepo.New(db), fetchers, ingestion, auditservice.New(auditrepo.New(db)), maxRanges)
}
