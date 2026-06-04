package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/TSM-061/Raggy/dashboard/internal/services"
	"github.com/TSM-061/Raggy/dashboard/internal/upload"
	"github.com/TSM-061/Raggy/shared/logger"
	uploadmsg "github.com/TSM-061/Raggy/shared/message/upload"
	"github.com/TSM-061/Raggy/shared/telemetry"
	"github.com/google/uuid"
	"github.com/twmb/franz-go/pkg/kgo"
)

type Config struct {
	MaxPollRecords int      `env:"MAX_POLL_RECORDS" envDefault:"10"`
	SeedBrokers    []string `env:"SEED_BROKERS,required"`
}

type Runner struct {
	config  *Config
	client  *kgo.Client
	uploads *services.Upload
}

func New(cfg *Config, uploads *services.Upload) (*Runner, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(cfg.SeedBrokers...),
		kgo.ConsumerGroup("dashboard-service"),
		kgo.ConsumeTopics("s3-events"),
	)
	if err != nil {
		return nil, err
	}

	return &Runner{
		config:  cfg,
		client:  client,
		uploads: uploads,
	}, nil
}

func (r *Runner) Start(ctx context.Context) {
	log := logger.FromContext(ctx)
	log.InfoContext(ctx, "kafka consumer ready")

	for {
		fetches := r.client.PollRecords(ctx, r.config.MaxPollRecords)

		if fetches.IsClientClosed() {
			return
		}

		if errs := fetches.Errors(); len(errs) > 0 {
			log.ErrorContext(ctx, "Kafka poll errors encountered", slog.Any("errors", errs))
		}

		for record := range fetches.RecordsAll() {
			ctx = telemetry.WithKafkaMeta(ctx, record)

			if err := r.ProcessMessage(ctx, record); err != nil {
				log.ErrorContext(ctx, "error processing message", slog.Any("error", err))
			}
		}
	}
}

func (r *Runner) ProcessMessage(ctx context.Context, record *kgo.Record) error {
	var msg S3Message

	if err := json.Unmarshal(record.Value, &msg); err != nil {
		return fmt.Errorf("unmarhsal s3 message: %w", err)
	}

	errs := make([]error, 0)

	for _, s3Record := range msg.Records {
		if err := r.handleS3Record(ctx, s3Record); err != nil {
			errs = append(errs, err)
		}
	}

	return nil
}

func (r *Runner) handleS3Record(ctx context.Context, s3Record S3Record) error {
	log := logger.FromContext(ctx)

	uploadID, err := uuid.Parse(s3Record.S3.Object.Key)
	if err != nil {
		return fmt.Errorf("parse object key %q: %w", s3Record.S3.Object.Key, err)
	}

	if s3Record.EventName != "s3:ObjectCreated:Put" {
		log.DebugContext(
			ctx,
			"ignoring unsupported event name",
			slog.String("s3_event_name", s3Record.EventName),
		)
		return nil
	}

	if err := r.uploads.UpdateStatus(ctx, uploadID, upload.StatusUploaded); err != nil {
		return fmt.Errorf("update upload status: %w", err)
	}

	evt := uploadmsg.CompletedMessage{
		Type:        uploadmsg.Completed,
		UploadID:    uploadID.String(),
		ProfileHint: uploadmsg.ProjectReportMd,
	}
	eventJson, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal %q message: %w", uploadmsg.Completed, err)
	}

	record := &kgo.Record{
		Topic: uploadmsg.TopicName,
		Key:   []byte(evt.UploadID),
		Value: eventJson,
	}

	if err := r.client.ProduceSync(ctx, record).FirstErr(); err != nil {
		return fmt.Errorf("produce %q message: %w", uploadmsg.Completed, err)
	}

	return nil
}

func (c *Runner) Close() {
	c.client.Close()
}
