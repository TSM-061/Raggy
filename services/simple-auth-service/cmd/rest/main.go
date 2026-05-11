package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/TSM-061/Raggy/shared/env"
	"github.com/TSM-061/Raggy/simple-auth-service/internal/config"
	"github.com/TSM-061/Raggy/simple-auth-service/internal/web"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()

	env := env.NewHelper(os.LookupEnv)
	cfg := config.LoadConfig(env)

	pool, err := pgxpool.New(ctx, cfg.DbConnectionString)
	if err != nil {
		panic("failed to open database connection")
	}
	defer pool.Close()

	server, err := web.NewServer(cfg, pool)
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/auth/signin", server.HandleSignin)
	mux.HandleFunc("POST /api/auth/refresh", server.HandleRefresh)
	mux.HandleFunc("POST /api/auth/signout", server.HandleSignout)

	log.Printf("Simple Auth Service listening on :%d", cfg.Port)
	port := fmt.Sprintf(":%d", cfg.Port)
	log.Fatal(http.ListenAndServe(port, mux))
}
