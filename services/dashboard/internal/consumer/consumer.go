package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/TSM-061/Raggy/dashboard/internal/services"
	"github.com/TSM-061/Raggy/dashboard/internal/upload"
	"github.com/TSM-061/Raggy/shared/logger"
	"github.com/TSM-061/Raggy/shared/message/s3"
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
		kgo.WithHooks(telemetry.NewKafkaHook()),

		kgo.SessionTimeout(2*time.Second),
		kgo.HeartbeatInterval(800*time.Millisecond),
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
			recordCtx := telemetry.Extract(ctx, record)

			if errs := r.processMessage(recordCtx, record); len(errs) > 0 {
				log.ErrorContext(recordCtx, "error processing message", slog.Any("errors", errs))
			}

		}
	}
}

func (r *Runner) processMessage(ctx context.Context, record *kgo.Record) []error {
	var msg s3.Message

	errs := make([]error, 0)

	if err := json.Unmarshal(record.Value, &msg); err != nil {
		errs = append(errs, fmt.Errorf("unmarhsal s3 message: %w", err))
		return errs
	}

	for _, s3Record := range msg.Records {
		recordCtx := telemetry.ExtractFromS3Message(ctx, s3Record)

		if err := r.handleS3Record(recordCtx, s3Record); err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

func (r *Runner) handleS3Record(ctx context.Context, s3Record s3.Record) error {
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

	if err := r.produceCompletedMsg(ctx, uploadID); err != nil {
		return err
	}

	return nil
}

func (r *Runner) produceCompletedMsg(ctx context.Context, id uuid.UUID) error {
	event, err := json.Marshal(
		uploadmsg.CompletedMessage{
			Type:        uploadmsg.Completed,
			UploadID:    id.String(),
			ProfileHint: uploadmsg.ProjectReportMd,
		})
	if err != nil {
		return fmt.Errorf("marshal %q message: %w", uploadmsg.Completed, err)
	}

	record := &kgo.Record{
		Topic: uploadmsg.TopicName,
		Key:   []byte(id.String()),
		Value: event,
	}

	if err := r.client.ProduceSync(ctx, record).FirstErr(); err != nil {
		return fmt.Errorf("produce %q message: %w", uploadmsg.Completed, err)
	}

	return nil
}

func (c *Runner) Close() {
	if c.client == nil {
		return
	}

	c.client.Close()
}
