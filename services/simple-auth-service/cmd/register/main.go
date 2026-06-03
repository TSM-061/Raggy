package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"

	"github.com/TSM-061/Raggy/shared/logger"
	"github.com/TSM-061/Raggy/simple-auth-service/internal/config"
	"github.com/TSM-061/Raggy/simple-auth-service/internal/web"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	args := os.Args[1:]

	if len(args) > 2 {
		fmt.Fprintln(os.Stderr, "Usage: go run ./cmd/register <username>")
		os.Exit(1)
	}

	username := args[0]

	passwordBytes := make([]byte, 24)
	rand.Read(passwordBytes)
	password := base64.StdEncoding.EncodeToString(passwordBytes)

	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	log := logger.New(cfg.LogLevel)

	baseCtx := context.Background()
	baseCtx = logger.ToContext(baseCtx, log)

	pool, err := pgxpool.New(baseCtx, cfg.DbConnectionString)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect to database: %v", err)
		os.Exit(1)
	}
	defer pool.Close()

	server := web.NewServer(cfg, log, pool)

	if _, err := server.Auth().Register(baseCtx, username, password); err != nil {
		fmt.Fprintf(os.Stderr, "user registration failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nuser created successfully!\n\n")
	fmt.Printf("username: %s\n", username)
	fmt.Printf("password: %s\n", password)
}
