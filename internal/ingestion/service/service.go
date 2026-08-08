package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/example/adnova/internal/common/apperror"
	"github.com/example/adnova/internal/ingestion/domain"
	"github.com/example/adnova/internal/ingestion/repository"
	"github.com/example/adnova/pkg/provider"
	"github.com/google/uuid"
)

const (
	ImportAd       = "AD_METRICS"
	ImportMMP      = "MMP_METRICS"
	ImportRevenue  = "GAME_REVENUE"
	ImportCreative = "CREATIVE_METRICS"
)

type ImportInput struct {
	TenantID          string
	UserID            string
	GameID            string
	Source            string
	FileName          string
	Data              []byte
	ImportType        string
	Authoritative     bool
	PeriodStart       time.Time
	PeriodEnd         time.Time
	BatchID           string
	ProducerSystem    string
	SchemaVersion     string
	CollectedAt       *time.Time
	PayloadHash       string
	AllowCustomSource bool
	AllowResume       bool
}

type Service struct{ repo *repository.Repository }

func New(repo *repository.Repository) *Service { return &Service{repo: repo} }

func (s *Service) Import(ctx context.Context, input ImportInput) (*domain.ImportJob, error) {
	input.Source = strings.ToUpper(strings.TrimSpace(input.Source))
	if input.GameID == "" || input.Source == "" || len(input.Data) == 0 {
		return nil, apperror.Validation("game_id、source 和文件内容不能为空")
	}
	if !input.AllowCustomSource && !sourceAllowed(input.ImportType, input.Source) {
		return nil, apperror.Validation("source 与导入类型不匹配")
	}
	exists, err := s.repo.GameExists(ctx, input.TenantID, input.GameID)
	if err != nil {
		return nil, fmt.Errorf("check import game: %w", err)
	}
	if !exists {
		return nil, apperror.Validation("game_id 不存在或不属于当前公司")
	}
	decoder, err := decoderFor(input.FileName)
	if err != nil {
		return nil, apperror.Validation(err.Error())
	}
	records, err := provider.DecodeBytes(decoder, input.Data)
	if err != nil {
		return nil, apperror.Validation(err.Error())
	}
	if len(records) == 0 && !input.Authoritative {
		return nil, apperror.Validation("导入文件没有数据行")
	}

	parsed, periodStart, err := s.parseRows(ctx, input, records)
	hash := sha256.Sum256(input.Data)
	fileHash := hex.EncodeToString(hash[:])
	if input.PayloadHash != "" {
		fileHash = input.PayloadHash
	}
	if (input.Authoritative || input.BatchID != "") && !input.PeriodStart.IsZero() {
		periodStart = input.PeriodStart.UTC()
	}
	if periodStart.IsZero() {
		periodStart = time.Now().UTC().Truncate(24 * time.Hour)
	}
	keyParts := []string{input.TenantID, input.Source, input.GameID, periodStart.Format("2006-01-02"), fileHash}
	if input.BatchID != "" {
		keyParts = []string{input.TenantID, input.ProducerSystem, input.BatchID}
	}
	if input.Authoritative && input.BatchID == "" {
		keyParts = append(keyParts, input.PeriodEnd.UTC().Format("2006-01-02"), "authoritative-v1")
	}
	key := strings.Join(keyParts, ":")
	if existing, findErr := s.repo.FindJobByKey(ctx, input.TenantID, key); findErr == nil {
		if existing.FileHash != fileHash {
			return nil, apperror.New(10005, 409, "batch_id 已被不同内容使用")
		}
		if existing.Status == "SUCCEEDED" {
			return existing, nil
		}
		if !input.AllowResume && existing.Status == "FAILED" {
			return nil, apperror.Validation(existing.ErrorMessage)
		}
		if !input.AllowResume {
			return existing, nil
		}
		if err != nil {
			return nil, apperror.Validation(err.Error())
		}
		existing.Status, existing.ErrorMessage, existing.FinishedAt = "PROCESSING", "", nil
		existing.TotalRows, existing.ImportedRows, existing.SkippedRows = len(records), 0, 0
		if updateErr := s.repo.UpdateJob(ctx, existing); updateErr != nil {
			return nil, fmt.Errorf("resume import job: %w", updateErr)
		}
		return s.finishImport(ctx, input, existing, parsed)
	} else if !errors.Is(findErr, repository.ErrNotFound) {
		return nil, fmt.Errorf("find import job: %w", findErr)
	}

	job := &domain.ImportJob{ID: uuid.NewString(), TenantID: input.TenantID, GameID: input.GameID, ImportType: input.ImportType, Source: input.Source, FileName: filepath.Base(input.FileName), FileHash: fileHash, IdempotencyKey: key, BatchID: input.BatchID, ProducerSystem: input.ProducerSystem, SchemaVersion: input.SchemaVersion, PeriodStart: periodStart, CollectedAt: input.CollectedAt, Status: "PROCESSING", TotalRows: len(records), CreatedBy: input.UserID}
	if !input.PeriodEnd.IsZero() {
		periodEnd := input.PeriodEnd.UTC()
		job.PeriodEnd = &periodEnd
	}
	if err != nil {
		now := time.Now().UTC()
		job.Status, job.ErrorMessage, job.FinishedAt = "FAILED", err.Error(), &now
		if createErr := s.repo.CreateJob(ctx, job); createErr != nil {
			return nil, fmt.Errorf("record failed import: %w", createErr)
		}
		return nil, apperror.Validation(err.Error())
	}
	if err := s.repo.CreateJob(ctx, job); err != nil {
		if existing, findErr := s.repo.FindJobByKey(ctx, input.TenantID, key); findErr == nil {
			if existing.FileHash != fileHash {
				return nil, apperror.New(10005, 409, "batch_id 已被不同内容使用")
			}
			return existing, nil
		}
		return nil, fmt.Errorf("create import job: %w", err)
	}
	return s.finishImport(ctx, input, job, parsed)
}

