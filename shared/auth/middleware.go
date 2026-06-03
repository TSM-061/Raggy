package auth

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/TSM-061/Raggy/shared/logger"
	"github.com/TSM-061/Raggy/shared/serviceerr"
	"github.com/google/uuid"
)

const AccessTokenCookieName = "access_token"

type contextKey struct{}

var userIDContextKey contextKey = contextKey{}

type Middleware struct {
	verifier *TokenVerifier
}

func NewMiddleware(verifier *TokenVerifier) *Middleware {
	return &Middleware{verifier: verifier}
}

func (m *Middleware) WrapFn(nextFunc http.HandlerFunc) http.Handler {
	handler := http.HandlerFunc(nextFunc)
	return m.Wrap(handler)
}

func (m *Middleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := logger.FromContext(ctx)

		cookie, err := r.Cookie(AccessTokenCookieName)
		if err != nil {
			log.WarnContext(
				ctx,
				"failed to verify access token",
				slog.Any("error", err),
			)

			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		claims, err := m.verifier.Verify(cookie.Value)
		if err != nil {
			log.WarnContext(
				ctx,
				"failed to verify access token",
				slog.String("cookie_value", cookie.Value),
				slog.Any("error", err),
			)
			serviceerr.WriteHTTPError(w, err)
			return
		}

		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx = context.WithValue(ctx, userIDContextKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetUserID(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(userIDContextKey).(uuid.UUID)
	return userID, ok
}

func MustGetUserID(ctx context.Context) uuid.UUID {
	userID, ok := GetUserID(ctx)
	if !ok {
		panic("auth user id missing from context. forgot to wrap handler with auth middleware.")
	}

	return userID
}
