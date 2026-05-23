package config

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/TSM-061/Raggy/shared/configerr"
	"github.com/TSM-061/Raggy/shared/env"
)

const ServiceUploadsBucketName = "uploads"

type Config struct {
	Port               int
	DbConnectionString string

	S3Region          string
	S3Endpoint        string
	S3AccessKeyID     string
	S3SecretAccessKey string
	S3UsePathStyle    bool
	S3DisableSSL      bool

	AccessTokenPublicKey ed25519.PublicKey

	UploadURLTTL time.Duration
}

func LoadConfig(h *env.Helper) *Config {
	accessTokenPublicKeyBase64 := h.GetStringRequired("ACCESS_TOKEN_PUBLIC_KEY")
	accessTokenPublicKey, err := base64.StdEncoding.DecodeString(accessTokenPublicKeyBase64)
	if err != nil {
		panic(&configerr.ConfigError{
			Key:     "ACCESS_TOKEN_PUBLIC_KEY",
			Message: "couldn't decode base64",
			Err:     err,
		})
	}

	if len(accessTokenPublicKey) != ed25519.PublicKeySize {
		panic(&configerr.ConfigError{
			Key: "ACCESS_TOKEN_PUBLIC_KEY",
			Message: fmt.Sprintf(
				"invalid key size, want %d, got %d",
				ed25519.PublicKeySize,
				len(accessTokenPublicKey),
			),
		})
	}

	return &Config{
		Port:                 h.GetInt("PORT", 80),
		DbConnectionString:   h.GetStringRequired("DB_CONNECTION_STRING"),
		S3Region:             h.GetString("S3_REGION", "ap-southeast-2"),
		S3Endpoint:           h.GetString("S3_ENDPOINT", ""),
		S3AccessKeyID:        h.GetStringRequired("S3_ACCESS_KEY_ID"),
		S3SecretAccessKey:    h.GetStringRequired("S3_SECRET_ACCESS_KEY"),
		S3UsePathStyle:       h.GetBool("S3_USE_PATH_STYLE", true),
		S3DisableSSL:         h.GetBool("S3_DISABLE_SSL", false),
		AccessTokenPublicKey: ed25519.PublicKey(accessTokenPublicKey),
		UploadURLTTL:         h.GetDuration("UPLOAD_URL_TTL", "5m"),
	}
}
