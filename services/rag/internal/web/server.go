package web

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/TSM-061/Raggy/rag/internal/config"
	"github.com/TSM-061/Raggy/rag/internal/rag"
	"github.com/TSM-061/Raggy/shared/auth"
	"github.com/TSM-061/Raggy/shared/clock"
	"github.com/TSM-061/Raggy/shared/logger"
)

type Server struct {
	cfg        *config.Config
	httpServer *http.Server
	rag        *rag.RAG
	auth       *auth.Middleware
}

func NewServer(cfg *config.Config, ragService *rag.RAG) *Server {
	verifier := auth.NewTokenVerifier(&clock.LiveClock{}, cfg.AccessTokenPublicKey)

	server := &Server{
		cfg:  cfg,
		rag:  ragService,
		auth: auth.NewMiddleware(verifier),
	}

	server.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: server.GetEndpoints(),
	}

	return server
}

func (s *Server) GetEndpoints() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/search", s.HandleQuery)

	return s.auth.Wrap(mux)
}

func (s *Server) Start(ctx context.Context, onFatalErr func()) {
	log := logger.FromContext(ctx)

	log.InfoContext(ctx, "http server ready", slog.Int("port", s.cfg.Port))

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
