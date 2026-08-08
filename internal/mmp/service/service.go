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
	ingestiondomain "github.com/example/adnova/internal/ingestion/domain"
	ingestionservice "github.com/example/adnova/internal/ingestion/service"
	"github.com/example/adnova/internal/mmp/appsflyer"
	mmpdomain "github.com/example/adnova/internal/mmp/domain"
	"github.com/example/adnova/internal/mmp/repository"
	"github.com/google/uuid"
)

var appIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{1,254}$`)

type Repository interface {
	GameExists(context.Context, string, string) (bool, error)
	ListConnections(context.Context, string) ([]mmpdomain.Connection, error)
	GetConnection(context.Context, string, string) (*mmpdomain.Connection, error)
	FindConnection(context.Context, string, string, string) (*mmpdomain.Connection, error)
	UpsertConnection(context.Context, *mmpdomain.Connection) error
	TouchConnection(context.Context, string, string, time.Time) error
	ListSyncRuns(context.Context, string) ([]mmpdomain.SyncRun, error)
	FindSyncRunByKey(context.Context, string, string) (*mmpdomain.SyncRun, error)
	CreateSyncRun(context.Context, *mmpdomain.SyncRun) error
	UpdateSyncRun(context.Context, *mmpdomain.SyncRun) error
}

type Fetcher interface {
	Configured() bool
	Fetch(context.Context, appsflyer.FetchInput) (*appsflyer.FetchResult, error)
}

type Importer interface {
	Import(context.Context, ingestionservice.ImportInput) (*ingestiondomain.ImportJob, error)
}

type Service struct {
	repo         Repository
	appsflyer    Fetcher
	importer     Importer
	auditor      auditservice.Recorder
	maxRangeDays int
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

func New(repo Repository, fetcher Fetcher, importer Importer, auditor auditservice.Recorder, maxRangeDays int) *Service {
	return &Service{repo: repo, appsflyer: fetcher, importer: importer, auditor: auditor, maxRangeDays: maxRangeDays, syncLease: 2 * time.Minute, now: func() time.Time { return time.Now().UTC() }}
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
		return nil, fmt.Errorf("check AppsFlyer game: %w", err)
	}
	if !exists {
		return nil, apperror.Validation("game_id 不存在或不属于当前公司")
	}
	before, findErr := s.repo.FindConnection(ctx, input.TenantID, input.GameID, mmpdomain.ProviderAppsFlyer)
	if findErr == nil && before.ExternalAppID != input.ExternalAppID && before.LastSyncAt != nil {
		return nil, apperror.New(10005, 409, "已有成功同步，不能直接更换 AppsFlyer App ID")
	}
	row := &mmpdomain.Connection{ID: uuid.NewString(), TenantID: input.TenantID, GameID: input.GameID, Provider: mmpdomain.ProviderAppsFlyer, ExternalAppID: input.ExternalAppID, Status: input.Status, CreatedBy: input.UserID, UpdatedBy: input.UserID}
	if findErr == nil {
		row.ID, row.CreatedBy, row.CreatedAt, row.LastSyncAt = before.ID, before.CreatedBy, before.CreatedAt, before.LastSyncAt
	}
	if findErr != nil && !errors.Is(findErr, repository.ErrNotFound) {
		return nil, findErr
	}
	if err := s.repo.UpsertConnection(ctx, row); err != nil {
		return nil, fmt.Errorf("save AppsFlyer connection: %w", err)
	}
	saved, err := s.repo.FindConnection(ctx, input.TenantID, input.GameID, mmpdomain.ProviderAppsFlyer)
	if err != nil {
		return nil, err
	}
	if s.auditor != nil {
		if err := s.auditor.Record(ctx, domain.RecordInput{TenantID: input.TenantID, ActorID: input.UserID, ActorType: "USER", Action: "MMP_CONNECTION_CONFIGURED", ResourceType: "MMP_CONNECTION", ResourceID: saved.ID, Before: safeConnection(before), After: safeConnection(saved), Metadata: map[string]any{"credential_configured": s.appsflyer.Configured()}, RequestID: input.RequestID, TraceID: input.TraceID, IPAddress: input.IPAddress}); err != nil {
			return nil, fmt.Errorf("audit AppsFlyer connection: %w", err)
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
	if connection.Provider != mmpdomain.ProviderAppsFlyer || connection.Status != mmpdomain.ConnectionActive {
		return nil, apperror.Validation("AppsFlyer 连接未启用")
	}
	if !s.appsflyer.Configured() {
		return nil, apperror.Validation("AppsFlyer API Token 尚未配置")
	}
	from, to := dateOnly(input.From), dateOnly(input.To)
	if from.IsZero() || to.IsZero() || to.Before(from) {
		return nil, apperror.Validation("from 和 to 必须是有效日期，且 from 不晚于 to")
	}
	if days := int(to.Sub(from).Hours()/24) + 1; days > s.maxRangeDays {
		return nil, apperror.Validation(fmt.Sprintf("单次同步不能超过 %d 天", s.maxRangeDays))
	}
	today := dateOnly(s.now())
	if to.After(today) {
		return nil, apperror.Validation("不能同步未来日期")
	}
	appDigest := fmt.Sprintf("%x", sha256.Sum256([]byte(connection.ExternalAppID)))[:12]
	key := fmt.Sprintf("appsflyer:raw-v5:%s:%s:%s:%s", connection.ID, appDigest, from.Format("2006-01-02"), to.Format("2006-01-02"))
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
			return nil, fmt.Errorf("create AppsFlyer sync: %w", err)
		}
	} else {
		run.Status, run.ErrorCode, run.ErrorMessage, run.RequestedBy, run.StartedAt, run.FinishedAt = mmpdomain.SyncProcessing, "", "", input.UserID, started, nil
		run.ImportJobID, run.SourceRows, run.NormalizedRows, run.SkippedRows, run.WarningMessage = "", 0, 0, 0, ""
		if err := s.repo.UpdateSyncRun(ctx, run); err != nil {
			return nil, err
		}
	}

	result, fetchErr := s.appsflyer.Fetch(ctx, appsflyer.FetchInput{AppID: connection.ExternalAppID, From: from, To: to})
	if fetchErr != nil {
		return s.failRun(ctx, run, fetchErr, input)
	}
	run.SourceRows, run.NormalizedRows, run.SkippedRows, run.WarningMessage = result.SourceRows, len(result.Records), result.SkippedRows, result.WarningMessage
	data, err := json.Marshal(result.Records)
	if err != nil {
		return s.failRun(ctx, run, fmt.Errorf("encode normalized metrics: %w", err), input)
	}
	job, err := s.importer.Import(ctx, ingestionservice.ImportInput{TenantID: input.TenantID, UserID: input.UserID, GameID: connection.GameID, Source: mmpdomain.ProviderAppsFlyer, FileName: fmt.Sprintf("appsflyer-%s-%s-%s.json", appDigest, from.Format("20060102"), to.Format("20060102")), Data: data, ImportType: ingestionservice.ImportMMP, Authoritative: true, PeriodStart: from, PeriodEnd: to})
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
		if err := s.auditor.Record(ctx, domain.RecordInput{TenantID: input.TenantID, ActorID: input.UserID, ActorType: "USER", Action: "MMP_SYNC_SUCCEEDED", ResourceType: "MMP_SYNC_RUN", ResourceID: run.ID, After: map[string]any{"status": run.Status, "period_start": from, "period_end": to, "source_rows": run.SourceRows, "normalized_rows": run.NormalizedRows, "skipped_rows": run.SkippedRows, "import_job_id": run.ImportJobID}, Metadata: map[string]any{"provider": mmpdomain.ProviderAppsFlyer, "external_app_id": connection.ExternalAppID}, RequestID: input.RequestID, TraceID: input.TraceID, IPAddress: input.IPAddress}); err != nil {
			return nil, fmt.Errorf("audit AppsFlyer sync: %w", err)
		}
	}
	return run, nil
}

func (s *Service) failRun(ctx context.Context, run *mmpdomain.SyncRun, cause error, input SyncInput) (*mmpdomain.SyncRun, error) {
	finished := s.now()
	run.Status, run.FinishedAt = mmpdomain.SyncFailed, &finished
	var providerErr *appsflyer.ProviderError
	var appErr *apperror.Error
	switch {
	case errors.As(cause, &providerErr):
		run.ErrorCode, run.ErrorMessage = providerErr.Code, providerErr.Message
	case errors.As(cause, &appErr):
		run.ErrorCode, run.ErrorMessage = "IMPORT_VALIDATION_FAILED", appErr.Message
	default:
		run.ErrorCode, run.ErrorMessage = "INTERNAL_ERROR", "AppsFlyer 同步失败"
	}
	if err := s.repo.UpdateSyncRun(ctx, run); err != nil {
		return nil, fmt.Errorf("record AppsFlyer sync failure: %w", err)
	}
	if s.auditor != nil {
		if err := s.auditor.Record(ctx, domain.RecordInput{TenantID: input.TenantID, ActorID: input.UserID, ActorType: "USER", Action: "MMP_SYNC_FAILED", ResourceType: "MMP_SYNC_RUN", ResourceID: run.ID, After: map[string]any{"status": run.Status, "error_code": run.ErrorCode, "period_start": run.PeriodStart, "period_end": run.PeriodEnd}, Metadata: map[string]any{"provider": run.Provider, "connection_id": run.ConnectionID}, RequestID: input.RequestID, TraceID: input.TraceID, IPAddress: input.IPAddress}); err != nil {
			return run, fmt.Errorf("audit AppsFlyer sync failure: %w", err)
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
	} else if !s.appsflyer.Configured() {
		health = "NOT_CONFIGURED"
	}
	return mmpdomain.ConnectionView{Connection: row, CredentialConfigured: s.appsflyer.Configured(), Health: health}
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
