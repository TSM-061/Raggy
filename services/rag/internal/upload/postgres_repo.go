package upload

import (
	"context"
	"errors"
	"fmt"

	"github.com/TSM-061/Raggy/shared/serviceerr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) Repo {
	return &PostgresRepo{pool: pool}
}

func (r *PostgresRepo) Upsert(ctx context.Context, upload *Upload) error {
	query := `
		INSERT INTO uploads (id, status, chunk_total)
		VALUES ($1, 'processing', $2)
		ON CONFLICT (id) DO NOTHING;
	`

	_, err := r.pool.Exec(
		ctx,
		query,
		upload.ID,
		upload.ChunkTotal,
	)
	if err != nil {
		return fmt.Errorf(
			"upload upsert failed: %w",
			serviceerr.WrapPostgresError(err),
		)
	}

	return nil
}

func (r *PostgresRepo) GetByID(ctx context.Context, id uuid.UUID) (*Upload, error) {
	query := `
		SELECT id, status, chunk_total, created_at, updated_at
		FROM uploads
		WHERE id = $1;
	`

	var upload Upload
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&upload.ID,
		&upload.Status,
		&upload.ChunkTotal,
		&upload.CreatedAt,
		&upload.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("upload not found (id=%s): %w", id, serviceerr.NotFound)
		}

		return nil, fmt.Errorf(
			"upload fetch failed (id=%s): %w",
			id,
			serviceerr.WrapPostgresError(err),
		)
	}

	return &upload, nil
}

func (r *PostgresRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status Status) error {
	if !status.IsValid() {
		return fmt.Errorf("%w: invalid upload status '%s'", serviceerr.InvalidInput, status)
	}

	query := `
		UPDATE uploads
		SET status = $2,
			updated_at = NOW()
		WHERE id = $1;
	`

	result, err := r.pool.Exec(ctx, query, id, status)
	if err != nil {
		return fmt.Errorf(
			"upload status update failed (id=%s): %w",
			id,
			serviceerr.WrapPostgresError(err),
		)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("upload not found (id=%s): %w", id, serviceerr.NotFound)
	}

	return nil
}
