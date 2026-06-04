package telemetry

import (
	"context"

	"github.com/twmb/franz-go/pkg/kgo"
)

type contextKey struct{}

var kafkaFields contextKey = contextKey{}

type KafkaMeta struct {
	Topic     string
	Partition int64
	Offset    int64
}

func WithKafkaMeta(ctx context.Context, record *kgo.Record) context.Context {
	km := &KafkaMeta{
		Topic:     record.Topic,
		Partition: int64(record.Partition),
		Offset:    record.Offset,
	}

	return context.WithValue(ctx, kafkaFields, km)
}

func KafkaMetaFromContext(ctx context.Context) (*KafkaMeta, bool) {
	km, ok := ctx.Value(kafkaFields).(*KafkaMeta)
	return km, ok
}
