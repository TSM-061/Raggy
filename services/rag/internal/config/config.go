package config

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"

	"github.com/TSM-061/Raggy/rag/internal/ai"
	"github.com/TSM-061/Raggy/rag/internal/consumer"
	"github.com/TSM-061/Raggy/shared/configerr"
	"github.com/TSM-061/Raggy/shared/env"
)

type Config struct {
	Port               int
	DbConnectionString string

	AccessTokenPublicKey ed25519.PublicKey

	GeminiConfig   *ai.GeminiEmbedderConfig
	ConsumerConfig *consumer.Config
}

func LoadConfig(h *env.Helper) (*Config, error) {

	dimensions := int32(h.GetInt("GEMINI_EMBEDDING_DIMENSIONS", 1536))

	accessTokenPublicKeyBase64 := h.GetStringRequired("ACCESS_TOKEN_PUBLIC_KEY")
	accessTokenPublicKey, err := base64.StdEncoding.DecodeString(accessTokenPublicKeyBase64)
	if err != nil {
		return nil, &configerr.ConfigError{
			Key:     "ACCESS_TOKEN_PUBLIC_KEY",
			Message: "couldn't decode base64",
			Err:     err,
		}
	}

	if len(accessTokenPublicKey) != ed25519.PublicKeySize {
		return nil, (&configerr.ConfigError{
			Key: "ACCESS_TOKEN_PUBLIC_KEY",
			Message: fmt.Sprintf(
				"invalid key size, want %d, got %d",
				ed25519.PublicKeySize,
				len(accessTokenPublicKey),
			),
		})
	}

	return &Config{
		Port:               h.GetInt("PORT", 80),
		DbConnectionString: h.GetStringRequired("DB_CONNECTION_STRING"),

		AccessTokenPublicKey: accessTokenPublicKey,

		GeminiConfig: &ai.GeminiEmbedderConfig{
			APIKey:              h.GetStringRequired("GEMINI_API_KEY"),
			EmbeddingModel:      h.GetString("GEMINI_EMBEDDING_MODEL", "gemini-embedding-2"),
			EmbeddingDimensions: &dimensions,

			GenerationModel: h.GetString("GEMINI_GENERATION_MODEL", "gemini-3.5-flash"),
		},

		ConsumerConfig: &consumer.Config{
			MaxPollRecords: h.GetInt("KAFKA_MAX_POLL_RECORDS", 10),
			SeedBrokers:    h.GetStringSliceRequired("KAFKA_SEED_BROKERS"),
		},
	}, nil
}
