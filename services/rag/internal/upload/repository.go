package upload

import (
	"context"

	"github.com/google/uuid"
)

type Repo interface {
	Upsert(ctx context.Context, upload *Upload) error
	GetByID(ctx context.Context, id uuid.UUID) (*Upload, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status Status) error
}
