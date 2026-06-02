package config

import (
	"crypto/ed25519"
	"log/slog"

	"github.com/TSM-061/Raggy/rag/internal/ai"
	"github.com/TSM-061/Raggy/rag/internal/consumer"
	"github.com/TSM-061/Raggy/shared/env"
)

type Config struct {
	Port               int        `env:"PORT" envDefault:"80"`
	DbConnectionString string     `env:"DB_CONNECTION_STRING"`
	LogLevel           slog.Level `env:"LOG_LEVEL" envDefault:"info"`

	AccessTokenPublicKey ed25519.PublicKey `env:"ACCESS_TOKEN_PUBLIC_KEY,required"`

	GeminiConfig   *ai.GeminiConfig `env:",init" envPrefix:"GEMINI_"`
	ConsumerConfig *consumer.Config `env:",init" envPrefix:"KAFKA_"`
}

func LoadConfig() (*Config, error) {
	cfg := Config{}

	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
