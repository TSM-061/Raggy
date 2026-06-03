package config

import (
	"log/slog"

	"github.com/TSM-061/Raggy/shared/auth"
	"github.com/TSM-061/Raggy/shared/env"
	"github.com/TSM-061/Raggy/simple-auth-service/internal/password"
)

type Config struct {
	Port               int        `env:"PORT" envDefault:"80"`
	DbConnectionString string     `env:"DB_CONNECTION_STRING,required"`
	LogLevel           slog.Level `env:"LOG_LEVEL" envDefault:"info"`

	Argon2Config *password.Argon2Config `env:",init" envPrefix:"ARGON2_"`

	AccessTokenConfig  *auth.TokenSignerConfig `env:",init" envPrefix:"ACCESS_TOKEN_"`
	RefreshTokenSecret string                  `env:"REFRESH_TOKEN_SECRET,required"`
}

func LoadConfig() (*Config, error) {
	cfg := Config{}

	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
