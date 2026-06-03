package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/TSM-061/Raggy/rag/internal/rag"
	"github.com/TSM-061/Raggy/shared/logger"
	"github.com/TSM-061/Raggy/shared/message/chunk"
	"github.com/google/uuid"
	"github.com/twmb/franz-go/pkg/kgo"
)

type Config struct {
	MaxPollRecords int      `env:"MAX_POLL_RECORDS" envDefault:"10"`
	SeedBrokers    []string `env:"SEED_BROKERS,required"`
}

type Runner struct {
	client     *kgo.Client
	ragService *rag.RAG
	config     *Config
}

func New(config *Config, ragService *rag.RAG) (*Runner, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(config.SeedBrokers...),
		kgo.ConsumerGroup("rag-service"),
		kgo.ConsumeTopics(chunk.TopicName),
	)
	if err != nil {
		return nil, err
	}

	return &Runner{
		client:     client,
		ragService: ragService,
		config:     config,
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

			if err := c.ProcessMessage(ctx, record); err != nil {
				log.ErrorContext(
					ctx,
					"error processing message",
					slog.String("kafka_topic", record.Topic),
					slog.Int("kafka_partition", int(record.Partition)),
					slog.Int64("kafka_offset", record.Offset),
					slog.Any("error", err),
				)
			}
		}
	}
}

func (c *Runner) ProcessMessage(ctx context.Context, record *kgo.Record) error {
	log := logger.FromContext(ctx)

	var msg chunk.Message

	if err := json.Unmarshal(record.Value, &msg); err != nil {
		return fmt.Errorf("unmarhsal chunk message: %w", err)
	}

	log = log.With(
		slog.String("upload_id", msg.UploadID),
		slog.String("type", string(msg.Type)),
	)
	ctx = logger.ToContext(ctx, log)

	switch msg.Type {
	case chunk.StreamEvent:
		if err := c.HandleChunkStream(ctx, msg); err != nil {
			return fmt.Errorf("upload %s: %w", msg.UploadID, err)
		}
	default:
		log.DebugContext(
			ctx,
			"ignoring unsupported chunk message",
			slog.String("type", string(msg.Type)),
		)
	}

	return nil
}

func (c *Runner) HandleChunkStream(ctx context.Context, msg chunk.Message) error {
	var payload chunk.StreamPayload

	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal chunk stream payload: %w", err)
	}

	uploadID, err := uuid.Parse(msg.UploadID)
	if err != nil {
		return fmt.Errorf("parse upload id: %w", err)
	}

	if err := c.ragService.IngestChunk(ctx, &rag.ChunkInformation{
		UploadID:      uploadID,
		ChunkIndex:    payload.Index,
		ChunkTotal:    payload.Total,
		Content:       payload.Content,
		ContentString: payload.ContentString,
	}); err != nil {
		return fmt.Errorf("ingest chunk[%d]: %w", payload.Index, err)
	}

	return nil
}

func (c *Runner) Close() {
	c.client.Close()
}
