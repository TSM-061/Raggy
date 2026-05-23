package upload

import (
	"context"

	"github.com/google/uuid"
)

type Repo interface {
	Create(ctx context.Context, upload *Upload) error
	List(ctx context.Context, limit int, offset int) ([]Upload, int64, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status Status) error
	DeleteByID(ctx context.Context, id uuid.UUID) error
}