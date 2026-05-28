package chunk

import (
	"time"

	"github.com/google/uuid"
)

type Chunk struct {
	ID         uuid.UUID
	UploadID   uuid.UUID
	ChunkIndex int
	Content    []byte
	// Omitted as not returned in queries / or used in creation
	// Embedding  []float32
	CreatedAt time.Time
}
