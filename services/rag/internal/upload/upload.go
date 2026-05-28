package upload

import (
	"time"

	"github.com/google/uuid"
)

type Upload struct {
	ID         uuid.UUID
	Status     Status
	ChunkTotal int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
