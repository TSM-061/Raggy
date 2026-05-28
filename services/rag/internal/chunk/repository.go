package chunk

import (
	"context"

	"github.com/google/uuid"
)

type Repo interface {
	Create(ctx context.Context, chunk *Chunk) error
	Search(ctx context.Context, queryVector []float32, limit int) ([]Chunk, error)
	UpdateEmbedding(ctx context.Context, id uuid.UUID, embedding []float32) error
	DeleteByUploadID(ctx context.Context, uploadID uuid.UUID) error
}
