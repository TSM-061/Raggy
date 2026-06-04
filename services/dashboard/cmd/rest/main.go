package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TSM-061/Raggy/dashboard/internal/config"
	"github.com/TSM-061/Raggy/dashboard/internal/consumer"
	"github.com/TSM-061/Raggy/dashboard/internal/services"
	"github.com/TSM-061/Raggy/dashboard/internal/upload"
	"github.com/TSM-061/Raggy/dashboard/internal/web"
	"github.com/TSM-061/Raggy/shared/auth"
	"github.com/TSM-061/Raggy/shared/clock"
	"github.com/TSM-061/Raggy/shared/logger"
	"github.com/TSM-061/Raggy/shared/storage"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	log := logger.New(cfg.LogLevel)

	baseCtx := context.Background()
	baseCtx = logger.ToContext(baseCtx, log)

	runCtx, stop := signal.NotifyContext(
		baseCtx,
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	pool, err := pgxpool.New(baseCtx, cfg.DbConnectionString)
	if err != nil {
		log.Error("failed to open database connection pool", slog.Any("error", err))
		os.Exit(1)
	}
	defer pool.Close()

	bucket, err := storage.OpenS3Bucket(baseCtx, cfg.S3Credentials, cfg.S3Config)
	if err != nil {
		log.Error("failed to open bucket", slog.Any("error", err))
		os.Exit(1)
	}
	defer bucket.Close()

	uploads := upload.NewPostgresRepo(pool)
	uploadService := services.NewUploadService(uploads, bucket, cfg.UploadURLTTL, validator.New())

	consumer, err := consumer.New(cfg.ConsumerConfig, uploadService)
	if err != nil {
		log.Error("failed to create kafka consumer", slog.Any("error", err))
		os.Exit(1)
	}
	defer consumer.Close()

	clock := clock.LiveClock{}
	verifier := auth.NewTokenVerifier(&clock, cfg.AccessTokenPublicKey)
	auth := auth.NewMiddleware(verifier)
	server := web.NewServer(cfg, log, auth, uploadService)

	go consumer.Start(runCtx)
	go server.Start(runCtx, stop)
	<-runCtx.Done()

	log.InfoContext(baseCtx, "http server shutting down")
	shutdownCtx, cancel := context.WithTimeout(baseCtx, 5*time.Second)
	defer cancel()
	server.Shutdown(shutdownCtx)
}
