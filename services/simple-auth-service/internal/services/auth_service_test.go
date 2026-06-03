package services

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	auth "github.com/TSM-061/Raggy/shared/auth"
	"github.com/TSM-061/Raggy/shared/clock"
	"github.com/TSM-061/Raggy/shared/serviceerr"
	"github.com/TSM-061/Raggy/simple-auth-service/internal/password"
	"github.com/TSM-061/Raggy/simple-auth-service/internal/session"
	"github.com/TSM-061/Raggy/simple-auth-service/internal/testutil"
	"github.com/TSM-061/Raggy/simple-auth-service/internal/user"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"
)

type testEnv struct {
	verifier *auth.TokenVerifier
	service  *Auth
	users    user.Repo
	hasher   *password.Argon2Hasher
}

func newTestEnv(ctx context.Context) *testEnv {
	pool, err := testutil.StartDatabase(ctx)
	if err != nil {
		panic(err)
	}

	users := user.NewPostgresRepo(pool)
	sessions := session.NewPostgresRepo(pool)
	hasher := password.NewArgon2Hasher(&password.Argon2Config{
		Pepper:  "PEPPER",
		KeyLen:  32,
		Memory:  64 * 1024,
		Time:    3,
		Threads: 2,
	})

	clock := &clock.MockClock{CurrentTime: time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)}

	publicKey, privateKey := newTestKeyPair(7)
	signer, err := auth.NewTokenSigner(&auth.TokenSignerConfig{
		PrivateKeyBase64: privateKey,
		Issuer:           "raggy-auth-test",
		TTL:              15 * time.Minute,
	}, clock)
	if err != nil {
		panic(err)
	}

	return &testEnv{
		users:    users,
		verifier: auth.NewTokenVerifier(clock, publicKey),
		service: NewAuthService(
			users,
			hasher,
			signer,
			session.NewManager("SECRET", sessions),
		),
		hasher: hasher,
	}
}

func newTestKeyPair(seedByte byte) (ed25519.PublicKey, string) {
	seed := bytes.Repeat([]byte{seedByte}, ed25519.SeedSize)
	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := privateKey.Public().(ed25519.PublicKey)

	return publicKey, base64.StdEncoding.EncodeToString(privateKey)
}

func TestAuthService_Register(t *testing.T) {
	ctx := context.Background()
	e := newTestEnv(ctx)

	t.Run("Success", func(t *testing.T) {
		username := testUsername()
		plaintextPassword := testPassword()

		createdUser, err := e.service.Register(ctx, username, plaintextPassword)
		if err != nil {
			t.Fatalf("Register() unexpected error: %v", err)
		}

		if createdUser == nil {
			t.Fatal("Register() user: got nil, want created user")
		}

		if createdUser.Username != username {
			t.Fatalf("Register() Username: got %q, want %q", createdUser.Username, username)
		}

		if createdUser.PasswordHash == "" {
			t.Fatal("Register() PasswordHash: got empty, want hashed password")
		}

		if createdUser.PasswordHash == plaintextPassword {
			t.Fatal("Register() PasswordHash: got plaintext password, want hash")
		}
	})

	t.Run("DuplicateUsername", func(t *testing.T) {
		username := testUsername()
		plaintextPassword := testPassword()

		firstUser, err := e.service.Register(ctx, username, plaintextPassword)
		if err != nil {
			t.Fatalf("first Register() unexpected error: %v", err)
		}

		if firstUser == nil {
			t.Fatal("first Register() user: got nil, want created user")
		}

		duplicateUser, err := e.service.Register(ctx, username, plaintextPassword)
		if err == nil {
			t.Fatal("second Register() error: got nil, want duplicate error")
		}

		if !errors.Is(err, serviceerr.Conflict) {
			t.Fatalf("second Register() error: got %v, want conflict error", err)
		}

		if duplicateUser != nil {
			t.Fatalf("second Register() user: got %#v, want nil", duplicateUser)
		}
	})
}

