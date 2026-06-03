package auth

import (
	"crypto/ed25519"
	"time"

	"github.com/TSM-061/Raggy/shared/clock"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenSignerConfig struct {
	PrivateKey ed25519.PrivateKey `env:"PRIVATE_KEY,required"`
	Issuer     string             `env:"ISSUER" envDefault:"raggy-auth"`
	TTL        time.Duration      `env:"TTL" envDefault:"15m"`
}

type TokenSigner struct {
	clock  clock.Clock
	config *TokenSignerConfig
}

func NewTokenSigner(config *TokenSignerConfig, clk clock.Clock) *TokenSigner {
	return &TokenSigner{
		clock:  clk,
		config: config,
	}
}

func (s *TokenSigner) Sign(userID uuid.UUID) (string, error) {
	iat := s.clock.Now().UTC().Truncate(time.Second)
	exp := iat.Add(s.config.TTL)

	claims := &UserClaims{
		UserID: userID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.config.Issuer,
			IssuedAt:  jwt.NewNumericDate(iat),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)

	return token.SignedString(s.config.PrivateKey)
}
