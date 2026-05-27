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
			default:
				log.Printf("kafka client misconfigured to consume topic: %q", record.Topic)
			}
		}
	}
}

func (c *Consumer) handleUploadMessages(ctx context.Context, record *kgo.Record) {
	var msg upload.CompletedMessage

	if err := json.Unmarshal(record.Value, &msg); err != nil {
		log.Printf(
			"failed to decode upload message (topic=%s partition=%d offset=%d): %v",
			record.Topic,
			record.Partition,
			record.Offset,
			err,
		)
		return
	}

	var err error

	switch msg.EventType {
	case upload.Completed:
		err = c.handleUploadCompleted(ctx, msg)
	default:
		log.Printf(
			"ignoring unsupported upload event (upload_id=%s event=%s)",
			msg.UploadID,
			msg.EventType,
		)
		return
	}

	if err != nil {
		log.Printf(
			"upload message handling failed (upload_id=%s event=%s): %v",
			msg.UploadID,
			msg.EventType,
			err,
		)
	}

}

func (c *Consumer) handleUploadCompleted(
	ctx context.Context, msg upload.CompletedMessage) error {

	data, err := c.uploadsBucket.ReadAll(ctx, msg.UploadID)
	if err != nil {
		return fmt.Errorf(
			"failed to read upload source from bucket (upload_id=%s): %w",
			msg.UploadID,
			err,
		)
	}

	switch msg.ProfileHint {
	case upload.ProjectReportMd:
		return c.handleProjectReportMarkdown(ctx, msg, data)
	default:
		return fmt.Errorf(
			"%w: unsupported profile hint %q for upload %q",
			serviceerr.InvalidInput,
			msg.ProfileHint,
			msg.UploadID,
		)
	}
}

func (c *Consumer) handleProjectReportMarkdown(ctx context.Context, uploadMsg upload.CompletedMessage, data []byte) error {
	report, err := projectreportmd.Parse(ctx, data)
	if err != nil {
		return fmt.Errorf(
			"failed to parse upload %q as %q: %w",
			uploadMsg.UploadID,
			uploadMsg.ProfileHint,
			err,
		)
	}

	messages := make([]*kgo.Record, len(report.Sections))

	for i, section := range report.Sections {
		content := projectreportmd.Content{
			Title:      report.Frontmatter.Title,
			Subject:    report.Frontmatter.Subject,
			Subheading: section.Subheading,
			Content:    section.Content,
		}

		contentBytes, err := json.Marshal(content)
		if err != nil {
			return fmt.Errorf(
				"failed to marshal chunk content (upload_id=%s, chunk_index=%d): %w",
				uploadMsg.UploadID,
				i,
				err,
			)
		}

		msg, err := json.Marshal(chunk.StreamMessage{
			UploadID:   uploadMsg.UploadID,
			EventName:  chunk.Stream,
			ChunkIndex: i,
			ChunkTotal: len(report.Sections),
			Content:    contentBytes,
			Raw:        projectreportmd.ContentAsString(content),
		})
		if err != nil {
			return fmt.Errorf(
				"failed to marshal chunk stream message (upload_id=%s chunk_index=%d): %w",
				uploadMsg.UploadID,
				i,
				err,
			)
		}

		messages[i] = &kgo.Record{
			Key:   []byte(uploadMsg.UploadID),
			Topic: chunk.TopicName,
			Value: msg,
		}
	}

	if err := c.client.ProduceSync(ctx, messages...).FirstErr(); err != nil {
		return fmt.Errorf(
			"failed to produce chunk stream messages (upload_id=%s chunk_total=%d): %w",
			uploadMsg.UploadID,
			len(messages),
			err,
		)
	}

	return nil
}
