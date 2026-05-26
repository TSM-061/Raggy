package config

import (
	"strings"

	"github.com/TSM-061/Raggy/shared/env"
	"github.com/TSM-061/Raggy/shared/storage"
)

type Config struct {
	KafkaSeedBrokers []string

	S3Credentials *storage.S3Credentials
	S3Config      *storage.S3Config
}

func LoadConfig(h *env.Helper) *Config {
	brokersStr := h.GetStringRequired("KAFKA_SEED_BROKERS")
	brokers := strings.Split(brokersStr, ",")

	return &Config{
		KafkaSeedBrokers: brokers,

		S3Config: &storage.S3Config{
			Bucket:       "uploads",
			Endpoint:     h.GetStringRequired("S3_ENDPOINT"),
			UsePathStyle: h.GetBool("S3_USE_PATH_STYLE", true),
			DisableSSL:   h.GetBool("S3_DISABLE_SSL", false),
		},

		S3Credentials: &storage.S3Credentials{
			AccessKeyID:     h.GetStringRequired("S3_ACCESS_KEY_ID"),
			SecretAccessKey: h.GetStringRequired("S3_SECRET_ACCESS_KEY"),
		},
	}
}
