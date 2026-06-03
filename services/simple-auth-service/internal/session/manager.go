package session

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"log/slog"

	"github.com/TSM-061/Raggy/shared/logger"
	"github.com/TSM-061/Raggy/shared/serviceerr"
	"github.com/google/uuid"
)

type Manager struct {
	secret   string
	sessions Repo
}

func NewManager(secret string, sessions Repo) *Manager {
	return &Manager{
		secret:   secret,
		sessions: sessions,
	}
}

func (m *Manager) Create(ctx context.Context, userId uuid.UUID) (string, error) {
	log := logger.FromContext(ctx)

	validator := generateValidator()

	s := &Session{
		ValidatorHash: m.hash(validator),
		UserID:        userId,
	}

	err := m.sessions.Create(ctx, s)
	if err != nil {
		return "", err
	}

	log.InfoContext(
		ctx,
		"user signed in",
		slog.String("user_id", userId.String()),
	)

	return encodeRefreshToken(s.Selector, validator), nil
}

func generateValidator() []byte {
	validator := make([]byte, 32)
	rand.Read(validator)

	return validator
}

func (m *Manager) hash(validator []byte) []byte {
	h := sha256.New()

	h.Write(validator)
	h.Write([]byte(m.secret))

	return h.Sum(nil)
}

type RefreshResult struct {
	UserID       uuid.UUID
	RefreshToken string
}

func (m *Manager) Refresh(ctx context.Context, token string) (*RefreshResult, error) {
	log := logger.FromContext(ctx)

	selector, validator, err := parseRefreshToken(token)
	if err != nil {
		return nil, err
	}

	session, err := m.sessions.GetBySelector(ctx, selector)
	if err != nil {
		return nil, err
	}

	isMatch := subtle.ConstantTimeCompare(
		m.hash(validator),
		session.ValidatorHash) == 1

	if !isMatch {
		return nil, fmt.Errorf("%w: couldn't verify token", serviceerr.Unauthorized)
	}

	newValidator := generateValidator()
	session.ValidatorHash = m.hash(newValidator)

	if err := m.sessions.Update(ctx, session); err != nil {
		return nil, err
	}

	log.InfoContext(
		ctx,
		"session token refreshed",
		slog.String("user_id", session.UserID.String()),
		slog.String("session_selector", session.Selector.String()),
	)

	return &RefreshResult{
		UserID:       session.UserID,
		RefreshToken: encodeRefreshToken(session.Selector, newValidator),
	}, nil
}

func (m *Manager) Delete(ctx context.Context, token string) error {
	log := logger.FromContext(ctx)

	selector, validator, err := parseRefreshToken(token)
	if err != nil {
		return err
	}

	session, err := m.sessions.GetBySelector(ctx, selector)
	if err != nil {
		return err
	}

	isMatch := subtle.ConstantTimeCompare(
		m.hash(validator),
		session.ValidatorHash) == 1

	if !isMatch {
		return fmt.Errorf("%w: couldn't verify token", serviceerr.Unauthorized)
	}

	if err := m.sessions.Delete(ctx, session); err != nil {
		return err
	}

	log.InfoContext(
		ctx,
		"user signed out",
		slog.String("user_id", session.UserID.String()),
		slog.String("session_selector", session.Selector.String()),
	)

	return nil
}
