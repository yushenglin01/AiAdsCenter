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
	t.Setenv("GAI_JWT_SECRET", "a-production-secret-that-is-not-the-default")
	t.Setenv("GAI_REGISTRATION_PUBLIC_BASE_URL", "https://ads.example.com")
	t.Setenv("GAI_REGISTRATION_MAIL_PROVIDER", "log")
	t.Setenv("GAI_REGISTRATION_ALLOWED_EMAIL_DOMAINS", "")
	_, err := Load()
	require.ErrorContains(t, err, "requires SMTP and allowed company email domains")
}

func TestProductionRegistrationSMTPConfiguration(t *testing.T) {
	t.Setenv("GAI_ENVIRONMENT", "production")
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
