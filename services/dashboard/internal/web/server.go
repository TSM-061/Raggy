package web

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/TSM-061/Raggy/dashboard/internal/config"
	"github.com/TSM-061/Raggy/dashboard/internal/services"
	"github.com/TSM-061/Raggy/shared/auth"
	"github.com/TSM-061/Raggy/shared/logger"
	"github.com/TSM-061/Raggy/shared/telemetry"
	"github.com/go-playground/validator/v10"
)

type Server struct {
	config     *config.Config
	httpServer *http.Server

	validator *validator.Validate
	uploads   *services.Upload
}

func NewServer(
	cfg *config.Config,
	log *slog.Logger,
	auth *auth.Middleware,
	uploads *services.Upload,
) *Server {

	server := &Server{
		config:    cfg,
		validator: validator.New(),
		uploads:   uploads,
	}

	server.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: server.BuildEndpoints(log, auth),
	}

	return server
}

func (s *Server) BuildEndpoints(log *slog.Logger, auth *auth.Middleware) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/uploads", s.HandleUpload)
	mux.HandleFunc("GET /api/uploads", s.HandleListUploads)
	mux.HandleFunc("DELETE /api/uploads/{id}", s.HandleDeleteUpload)

	var handler http.Handler = mux

	handler = auth.Wrap(handler)
	handler = logger.Wrap(handler, log)
	handler = telemetry.Middleware(handler)

	return handler
}

func (s *Server) Start(ctx context.Context, onFatalErr func()) {
	log := logger.FromContext(ctx)

	log.InfoContext(ctx, "http server ready", slog.Int("port", s.config.Port))

	if err := s.httpServer.ListenAndServe(); err != nil {
		// http server exiting from a fatal error
		if !errors.Is(err, http.ErrServerClosed) {
			log.ErrorContext(ctx, "http server exited", slog.Any("error", err))
			onFatalErr()
		}
	}
}

func (s *Server) Shutdown(ctx context.Context) {
	log := logger.FromContext(ctx)

	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.InfoContext(ctx, "http server shutdown failed", slog.Any("error", err))
	} else {
		log.InfoContext(ctx, "http server shutdown completed")
	}
}
