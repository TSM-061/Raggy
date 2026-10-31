package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/TSM-061/Raggy/rag/internal/rag"
	"github.com/TSM-061/Raggy/shared/kafkawork"
	"github.com/TSM-061/Raggy/shared/logger"
	"github.com/TSM-061/Raggy/shared/message/chunk"
	"github.com/TSM-061/Raggy/shared/message/upload"
	"github.com/TSM-061/Raggy/shared/serviceerr"
	"github.com/TSM-061/Raggy/shared/telemetry"
	"github.com/google/uuid"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel"
)

type Config struct {
	MaxPollRecords int      `env:"MAX_POLL_RECORDS" envDefault:"10"`
	SeedBrokers    []string `env:"SEED_BROKERS,required"`
}

type Runner struct {
	client  *kgo.Client
	config  *Config
	tracker *tracker
}

var tracer = otel.Tracer("github.com/TSM-061/Raggy/rag/internal/consumer")

func New(config *Config, tracker *tracker) (*Runner, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(config.SeedBrokers...),
		kgo.ConsumerGroup("rag-service"),
		kgo.ConsumeTopics(chunk.TopicName),
		kgo.WithHooks(telemetry.NewKafkaHook()),
		kgo.DisableAutoCommit(),
		kgo.BlockRebalanceOnPoll(),

		kgo.SessionTimeout(2*time.Second),
		kgo.HeartbeatInterval(800*time.Millisecond),
	)
	if err != nil {
		return nil, err
	}

	return &Runner{
		client:  client,
		config:  config,
		tracker: tracker,
	}, nil
}

func (c *Runner) Start(ctx context.Context) {
	defer c.client.AllowRebalance()
	log := logger.FromContext(ctx)
	log.InfoContext(ctx, "kafka consumer ready")

	for {
		fetches := c.client.PollRecords(ctx, c.config.MaxPollRecords)

		if fetches.IsClientClosed() || ctx.Err() != nil {
			return
		}

		if errs := fetches.Errors(); len(errs) > 0 {
			log.ErrorContext(ctx, "Kafka poll errors encountered", slog.Any("errors", errs))
		}

		for record := range fetches.RecordsAll() {
			recordCtx := telemetry.Extract(ctx, record)
			if !kafkawork.ProcessRecord(recordCtx, c.client, record, c.ProcessMessage) {
				c.client.AllowRebalance()
				return
			}
		}
		c.client.AllowRebalance()
	}
}

func (c *Runner) ProcessMessage(ctx context.Context, record *kgo.Record) error {
	log := logger.FromContext(ctx)

	var msg chunk.Message

	if err := json.Unmarshal(record.Value, &msg); err != nil {
		return fmt.Errorf("%w: decode chunk message", serviceerr.InvalidInput)
	}

	log = log.With(slog.String("upload_id", msg.UploadID))
	ctx = logger.ToContext(ctx, log)

	switch msg.Type {
	case chunk.StreamEvent:
		if err := c.HandleChunkStream(ctx, msg); err != nil {
			return err
		}
	default:
		log.DebugContext(ctx, "ignoring unsupported chunk message")
	}

	return nil
}

func (c *Runner) HandleChunkStream(ctx context.Context, msg chunk.Message) error {
	var payload chunk.StreamPayload

	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return fmt.Errorf("%w: decode chunk payload", serviceerr.InvalidInput)
	}

	uploadID, err := uuid.Parse(msg.UploadID)
	if err != nil {
		return fmt.Errorf("%w: invalid upload id", serviceerr.InvalidInput)
	}

	progress, err := c.tracker.TrackAndProcessChunk(ctx, &rag.ChunkInformation{
		UploadID:      uploadID,
		ChunkIndex:    payload.Index,
		ChunkTotal:    payload.Total,
		Content:       payload.Content,
		ContentString: payload.ContentString,
	})
	if err != nil {
		return fmt.Errorf("ingest chunk[%d]: %w", payload.Index, err)
	}
	data, err := json.Marshal(progress)
	if err != nil {
		return err
	}
	if err := c.client.ProduceSync(ctx, &kgo.Record{Topic: upload.ProgressTopic, Key: []byte(msg.UploadID), Value: data}).FirstErr(); err != nil {
		return fmt.Errorf("publish chunk progress: %w", err)
	}

	return nil
}

func (c *Runner) Close() {
	c.client.Close()
}
