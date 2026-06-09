package telemetry

import (
	"context"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel"
)

type contextKey struct{}

var kafkaFields contextKey = contextKey{}

type KafkaMeta struct {
	Topic     string
	Partition int64
	Offset    int64
}

// Extract reads cross-cutting concerns from a Kafka record (carrier) into a Context.
// Includes metadata fields about the particular message for tracing.
func Extract(ctx context.Context, record *kgo.Record) context.Context {
	ctx = otel.GetTextMapPropagator().Extract(ctx, NewKafkaCarrier(record))

	km := &KafkaMeta{
		Topic:     record.Topic,
		Partition: int64(record.Partition),
		Offset:    record.Offset,
	}
	ctx = context.WithValue(ctx, kafkaFields, km)

	return ctx
}

func MetadataFromContext(ctx context.Context) (*KafkaMeta, bool) {
	km, ok := ctx.Value(kafkaFields).(*KafkaMeta)
	return km, ok
}

type KafkaClientHook struct{}

func NewKafkaHook() KafkaClientHook {
	return KafkaClientHook{}
}

func (h KafkaClientHook) OnProduceRecordBuffered(r *kgo.Record) {
	if r.Context == nil {
		return
	}

	otel.GetTextMapPropagator().Inject(
		r.Context,
		NewKafkaCarrier(r),
	)
}
