package session

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/TSM-061/Raggy/simple-auth-service/internal/testutil"
	"github.com/TSM-061/Raggy/simple-auth-service/internal/user"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"
)

type testEnv struct {
	sessions Repo
	users    user.Repo
	manager  *Manager
}

func newTestEnv(ctx context.Context) *testEnv {
	pool, err := testutil.StartDatabase(ctx)
	if err != nil {
		panic(err)
	}

	sessions := NewPostgresRepo(pool)
	users := user.NewPostgresRepo(pool)

	manager := NewManager("SECRET", sessions)

	return &testEnv{
		users:    users,
		sessions: sessions,
		manager:  manager,
	}
}

func TestSessionManagerService_Create(t *testing.T) {
	ctx := context.Background()

	e := newTestEnv(ctx)

	t.Run("UserExists", func(t *testing.T) {
		user := seedTestUser(ctx, t, e.users)

		refreshToken, err := e.manager.Create(ctx, user.ID)

		if err != nil {
			t.Fatalf("Create() unexpected error: %v", err)
		}

		if refreshToken == "" {
			t.Fatalf("Create() token: got %q, want valid refreshToken", refreshToken)
		}

		parts := strings.Split(refreshToken, ".")
		if len(parts) != 2 {
			t.Fatalf("Create() token: got %q, want selector.validator format", refreshToken)
		}

		decodedSelector, err := base64.RawURLEncoding.DecodeString(parts[0])
		if err != nil {
			t.Fatalf("Create() token.selector: got %q, want base64 encoded string", parts[0])
		}

		if _, err := uuid.FromBytes(decodedSelector); err != nil {
			t.Fatalf("Create() token.selector: got %q, want valid uuid", decodedSelector)
		}

	})
}

func TestSessionManagerService_Refresh(t *testing.T) {
	ctx := context.Background()

	e := newTestEnv(ctx)

	t.Run("ValidToken", func(t *testing.T) {
		user := seedTestUser(ctx, t, e.users)

		refreshToken, err := e.manager.Create(ctx, user.ID)
		if err != nil {
			t.Fatalf("Create() unexpected error: %v", err)
		}

		result, err := e.manager.Refresh(ctx, refreshToken)
		if err != nil {
			t.Fatalf("Refresh() unexpected error: %v", err)
		}

		if result.UserID != user.ID {
			t.Fatalf("Refresh() UserID: got %v, want %v", result.UserID, user.ID)
		}

		if result.RefreshToken == "" {
			t.Fatalf("Refresh() RefreshToken: got empty, want valid token")
		}

		// Verify token format
		parts := strings.Split(result.RefreshToken, ".")
		if len(parts) != 2 {
			t.Fatalf("Refresh() token format: got %q, want selector.validator format", result.RefreshToken)
		}

		// Verify token can't be reused
		_, err = e.manager.Refresh(ctx, refreshToken)
		if err == nil {
			t.Fatalf("Refresh() with used token: got nil error, want error")
		}
	})

	t.Run("IncorrectValidator", func(t *testing.T) {
		user := seedTestUser(ctx, t, e.users)

		refreshToken, err := e.manager.Create(ctx, user.ID)
		if err != nil {
			t.Fatalf("Create() unexpected error: %v", err)
		}

		selector, _, err := parseRefreshToken(refreshToken)
		if err != nil {
			t.Fatalf("Refresh() unexpected error: %v", err)
		}

		guessedToken := encodeRefreshToken(selector, generateValidator())

		if _, err := e.manager.Refresh(ctx, guessedToken); err == nil {
			t.Fatalf("Refresh() with incorrect validator: got nil error, want error")
		}

	})

	t.Run("InvalidToken", func(t *testing.T) {
		invalidToken := "invalid.token"

		_, err := e.manager.Refresh(ctx, invalidToken)
		if err == nil {
			t.Fatalf("Refresh() with invalid token: got nil error, want error")
		}
	})
}

func TestSessionManagerService_Delete(t *testing.T) {
	ctx := context.Background()

	e := newTestEnv(ctx)

	t.Run("ValidToken", func(t *testing.T) {
		user := seedTestUser(ctx, t, e.users)

		refreshToken, err := e.manager.Create(ctx, user.ID)
		if err != nil {
			t.Fatalf("Create() unexpected error: %v", err)
		}

		err = e.manager.Delete(ctx, refreshToken)
		if err != nil {
			t.Fatalf("Delete() unexpected error: %v", err)
		}

		// Verify token can't be used after deletion
		_, err = e.manager.Refresh(ctx, refreshToken)
		if err == nil {
			t.Fatalf("Refresh() with deleted token: got nil error, want error")
		}
	})

	t.Run("InvalidToken", func(t *testing.T) {
		invalidToken := "invalid.token"

		err := e.manager.Delete(ctx, invalidToken)
		if err == nil {
			t.Fatalf("Delete() with invalid token: got nil error, want error")
		}
	})
}

func seedTestUser(ctx context.Context, t *testing.T, users user.Repo) *user.User {
	t.Helper()

	user := &user.User{
		Username:     gofakeit.Username(),
		PasswordHash: "fake-hash",
	}

	if err := users.Create(ctx, user); err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}

	return user
}