func TestAuthService_Register_InvalidUsernameOrPassword(t *testing.T) {
	ctx := context.Background()
	e := newTestEnv(ctx)

	tests := []struct {
		name     string
		username string
		password string
	}{
		{
			name:     "UsernameTooShort",
			username: strings.Repeat("a", user.MinUsernameLength-1),
			password: testPassword(),
		},
		{
			name:     "UsernameTooLong",
			username: strings.Repeat("a", user.MaxUsernameLength+1),
			password: testPassword(),
		},
		{
			name:     "PasswordTooShort",
			username: testUsername(),
			password: strings.Repeat("a", user.MinPasswordLength-1),
		},
		{
			name:     "PasswordTooLong",
			username: testUsername(),
			password: strings.Repeat("a", user.MaxPasswordLength+1),
		},
		{
			name:     "UsernameStartsWithUnderscore",
			username: "_" + testUsername(),
			password: testPassword(),
		},
		{
			name:     "UsernameEndsWithDash",
			username: "valid-user-",
			password: testPassword(),
		},
		{
			name:     "UsernameContainsInvalidCharacter",
			username: "invalid@user",
			password: testPassword(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createdUser, err := e.service.Register(ctx, tt.username, tt.password)
			if err == nil {
				t.Fatalf("Register() error: got nil, want invalid input error")
			}

			if !errors.Is(err, serviceerr.InvalidInput) {
				t.Fatalf("Register() error: got generic error, want invaid input error")
			}

			if createdUser != nil {
				t.Fatalf("Register() user: got %#v, want nil", createdUser)
			}
		})
	}
}

func TestAuthService_Signin(t *testing.T) {
	ctx := context.Background()
	e := newTestEnv(ctx)

	user, password := e.registerTestUser(ctx, t)

	authResult, err := e.service.Signin(ctx, user.Username, password)
	if err != nil {
		t.Fatalf("Signin() unexpected error: %v", err)
	}

	assertAuthResult(t, e, user.ID, authResult)
}

func TestAuthService_Refresh(t *testing.T) {
	ctx := context.Background()
	e := newTestEnv(ctx)

	t.Run("Success", func(t *testing.T) {
		user, password := e.registerTestUser(ctx, t)

		authResult, err := e.service.Signin(ctx, user.Username, password)
		if err != nil {
			t.Fatalf("Signin() unexpected error: %v", err)
		}

		refreshedAuthResult, err := e.service.Refresh(ctx, authResult.Tokens.RefreshToken)
		if err != nil {
			t.Fatalf("Refresh() unexpected error: %v", err)
		}

		assertAuthResult(t, e, user.ID, refreshedAuthResult)

		if refreshedAuthResult.Tokens.RefreshToken == authResult.Tokens.RefreshToken {
			t.Fatal("Refresh() RefreshToken: got unchanged token, want rotated token")
		}
	})

	t.Run("InvalidToken", func(t *testing.T) {
		authResult, err := e.service.Refresh(ctx, "invalid.token")
		if err == nil {
			t.Fatal("Refresh() error: got nil, want invalid input error")
		}

		if !errors.Is(err, serviceerr.InvalidInput) {
			t.Fatalf("Refresh() error: got %v, want invalid input error", err)
		}

		if authResult != nil {
			t.Fatalf("Refresh() result: got %#v, want nil", authResult)
		}
	})
}

func assertAuthResult(t *testing.T, e *testEnv, userID uuid.UUID, authResult *AuthResult) {
	t.Helper()

	if authResult == nil {
		t.Fatal("result: got nil, want auth result")
	}

	if authResult.Tokens == nil {
		t.Fatal("Tokens: got nil, want token pair")
	}

	if authResult.Tokens.AccessToken == "" {
		t.Fatal("AccessToken: got empty, want signed token")
	}

	if authResult.Tokens.RefreshToken == "" {
		t.Fatal("RefreshToken: got empty, want session token")
	}

	claims, err := e.verifier.Verify(authResult.Tokens.AccessToken)
	if err != nil {
		t.Fatalf("Verify() unexpected error: %v", err)
	}

	if claims.UserID != userID.String() {
		t.Fatalf("AccessToken subject: got %q, want %q", claims.UserID, userID.String())
	}

	parts := strings.Split(authResult.Tokens.RefreshToken, ".")
	if len(parts) != 2 {
		t.Fatalf("RefreshToken format: got %q, want selector.validator format", authResult.Tokens.RefreshToken)
	}

	selector, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatalf("RefreshToken selector decode: unexpected error: %v", err)
	}

	if _, err := uuid.FromBytes(selector); err != nil {
		t.Fatalf("RefreshToken selector UUID: unexpected error: %v", err)
	}
}

func (e *testEnv) registerTestUser(
	ctx context.Context, t *testing.T) (*user.User, string) {

	t.Helper()

	password := "password"

	user := &user.User{
		Username:     gofakeit.Username(),
		PasswordHash: e.hasher.Hash("password"),
	}

	err := e.users.Create(ctx, user)
	if err != nil {
		t.Fatalf("Register() test setup failed: %v", err)
	}

	return user, password
}

func testUsername() string {
	return gofakeit.Regex(UsernameRegexStr)
}

func testPassword() string {
	return gofakeit.Password(true, true, true, true, false, 32)
}
