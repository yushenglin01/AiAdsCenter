package config

import (
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/viper"
)

type Config struct {
	Environment  string
	HTTP         HTTPConfig
	Database     DatabaseConfig
	Redis        RedisConfig
	JWT          JWTConfig
	Tenant       TenantConfig
	Demo         DemoConfig
	LLM          LLMConfig
	Queue        QueueConfig
	Kafka        KafkaConfig
	AppsFlyer    AppsFlyerConfig
	Registration RegistrationConfig
}

type HTTPConfig struct {
	Address       string
	AllowedOrigin string
	ReadTimeout   time.Duration
	WriteTimeout  time.Duration
}

type DatabaseConfig struct {
	DSN          string
	MigrationDir string
}

type RedisConfig struct {
	Address  string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	Issuer     string
}

type TenantConfig struct {
	DefaultID string
}

type DemoConfig struct {
	Seed bool
}

type LLMConfig struct {
	Provider  string
	BaseURL   string
	APIKey    string
	Model     string
	Timeout   time.Duration
	PromptDir string
	SchemaDir string
}

type QueueConfig struct {
	Name             string
	Concurrency      int
	MaxRetry         int
	TaskTimeout      time.Duration
	Retention        time.Duration
	DispatchInterval time.Duration
	ShutdownTimeout  time.Duration
}

type KafkaConfig struct {
	Enabled        bool
	Brokers        []string
	Topics         []string
	GroupID        string
	DLQTopic       string
	TenantID       string
	ProducerSystem string
	Username       string
	Password       string
	TLSEnabled     bool
	Debounce       time.Duration
	MessageLease   time.Duration
	AnalysisLease  time.Duration
	ProcessRetries int
}

type AppsFlyerConfig struct {
	BaseURL        string
	APIToken       string
	Timeout        time.Duration
	MaxRetries     int
	PurchaseEvents []string
	MaxRangeDays   int
}

type RegistrationConfig struct {
	Enabled             bool
	PublicBaseURL       string
	VerificationTTL     time.Duration
	ResendCooldown      time.Duration
	AllowedEmailDomains []string
	Mail                MailConfig
}

type MailConfig struct {
	Provider     string
	SMTPAddress  string
	SMTPUsername string
	SMTPPassword string
	FromAddress  string
	FromName     string
}

