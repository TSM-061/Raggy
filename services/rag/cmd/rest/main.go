package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TSM-061/Raggy/rag/internal/ai"
	"github.com/TSM-061/Raggy/rag/internal/chunk"
	"github.com/TSM-061/Raggy/rag/internal/config"
	"github.com/TSM-061/Raggy/rag/internal/consumer"
	"github.com/TSM-061/Raggy/rag/internal/rag"
	"github.com/TSM-061/Raggy/rag/internal/upload"
	"github.com/TSM-061/Raggy/rag/internal/web"
	"github.com/TSM-061/Raggy/shared/logger"
	"github.com/TSM-061/Raggy/shared/telemetry"
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

	tp, err := telemetry.InitTracerProvider(baseCtx, "rag-service", cfg.Telemetry)
	if err != nil {
		slog.Error("failed to init telemetry", slog.Any("error", err))
		os.Exit(1)
	}
	defer tp.Shutdown(baseCtx)

	pool, err := pgxpool.New(baseCtx, cfg.DbConnectionString)
	if err != nil {
		log.Error(
			"failed to open database connection pool",
			slog.Any("error", err),
		)
		os.Exit(1)
	}
	defer pool.Close()

	chunks := chunk.NewPostgresRepo(pool)
	uploads := upload.NewPostgresRepo(pool)

	// geminiClient, err := ai.NewGeminiClient(baseCtx, cfg.GeminiConfig)
	// if err != nil {
	// 	log.Error(
	// 		"failed to create gemini ai client",
	// 		slog.Any("error", err),
	// 	)
	// 	os.Exit(1)
	// }

	mockAIClient := &ai.MockAI{}
	ragService := rag.New(uploads, chunks, mockAIClient, mockAIClient)

	tracker := consumer.NewJobTracker(ragService)

	consumer, err := consumer.New(cfg.ConsumerConfig, tracker)
	if err != nil {
		log.Error(
			"failed to create kafka consumer",
			slog.Any("error", err),
		)
		os.Exit(1)
	}
	defer consumer.Close()

	server := web.NewServer(cfg, log, ragService)

	go server.Start(runCtx, stop)
	go consumer.Start(runCtx)
	<-runCtx.Done()

	log.InfoContext(baseCtx, "http server shutting down")
	shutdownCtx, cancel := context.WithTimeout(baseCtx, 5*time.Second)
	defer cancel()
	server.Shutdown(shutdownCtx)
}
