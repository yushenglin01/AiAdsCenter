package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestKafkaDisabledDoesNotRequireConnection(t *testing.T) {
	t.Setenv("GAI_KAFKA_ENABLED", "false")
	cfg, err := Load()
	require.NoError(t, err)
	require.False(t, cfg.Kafka.Enabled)
}

func TestKafkaEnabledRequiresTenantAndProducer(t *testing.T) {
	t.Setenv("GAI_KAFKA_ENABLED", "true")
	t.Setenv("GAI_KAFKA_BROKERS", "kafka:9092")
	t.Setenv("GAI_KAFKA_TENANT_ID", "")
	t.Setenv("GAI_KAFKA_PRODUCER_SYSTEM", "")
	_, err := Load()
	require.ErrorContains(t, err, "tenant_id and producer_system")
}

func TestKafkaEnabledConfiguration(t *testing.T) {
	t.Setenv("GAI_KAFKA_ENABLED", "true")
	t.Setenv("GAI_KAFKA_BROKERS", "kafka-a:9092,kafka-b:9092")
	t.Setenv("GAI_KAFKA_TENANT_ID", "00000000-0000-4000-8000-000000000001")
	t.Setenv("GAI_KAFKA_PRODUCER_SYSTEM", "partner-data-hub")
	t.Setenv("GAI_KAFKA_TLS_ENABLED", "false")
	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, []string{"kafka-a:9092", "kafka-b:9092"}, cfg.Kafka.Brokers)
	require.Equal(t, "PARTNER-DATA-HUB", cfg.Kafka.ProducerSystem)
	require.False(t, cfg.Kafka.TLSEnabled)
	require.Equal(t, 15*time.Minute, cfg.Kafka.AnalysisLease)
}

func TestProductionRegistrationRequiresSMTPAndCompanyDomains(t *testing.T) {
	t.Setenv("GAI_ENVIRONMENT", "production")
	t.Setenv("GAI_DEMO_SEED", "false")
	t.Setenv("GAI_JWT_SECRET", "a-production-secret-that-is-not-the-default")
	t.Setenv("GAI_REGISTRATION_PUBLIC_BASE_URL", "https://ads.example.com")
	t.Setenv("GAI_REGISTRATION_MAIL_PROVIDER", "log")
	t.Setenv("GAI_REGISTRATION_ALLOWED_EMAIL_DOMAINS", "")
	_, err := Load()
	require.ErrorContains(t, err, "requires SMTP and allowed company email domains")
}

func TestProductionRegistrationSMTPConfiguration(t *testing.T) {
	t.Setenv("GAI_ENVIRONMENT", "production")
	t.Setenv("GAI_DEMO_SEED", "false")
	t.Setenv("GAI_JWT_SECRET", "a-production-secret-that-is-not-the-default")
	t.Setenv("GAI_REGISTRATION_PUBLIC_BASE_URL", "https://ads.example.com")
	t.Setenv("GAI_REGISTRATION_MAIL_PROVIDER", "smtp")
	t.Setenv("GAI_REGISTRATION_MAIL_SMTP_ADDRESS", "smtp.example.com:587")
	t.Setenv("GAI_REGISTRATION_MAIL_FROM_ADDRESS", "no-reply@example.com")
	t.Setenv("GAI_REGISTRATION_ALLOWED_EMAIL_DOMAINS", "Example.com, studio.example.com")
	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, []string{"example.com", "studio.example.com"}, cfg.Registration.AllowedEmailDomains)
}

func TestProductionRejectsDemoSeed(t *testing.T) {
	t.Setenv("GAI_ENVIRONMENT", " PRODUCTION ")
	t.Setenv("GAI_DEMO_SEED", "true")
	t.Setenv("GAI_JWT_SECRET", "a-production-secret-that-is-not-the-default")
	_, err := Load()
	require.ErrorContains(t, err, "GAI_DEMO_SEED must be false")
}

