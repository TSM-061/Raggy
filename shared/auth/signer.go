package auth

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/TSM-061/Raggy/shared/clock"
	"github.com/TSM-061/Raggy/shared/configerr"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenSigner struct {
	clock clock.Clock

	privateKey ed25519.PrivateKey
	issuer     string
	ttl        time.Duration
}

func NewTokenSigner(clock clock.Clock, privateKeyBase64 string, iss string, ttl time.Duration) (*TokenSigner, error) {
	privateKey, err := base64.StdEncoding.DecodeString(privateKeyBase64)
	if err != nil {
		return nil, &configerr.ConfigError{
			Key:     "privateKeyBase64",
			Message: "couldn't decode base64",
			Err:     err,
		}

	}

	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, &configerr.ConfigError{
			Key: "privateKeyBase64",
			Message: fmt.Sprintf(
				"invalid key size, want %d, got %d",
				ed25519.PrivateKeySize,
				len(privateKey),
			),
		}
	}

	return &TokenSigner{
		clock:      clock,
		privateKey: ed25519.PrivateKey(privateKey),
		issuer:     iss,
		ttl:        ttl,
	}, nil
}

func (s *TokenSigner) Sign(userID uuid.UUID) (string, error) {
	iat := s.clock.Now().UTC().Truncate(time.Second)
	exp := iat.Add(s.ttl)

	claims := &UserClaims{
		UserID: userID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			IssuedAt:  jwt.NewNumericDate(iat),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	return token.SignedString(s.privateKey)
}
