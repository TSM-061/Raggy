package token

import (
	"crypto/ed25519"
	"fmt"

	"github.com/TSM-061/Raggy/shared/serviceerr"
	"github.com/golang-jwt/jwt/v5"
)

type Verifier struct {
	publicKey ed25519.PublicKey
}

func NewVerifier(pubKey ed25519.PublicKey) *Verifier {
	return &Verifier{
		publicKey: pubKey,
	}
}

func (v *Verifier) Verify(tokenStr string) (*UserClaims, error) {
	claims := &UserClaims{}

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}

		return v.publicKey, nil
	})

	if err != nil || !token.Valid {
		return nil, fmt.Errorf("%w: %w", serviceerr.Unauthorized, err)
	}

	return claims, nil
}
