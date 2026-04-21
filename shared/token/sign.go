package token

import (
	"crypto/ed25519"

	"github.com/golang-jwt/jwt/v5"
)

type Signer struct {
	privateKey ed25519.PrivateKey
	issuer     string
}

func NewSigner(privKey ed25519.PrivateKey, iss string) *Signer {
	return &Signer{
		privateKey: privKey,
		issuer:     iss,
	}
}

func (s *Signer) Generate(claims UserClaims) (string, error) {
	if s.issuer != "" {
		claims.Issuer = s.issuer
	}

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	return token.SignedString(s.privateKey)
}
