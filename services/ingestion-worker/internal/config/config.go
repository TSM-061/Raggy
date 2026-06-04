package config

import (
	"log/slog"

	"github.com/TSM-061/Raggy/ingestion-worker/internal/consumer"
	"github.com/TSM-061/Raggy/shared/env"
	"github.com/TSM-061/Raggy/shared/storage"
)

type Config struct {
	LogLevel slog.Level `env:"LOG_LEVEL" envDefault:"info"`

	S3Credentials *storage.S3Credentials `env:",init" envPrefix:"S3_"`
	S3Config      *storage.S3Config      `env:",init" envPrefix:"S3_"`

	ConsumerConfig *consumer.Config `env:",init" envPrefix:"KAFKA_"`
}

func LoadConfig() (*Config, error) {
	cfg := Config{}

	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
