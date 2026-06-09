package telemetry

import (
	"context"
	"strings"

	"github.com/TSM-061/Raggy/shared/message/s3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

const amzPrefix = "x-amz-meta-"

// carrierFromS3Record extracts user metadata fields prefixed with custom S3-compatible
// metadata headers and formats them into an OpenTelemetry MapCarrier.
func carrierFromS3Record(r s3.Record) propagation.MapCarrier {
	result := make(map[string]string)

	for header, value := range r.S3.Object.UserMetadata {
		normalised := strings.ToLower(header)

		if !strings.HasPrefix(normalised, amzPrefix) {
			continue
		}

		headerNoPrefix := strings.TrimPrefix(normalised, amzPrefix)
		result[headerNoPrefix] = value
	}

	return result
}

// ExtractFromS3Message extracts cross-cutting concerns from s3 event record into context.
func ExtractFromS3Message(ctx context.Context, r s3.Record) context.Context {
	carrier := carrierFromS3Record(r)
	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)

	return ctx
}
