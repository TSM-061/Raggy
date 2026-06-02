package env

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"log/slog"
	"reflect"

	"github.com/caarlos0/env/v11"
)

func Parse(value any) error {
	return env.ParseWithOptions(value, env.Options{
		FuncMap: map[reflect.Type]env.ParserFunc{
			reflect.TypeFor[slog.Level]():         parseLogLevel,
			reflect.TypeFor[ed25519.PublicKey]():  parsePublicKey,
			reflect.TypeFor[ed25519.PrivateKey](): parsePrivateKey,
		},
	})
}

func parseLogLevel(value string) (any, error) {
	switch value {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf(
			"invalid log level %q: must be debug, info, warn, or error",
			value,
		)
	}
}

func parsePublicKey(value string) (any, error) {
	publicKey, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("decode base6: %w", err)
	}

	if len(publicKey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf(
			"invalid key size: want %d, got %d",
			ed25519.PublicKeySize,
			len(publicKey),
		)
	}

	return publicKey, nil
}

func parsePrivateKey(value string) (any, error) {
	privateKey, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("decode base6: %w", err)
	}

	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf(
			"invalid key size: want %d, got %d",
			ed25519.PrivateKeySize,
			len(privateKey),
		)
	}

	return privateKey, nil
}
