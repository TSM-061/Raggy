package config

import (
	"github.com/TSM-061/Raggy/shared/env"
)

type Config struct {
	Port               int
	DbConnectionString string

	AuthPasswordPepper string

	Argon2KeyLength uint32

	Argon2Memory  uint32
	Argon2Time    uint32
	Argon2Threads uint8
}

func LoadConfig(h *env.Helper) *Config {
	return &Config{
		Port:               h.GetInt("PORT", 80),
		DbConnectionString: h.GetStringRequired("DB_CONNECTION_STRING"),

		AuthPasswordPepper: h.GetStringRequired("AUTH_PASSWORD_PEPPER"),

		// as recommended by RFC9106
		Argon2KeyLength: uint32(h.GetInt("ARGON2_KEY_LENGTH", 32)),
		Argon2Memory:    uint32(h.GetInt("ARGON2_MEMORY", 64*1024)),
		Argon2Time:      uint32(h.GetInt("ARGON2_TIME", 3)),
		Argon2Threads:   uint8(h.GetInt("ARGON2_THREADS", 4)),
	}
}
