package web

import (
	"github.com/TSM-061/Raggy/shared/accesstoken"
	"github.com/TSM-061/Raggy/shared/clock"
	"github.com/TSM-061/Raggy/simple-auth-service/internal/config"
	"github.com/TSM-061/Raggy/simple-auth-service/internal/password"
	"github.com/TSM-061/Raggy/simple-auth-service/internal/services"
	"github.com/TSM-061/Raggy/simple-auth-service/internal/session"
	"github.com/TSM-061/Raggy/simple-auth-service/internal/user"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	config    *config.Config
	Users     user.Repo
	Sessions  session.Repo
	Auth      *services.AuthService
	Validator *validator.Validate
}

func NewServer(
	config *config.Config,
	pool *pgxpool.Pool,
) (*Server, error) {

	clock := &clock.LiveClock{}

	users := user.NewPostgresRepo(pool)
	sessions := session.NewPostgresRepo(pool)

	sessionManager := session.NewManager(config.RefreshTokenSecret, sessions)

	hasher := password.NewArgon2Hasher(
		config.PasswordSecret,
		config.Argon2KeyLength,
		config.Argon2Memory,
		config.Argon2Time,
		config.Argon2Threads,
	)

	signer, err := accesstoken.NewSigner(
		clock,
		config.AccessTokenPrivateKey,
		"raggy-auth",
		config.AccessTokenTTL,
	)
	if err != nil {
		return nil, err
	}

	authService := services.NewAuthService(users, hasher, signer, sessionManager)

	return &Server{
		config:    config,
		Users:     users,
		Sessions:  sessions,
		Auth:      authService,
		Validator: validator.New(validator.WithRequiredStructEnabled()),
	}, nil
}
