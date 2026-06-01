package web

import (
	"net/http"

	"github.com/TSM-061/Raggy/rag/internal/config"
	services "github.com/TSM-061/Raggy/rag/internal/rag"
	"github.com/TSM-061/Raggy/shared/auth"
	"github.com/TSM-061/Raggy/shared/clock"
)

type Server struct {
	cfg            *config.Config
	rag            *services.RAG
	authMiddleware *auth.Middleware
}

func NewServer(cfg *config.Config, ragService *services.RAG) *Server {
	verifier := auth.NewTokenVerifier(&clock.LiveClock{}, cfg.AccessTokenPublicKey)

	return &Server{
		cfg:            cfg,
		rag:            ragService,
		authMiddleware: auth.NewMiddleware(verifier),
	}
}

func (s *Server) GetEndpoints() *http.ServeMux {
	mux := http.NewServeMux()

	mux.Handle("GET /api/search", s.authMiddleware.WrapFn(s.HandleQuery))

	return mux
}
