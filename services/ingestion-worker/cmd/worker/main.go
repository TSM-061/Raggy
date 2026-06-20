package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/TSM-061/Raggy/ingestion-worker/internal/config"
	"github.com/TSM-061/Raggy/ingestion-worker/internal/consumer"
	"github.com/TSM-061/Raggy/shared/logger"
	"github.com/TSM-061/Raggy/shared/storage"
	"github.com/TSM-061/Raggy/shared/telemetry"
)

func main() {

	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	ctx := context.Background()

	log := logger.New(cfg.LogLevel)
	ctx = logger.ToContext(ctx, log)

	tp, err := telemetry.InitTracerProvider(ctx, "ingestion-worker", cfg.Telemetry)
	if err != nil {
		slog.Error("failed to init telemetry", slog.Any("error", err))
		os.Exit(1)
	}
	defer tp.Shutdown(ctx)

	uploadsBucket, err := storage.OpenS3Bucket(ctx, cfg.S3Credentials, cfg.S3Config)
	if err != nil {
		log.Error("failed to open bucket", slog.Any("error", err))
		os.Exit(1)
	}
	defer uploadsBucket.Close()

	consumer, err := consumer.New(cfg.ConsumerConfig, uploadsBucket)
	if err != nil {
		log.Error("failed to create kafka consumer", slog.Any("error", err))
		os.Exit(1)
	}
	defer consumer.Close()

	consumer.Start(ctx)
}
