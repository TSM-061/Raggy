package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
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

	e := env.NewHelper(os.LookupEnv)

	cfg, err := config.LoadConfig(e)
	if err != nil {
		log.Error(
			"failed to load config",
			slog.Any("error", err),
		)
		os.Exit(1)

	}

	ctx := context.Background()

	ctx = logger.ToContext(ctx, log)

	ctx, stop := signal.NotifyContext(
		ctx,
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DbConnectionString)
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

	geminiClient, err := ai.NewGeminiClient(ctx, cfg.GeminiConfig)
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
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: server.GetEndpoints(),
	}

	go func() {
		log.InfoContext(ctx, "http server ready", slog.Int("port", cfg.Port))

		if err := httpServer.ListenAndServe(); err != nil {
			isExit := errors.Is(err, http.ErrServerClosed)

			if !isExit {
				log.ErrorContext(ctx, "http server exited", slog.Any("error", err))
			}

			stop()
		}

	}()

	go consumer.Start(ctx)

	<-ctx.Done()

	log.InfoContext(ctx, "http server shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.InfoContext(ctx, "http server shutdown failed", slog.Any("error", err))
		return
	}

	log.InfoContext(ctx, "http server shutdown completed")
}
