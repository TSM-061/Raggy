package accesstoken

import (
	"crypto/ed25519"
	"time"

	"github.com/TSM-061/Raggy/shared/clock"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Signer struct {
	clock clock.Clock

	privateKey ed25519.PrivateKey
	issuer     string
	ttl        time.Duration
}

func NewSigner(clk clock.Clock, privKey ed25519.PrivateKey, iss string, ttl time.Duration) *Signer {
	return &Signer{
		clock:      clk,
		privateKey: privKey,
		issuer:     iss,
		ttl:        ttl,
	}
}

func (s *Signer) Sign(userID uuid.UUID) (string, error) {
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
