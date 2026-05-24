package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TSM-061/Raggy/dashboard/internal/app"
	"github.com/TSM-061/Raggy/dashboard/internal/config"
	"github.com/TSM-061/Raggy/dashboard/internal/events"
	"github.com/TSM-061/Raggy/dashboard/internal/web"
	"github.com/TSM-061/Raggy/shared/env"
	"github.com/TSM-061/Raggy/shared/storage"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/twmb/franz-go/pkg/kgo"
)

var uploadProfiles = []string{
	"project_summary_markdown",
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	envHelper := env.NewHelper(os.LookupEnv)
	cfg := config.LoadConfig(envHelper)

	pool, err := pgxpool.New(ctx, cfg.DbConnectionString)
	if err != nil {
		panic("failed to open database connection")
	}
	defer pool.Close()

	bucket, err := storage.OpenS3Bucket(ctx, cfg.S3Credentials, cfg.S3Config)
	if err != nil {
		panic(err)
	}
	defer bucket.Close()

	kafkaClient, err := kgo.NewClient(
		kgo.SeedBrokers("kafka:9092"),
		kgo.ConsumerGroup("dashboard-service"),
		kgo.ConsumeTopics("s3-events"),
	)
	if err != nil {
		log.Fatalf("failed to create kafka client: %v", err)
	}
	defer kafkaClient.Close()

	application := app.New(cfg, pool, bucket, uploadProfiles, validator.New())

	server, err := web.NewServer(cfg, application)
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	uploadHandler := server.Auth().Wrap(http.HandlerFunc(server.HandleUpload))
	listUploadsHandler := server.Auth().Wrap(http.HandlerFunc(server.HandleListUploads))
	deleteUploadHandler := server.Auth().Wrap(http.HandlerFunc(server.HandleDeleteUpload))
	mux.Handle("POST /api/uploads", uploadHandler)
	mux.Handle("GET /api/uploads", listUploadsHandler)
	mux.Handle("DELETE /api/uploads/{id}", deleteUploadHandler)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: mux,
	}

	consumer := events.NewConsumer(kafkaClient, application)
	go consumer.Start(ctx)

	go func() {
		log.Printf("Rest listening on :%d", cfg.Port)

		if err := httpServer.ListenAndServe(); err != nil {
			isStopping := errors.Is(err, http.ErrServerClosed)

			if !isStopping {
				log.Fatalf("(Rest) Error Starting: %v", err)
			}
		}
	}()

	<-ctx.Done() // wait for shutdown signal

	log.Println("Shutting down...")

	shutdownCtx, cancelShutdown := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancelShutdown()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP shutdown error: %v", err)
	} else {
		log.Println("HTTP server stopped successfully.")
	}
}
