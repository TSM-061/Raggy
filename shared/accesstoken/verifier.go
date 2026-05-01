package accesstoken

import (
	"crypto/ed25519"
	"fmt"

	"github.com/TSM-061/Raggy/shared/clock"
	"github.com/TSM-061/Raggy/shared/serviceerr"
	"github.com/golang-jwt/jwt/v5"
)

type Verifier struct {
	publicKey ed25519.PublicKey
	clock     clock.Clock
}

func NewVerifier(clock clock.Clock, pubKey ed25519.PublicKey) *Verifier {
	return &Verifier{
		publicKey: pubKey,
		clock:     clock,
	}
}

func (v *Verifier) Verify(tokenStr string) (*UserClaims, error) {
	claims := &UserClaims{}

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}

		return v.publicKey, nil
	},
		jwt.WithTimeFunc(v.clock.Now),
	)

	if err != nil || !token.Valid {
		return nil, fmt.Errorf("%w: %w", serviceerr.Unauthorized, err)
	}

	return claims, nil
}
