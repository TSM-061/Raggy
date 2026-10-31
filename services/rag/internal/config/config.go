package config

import (
	"crypto/ed25519"
	"fmt"
	"log/slog"
	"time"

	"github.com/TSM-061/Raggy/rag/internal/ai"
	"github.com/TSM-061/Raggy/rag/internal/consumer"
	"github.com/TSM-061/Raggy/shared/env"
	"github.com/TSM-061/Raggy/shared/telemetry"
)

type Config struct {
	Port               int           `env:"PORT" envDefault:"80"`
	DbConnectionString string        `env:"DB_CONNECTION_STRING"`
	LogLevel           slog.Level    `env:"LOG_LEVEL" envDefault:"info"`
	AIProvider         string        `env:"AI_PROVIDER" envDefault:"gemini"`
	FakeChunkDelay     time.Duration `env:"FAKE_CHUNK_DELAY" envDefault:"1s"`

	AccessTokenPublicKey ed25519.PublicKey `env:"ACCESS_TOKEN_PUBLIC_KEY,required"`

	GeminiConfig   *ai.GeminiConfig `env:",init" envPrefix:"GEMINI_"`
	ConsumerConfig *consumer.Config `env:",init" envPrefix:"KAFKA_"`

	Telemetry *telemetry.Config `env:",init" envPrefix:"TELEMETRY_"`
}

func LoadConfig() (*Config, error) {
	cfg := Config{}

	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}
	if err := ai.ValidateProvider(cfg.AIProvider, cfg.GeminiConfig); err != nil {
		return nil, err
	}
	if cfg.AIProvider == ai.ProviderFake && cfg.FakeChunkDelay < 0 {
		return nil, fmt.Errorf("FAKE_CHUNK_DELAY must not be negative")
	}

	return &cfg, nil
}
