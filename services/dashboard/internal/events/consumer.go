package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/TSM-061/Raggy/dashboard/internal/app"
	"github.com/TSM-061/Raggy/dashboard/internal/upload"
	"github.com/google/uuid"
	"github.com/twmb/franz-go/pkg/kgo"
)

type Consumer struct {
	client *kgo.Client
	app    *app.App
}

func NewConsumer(client *kgo.Client, app *app.App) *Consumer {
	return &Consumer{
		client: client,
		app:    app,
	}
}

func (c *Consumer) Start(ctx context.Context) {
	log.Println("Kafka ready")

	for {
		fetches := c.client.PollFetches(ctx)
		if fetches.IsClientClosed() {
			return
		}

		if errs := fetches.Errors(); len(errs) > 0 {
			log.Printf("Kafka poll errors encountered: %v", errs)
		}

		iter := fetches.RecordIter()
		for !iter.Done() {
			record := iter.Next()
			c.processEvent(ctx, record)
		}
	}
}

func (c *Consumer) processEvent(ctx context.Context, record *kgo.Record) {
	fmt.Printf("Message received - Topic: %s, Key: %s\n",
		record.Topic,
		string(record.Key),
	)

	switch record.Topic {
	case "s3-events":
		c.handleS3Event(ctx, record)
	}
}

func (c *Consumer) handleS3Event(ctx context.Context, record *kgo.Record) {
	var payload MinioPayload

	if err := json.Unmarshal(record.Value, &payload); err != nil {
		log.Printf("failed to decode s3 event payload: %v", err)
		return
	}

	for _, s3Record := range payload.Records {
		if s3Record.EventName != "s3:ObjectCreated:Put" {
			continue
		}

		objectKey := strings.TrimSpace(s3Record.S3.Object.Key)

		uploadID, err := uuid.Parse(objectKey)
		if err != nil {
			log.Printf("invalid upload id in s3 event key %q: %v", objectKey, err)
			continue
		}

		if err := c.app.UploadService.UpdateStatus(ctx, uploadID, upload.StatusUploaded); err != nil {
			log.Printf("failed to update upload status for %s: %v", uploadID, err)
		}
	}
}
