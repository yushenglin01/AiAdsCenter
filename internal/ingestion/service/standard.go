package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/example/adnova/internal/common/apperror"
	"github.com/example/adnova/internal/common/identity"
	"github.com/example/adnova/internal/ingestion/contract"
	"github.com/example/adnova/internal/ingestion/domain"
	"github.com/example/adnova/internal/ingestion/repository"
)

type PermanentError struct {
	Code string
	Err  error
}

func (e *PermanentError) Error() string { return e.Code + ": " + e.Err.Error() }
func (e *PermanentError) Unwrap() error { return e.Err }

type KafkaMessage struct {
	ExpectedTenantID string
	ExpectedProducer string
	Topic            string
	Partition        int32
	Offset           int64
	Payload          []byte
	Debounce         time.Duration
	Lease            time.Duration
}

func (s *Service) ImportBatch(ctx context.Context, tenantID, userID string, batch contract.Batch) (*domain.ImportJob, error) {
	return s.importBatch(ctx, tenantID, userID, batch, false)
}

func (s *Service) importBatch(ctx context.Context, tenantID, userID string, batch contract.Batch, allowResume bool) (*domain.ImportJob, error) {
	batch.Normalize()
	if err := batch.Validate(); err != nil {
		return nil, apperror.Validation(err.Error())
	}
	if batch.TenantID != "" && batch.TenantID != tenantID {
		return nil, apperror.Forbidden
	}
	batch.TenantID = tenantID
	game, err := s.repo.FindGameByCode(ctx, tenantID, batch.GameCode)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, apperror.Validation("game_code 不存在或不属于当前公司")
	}
	if err != nil {
		return nil, fmt.Errorf("find import game: %w", err)
	}
	records, err := json.Marshal(batch.Records)
	if err != nil {
		return nil, apperror.Validation("records 无法编码")
	}
	payload, _ := json.Marshal(batch)
	digest := sha256.Sum256(payload)
	start, end := batch.PeriodTimes()
	collectedAt, _ := time.Parse(time.RFC3339, batch.CollectedAt)
	importType := map[string]string{
		contract.DatasetAd: ImportAd, contract.DatasetMMP: ImportMMP,
		contract.DatasetRevenue: ImportRevenue, contract.DatasetCreative: ImportCreative,
	}[batch.DatasetType]
	return s.Import(ctx, ImportInput{
		TenantID: tenantID, UserID: userID, GameID: game.ID, Source: batch.Source,
		FileName: batch.BatchID + ".json", Data: records, ImportType: importType,
		Authoritative:     batch.WriteMode == contract.WriteReplaceRange,
		AllowCustomSource: true, AllowResume: allowResume,
		PeriodStart: start, PeriodEnd: end, BatchID: batch.BatchID,
		ProducerSystem: batch.Producer.System, SchemaVersion: batch.SchemaVersion,
		CollectedAt: &collectedAt, PayloadHash: hex.EncodeToString(digest[:]),
	})
}

func (s *Service) ProcessKafkaMessage(ctx context.Context, input KafkaMessage) error {
	batch, err := contract.Decode(input.Payload)
	if err != nil {
		return &PermanentError{Code: "SCHEMA_INVALID", Err: fmt.Errorf("decode batch: %w", err)}
	}
	batch.Normalize()
	if batch.TenantID == "" || batch.TenantID != input.ExpectedTenantID {
		return &PermanentError{Code: "TENANT_MISMATCH", Err: errors.New("payload tenant does not match consumer tenant")}
	}
	if batch.Producer.System != input.ExpectedProducer {
		return &PermanentError{Code: "PRODUCER_MISMATCH", Err: errors.New("payload producer does not match consumer producer")}
	}
	if err := batch.Validate(); err != nil {
		return &PermanentError{Code: "SCHEMA_INVALID", Err: err}
	}
	digest := sha256.Sum256(input.Payload)
	message, claimed, err := s.repo.ClaimMessage(ctx, repository.MessageClaim{
		TenantID: batch.TenantID, EventID: batch.BatchID, ProducerSystem: batch.Producer.System,
		SchemaVersion: batch.SchemaVersion, Topic: input.Topic, Partition: input.Partition,
		Offset: input.Offset, PayloadHash: hex.EncodeToString(digest[:]), Lease: input.Lease,
	})
	if err != nil {
		if errors.Is(err, repository.ErrMessageBusy) {
			return err
		}
		if errors.Is(err, repository.ErrMessageConflict) {
			return &PermanentError{Code: "IDEMPOTENCY_CONFLICT", Err: err}
		}
		return err
	}
	if !claimed {
		return nil
	}
	job, err := s.importBatch(ctx, batch.TenantID, identity.SystemAgentUserID, batch, true)
	if err != nil {
		var appErr *apperror.Error
		if errors.As(err, &appErr) && appErr.HTTPStatus < 500 {
			_ = s.repo.FailMessage(ctx, message.ID, "IMPORT_VALIDATION_FAILED", appErr.Message)
			return &PermanentError{Code: "IMPORT_VALIDATION_FAILED", Err: err}
		}
		_ = s.repo.FailMessage(ctx, message.ID, "IMPORT_TEMPORARY_ERROR", err.Error())
		return err
	}
	start, end := batch.PeriodTimes()
	if err := s.repo.TouchAnalysisWindow(ctx, batch.TenantID, job.GameID, start, end, batch.DatasetType, input.Debounce); err != nil {
		_ = s.repo.FailMessage(ctx, message.ID, "ANALYSIS_WINDOW_ERROR", err.Error())
		return err
	}
	return s.repo.CompleteMessage(ctx, message.ID)
}