func TestProductionRejectsComposeDemoJWTSecret(t *testing.T) {
	t.Setenv("GAI_ENVIRONMENT", "production")
	t.Setenv("GAI_DEMO_SEED", "false")
	t.Setenv("GAI_JWT_SECRET", "local-demo-secret-change-before-production")
	_, err := Load()
	require.ErrorContains(t, err, "unique value of at least 32 characters")
}

func TestAdjustTokenRequiresMetricMappings(t *testing.T) {
	t.Setenv("GAI_ADJUST_API_TOKEN", "secret")
	t.Setenv("GAI_ADJUST_ACTIVATION_METRIC", "")
	_, err := Load()
	require.ErrorContains(t, err, "activation, payer and revenue")
}

func TestAdjustAndAutoSyncConfiguration(t *testing.T) {
	t.Setenv("GAI_ADJUST_API_TOKEN", "secret")
	t.Setenv("GAI_ADJUST_ACTIVATION_METRIC", "first_session")
	t.Setenv("GAI_ADJUST_PAYER_METRIC", "unique_purchase")
	t.Setenv("GAI_ADJUST_REVENUE_METRIC", "purchase_revenue")
	t.Setenv("GAI_MMP_AUTO_SYNC_ENABLED", "true")
	t.Setenv("GAI_MMP_AUTO_SYNC_INTERVAL", "30m")
	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "unique_purchase", cfg.Adjust.PayerMetric)
	require.True(t, cfg.MMPAutoSync.Enabled)
	require.Equal(t, 30*time.Minute, cfg.MMPAutoSync.Interval)
}

func TestWebSearchCanBeConfiguredWithoutExposingDefaults(t *testing.T) {
	t.Setenv("GAI_WEB_SEARCH_API_KEY", "server-only-key")
	t.Setenv("GAI_WEB_SEARCH_MAX_RESULTS", "10")
	t.Setenv("GAI_WEB_SEARCH_IMPORT_ENABLED", "true")
	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "brave", cfg.WebSearch.Provider)
	require.Equal(t, "server-only-key", cfg.WebSearch.APIKey)
	require.Equal(t, 10, cfg.WebSearch.MaxResults)
	require.True(t, cfg.WebSearch.ImportEnabled)
}

func TestWebSearchImportRequiresCredential(t *testing.T) {
	t.Setenv("GAI_WEB_SEARCH_API_KEY", "")
	t.Setenv("GAI_WEB_SEARCH_IMPORT_ENABLED", "true")
	_, err := Load()
	require.ErrorContains(t, err, "requires a configured provider API key")
}

func TestWebSearchRejectsUnexpectedBaseURL(t *testing.T) {
	t.Setenv("GAI_WEB_SEARCH_BASE_URL", "https://example.com")
	_, err := Load()
	require.ErrorContains(t, err, "https://api.search.brave.com")
}

func TestResearchSchedulerRequiresStoragePermission(t *testing.T) {
	t.Setenv("GAI_RESEARCH_SCHEDULER_ENABLED", "true")
	t.Setenv("GAI_WEB_SEARCH_IMPORT_ENABLED", "false")
	_, err := Load()
	require.ErrorContains(t, err, "requires web search import")
}

func TestResearchSchedulerConfiguration(t *testing.T) {
	t.Setenv("GAI_WEB_SEARCH_API_KEY", "server-only-key")
	t.Setenv("GAI_WEB_SEARCH_IMPORT_ENABLED", "true")
	t.Setenv("GAI_RESEARCH_SCHEDULER_ENABLED", "true")
	t.Setenv("GAI_RESEARCH_SCHEDULER_POLL_INTERVAL", "30s")
	t.Setenv("GAI_RESEARCH_SCHEDULER_BATCH_SIZE", "12")
	t.Setenv("GAI_RESEARCH_SCHEDULER_LEASE", "3m")
	cfg, err := Load()
	require.NoError(t, err)
	require.True(t, cfg.ResearchScheduler.Enabled)
	require.Equal(t, 30*time.Second, cfg.ResearchScheduler.PollInterval)
	require.Equal(t, 12, cfg.ResearchScheduler.BatchSize)
	require.Equal(t, 3*time.Minute, cfg.ResearchScheduler.Lease)
}
