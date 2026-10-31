package kafkawork

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/TSM-061/Raggy/shared/logger"
	"github.com/TSM-061/Raggy/shared/serviceerr"
	"github.com/twmb/franz-go/pkg/kgo"
)

type Committer interface {
	CommitRecords(context.Context, ...*kgo.Record) error
}

// ProcessRecord never commits past failed work. Permanent validation errors are
// rejected explicitly; transient failures retry until shutdown. Commit retries do
// not repeat processing. Consumers must disable autocommit and block rebalancing.
func ProcessRecord(ctx context.Context, client Committer, record *kgo.Record, process func(context.Context, *kgo.Record) error) bool {
	log := logger.FromContext(ctx)
	for ctx.Err() == nil {
		err := process(ctx, record)
		if err == nil {
			break
		}
		if errors.Is(err, serviceerr.InvalidInput) {
			log.ErrorContext(ctx, "rejecting invalid Kafka record", slog.String("topic", record.Topic), slog.Int64("offset", record.Offset))
			break
		}
		log.ErrorContext(ctx, "Kafka processing failed; retrying", slog.String("topic", record.Topic), slog.Any("error", err))
		if !wait(ctx) {
			return false
		}
	}
	for ctx.Err() == nil {
		if err := client.CommitRecords(ctx, record); err == nil {
			return true
		}
		log.WarnContext(ctx, "Kafka offset commit failed; retrying")
		if !wait(ctx) {
			return false
		}
	}
	return false
}

func wait(ctx context.Context) bool {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
