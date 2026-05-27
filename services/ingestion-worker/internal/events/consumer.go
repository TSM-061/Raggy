package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/TSM-061/Raggy/ingestion-worker/internal/projectreportmd"
	"github.com/TSM-061/Raggy/shared/message/chunk"
	"github.com/TSM-061/Raggy/shared/message/upload"
	"github.com/TSM-061/Raggy/shared/serviceerr"
	"github.com/twmb/franz-go/pkg/kgo"

	"gocloud.dev/blob"
)

type Consumer struct {
	client        *kgo.Client
	uploadsBucket *blob.Bucket
}

func NewConsumer(client *kgo.Client, uploadsBucket *blob.Bucket) *Consumer {
	return &Consumer{
		uploadsBucket: uploadsBucket,
		client:        client,
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

			switch record.Topic {
			case upload.TopicName:
				c.handleUploadMessages(ctx, record)
			}
		}
	}
}

func (c *Consumer) handleUploadMessages(ctx context.Context, record *kgo.Record) {
	var msg upload.CompletedMessage

	if err := json.Unmarshal(record.Value, &msg); err != nil {
		log.Printf("failed to decode s3 event payload: %v", err)
		return
	}

	var err error

	switch msg.EventType {
	case upload.Completed:
		err = c.handleUploadCompleted(ctx, msg)
	default:
		err = nil
	}

	if err != nil {
		log.Printf("error handling upload message: %v", err)
	}

}

func (c *Consumer) handleUploadCompleted(
	ctx context.Context, msg upload.CompletedMessage) error {

	data, err := c.uploadsBucket.ReadAll(ctx, msg.UploadID)
	if err != nil {
		log.Printf("failed to read uploads bucket: %v", err)
		return err
	}

	switch msg.ProfileHint {
	case upload.ProjectReportMd:
		return c.handleProjectReportMarkdown(ctx, msg, data)
	default:
		return fmt.Errorf(
			"%w: profile hint not supported %q",
			serviceerr.InvalidInput,
			msg.ProfileHint,
		)
	}
}

func (c *Consumer) handleProjectReportMarkdown(ctx context.Context, msg upload.CompletedMessage, data []byte) error {
	report, err := projectreportmd.Parse(ctx, data)
	if err != nil {
		return err
	}

	messages := make([]*kgo.Record, len(report.Sections))

	for i, s := range report.Sections {
		chunkMsg, err := json.Marshal(chunk.StreamMessage{
			UploadID:   msg.UploadID,
			EventName:  chunk.Stream,
			ChunkIndex: i,
			ChunkTotal: len(report.Sections),
			Payload: fmt.Sprintf(
				"Project:%s\nSubject:%s\nSubheading:%s\nContent:%s",
				report.Frontmatter.Title,
				report.Frontmatter.Subject,
				s.Subheading,
				s.Content,
			),
		})
		if err != nil {
			return err
		}

		messages[i] = &kgo.Record{
			Key:   []byte(msg.UploadID),
			Topic: chunk.TopicName,
			Value: chunkMsg,
		}
	}

	if err := c.client.ProduceSync(ctx, messages...).FirstErr(); err != nil {
		log.Printf("failed to produce messages for %s: %v", msg.UploadID, err)
		return err
	}

	return nil
}
