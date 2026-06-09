package logger

import (
	"context"
	"log/slog"

	"github.com/TSM-061/Raggy/shared/telemetry"
	"go.opentelemetry.io/otel/trace"
)

type ContextHandler struct {
	next slog.Handler
}

func NewContextHandler(next slog.Handler) *ContextHandler {
	return &ContextHandler{next: next}
}

func (h *ContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{next: h.next.WithAttrs(attrs)}
}

func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{next: h.next.WithGroup(name)}
}

func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if ctx == nil {
		return h.next.Handle(ctx, r)
	}

	if span := trace.SpanContextFromContext(ctx); span.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", span.TraceID().String()),
			slog.String("span_id", span.SpanID().String()),
		)
	}

	// add kafka metadata on error messages
	if r.Level == slog.LevelError {
		if meta, ok := telemetry.MetadataFromContext(ctx); ok {
			r.AddAttrs(slog.Attr{
				Key: "kafka",
				Value: slog.GroupValue(
					slog.String("topic", meta.Topic),
					slog.Int64("partition", meta.Partition),
					slog.Int64("offset", meta.Offset),
				),
			})
		}
	}

	return h.next.Handle(ctx, r)
}
