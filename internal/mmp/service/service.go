package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/example/adnova/internal/audit/domain"
	auditservice "github.com/example/adnova/internal/audit/service"
	"github.com/example/adnova/internal/common/apperror"
	"github.com/example/adnova/internal/common/identity"
	ingestiondomain "github.com/example/adnova/internal/ingestion/domain"
	ingestionservice "github.com/example/adnova/internal/ingestion/service"
	mmpdomain "github.com/example/adnova/internal/mmp/domain"
	mmpprovider "github.com/example/adnova/internal/mmp/provider"
	"github.com/example/adnova/internal/mmp/repository"
	"github.com/google/uuid"
)

var appIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{1,254}$`)

type Repository interface {
	GameExists(context.Context, string, string) (bool, error)
	ListConnections(context.Context, string) ([]mmpdomain.Connection, error)
	ListAllConnections(context.Context) ([]mmpdomain.Connection, error)
	GetConnection(context.Context, string, string) (*mmpdomain.Connection, error)
	FindConnection(context.Context, string, string, string) (*mmpdomain.Connection, error)
	UpsertConnection(context.Context, *mmpdomain.Connection) error
	TouchConnection(context.Context, string, string, time.Time) error
	ListSyncRuns(context.Context, string) ([]mmpdomain.SyncRun, error)
	FindSyncRunByKey(context.Context, string, string) (*mmpdomain.SyncRun, error)
	CreateSyncRun(context.Context, *mmpdomain.SyncRun) error
	UpdateSyncRun(context.Context, *mmpdomain.SyncRun) error
}

type Importer interface {
	Import(context.Context, ingestionservice.ImportInput) (*ingestiondomain.ImportJob, error)
}

type Service struct {
	repo         Repository
	fetchers     map[string]mmpprovider.Fetcher
	importer     Importer
	auditor      auditservice.Recorder
	maxRangeDays map[string]int
	syncLease    time.Duration
	now          func() time.Time
}

type ConfigureInput struct {
	TenantID, UserID, GameID, ExternalAppID, Status string
	RequestID, TraceID, IPAddress                   string
}

type SyncInput struct {
	TenantID, UserID, ConnectionID string
	From, To                       time.Time
	RequestID, TraceID, IPAddress  string
}

func New(repo Repository, fetchers map[string]mmpprovider.Fetcher, importer Importer, auditor auditservice.Recorder, maxRangeDays map[string]int) *Service {
	return &Service{repo: repo, fetchers: fetchers, importer: importer, auditor: auditor, maxRangeDays: maxRangeDays, syncLease: 2 * time.Minute, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Service) ListConnections(ctx context.Context, tenantID string) ([]mmpdomain.ConnectionView, error) {
	rows, err := s.repo.ListConnections(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	views := make([]mmpdomain.ConnectionView, 0, len(rows))
	for _, row := range rows {
		views = append(views, s.connectionView(row))
	}
	return views, nil
}

func (s *Service) ConfigureAppsFlyer(ctx context.Context, input ConfigureInput) (*mmpdomain.ConnectionView, error) {
	return s.configure(ctx, mmpdomain.ProviderAppsFlyer, input)
}

func (s *Service) ConfigureAdjust(ctx context.Context, input ConfigureInput) (*mmpdomain.ConnectionView, error) {
	return s.configure(ctx, mmpdomain.ProviderAdjust, input)
}

func (s *Service) configure(ctx context.Context, providerName string, input ConfigureInput) (*mmpdomain.ConnectionView, error) {
	input.ExternalAppID = strings.TrimSpace(input.ExternalAppID)
	input.Status = strings.ToUpper(strings.TrimSpace(input.Status))
	if input.Status == "" {
		input.Status = mmpdomain.ConnectionActive
	}
	if input.TenantID == "" || input.UserID == "" || input.GameID == "" || !appIDPattern.MatchString(input.ExternalAppID) {
		return nil, apperror.Validation("game_id 和合法的 external_app_id 不能为空")
	}
	if input.Status != mmpdomain.ConnectionActive && input.Status != mmpdomain.ConnectionDisabled {
		return nil, apperror.Validation("status 必须是 ACTIVE 或 DISABLED")
	}
	exists, err := s.repo.GameExists(ctx, input.TenantID, input.GameID)
	if err != nil {
		return nil, fmt.Errorf("check %s game: %w", providerName, err)
	}
	if !exists {
		return nil, apperror.Validation("game_id 不存在或不属于当前公司")
	}
	before, findErr := s.repo.FindConnection(ctx, input.TenantID, input.GameID, providerName)
	if findErr == nil && before.ExternalAppID != input.ExternalAppID && before.LastSyncAt != nil {
		return nil, apperror.New(10005, 409, fmt.Sprintf("已有成功同步，不能直接更换 %s App ID", providerName))
	}
	row := &mmpdomain.Connection{ID: uuid.NewString(), TenantID: input.TenantID, GameID: input.GameID, Provider: providerName, ExternalAppID: input.ExternalAppID, Status: input.Status, CreatedBy: input.UserID, UpdatedBy: input.UserID}
	if findErr == nil {
		row.ID, row.CreatedBy, row.CreatedAt, row.LastSyncAt = before.ID, before.CreatedBy, before.CreatedAt, before.LastSyncAt
	}
	if findErr != nil && !errors.Is(findErr, repository.ErrNotFound) {
		return nil, findErr
	}
	if err := s.repo.UpsertConnection(ctx, row); err != nil {
		return nil, fmt.Errorf("save %s connection: %w", providerName, err)
	}
	saved, err := s.repo.FindConnection(ctx, input.TenantID, input.GameID, providerName)
	if err != nil {
		return nil, err
	}
	if s.auditor != nil {
		if err := s.auditor.Record(ctx, domain.RecordInput{TenantID: input.TenantID, ActorID: input.UserID, ActorType: "USER", Action: "MMP_CONNECTION_CONFIGURED", ResourceType: "MMP_CONNECTION", ResourceID: saved.ID, Before: safeConnection(before), After: safeConnection(saved), Metadata: map[string]any{"provider": providerName, "credential_configured": s.fetcherConfigured(providerName)}, RequestID: input.RequestID, TraceID: input.TraceID, IPAddress: input.IPAddress}); err != nil {
			return nil, fmt.Errorf("audit %s connection: %w", providerName, err)
		}
	}
	view := s.connectionView(*saved)
	return &view, nil
}

func (s *Service) ListSyncRuns(ctx context.Context, tenantID string) ([]mmpdomain.SyncRun, error) {
	return s.repo.ListSyncRuns(ctx, tenantID)
}

func (s *Service) Sync(ctx context.Context, input SyncInput) (*mmpdomain.SyncRun, error) {
	connection, err := s.repo.GetConnection(ctx, input.TenantID, input.ConnectionID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, apperror.NotFound
	}
	if err != nil {
		return nil, err
	}
	if connection.Status != mmpdomain.ConnectionActive {
		return nil, apperror.Validation(connection.Provider + " 连接未启用")
	}
	fetcher := s.fetchers[connection.Provider]
	if fetcher == nil || !fetcher.Configured() {
		return nil, apperror.Validation(connection.Provider + " API 凭证或指标映射尚未配置")
	}
	from, to := dateOnly(input.From), dateOnly(input.To)
	if from.IsZero() || to.IsZero() || to.Before(from) {
		return nil, apperror.Validation("from 和 to 必须是有效日期，且 from 不晚于 to")
	}
	maxDays := s.maxRangeDays[connection.Provider]
	if maxDays < 1 {
		return nil, apperror.Validation("不支持的 MMP Provider")
	}
	if days := int(to.Sub(from).Hours()/24) + 1; days > maxDays {
		return nil, apperror.Validation(fmt.Sprintf("单次同步不能超过 %d 天", maxDays))
	}
	today := dateOnly(s.now())
	if to.After(today) {
		return nil, apperror.Validation("不能同步未来日期")
	}
	appDigest := fmt.Sprintf("%x", sha256.Sum256([]byte(connection.ExternalAppID)))[:12]
	key := fmt.Sprintf("%s:report:%s:%s:%s:%s", strings.ToLower(connection.Provider), connection.ID, appDigest, from.Format("2006-01-02"), to.Format("2006-01-02"))
	run, findErr := s.repo.FindSyncRunByKey(ctx, input.TenantID, key)
	if findErr == nil {
		if run.Status == mmpdomain.SyncSucceeded || (run.Status == mmpdomain.SyncProcessing && s.now().Sub(run.StartedAt) <= s.syncLease) {
			return run, nil
		}
	}
	if findErr != nil && !errors.Is(findErr, repository.ErrNotFound) {
		return nil, findErr
	}
	started := s.now()
	if run == nil {
		run = &mmpdomain.SyncRun{ID: uuid.NewString(), TenantID: input.TenantID, ConnectionID: connection.ID, Provider: connection.Provider, PeriodStart: from, PeriodEnd: to, Status: mmpdomain.SyncProcessing, IdempotencyKey: key, RequestedBy: input.UserID, StartedAt: started}
		if err := s.repo.CreateSyncRun(ctx, run); err != nil {
			if existing, lookupErr := s.repo.FindSyncRunByKey(ctx, input.TenantID, key); lookupErr == nil {
				return existing, nil
			}
			return nil, fmt.Errorf("create %s sync: %w", connection.Provider, err)
		}
	} else {
		run.Status, run.ErrorCode, run.ErrorMessage, run.RequestedBy, run.StartedAt, run.FinishedAt = mmpdomain.SyncProcessing, "", "", input.UserID, started, nil
		run.ImportJobID, run.SourceRows, run.NormalizedRows, run.SkippedRows, run.WarningMessage = "", 0, 0, 0, ""
		if err := s.repo.UpdateSyncRun(ctx, run); err != nil {
			return nil, err
		}
	}

	result, fetchErr := fetcher.Fetch(ctx, mmpprovider.FetchInput{AppID: connection.ExternalAppID, From: from, To: to})
	if fetchErr != nil {
		return s.failRun(ctx, run, fetchErr, input)
	}
	run.SourceRows, run.NormalizedRows, run.SkippedRows, run.WarningMessage = result.SourceRows, len(result.Records), result.SkippedRows, result.WarningMessage
	data, err := json.Marshal(result.Records)
	if err != nil {
		return s.failRun(ctx, run, fmt.Errorf("encode normalized metrics: %w", err), input)
	}
	job, err := s.importer.Import(ctx, ingestionservice.ImportInput{TenantID: input.TenantID, UserID: input.UserID, GameID: connection.GameID, Source: connection.Provider, FileName: fmt.Sprintf("%s-%s-%s-%s.json", strings.ToLower(connection.Provider), appDigest, from.Format("20060102"), to.Format("20060102")), Data: data, ImportType: ingestionservice.ImportMMP, Authoritative: true, PeriodStart: from, PeriodEnd: to})
	if err != nil {
		return s.failRun(ctx, run, err, input)
	}
	run.ImportJobID = job.ID
	finished := s.now()
	run.Status, run.FinishedAt = mmpdomain.SyncSucceeded, &finished
	if err := s.repo.UpdateSyncRun(ctx, run); err != nil {
		return nil, err
	}
	if err := s.repo.TouchConnection(ctx, input.TenantID, connection.ID, finished); err != nil {
		return nil, err
	}
	if s.auditor != nil {
		if err := s.auditor.Record(ctx, domain.RecordInput{TenantID: input.TenantID, ActorID: input.UserID, ActorType: actorType(input.UserID), Action: "MMP_SYNC_SUCCEEDED", ResourceType: "MMP_SYNC_RUN", ResourceID: run.ID, After: map[string]any{"status": run.Status, "period_start": from, "period_end": to, "source_rows": run.SourceRows, "normalized_rows": run.NormalizedRows, "skipped_rows": run.SkippedRows, "import_job_id": run.ImportJobID}, Metadata: map[string]any{"provider": connection.Provider, "external_app_id": connection.ExternalAppID}, RequestID: input.RequestID, TraceID: input.TraceID, IPAddress: input.IPAddress}); err != nil {
			return nil, fmt.Errorf("audit %s sync: %w", connection.Provider, err)
		}
	}
	return run, nil
}

func (s *Service) failRun(ctx context.Context, run *mmpdomain.SyncRun, cause error, input SyncInput) (*mmpdomain.SyncRun, error) {
	finished := s.now()
	run.Status, run.FinishedAt = mmpdomain.SyncFailed, &finished
	var providerErr *mmpprovider.Error
	var appErr *apperror.Error
	switch {
	case errors.As(cause, &providerErr):
		run.ErrorCode, run.ErrorMessage = providerErr.Code, providerErr.Message
	case errors.As(cause, &appErr):
		run.ErrorCode, run.ErrorMessage = "IMPORT_VALIDATION_FAILED", appErr.Message
	default:
		run.ErrorCode, run.ErrorMessage = "INTERNAL_ERROR", run.Provider+" 同步失败"
	}
	if err := s.repo.UpdateSyncRun(ctx, run); err != nil {
		return nil, fmt.Errorf("record %s sync failure: %w", run.Provider, err)
	}
	if s.auditor != nil {
		if err := s.auditor.Record(ctx, domain.RecordInput{TenantID: input.TenantID, ActorID: input.UserID, ActorType: actorType(input.UserID), Action: "MMP_SYNC_FAILED", ResourceType: "MMP_SYNC_RUN", ResourceID: run.ID, After: map[string]any{"status": run.Status, "error_code": run.ErrorCode, "period_start": run.PeriodStart, "period_end": run.PeriodEnd}, Metadata: map[string]any{"provider": run.Provider, "connection_id": run.ConnectionID}, RequestID: input.RequestID, TraceID: input.TraceID, IPAddress: input.IPAddress}); err != nil {
			return run, fmt.Errorf("audit %s sync failure: %w", run.Provider, err)
		}
	}
	if providerErr != nil {
		switch providerErr.Code {
		case "RATE_LIMITED":
			return run, apperror.New(10201, 429, run.ErrorMessage)
		case "UPSTREAM_UNAVAILABLE", "UPSTREAM_ERROR", "INVALID_RESPONSE", "RESPONSE_TOO_LARGE", "ROW_LIMIT_REACHED":
			return run, apperror.New(10202, 502, run.ErrorMessage)
		}
	}
	return run, apperror.Validation(run.ErrorMessage)
}

func (s *Service) connectionView(row mmpdomain.Connection) mmpdomain.ConnectionView {
	health := "READY"
	if row.Status == mmpdomain.ConnectionDisabled {
		health = "DISABLED"
	} else if !s.fetcherConfigured(row.Provider) {
		health = "NOT_CONFIGURED"
	}
	return mmpdomain.ConnectionView{Connection: row, CredentialConfigured: s.fetcherConfigured(row.Provider), Health: health}
}

func (s *Service) fetcherConfigured(providerName string) bool {
	fetcher := s.fetchers[providerName]
	return fetcher != nil && fetcher.Configured()
}

func (s *Service) AutoSync(ctx context.Context, tenantID string, lookbackDays int) (int, error) {
	connections, err := s.repo.ListConnections(ctx, tenantID)
	if err != nil {
		return 0, err
	}
	return s.autoSyncConnections(ctx, connections, lookbackDays)
}

func (s *Service) AutoSyncAll(ctx context.Context, lookbackDays int) (int, error) {
	connections, err := s.repo.ListAllConnections(ctx)
	if err != nil {
		return 0, err
	}
	return s.autoSyncConnections(ctx, connections, lookbackDays)
}

func (s *Service) autoSyncConnections(ctx context.Context, connections []mmpdomain.Connection, lookbackDays int) (int, error) {
	if lookbackDays < 1 {
		return 0, fmt.Errorf("MMP auto sync lookback_days must be positive")
	}
	today := dateOnly(s.now())
	to := today.AddDate(0, 0, -1)
	synced := 0
	var failures []string
	for _, connection := range connections {
		if connection.Status != mmpdomain.ConnectionActive || !s.fetcherConfigured(connection.Provider) {
			continue
		}
		rangeDays := lookbackDays
		if maximum := s.maxRangeDays[connection.Provider]; maximum > 0 && rangeDays > maximum {
			rangeDays = maximum
		}
		from := to.AddDate(0, 0, -(rangeDays - 1))
		_, syncErr := s.Sync(ctx, SyncInput{TenantID: connection.TenantID, UserID: identity.SystemAgentUserID, ConnectionID: connection.ID, From: from, To: to})
		if syncErr != nil {
			failures = append(failures, connection.Provider+":"+connection.ID)
			continue
		}
		synced++
	}
	if len(failures) > 0 {
		return synced, fmt.Errorf("MMP auto sync failed for %s", strings.Join(failures, ","))
	}
	return synced, nil
}

func actorType(userID string) string {
	if userID == identity.SystemAgentUserID {
		return "SYSTEM_AGENT"
	}
	return "USER"
}

func safeConnection(row *mmpdomain.Connection) any {
	if row == nil {
		return nil
	}
	return map[string]any{"id": row.ID, "game_id": row.GameID, "provider": row.Provider, "external_app_id": row.ExternalAppID, "status": row.Status}
}

func dateOnly(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}
	year, month, day := value.UTC().Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
