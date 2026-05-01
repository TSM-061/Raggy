package user

import (
	"github.com/google/uuid"
	"time"
)

const (
	MinUsernameLength = 3
	MaxUsernameLength = 32
	MinPasswordLength = 15
	MaxPasswordLength = 32
)

type User struct {
	ID           uuid.UUID
	Username     string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
