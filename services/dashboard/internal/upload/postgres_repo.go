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

func (r *PostgresRepo) Create(ctx context.Context, upload *Upload) error {
	query := `
		INSERT INTO uploads (
			uploaded_by,
			original_name,
			content_type,
			profile_hint,
			size_bytes,
			status
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at;
	`

	err := r.pool.QueryRow(
		ctx,
		query,
		upload.UploadedBy,
		upload.OriginalName,
		upload.ContentType,
		upload.ProfileHint,
		upload.SizeBytes,
		upload.Status,
	).Scan(&upload.ID, &upload.CreatedAt, &upload.UpdatedAt)
	if err != nil {
		return fmt.Errorf(
			"upload creation failed: %w",
			serviceerr.WrapPostgresError(err),
		)
	}

	return nil
}

func (r *PostgresRepo) List(
	ctx context.Context,
	limit int,
	offset int,
) ([]Upload, int64, error) {
	countQuery := `
		SELECT COUNT(*)
		FROM uploads;
	`

	var total int64
	if err := r.pool.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf(
			"upload count fetch failed: %w",
			serviceerr.WrapPostgresError(err),
		)
	}

	query := `
		SELECT
			id,
			uploaded_by,
			original_name,
			content_type,
			profile_hint,
			size_bytes,
			status,
			created_at,
			updated_at
		FROM uploads
		ORDER BY created_at DESC, id DESC
		LIMIT $1 OFFSET $2;
	`

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf(
			"upload list fetch failed: %w",
			serviceerr.WrapPostgresError(err),
		)
	}

	uploads, err := pgx.CollectRows(rows, pgx.RowToStructByName[Upload])
	if err != nil {
		return nil, 0, fmt.Errorf(
			"upload list collect failed: %w",
			serviceerr.WrapPostgresError(err),
		)
	}

	return uploads, total, nil
}

func (r *PostgresRepo) GetByID(ctx context.Context, id uuid.UUID) (*Upload, error) {
	query := `
		SELECT
			id,
			uploaded_by,
			original_name,
			content_type,
			profile_hint,
			size_bytes,
			status,
			created_at,
			updated_at
		FROM uploads
		WHERE id = $1;
	`

	var upload Upload
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&upload.ID,
		&upload.UploadedBy,
		&upload.OriginalName,
		&upload.ContentType,
		&upload.ProfileHint,
		&upload.SizeBytes,
		&upload.Status,
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

func (r *PostgresRepo) DeleteByID(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM uploads
		WHERE id = $1;
	`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf(
			"upload delete failed (id=%s): %w",
			id,
			serviceerr.WrapPostgresError(err),
		)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("upload not found (id=%s): %w", id, serviceerr.NotFound)
	}

	return nil
}