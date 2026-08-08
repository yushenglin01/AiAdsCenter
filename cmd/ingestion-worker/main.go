package main

import (
	"context"
	"errors"
	"os/signal"
	"syscall"
	"time"

	analysisservice "github.com/example/adnova/internal/analysis/service"
	attributionrepo "github.com/example/adnova/internal/attribution/repository"
	attributionservice "github.com/example/adnova/internal/attribution/service"
	"github.com/example/adnova/internal/bootstrap"
	"github.com/example/adnova/internal/config"
	creativeanalysisrepo "github.com/example/adnova/internal/creative/analysis/repository"
	creativeanalysisservice "github.com/example/adnova/internal/creative/analysis/service"
	"github.com/example/adnova/internal/ingestion/consumer"
	"github.com/example/adnova/internal/ingestion/dispatcher"
	ingestionrepo "github.com/example/adnova/internal/ingestion/repository"
	ingestionservice "github.com/example/adnova/internal/ingestion/service"
	metricsrepo "github.com/example/adnova/internal/metrics/repository"
	metricsservice "github.com/example/adnova/internal/metrics/service"
	rulesrepo "github.com/example/adnova/internal/rules/repository"
	rulesservice "github.com/example/adnova/internal/rules/service"
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
	if !cfg.Kafka.Enabled {
		logger.Info("Kafka ingestion worker disabled")
		return
	}
	db, err := bootstrap.OpenDatabase(cfg.Database)
	if err != nil {
		logger.Fatal("ingestion worker database initialization failed", zap.Error(err))
	}
	repo := ingestionrepo.New(db)
	service := ingestionservice.New(repo)
	metrics := metricsservice.New(metricsrepo.New(db))
	pipeline := analysisservice.NewPipeline(
		metrics,
		rulesservice.New(rulesrepo.New(db), metrics),
		attributionservice.New(attributionrepo.New(db)),
		creativeanalysisservice.New(creativeanalysisrepo.New(db)),
	)
	dispatch := dispatcher.New(repo, pipeline, cfg.Kafka.AnalysisLease)
	kafkaConsumer, err := consumer.New(cfg.Kafka, service, logger)
	if err != nil {
		logger.Fatal("Kafka consumer initialization failed", zap.Error(err))
	}
	defer kafkaConsumer.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	pingCtx, cancelPing := context.WithTimeout(ctx, 10*time.Second)
	if err := kafkaConsumer.Ping(pingCtx); err != nil {
		cancelPing()
		logger.Fatal("Kafka broker health check failed", zap.Error(err))
	}
	cancelPing()

	dispatchErr := make(chan error, 1)
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			if err := dispatch.RunOnce(ctx, 20); err != nil && !errors.Is(err, context.Canceled) {
				dispatchErr <- err
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	logger.Info("Kafka ingestion worker ready", zap.Strings("brokers", cfg.Kafka.Brokers), zap.Strings("topics", cfg.Kafka.Topics), zap.String("group", cfg.Kafka.GroupID))
	consumerErr := make(chan error, 1)
	go func() { consumerErr <- kafkaConsumer.Run(ctx) }()
	select {
	case <-ctx.Done():
	case err := <-dispatchErr:
		logger.Fatal("analysis dispatcher stopped", zap.Error(err))
	case err := <-consumerErr:
		if err != nil && !errors.Is(err, context.Canceled) {
			logger.Fatal("Kafka consumer stopped", zap.Error(err))
		}
	}
}
