package session

import (
	"context"

	"github.com/google/uuid"
)

type Repo interface {
	Create(ctx context.Context, session *Session) error
	GetBySelector(ctx context.Context, selector uuid.UUID) (*Session, error)
	Update(ctx context.Context, s *Session) error
	Delete(ctx context.Context, session *Session) error
}
