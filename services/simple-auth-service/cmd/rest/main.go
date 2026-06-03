package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TSM-061/Raggy/shared/auth"
	"github.com/TSM-061/Raggy/shared/clock"
	"github.com/TSM-061/Raggy/shared/logger"
	"github.com/TSM-061/Raggy/simple-auth-service/internal/config"
	"github.com/TSM-061/Raggy/simple-auth-service/internal/password"
	"github.com/TSM-061/Raggy/simple-auth-service/internal/services"
	"github.com/TSM-061/Raggy/simple-auth-service/internal/session"
	"github.com/TSM-061/Raggy/simple-auth-service/internal/user"
	"github.com/TSM-061/Raggy/simple-auth-service/internal/web"
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
		log.Error(
			"failed to open database connection pool",
			slog.Any("error", err),
		)
		os.Exit(1)
	}
	defer pool.Close()

	clock := &clock.LiveClock{}

	hasher := password.NewArgon2Hasher(cfg.Argon2Config)
	signer := auth.NewTokenSigner(cfg.AccessTokenConfig, clock)

	users := user.NewPostgresRepo(pool)
	sessions := session.NewPostgresRepo(pool)

	sessionManager := session.NewManager(cfg.RefreshTokenSecret, sessions)
	auth := services.NewAuthService(users, hasher, signer, sessionManager)

	server := web.NewServer(cfg, log, auth)

	go server.Start(runCtx, stop)

	<-runCtx.Done()

	log.InfoContext(baseCtx, "http server shutting down")

	shutdownCtx, cancel := context.WithTimeout(baseCtx, 5*time.Second)
	defer cancel()
	server.Shutdown(shutdownCtx)
}
