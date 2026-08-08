package dispatcher

import (
	"context"
	"time"

	analysisservice "github.com/example/adnova/internal/analysis/service"
	"github.com/example/adnova/internal/ingestion/domain"
)

type Pipeline interface {
	Run(context.Context, string, string) (*analysisservice.Result, error)
}

type WindowRepository interface {
	ListReadyAnalysisWindows(context.Context, int) ([]domain.AnalysisWindow, error)
	ClaimAnalysisWindow(context.Context, string, int, time.Duration) (string, bool, error)
	RetryAnalysisWindow(context.Context, string, int, string, int, time.Time, string) error
	CompleteAnalysisWindow(context.Context, string, int, string) error
}

type Dispatcher struct {
	repo     WindowRepository
	pipeline Pipeline
	lease    time.Duration
}

func New(repo WindowRepository, pipeline Pipeline, lease time.Duration) *Dispatcher {
	return &Dispatcher{repo: repo, pipeline: pipeline, lease: lease}
}

func (d *Dispatcher) RunOnce(ctx context.Context, limit int) error {
	windows, err := d.repo.ListReadyAnalysisWindows(ctx, limit)
	if err != nil {
		return err
	}
	for _, window := range windows {
		token, claimed, err := d.repo.ClaimAnalysisWindow(ctx, window.ID, window.Version, d.lease)
		if err != nil {
			return err
		}
		if !claimed {
			continue
		}
		if _, err := d.pipeline.Run(ctx, window.TenantID, window.GameID); err != nil {
			attempts := window.AttemptCount + 1
			delay := time.Duration(1<<min(attempts, 8)) * time.Second
			if delay > 5*time.Minute {
				delay = 5 * time.Minute
			}
			if retryErr := d.repo.RetryAnalysisWindow(ctx, window.ID, window.Version, token, attempts, time.Now().UTC().Add(delay), err.Error()); retryErr != nil {
				return retryErr
			}
			continue
		}
		if err := d.repo.CompleteAnalysisWindow(ctx, window.ID, window.Version, token); err != nil {
			return err
		}
	}
	return nil
}
