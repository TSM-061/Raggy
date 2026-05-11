package accesstoken

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/TSM-061/Raggy/shared/clock"
	"github.com/TSM-061/Raggy/shared/configerr"
	"github.com/TSM-061/Raggy/shared/serviceerr"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func testKeyPair(seedByte byte) (ed25519.PublicKey, string) {
	seed := bytes.Repeat([]byte{seedByte}, ed25519.SeedSize)
	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := privateKey.Public().(ed25519.PublicKey)

	return publicKey, base64.StdEncoding.EncodeToString(privateKey)
}

func TestSignerAndVerifierIntegration(t *testing.T) {
	fixedNow := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	validPublicKey, validPrivateKey := testKeyPair(1)
	wrongPublicKey, _ := testKeyPair(2)

	tests := []struct {
		name          string
		token         func(t *testing.T) string
		verifierKey   ed25519.PublicKey
		verifierClock clock.Clock
		assert        func(t *testing.T, claims *UserClaims, err error)
	}{
		{
			name: "success",
			token: func(t *testing.T) string {
				t.Helper()

				clk := &clock.MockClock{CurrentTime: fixedNow}
				signer, err := NewSigner(clk, validPrivateKey, "raggy-test", 15*time.Minute)
				if err != nil {
					t.Fatalf("NewSigner() error = %v", err)
				}
				token, err := signer.Sign(userID)
				if err != nil {
					t.Fatalf("Sign() error = %v", err)
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

				if claims.UserID != userID.String() {
					t.Fatalf("Verify() UserID = %q, want %q", claims.UserID, userID.String())
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

				clk := &clock.MockClock{CurrentTime: fixedNow}
				signer, err := NewSigner(clk, validPrivateKey, "raggy-test", 15*time.Minute)
				if err != nil {
					t.Fatalf("NewSigner() error = %v", err)
				}
				token, err := signer.Sign(userID)
				if err != nil {
					t.Fatalf("Sign() error = %v", err)
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

				// Issue a token in the past with a 1-second TTL so it is
				// already expired by the time Verify is called.
				pastTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
				clk := &clock.MockClock{CurrentTime: pastTime}
				signer, err := NewSigner(clk, validPrivateKey, "raggy-test", 1*time.Second)
				if err != nil {
					t.Fatalf("NewSigner() error = %v", err)
				}
				token, err := signer.Sign(userID)
				if err != nil {
					t.Fatalf("Sign() error = %v", err)
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
			verifierClock := tt.verifierClock
			if verifierClock == nil {
				verifierClock = &clock.MockClock{CurrentTime: fixedNow}
			}
			verifier := NewVerifier(verifierClock, tt.verifierKey)
			token := tt.token(t)

			claims, err := verifier.Verify(token)
			tt.assert(t, claims, err)
		})
	}
}

func TestNewSigner_InvalidKey(t *testing.T) {
	clk := &clock.MockClock{CurrentTime: time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)}

	tests := []struct {
		name string
		key  string
	}{
		{
			name: "not valid base64",
			key:  "not-valid-base64!!!",
		},
		{
			name: "wrong key size",
			key:  base64.StdEncoding.EncodeToString([]byte("too-short")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signer, err := NewSigner(clk, tt.key, "raggy-test", 15*time.Minute)

			if signer != nil {
				t.Fatal("NewSigner() signer = non-nil, want nil")
			}

			var configErr *configerr.ConfigError
			if !errors.As(err, &configErr) {
				t.Fatalf("NewSigner() error = %v, want *configerr.ConfigError", err)
			}
		})
	}
}
