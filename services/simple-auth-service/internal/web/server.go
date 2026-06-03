package web

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/TSM-061/Raggy/shared/logger"
	"github.com/TSM-061/Raggy/simple-auth-service/internal/config"
	"github.com/TSM-061/Raggy/simple-auth-service/internal/services"
	"github.com/go-playground/validator/v10"
)

type Server struct {
	config *config.Config

	httpServer *http.Server

	validator *validator.Validate

	auth *services.Auth
}

func NewServer(
	cfg *config.Config,
	log *slog.Logger,
	auth *services.Auth,
) *Server {

	server := &Server{
		config:    cfg,
		auth:      auth,
		validator: validator.New(validator.WithRequiredStructEnabled()),
	}

	server.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: server.GetEndpoints(log),
	}

	return server
}

func (s *Server) GetEndpoints(log *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/auth/signin", s.HandleSignin)
	mux.HandleFunc("POST /api/auth/refresh", s.HandleRefresh)
	mux.HandleFunc("POST /api/auth/signout", s.HandleSignout)

	var handler http.Handler = mux

	handler = logger.Wrap(handler, log)

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

func (s *Server) Auth() *services.Auth {
	return s.auth
}
