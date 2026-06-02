package logger

import (
	"context"
	"log/slog"
	"net/http"
)

type contextKey struct{}

var loggerCtxKey contextKey = contextKey{}

func ToContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerCtxKey, logger)
}

func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerCtxKey).(*slog.Logger); ok {
		return logger
	}

	return slog.Default()
}

func Wrap(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = ToContext(ctx, logger)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