func Load() (Config, error) {
	v := viper.New()
	v.SetEnvPrefix("GAI")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	defaults := map[string]any{
		"environment":                        "development",
		"http.address":                       ":8080",
		"http.allowed_origin":                "http://localhost:5173",
		"http.read_timeout":                  "10s",
		"http.write_timeout":                 "30s",
		"database.dsn":                       "gai:gai@tcp(localhost:3306)/game_ads?charset=utf8mb4&parseTime=True&loc=UTC",
		"database.migration_dir":             "migrations",
		"redis.address":                      "localhost:6379",
		"redis.password":                     "",
		"redis.db":                           0,
		"jwt.secret":                         "change-me-in-production",
		"jwt.access_ttl":                     "15m",
		"jwt.refresh_ttl":                    "168h",
		"jwt.issuer":                         "game-ads-intelligence",
		"tenant.default_id":                  "00000000-0000-4000-8000-000000000001",
		"demo.seed":                          true,
		"llm.provider":                       "mock",
		"llm.base_url":                       "",
		"llm.api_key":                        "",
		"llm.model":                          "mock-business-v1",
		"llm.timeout":                        "30s",
		"llm.prompt_dir":                     "configs/prompts",
		"llm.schema_dir":                     "configs/schemas",
		"queue.name":                         "business-analysis",
		"queue.concurrency":                  4,
		"queue.max_retry":                    3,
		"queue.task_timeout":                 "2m",
		"queue.retention":                    "24h",
		"queue.dispatch_interval":            "2s",
		"queue.shutdown_timeout":             "30s",
		"kafka.enabled":                      false,
		"kafka.brokers":                      "",
		"kafka.topics":                       "adnova.ingestion.ad-metrics.v1,adnova.ingestion.mmp-metrics.v1,adnova.ingestion.game-revenue.v1,adnova.ingestion.creative-metrics.v1",
		"kafka.group_id":                     "adnova-ingestion-v1",
		"kafka.dlq_topic":                    "adnova.ingestion.dlq.v1",
		"kafka.tenant_id":                    "",
		"kafka.producer_system":              "",
		"kafka.username":                     "",
		"kafka.password":                     "",
		"kafka.tls_enabled":                  true,
		"kafka.debounce":                     "60s",
		"kafka.message_lease":                "5m",
		"kafka.analysis_lease":               "15m",
		"kafka.process_retries":              3,
		"appsflyer.base_url":                 "https://hq1.appsflyer.com",
		"appsflyer.api_token":                "",
		"appsflyer.timeout":                  "8s",
		"appsflyer.max_retries":              2,
		"appsflyer.purchase_events":          "af_purchase",
		"appsflyer.max_range_days":           7,
		"registration.enabled":               true,
		"registration.public_base_url":       "http://localhost:5173",
		"registration.verification_ttl":      "24h",
		"registration.resend_cooldown":       "2m",
		"registration.allowed_email_domains": "",
		"registration.mail.provider":         "log",
		"registration.mail.smtp_address":     "",
		"registration.mail.smtp_username":    "",
		"registration.mail.smtp_password":    "",
		"registration.mail.from_address":     "no-reply@adnova.local",
		"registration.mail.from_name":        "AdNova",
	}
	for key, value := range defaults {
		v.SetDefault(key, value)
	}

	cfg := Config{
		Environment: v.GetString("environment"),
		HTTP: HTTPConfig{
			Address:       v.GetString("http.address"),
			AllowedOrigin: v.GetString("http.allowed_origin"),
			ReadTimeout:   v.GetDuration("http.read_timeout"),
			WriteTimeout:  v.GetDuration("http.write_timeout"),
		},
		Database: DatabaseConfig{DSN: v.GetString("database.dsn"), MigrationDir: v.GetString("database.migration_dir")},
		Redis: RedisConfig{
			Address:  v.GetString("redis.address"),
			Password: v.GetString("redis.password"),
			DB:       v.GetInt("redis.db"),
		},
		JWT: JWTConfig{
			Secret:     v.GetString("jwt.secret"),
			AccessTTL:  v.GetDuration("jwt.access_ttl"),
			RefreshTTL: v.GetDuration("jwt.refresh_ttl"),
			Issuer:     v.GetString("jwt.issuer"),
		},
		Tenant: TenantConfig{DefaultID: v.GetString("tenant.default_id")},
		Demo:   DemoConfig{Seed: v.GetBool("demo.seed")},
		LLM:    LLMConfig{Provider: v.GetString("llm.provider"), BaseURL: v.GetString("llm.base_url"), APIKey: v.GetString("llm.api_key"), Model: v.GetString("llm.model"), Timeout: v.GetDuration("llm.timeout"), PromptDir: v.GetString("llm.prompt_dir"), SchemaDir: v.GetString("llm.schema_dir")},
		Queue:  QueueConfig{Name: v.GetString("queue.name"), Concurrency: v.GetInt("queue.concurrency"), MaxRetry: v.GetInt("queue.max_retry"), TaskTimeout: v.GetDuration("queue.task_timeout"), Retention: v.GetDuration("queue.retention"), DispatchInterval: v.GetDuration("queue.dispatch_interval"), ShutdownTimeout: v.GetDuration("queue.shutdown_timeout")},
		Kafka: KafkaConfig{
			Enabled: v.GetBool("kafka.enabled"), Brokers: splitNonEmpty(v.GetString("kafka.brokers")), Topics: splitNonEmpty(v.GetString("kafka.topics")),
			GroupID: strings.TrimSpace(v.GetString("kafka.group_id")), DLQTopic: strings.TrimSpace(v.GetString("kafka.dlq_topic")),
			TenantID: strings.TrimSpace(v.GetString("kafka.tenant_id")), ProducerSystem: strings.ToUpper(strings.TrimSpace(v.GetString("kafka.producer_system"))),
			Username: strings.TrimSpace(v.GetString("kafka.username")), Password: v.GetString("kafka.password"), TLSEnabled: v.GetBool("kafka.tls_enabled"),
			Debounce: v.GetDuration("kafka.debounce"), MessageLease: v.GetDuration("kafka.message_lease"), AnalysisLease: v.GetDuration("kafka.analysis_lease"), ProcessRetries: v.GetInt("kafka.process_retries"),
		},
		AppsFlyer: AppsFlyerConfig{
			BaseURL:        strings.TrimRight(strings.TrimSpace(v.GetString("appsflyer.base_url")), "/"),
			APIToken:       strings.TrimSpace(v.GetString("appsflyer.api_token")),
			Timeout:        v.GetDuration("appsflyer.timeout"),
			MaxRetries:     v.GetInt("appsflyer.max_retries"),
			PurchaseEvents: splitNonEmpty(v.GetString("appsflyer.purchase_events")),
			MaxRangeDays:   v.GetInt("appsflyer.max_range_days"),
		},
		Registration: RegistrationConfig{
			Enabled:             v.GetBool("registration.enabled"),
			PublicBaseURL:       strings.TrimRight(strings.TrimSpace(v.GetString("registration.public_base_url")), "/"),
			VerificationTTL:     v.GetDuration("registration.verification_ttl"),
			ResendCooldown:      v.GetDuration("registration.resend_cooldown"),
			AllowedEmailDomains: lowerNonEmpty(v.GetString("registration.allowed_email_domains")),
			Mail: MailConfig{
				Provider:     strings.ToLower(strings.TrimSpace(v.GetString("registration.mail.provider"))),
				SMTPAddress:  strings.TrimSpace(v.GetString("registration.mail.smtp_address")),
				SMTPUsername: strings.TrimSpace(v.GetString("registration.mail.smtp_username")),
				SMTPPassword: v.GetString("registration.mail.smtp_password"),
				FromAddress:  strings.TrimSpace(v.GetString("registration.mail.from_address")),
				FromName:     strings.TrimSpace(v.GetString("registration.mail.from_name")),
			},
		},
	}
	if cfg.JWT.Secret == "" {
		return Config{}, fmt.Errorf("GAI_JWT_SECRET must not be empty")
	}
	if cfg.Environment == "production" && cfg.JWT.Secret == "change-me-in-production" {
		return Config{}, fmt.Errorf("GAI_JWT_SECRET must be changed in production")
	}
	if cfg.Tenant.DefaultID == "" {
		return Config{}, fmt.Errorf("GAI_TENANT_DEFAULT_ID must not be empty")
	}
	if cfg.LLM.Provider != "mock" && cfg.LLM.Provider != "openai-compatible" {
		return Config{}, fmt.Errorf("GAI_LLM_PROVIDER must be mock or openai-compatible")
	}
	if cfg.LLM.Provider == "openai-compatible" && (cfg.LLM.BaseURL == "" || cfg.LLM.APIKey == "" || cfg.LLM.Model == "") {
		return Config{}, fmt.Errorf("OpenAI-compatible provider requires GAI_LLM_BASE_URL, GAI_LLM_API_KEY and GAI_LLM_MODEL")
	}
	if cfg.Queue.Name == "" || cfg.Queue.Concurrency < 1 || cfg.Queue.MaxRetry < 0 || cfg.Queue.TaskTimeout <= 0 || cfg.Queue.DispatchInterval <= 0 {
		return Config{}, fmt.Errorf("queue configuration is invalid")
	}
	if cfg.Kafka.Enabled {
		if len(cfg.Kafka.Brokers) == 0 || len(cfg.Kafka.Topics) == 0 || cfg.Kafka.GroupID == "" || cfg.Kafka.DLQTopic == "" || cfg.Kafka.TenantID == "" || cfg.Kafka.ProducerSystem == "" {
			return Config{}, fmt.Errorf("Kafka enabled requires brokers, topics, group_id, dlq_topic, tenant_id and producer_system")
		}
		if (cfg.Kafka.Username == "") != (cfg.Kafka.Password == "") {
			return Config{}, fmt.Errorf("Kafka username and password must be configured together")
		}
		if _, err := uuid.Parse(cfg.Kafka.TenantID); err != nil {
			return Config{}, fmt.Errorf("GAI_KAFKA_TENANT_ID must be a UUID")
		}
		if !regexp.MustCompile(`^[A-Z0-9][A-Z0-9._:-]{0,79}$`).MatchString(cfg.Kafka.ProducerSystem) {
			return Config{}, fmt.Errorf("GAI_KAFKA_PRODUCER_SYSTEM format is invalid")
		}
		if cfg.Kafka.Debounce <= 0 || cfg.Kafka.MessageLease <= 0 || cfg.Kafka.AnalysisLease <= 0 || cfg.Kafka.ProcessRetries < 0 || cfg.Kafka.ProcessRetries > 10 {
			return Config{}, fmt.Errorf("Kafka processing configuration is invalid")
		}
	}
	parsedAppsFlyerURL, err := url.Parse(cfg.AppsFlyer.BaseURL)
	if err != nil || parsedAppsFlyerURL.Scheme != "https" || parsedAppsFlyerURL.Host != "hq1.appsflyer.com" || parsedAppsFlyerURL.Path != "" || parsedAppsFlyerURL.RawQuery != "" {
		return Config{}, fmt.Errorf("GAI_APPSFLYER_BASE_URL must be https://hq1.appsflyer.com")
	}
	if cfg.AppsFlyer.Timeout <= 0 || cfg.AppsFlyer.MaxRetries < 0 || cfg.AppsFlyer.MaxRetries > 5 || cfg.AppsFlyer.MaxRangeDays < 1 || cfg.AppsFlyer.MaxRangeDays > 31 || len(cfg.AppsFlyer.PurchaseEvents) == 0 {
		return Config{}, fmt.Errorf("AppsFlyer configuration is invalid")
	}
	if cfg.Registration.Enabled {
		publicURL, err := url.Parse(cfg.Registration.PublicBaseURL)
		if err != nil || publicURL.Host == "" || (publicURL.Scheme != "https" && !(cfg.Environment != "production" && publicURL.Scheme == "http")) || publicURL.User != nil || publicURL.RawQuery != "" || publicURL.Fragment != "" {
			return Config{}, fmt.Errorf("GAI_REGISTRATION_PUBLIC_BASE_URL must be an absolute HTTPS URL (HTTP is allowed outside production)")
		}
		if cfg.Registration.VerificationTTL < 15*time.Minute || cfg.Registration.VerificationTTL > 72*time.Hour || cfg.Registration.ResendCooldown < time.Minute || cfg.Registration.ResendCooldown > time.Hour {
			return Config{}, fmt.Errorf("registration token timing configuration is invalid")
		}
		if cfg.Registration.Mail.Provider != "log" && cfg.Registration.Mail.Provider != "smtp" {
			return Config{}, fmt.Errorf("GAI_REGISTRATION_MAIL_PROVIDER must be log or smtp")
		}
		if cfg.Registration.Mail.Provider == "smtp" {
			if _, _, err := net.SplitHostPort(cfg.Registration.Mail.SMTPAddress); err != nil {
				return Config{}, fmt.Errorf("GAI_REGISTRATION_MAIL_SMTP_ADDRESS must be host:port")
			}
			from, err := mail.ParseAddress(cfg.Registration.Mail.FromAddress)
			if err != nil || from.Address != cfg.Registration.Mail.FromAddress {
				return Config{}, fmt.Errorf("GAI_REGISTRATION_MAIL_FROM_ADDRESS must be a valid email address")
			}
			if strings.ContainsAny(cfg.Registration.Mail.FromName, "\r\n") {
				return Config{}, fmt.Errorf("GAI_REGISTRATION_MAIL_FROM_NAME must not contain line breaks")
			}
		}
		if cfg.Environment == "production" && (cfg.Registration.Mail.Provider != "smtp" || len(cfg.Registration.AllowedEmailDomains) == 0) {
			return Config{}, fmt.Errorf("production registration requires SMTP and allowed company email domains")
		}
	}
	return cfg, nil
}

func splitNonEmpty(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func lowerNonEmpty(value string) []string {
	parts := splitNonEmpty(value)
	for index := range parts {
		parts[index] = strings.ToLower(parts[index])
	}
	return parts
}
