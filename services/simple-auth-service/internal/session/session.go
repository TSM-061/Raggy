package session

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	Selector uuid.UUID
	UserID   uuid.UUID

	ValidatorHash []byte

	CreatedAt time.Time
	UpdatedAt time.Time
}
