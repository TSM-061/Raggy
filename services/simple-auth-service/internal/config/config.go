package config

import (
	"time"

	"github.com/TSM-061/Raggy/shared/env"
)

type Config struct {
	Port               int
	DbConnectionString string

	PasswordSecret string

	Argon2KeyLength uint32

	Argon2Memory  uint32
	Argon2Time    uint32
	Argon2Threads uint8

	RefreshTokenSecret string

	// Base 64 representation of the key
	AccessTokenPrivateKey string
	AccessTokenTTL        time.Duration
}

func LoadConfig(h *env.Helper) *Config {
	return &Config{
		Port:               h.GetInt("PORT", 80),
		DbConnectionString: h.GetStringRequired("DB_CONNECTION_STRING"),

		PasswordSecret: h.GetStringRequired("PASSWORD_SECRET"),

		// as recommended by RFC9106
		Argon2KeyLength: uint32(h.GetInt("ARGON2_KEY_LENGTH", 32)),
		Argon2Memory:    uint32(h.GetInt("ARGON2_MEMORY", 64*1024)),
		Argon2Time:      uint32(h.GetInt("ARGON2_TIME", 3)),
		Argon2Threads:   uint8(h.GetInt("ARGON2_THREADS", 4)),

		RefreshTokenSecret: h.GetStringRequired("REFRESH_TOKEN_SECRET"),

		AccessTokenPrivateKey: h.GetStringRequired("ACCESS_TOKEN_PRIVATE_KEY"),
		AccessTokenTTL:        h.GetDuration("ACCESS_TOKEN_TTL", "15m"),
	}
}
