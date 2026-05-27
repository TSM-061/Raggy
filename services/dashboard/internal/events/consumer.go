package events

import (
	"context"
	"encoding/json"
	"log"

	"github.com/TSM-061/Raggy/dashboard/internal/app"
	"github.com/TSM-061/Raggy/dashboard/internal/upload"
	uploadmsg "github.com/TSM-061/Raggy/shared/message/upload"
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

		uploadID, err := uuid.Parse(s3Record.S3.Object.Key)
		if err != nil {
			log.Printf("invalid upload id in s3 event key %q: %v", s3Record.S3.Object.Key, err)
			continue
		}

		if err := c.app.UploadService.UpdateStatus(ctx, uploadID, upload.StatusUploaded); err != nil {
			log.Printf("failed to update upload status for %s: %v", uploadID, err)
			return
		}

		evt := uploadmsg.CompletedMessage{
			EventType:   uploadmsg.Completed,
			UploadID:    uploadID.String(),
			ProfileHint: uploadmsg.ProjectReportMd,
		}
		eventJson, err := json.Marshal(evt)
		if err != nil {
			return
		}

		record := &kgo.Record{
			Topic: uploadmsg.TopicName,
			Key:   []byte(evt.UploadID),
			Value: eventJson,
		}

		if err := c.client.ProduceSync(ctx, record).FirstErr(); err != nil {
			log.Printf("failed to produce message for %s: %v", uploadID, err)
			return
		}
	}
}
