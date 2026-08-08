package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/adnova/internal/bootstrap"
	"github.com/example/adnova/internal/config"
	"github.com/example/adnova/internal/taskqueue"
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

	db, err := bootstrap.NewDatabase(cfg.Database)
	if err != nil {
		logger.Fatal("database initialization failed", zap.Error(err))
	}
	if cfg.Demo.Seed {
		if err := bootstrap.SeedDemo(db); err != nil {
			logger.Fatal("demo seed failed", zap.Error(err))
		}
	}
	redisClient, err := bootstrap.NewRedis(context.Background(), cfg.Redis)
	if err != nil {
		logger.Fatal("redis initialization failed", zap.Error(err))
	}
	defer func() { _ = redisClient.Close() }()
	queueClient := taskqueue.NewAsynqEnqueuer(cfg.Redis, cfg.Queue)
	defer func() { _ = queueClient.Close() }()

	server := &http.Server{Addr: cfg.HTTP.Address, Handler: bootstrap.NewRouter(cfg, logger, db, redisClient, queueClient), ReadTimeout: cfg.HTTP.ReadTimeout, WriteTimeout: cfg.HTTP.WriteTimeout}
	go func() {
		logger.Info("api server started", zap.String("address", cfg.HTTP.Address))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("api server stopped unexpectedly", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", zap.Error(err))
	}
}
