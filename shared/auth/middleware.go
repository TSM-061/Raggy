package auth

import (
	"context"
	"net/http"

	"github.com/TSM-061/Raggy/shared/serviceerr"
)

const AccessTokenCookieName = "access_token"

type contextKey string

const userClaimsContextKey contextKey = "userClaims"

type Middleware struct {
	verifier *TokenVerifier
}

func NewMiddleware(verifier *TokenVerifier) *Middleware {
	return &Middleware{verifier: verifier}
}

func (m *Middleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(AccessTokenCookieName)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		claims, err := m.verifier.Verify(cookie.Value)
		if err != nil {
			serviceerr.WriteHTTPError(w, err)
			return
		}

		ctx := context.WithValue(r.Context(), userClaimsContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserClaimsFromContext(ctx context.Context) (*UserClaims, bool) {
	claims, ok := ctx.Value(userClaimsContextKey).(*UserClaims)
	return claims, ok
}
