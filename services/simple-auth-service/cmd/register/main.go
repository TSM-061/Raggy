package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/TSM-061/Raggy/shared/env"
	"github.com/TSM-061/Raggy/simple-auth-service/internal/config"
	"github.com/TSM-061/Raggy/simple-auth-service/internal/web"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
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

	// setup application to complete registration

	ctx := context.Background()

	env := env.NewHelper(os.LookupEnv)
	cfg := config.LoadConfig(env)

	pool, err := pgxpool.New(ctx, cfg.DbConnectionString)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect to database: %v", err)
		os.Exit(1)
	}
	defer pool.Close()

	server, err := web.NewServer(cfg, pool)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize application: %v", err)
		os.Exit(1)
	}

	if _, err := server.Auth.Register(ctx, username, password); err != nil {
		fmt.Fprintf(os.Stderr, "user registration failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nuser created successfully!\n\n")
	fmt.Printf("username: %s\n", username)
	fmt.Printf("password: %s\n", password)
}
