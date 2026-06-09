package telemetry

import (
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("github.com/TSM-061/Raggy/shared/telemetry")

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := otel.GetTextMapPropagator().Extract(
			r.Context(),
			propagation.HeaderCarrier(r.Header),
		)

		ctx, span := tracer.Start(
			ctx,
			fmt.Sprintf("%s %s", r.Method, r.Pattern),
			trace.WithSpanKind(trace.SpanKindServer),
		)
		defer span.End()

		otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(w.Header()))

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
