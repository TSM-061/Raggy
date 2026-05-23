package upload

import (
	"time"

	"github.com/google/uuid"
)

type Upload struct {
	ID         uuid.UUID
	UploadedBy uuid.UUID

	OriginalName string
	ContentType  string
	ProfileHint  string
	SizeBytes    int64

	Status Status

	CreatedAt time.Time
	UpdatedAt time.Time
}
