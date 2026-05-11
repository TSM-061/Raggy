package web

import (
	"encoding/json"
	"math"
	"net/http"

	"github.com/TSM-061/Raggy/shared/serviceerr"
	"github.com/TSM-061/Raggy/simple-auth-service/internal/services"
)

const (
	AccessTokenCookieName  = "access_token"
	RefreshTokenCookieName = "refresh_token"
)

func newAccessTokenCookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     AccessTokenCookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
}

func newRefreshTokenCookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    value,
		Path:     "/api/auth",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
}

func (s *Server) setAuthCookies(w http.ResponseWriter, authResult *services.AuthResult) {
	accessTokenCookie := newAccessTokenCookie(
		authResult.Tokens.AccessToken,
		int(s.config.AccessTokenTTL.Seconds()),
	)
	refreshTokenCookie := newRefreshTokenCookie(
		authResult.Tokens.RefreshToken,
		math.MaxInt,
	)
	http.SetCookie(w, accessTokenCookie)
	http.SetCookie(w, refreshTokenCookie)
}

func clearAuthCookies(w http.ResponseWriter) {
	accessTokenCookie := newAccessTokenCookie("", -1)
	refreshTokenCookie := newRefreshTokenCookie("", -1)
	http.SetCookie(w, accessTokenCookie)
	http.SetCookie(w, refreshTokenCookie)
}

type signinRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

func (s *Server) HandleSignin(w http.ResponseWriter, r *http.Request) {
	var body signinRequest

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.Validator.Struct(body); err != nil {
		// TODO use specific mapping for validation errors
		http.Error(w, "invalid request parameters", http.StatusBadRequest)
		return
	}
	result, err := s.Auth.Signin(r.Context(), body.Username, body.Password)
	if err != nil {
		serviceerr.WriteHTTPError(w, err)
		return
	}

	if result == nil || result.Tokens == nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	s.setAuthCookies(w, result)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(RefreshTokenCookieName)
	if err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	result, err := s.Auth.Refresh(r.Context(), cookie.Value)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}

	s.setAuthCookies(w, result)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) HandleSignout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(RefreshTokenCookieName)
	if err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if err := s.Auth.Signout(r.Context(), cookie.Value); err != nil {
		http.Error(w, "failed to signout", http.StatusInternalServerError)
	}

	clearAuthCookies(w)
	w.WriteHeader(http.StatusOK)
}