func (s *Service) finishImport(ctx context.Context, input ImportInput, job *domain.ImportJob, parsed []parsedRow) (*domain.ImportJob, error) {
	_, err := s.insert(ctx, input, job.ID, parsed)
	now := time.Now().UTC()
	job.FinishedAt = &now
	if err != nil {
		job.Status, job.ErrorMessage = "FAILED", err.Error()
	} else {
		imported, countErr := s.repo.CountJobRows(ctx, input.ImportType, job.ID)
		if countErr != nil {
			err = countErr
			job.Status, job.ErrorMessage = "FAILED", countErr.Error()
		} else {
			job.Status, job.ImportedRows, job.SkippedRows = "SUCCEEDED", imported, len(parsed)-imported
			job.ErrorMessage = ""
		}
	}
	if updateErr := s.repo.UpdateJob(ctx, job); updateErr != nil {
		return nil, fmt.Errorf("finish import job: %w", updateErr)
	}
	if err != nil {
		return nil, fmt.Errorf("import rows: %w", err)
	}
	return job, nil
}

func sourceAllowed(importType, source string) bool {
	allowed := map[string]map[string]bool{
		ImportAd:       {"META": true, "GOOGLE": true, "TIKTOK": true, "INTERNAL": true},
		ImportMMP:      {"APPSFLYER": true, "ADJUST": true},
		ImportRevenue:  {"GAME": true, "INTERNAL": true},
		ImportCreative: {"META": true, "GOOGLE": true, "TIKTOK": true, "INTERNAL": true},
	}
	return allowed[importType][source]
}

func (s *Service) List(ctx context.Context, tenantID string) ([]domain.ImportJob, error) {
	return s.repo.ListJobs(ctx, tenantID)
}
func (s *Service) Get(ctx context.Context, tenantID, id string) (*domain.ImportJob, error) {
	row, err := s.repo.GetJob(ctx, tenantID, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, apperror.NotFound
	}
	return row, err
}

func decoderFor(fileName string) (provider.DataProvider, error) {
	switch strings.ToLower(filepath.Ext(fileName)) {
	case ".csv":
		return provider.CSVProvider{}, nil
	case ".json":
		return provider.JSONProvider{}, nil
	default:
		return nil, fmt.Errorf("仅支持 .csv 和 .json 文件")
	}
}
