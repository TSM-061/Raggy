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
            chunk_total,
            chunks_processed,
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
            chunk_total,
            chunks_processed,
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
		&upload.ChunkTotal,
		&upload.ChunksProcessed,
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
		SET status = CASE WHEN $2 = 'uploaded' AND status IN ('processing', 'processed') THEN status ELSE $2 END,
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

// ApplyProgress atomically merges cumulative counts. Older snapshots cannot
// decrement progress, and the total cannot change once an upload is identified.
func (r *PostgresRepo) ApplyProgress(ctx context.Context, id uuid.UUID, total, processed int) error {
	if total < 0 || processed < 0 || processed > total {
		return serviceerr.InvalidInput
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	result, err := tx.Exec(ctx, `UPDATE uploads SET
		chunk_total = $2,
		chunks_processed = GREATEST(chunks_processed, $3),
		status = CASE WHEN GREATEST(chunks_processed, $3) = $2 THEN 'processed' ELSE 'processing' END,
		updated_at = NOW()
		WHERE id = $1 AND (chunk_total IS NULL OR chunk_total = $2)`, id, total, processed)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		var exists bool
		if err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM uploads WHERE id = $1)", id).Scan(&exists); err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("%w: conflicting chunk total", serviceerr.InvalidInput)
		}
		return serviceerr.NotFound
	}
	// Notifications are delivered only after the surrounding transaction commits.
	if _, err := tx.Exec(ctx, "SELECT pg_notify('upload_progress', 'changed')"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
