package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/adnova/internal/bootstrap"
	"github.com/example/adnova/internal/config"
	"github.com/example/adnova/internal/taskqueue"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	logger, err := bootstrap.NewLogger(cfg.Environment)
	if err != nil {
		panic(err)
	}
	defer func() { _ = logger.Sync() }()
	db, err := bootstrap.OpenDatabase(cfg.Database)
	if err != nil {
		logger.Fatal("worker database initialization failed", zap.Error(err))
	}
	queueClient := taskqueue.NewAsynqEnqueuer(cfg.Redis, cfg.Queue)
	defer func() { _ = queueClient.Close() }()
	businessService, err := bootstrap.NewBusinessService(cfg, db, queueClient)
	if err != nil {
		logger.Fatal("worker business service initialization failed", zap.Error(err))
	}
	server := asynq.NewServer(taskqueue.RedisOption(cfg.Redis), asynq.Config{
		Concurrency:     cfg.Queue.Concurrency,
		Queues:          map[string]int{cfg.Queue.Name: 1},
		ShutdownTimeout: cfg.Queue.ShutdownTimeout,
		ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, taskErr error) {
			logger.Error("asynq task failed", zap.String("type", task.Type()), zap.Error(taskErr))
		}),
	})
	mux := asynq.NewServeMux()
	mux.Handle(taskqueue.TypeBusinessAnalysis, taskqueue.NewBusinessHandler(businessService))
	if err := server.Start(mux); err != nil {
		logger.Fatal("asynq worker start failed", zap.Error(err))
	}
	dispatchCtx, cancelDispatch := context.WithCancel(context.Background())
	defer cancelDispatch()
	go func() {
		ticker := time.NewTicker(cfg.Queue.DispatchInterval)
		defer ticker.Stop()
		for {
			if err := businessService.DispatchPending(dispatchCtx, 50); err != nil && !errors.Is(err, context.Canceled) {
				logger.Warn("outbox dispatch failed", zap.Error(err))
			}
			select {
			case <-dispatchCtx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	if cfg.MMPAutoSync.Enabled {
		mmpService := bootstrap.NewMMPService(cfg, db)
		go func() {
			run := func() {
				synced, syncErr := mmpService.AutoSyncAll(dispatchCtx, cfg.MMPAutoSync.LookbackDays)
				if syncErr != nil && !errors.Is(syncErr, context.Canceled) {
					logger.Warn("MMP auto sync completed with failures", zap.Int("synced", synced), zap.Error(syncErr))
				} else if syncErr == nil {
					logger.Info("MMP auto sync completed", zap.Int("synced", synced))
				}
			}
			run()
			ticker := time.NewTicker(cfg.MMPAutoSync.Interval)
			defer ticker.Stop()
			for {
				select {
				case <-dispatchCtx.Done():
					return
				case <-ticker.C:
					run()
				}
			}
		}()
		logger.Info("MMP auto sync enabled", zap.Duration("interval", cfg.MMPAutoSync.Interval), zap.Int("lookback_days", cfg.MMPAutoSync.LookbackDays))
	}
	logger.Info("asynq worker ready", zap.String("queue", cfg.Queue.Name), zap.Int("concurrency", cfg.Queue.Concurrency))
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	cancelDispatch()
	server.Shutdown()
}
