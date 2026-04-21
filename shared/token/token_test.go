package token

import (
	"bytes"
	"crypto/ed25519"
	"errors"
	"testing"
	"time"

	"github.com/TSM-061/Raggy/shared/serviceerr"
	"github.com/golang-jwt/jwt/v5"
)

func testKeyPair(seedByte byte) (ed25519.PublicKey, ed25519.PrivateKey) {
	seed := bytes.Repeat([]byte{seedByte}, ed25519.SeedSize)
	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := privateKey.Public().(ed25519.PublicKey)

	return publicKey, privateKey
}

func TestSignerAndVerifierIntegration(t *testing.T) {
	issuedAt := time.Now().UTC().Truncate(time.Second)
	expiresAt := issuedAt.Add(15 * time.Minute)

	validPublicKey, validPrivateKey := testKeyPair(1)
	wrongPublicKey, _ := testKeyPair(2)

	tests := []struct {
		name        string
		token       func(t *testing.T) string
		verifierKey ed25519.PublicKey
		assert      func(t *testing.T, claims *UserClaims, err error)
	}{
		{
			name: "success",
			token: func(t *testing.T) string {
				t.Helper()

				signer := NewSigner(validPrivateKey, "raggy-test")
				token, err := signer.Generate(UserClaims{
					UserID: "user-123",
					RegisteredClaims: jwt.RegisteredClaims{
						IssuedAt:  jwt.NewNumericDate(issuedAt),
						ExpiresAt: jwt.NewNumericDate(expiresAt),
					},
				})
				if err != nil {
					t.Fatalf("Generate() error = %v", err)
				}

				return token
			},
			verifierKey: validPublicKey,
			assert: func(t *testing.T, claims *UserClaims, err error) {
				t.Helper()

				if err != nil {
					t.Fatalf("Verify() error = %v", err)
				}

				if claims == nil {
					t.Fatal("Verify() claims = nil")
				}

				if claims.UserID != "user-123" {
					t.Fatalf("Verify() UserID = %q, want %q", claims.UserID, "user-123")
				}

				if claims.Issuer != "raggy-test" {
					t.Fatalf("Verify() Issuer = %q, want %q", claims.Issuer, "raggy-test")
				}
			},
		},
		{
			name: "wrong verification key",
			token: func(t *testing.T) string {
				t.Helper()

				signer := NewSigner(validPrivateKey, "raggy-test")
				token, err := signer.Generate(UserClaims{
					UserID: "user-123",
				})
				if err != nil {
					t.Fatalf("Generate() error = %v", err)
				}

				return token
			},
			verifierKey: wrongPublicKey,
			assert: func(t *testing.T, claims *UserClaims, err error) {
				t.Helper()

				if claims != nil {
					t.Fatalf("Verify() claims = %#v, want nil", claims)
				}

				if err == nil {
					t.Fatal("Verify() error = nil, want unauthorized error")
				}

				if !errors.Is(err, serviceerr.Unauthorized) {
					t.Fatalf("Verify() error = %v, want unauthorized error", err)
				}
			},
		},
		{
			name: "malformed token",
			token: func(t *testing.T) string {
				t.Helper()
				return "not-a-jwt"
			},
			verifierKey: validPublicKey,
			assert: func(t *testing.T, claims *UserClaims, err error) {
				t.Helper()

				if claims != nil {
					t.Fatalf("Verify() claims = %#v, want nil", claims)
				}

				if err == nil {
					t.Fatal("Verify() error = nil, want unauthorized error")
				}

				if !errors.Is(err, serviceerr.Unauthorized) {
					t.Fatalf("Verify() error = %v, want unauthorized error", err)
				}
			},
		},
		{
			name: "unexpected signing method",
			token: func(t *testing.T) string {
				t.Helper()

				token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"sub": "user-123",
				})

				tokenString, err := token.SignedString([]byte("shared-secret"))
				if err != nil {
					t.Fatalf("SignedString() error = %v", err)
				}

				return tokenString
			},
			verifierKey: validPublicKey,
			assert: func(t *testing.T, claims *UserClaims, err error) {
				t.Helper()

				if claims != nil {
					t.Fatalf("Verify() claims = %#v, want nil", claims)
				}

				if err == nil {
					t.Fatal("Verify() error = nil, want unauthorized error")
				}

				if !errors.Is(err, serviceerr.Unauthorized) {
					t.Fatalf("Verify() error = %v, want unauthorized error", err)
				}
			},
		},
		{
			name: "expired token",
			token: func(t *testing.T) string {
				t.Helper()

				signer := NewSigner(validPrivateKey, "raggy-test")
				token, err := signer.Generate(UserClaims{
					UserID: "user-123",
					RegisteredClaims: jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(issuedAt.Add(-1 * time.Minute)),
					},
				})
				if err != nil {
					t.Fatalf("Generate() error = %v", err)
				}

				return token
			},
			verifierKey: validPublicKey,
			assert: func(t *testing.T, claims *UserClaims, err error) {
				t.Helper()

				if claims != nil {
					t.Fatalf("Verify() claims = %#v, want nil", claims)
				}

				if err == nil {
					t.Fatal("Verify() error = nil, want unauthorized error")
				}

				if !errors.Is(err, serviceerr.Unauthorized) {
					t.Fatalf("Verify() error = %v, want unauthorized error", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifier := NewVerifier(tt.verifierKey)
			token := tt.token(t)

			claims, err := verifier.Verify(token)
			tt.assert(t, claims, err)
		})
	}
}
