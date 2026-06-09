package logger

import (
	"context"
	"log/slog"
	"net/http"
	"os"
)

type contextKey struct{}

var loggerKey contextKey = contextKey{}

var defaultLogger = New(slog.LevelInfo)

func New(level slog.Level) *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	contextHandler := NewContextHandler(handler)

	return slog.New(contextHandler)
}

func ToContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
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

func init() {
	slog.SetDefault(defaultLogger)
}
