package main

import (
	"context"
	"log"
	"os"

	"github.com/TSM-061/Raggy/ingestion-worker/internal/config"
	"github.com/TSM-061/Raggy/ingestion-worker/internal/events"
	"github.com/TSM-061/Raggy/shared/env"
	"github.com/TSM-061/Raggy/shared/storage"
	"github.com/twmb/franz-go/pkg/kgo"
)

func main() {
	ctx := context.Background()

	h := env.NewHelper(os.LookupEnv)
	cfg := config.LoadConfig(h)

	bucket, err := storage.OpenS3Bucket(ctx, cfg.S3Credentials, cfg.S3Config)
	if err != nil {
		panic(err)
	}
	defer bucket.Close()

	kafkaClient, err := kgo.NewClient(
		kgo.SeedBrokers(cfg.KafkaSeedBrokers...),
		kgo.ConsumerGroup("ingestion-worker"),
		kgo.ConsumeTopics("uploads"),
	)
	if err != nil {
		log.Fatalf("failed to create kafka client: %v", err)
		return
	}
	defer kafkaClient.Close()
	// TODO handle graceful shutdown, commit on shutdown

	events.NewConsumer(kafkaClient, bucket).Start(ctx)
}
