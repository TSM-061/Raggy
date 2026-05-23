package web

import (
	"github.com/TSM-061/Raggy/dashboard/internal/app"
	"github.com/TSM-061/Raggy/dashboard/internal/config"
	"github.com/TSM-061/Raggy/shared/auth"
	"github.com/TSM-061/Raggy/shared/clock"
	"github.com/go-playground/validator/v10"
)

type Server struct {
	config      *config.Config
	Application *app.App

	validator      *validator.Validate
	authMiddleware *auth.Middleware
}

func NewServer(cfg *config.Config, app *app.App) (*Server, error) {
	validator := validator.New()

	verifier := auth.NewTokenVerifier(&clock.LiveClock{}, cfg.AccessTokenPublicKey)

	return &Server{
		config:         cfg,
		Application:    app,
		validator:      validator,
		authMiddleware: auth.NewMiddleware(verifier),
	}, nil
}

func (s *Server) Auth() *auth.Middleware {
	return s.authMiddleware
}

