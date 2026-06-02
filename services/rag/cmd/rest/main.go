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
	"github.com/TSM-061/Raggy/shared/env"
	"github.com/TSM-061/Raggy/shared/logger"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	baseCtx := context.Background()
	baseCtx = logger.ToContext(baseCtx, log)

	e := env.NewHelper(os.LookupEnv)

	cfg, err := config.LoadConfig(e)
	if err != nil {
		log.Error(
			"failed to load config",
			slog.Any("error", err),
		)
		os.Exit(1)
	}

	runCtx, stop := signal.NotifyContext(
		baseCtx,
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

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

	geminiClient, err := ai.NewGeminiClient(baseCtx, cfg.GeminiConfig)
	if err != nil {
		log.Error(
			"failed to create gemini ai client",
			slog.Any("error", err),
		)
		os.Exit(1)
	}

	ragService := rag.New(uploads, chunks, geminiClient, geminiClient)

	consumer, err := consumer.New(cfg.ConsumerConfig, ragService)
	if err != nil {
		log.Error(
			"failed to create kafka consumer",
			slog.Any("error", err),
		)
		os.Exit(1)
	}
	defer consumer.Close()

	server := web.NewServer(cfg, ragService)

	go server.Start(runCtx, stop)
	go consumer.Start(runCtx)
	<-runCtx.Done()

	log.InfoContext(baseCtx, "http server shutting down")

	shutdownCtx, cancel := context.WithTimeout(baseCtx, 5*time.Second)
	defer cancel()
	server.Shutdown(shutdownCtx)
}
