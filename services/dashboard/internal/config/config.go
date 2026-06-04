package config

import (
	"crypto/ed25519"
	"log/slog"
	"time"

	"github.com/TSM-061/Raggy/dashboard/internal/consumer"
	"github.com/TSM-061/Raggy/shared/env"
	"github.com/TSM-061/Raggy/shared/storage"
)

type Config struct {
	Port               int        `env:"PORT" envDefault:"80"`
	DbConnectionString string     `env:"DB_CONNECTION_STRING,required"`
	LogLevel           slog.Level `env:"LOG_LEVEL" envDefault:"info"`

	AccessTokenPublicKey ed25519.PublicKey `env:"ACCESS_TOKEN_PUBLIC_KEY,required"`

	S3Credentials *storage.S3Credentials `env:",init" envPrefix:"S3_"`
	S3Config      *storage.S3Config      `env:",init" envPrefix:"S3_"`

	ConsumerConfig *consumer.Config `env:",init" envPrefix:"KAFKA_"`

	UploadURLTTL time.Duration `env:"UPLOAD_URL_TTL" envDefault:"5m"`
}

func LoadConfig() (*Config, error) {
	cfg := Config{}

	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
