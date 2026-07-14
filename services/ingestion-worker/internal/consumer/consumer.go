package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/TSM-061/Raggy/ingestion-worker/internal/projectreportmd"
	"github.com/TSM-061/Raggy/shared/logger"
	"github.com/TSM-061/Raggy/shared/message/chunk"
	"github.com/TSM-061/Raggy/shared/message/upload"
	"github.com/TSM-061/Raggy/shared/serviceerr"
	"github.com/TSM-061/Raggy/shared/telemetry"
	"github.com/google/uuid"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("github.com/TSM-061/Raggy/ingestion-worker/internal/consumer")

type Downloader interface {
	Download(ctx context.Context, key uuid.UUID) ([]byte, error)
}

type Config struct {
	MaxPollRecords int      `env:"MAX_POLL_RECORDS" envDefault:"10"`
	SeedBrokers    []string `env:"SEED_BROKERS,required"`
}

type Runner struct {
	client  *kgo.Client
	uploads Downloader
	config  *Config
}

func New(cfg *Config, uploads Downloader) (*Runner, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(cfg.SeedBrokers...),
		kgo.ConsumerGroup("ingestion-worker"),
		kgo.ConsumeTopics(upload.TopicName),
		kgo.WithHooks(telemetry.NewKafkaHook()),

		kgo.SessionTimeout(2*time.Second),
		kgo.HeartbeatInterval(800*time.Millisecond),
	)
	if err != nil {
		return nil, err
	}

	return &Runner{
		config:  cfg,
		uploads: uploads,
		client:  client,
	}, nil
}

func (c *Runner) Start(ctx context.Context) {
	log := logger.FromContext(ctx)
	log.InfoContext(ctx, "kafka consumer ready")

	for {
		fetches := c.client.PollRecords(ctx, c.config.MaxPollRecords)

		if fetches.IsClientClosed() {
			return
		}

		if errs := fetches.Errors(); len(errs) > 0 {
			log.ErrorContext(ctx, "Kafka poll errors encountered", slog.Any("errors", errs))
		}

		for record := range fetches.RecordsAll() {
			// TODO separate this out into another file and add the upload_id as an attribute
			recordCtx, span := tracer.Start(
				ctx,
				"Upload.Ingest",
				trace.WithLinks(trace.LinkFromContext(telemetry.Extract(ctx, record))),
			)

			c.handleUploadMessage(recordCtx, record)

			span.End()
		}
	}
}

func (c *Runner) handleUploadMessage(ctx context.Context, record *kgo.Record) {
	log := logger.FromContext(ctx)
	var msg upload.CompletedMessage

	if err := json.Unmarshal(record.Value, &msg); err != nil {
		log.ErrorContext(ctx, "failed to unmarshal message", slog.Any("error", err))
		return
	}

	log = log.With(slog.String("upload_id", msg.UploadID))
	ctx = logger.ToContext(ctx, log)

	switch msg.Type {
	case upload.Completed:
		if err := c.handleUploadCompleted(ctx, msg); err != nil {
			log.ErrorContext(ctx, "failed to handle upload message", slog.Any("error", err))
			return
		}
	default:
		log.DebugContext(ctx,
			"ignoring unsupported upload message",
			slog.String("type", string(msg.Type)),
		)
	}

}

func (c *Runner) handleUploadCompleted(
	ctx context.Context, msg upload.CompletedMessage) error {
	log := logger.FromContext(ctx)

	uploadID, err := uuid.Parse(msg.UploadID)
	if err != nil {
		return fmt.Errorf("parse upload id: %w", err)
	}

	downloadCtx, downloadSpan := tracer.Start(ctx, "Upload.Download")

	data, err := c.uploads.Download(downloadCtx, uploadID)
	if err != nil {
		return fmt.Errorf("read from bucket: %w", err)
	}

	downloadSpan.End()

	log = log.With(
		slog.String("profile_hint", string(msg.ProfileHint)),
	)
	ctx = logger.ToContext(ctx, log)

	_, span := tracer.Start(ctx, "Upload.Chunking")

	switch msg.ProfileHint {
	case upload.ProjectReportMd:
		if err := c.handleProjectReportMarkdown(ctx, msg, data); err != nil {
			log.ErrorContext(ctx, "failed to process file", slog.Any("error", err))

		}
	default:
		log.ErrorContext(
			ctx,
			"unsupported profile hint",
			slog.Any("error", serviceerr.InvalidInput),
		)
	}

	span.End()

	return nil
}

func (c *Runner) handleProjectReportMarkdown(ctx context.Context, uploadMsg upload.CompletedMessage, data []byte) error {
	log := logger.FromContext(ctx)

	report, err := projectreportmd.Parse(ctx, data)
	if err != nil {
		return fmt.Errorf("parse project markdown: %w", err)
	}

	messages := make([]*kgo.Record, len(report.Sections))

	for i, section := range report.Sections {
		content := projectreportmd.Content{
			Title:      report.Frontmatter.Title,
			Subject:    report.Frontmatter.Subject,
			Subheading: section.Subheading,
			Text:       section.Content,
		}
		contentString := projectreportmd.ContentAsString(content)

		contentBytes, err := json.Marshal(content)
		if err != nil {
			return fmt.Errorf("marshal chunk[%d] content: %w", i, err)
		}

		payload, err := json.Marshal(chunk.StreamPayload{
			Index:         i,
			Total:         len(report.Sections),
			Content:       contentBytes,
			ContentString: contentString,
		})
		if err != nil {
			return fmt.Errorf(
				"marshal chunk[%d] stream payload: %w",
				i,
				err,
			)
		}

		msg, err := json.Marshal(chunk.Message{
			UploadID: uploadMsg.UploadID,
			Type:     chunk.StreamEvent,
			Payload:  payload,
		})
		if err != nil {
			return fmt.Errorf("marshal chunk[%d] stream message: %w", i, err)
		}

		messages[i] = &kgo.Record{
			Key:   []byte(uploadMsg.UploadID),
			Topic: chunk.TopicName,
			Value: msg,
		}
	}

	if err := c.client.ProduceSync(ctx, messages...).FirstErr(); err != nil {
		return fmt.Errorf(
			"produce chunk stream messages (chunk_total=%d): %w",
			len(messages),
			err,
		)
	}

	log.InfoContext(ctx, "upload chunked successfully",
		slog.Int("count", len(messages)),
	)

	return nil
}

func (c *Runner) Close() {
	c.client.Close()
}
