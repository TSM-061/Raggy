package telemetry

import (
	"github.com/twmb/franz-go/pkg/kgo"
)

// KafkaCarrier an adapter for otel to get/set telemetry headers on Kafka messages.
type KafkaCarrier struct {
	record *kgo.Record
}

func NewKafkaCarrier(r *kgo.Record) KafkaCarrier {
	return KafkaCarrier{record: r}
}

// Get looks up the value associated with a given key in the Kafka headers.
func (c KafkaCarrier) Get(key string) string {
	for _, h := range c.record.Headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}

// Set sets a key-value pair in the Kafka headers (used if you are producing messages).
func (c KafkaCarrier) Set(key string, value string) {
	for i, h := range c.record.Headers {
		if h.Key == key {
			c.record.Headers[i].Value = []byte(value)
			return
		}
	}

	c.record.Headers = append(
		c.record.Headers,
		kgo.RecordHeader{Key: key, Value: []byte(value)},
	)
}

// Keys returns all the keys present in the Kafka headers.
func (c KafkaCarrier) Keys() []string {
	keys := make([]string, len(c.record.Headers))

	for i, h := range c.record.Headers {
		keys[i] = h.Key
	}

	return keys
}
