package config

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/TSM-061/Raggy/shared/configerr"
	"github.com/TSM-061/Raggy/shared/env"
	"github.com/TSM-061/Raggy/shared/storage"
)

type Config struct {
	Port               int
	DbConnectionString string

	AccessTokenPublicKey ed25519.PublicKey

	S3Credentials *storage.S3Credentials
	S3Config      *storage.S3Config

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
		Port:               h.GetInt("PORT", 80),
		DbConnectionString: h.GetStringRequired("DB_CONNECTION_STRING"),
		S3Credentials: &storage.S3Credentials{
			AccessKeyID:     h.GetStringRequired("S3_ACCESS_KEY_ID"),
			SecretAccessKey: h.GetStringRequired("S3_SECRET_ACCESS_KEY"),
		},
		S3Config: &storage.S3Config{
			Bucket:       "uploads",
			Endpoint:     h.GetStringRequired("S3_ENDPOINT"),
			UsePathStyle: h.GetBool("S3_USE_PATH_STYLE", true),
			DisableSSL:   h.GetBool("S3_DISABLE_SSL", false),
		},
		AccessTokenPublicKey: ed25519.PublicKey(accessTokenPublicKey),
		UploadURLTTL:         h.GetDuration("UPLOAD_URL_TTL", "5m"),
	}
}
