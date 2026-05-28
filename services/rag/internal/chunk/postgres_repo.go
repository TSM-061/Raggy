package chunk

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/TSM-061/Raggy/shared/serviceerr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) Repo {
	return &PostgresRepo{pool: pool}
}

func (r *PostgresRepo) Create(ctx context.Context, chunk *Chunk) error {
	query := `
		INSERT INTO chunks (upload_id, chunk_index, content)
		VALUES ($1, $2, $3::jsonb)
		RETURNING id, created_at;
	`

	err := r.pool.QueryRow(
		ctx,
		query,
		chunk.UploadID,
		chunk.ChunkIndex,
		json.RawMessage(chunk.Content),
	).Scan(&chunk.ID, &chunk.CreatedAt)
	if err != nil {
		return fmt.Errorf(
			"chunk creation failed: %w",
			serviceerr.WrapPostgresError(err),
		)
	}

	return nil
}

func (r *PostgresRepo) Search(ctx context.Context, queryVector []float32, limit int) ([]Chunk, error) {
	query := `
		SELECT
			id,
			upload_id,
			chunk_index,
			content,
			created_at
		FROM chunks
		WHERE embedding IS NOT NULL
		ORDER BY embedding <=> $1::vector ASC, chunk_index ASC
		LIMIT $2;
	`

	rows, err := r.pool.Query(ctx, query, formatEmbedding(queryVector), limit)
	if err != nil {
		return nil, fmt.Errorf(
			"chunk search failed: %w",
			serviceerr.WrapPostgresError(err),
		)
	}
	defer rows.Close()

	results := make([]Chunk, 0)
	for rows.Next() {
		var result Chunk
		if err := rows.Scan(
			&result.ID,
			&result.UploadID,
			&result.ChunkIndex,
			&result.Content,
			&result.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf(
				"chunk search scan failed: %w",
				serviceerr.WrapPostgresError(err),
			)
		}

		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"chunk search iteration failed: %w",
			serviceerr.WrapPostgresError(err),
		)
	}

	return results, nil
}

func (r *PostgresRepo) UpdateEmbedding(ctx context.Context, id uuid.UUID, embedding []float32) error {
	query := `
		UPDATE chunks
		SET embedding = NULLIF($2, '')
		WHERE id = $1;
	`

	result, err := r.pool.Exec(ctx, query, id, formatEmbedding(embedding))
	if err != nil {
		return fmt.Errorf(
			"chunk embedding update failed (id=%s): %w",
			id,
			serviceerr.WrapPostgresError(err),
		)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("chunk not found (id=%s): %w", id, serviceerr.NotFound)
	}

	return nil
}

func (r *PostgresRepo) DeleteByUploadID(ctx context.Context, uploadID uuid.UUID) error {
	query := `
		DELETE FROM chunks
		WHERE upload_id = $1;
	`

	_, err := r.pool.Exec(ctx, query, uploadID)
	if err != nil {
		return fmt.Errorf(
			"chunk delete failed (upload_id=%s): %w",
			uploadID,
			serviceerr.WrapPostgresError(err),
		)
	}

	return nil
}

func formatEmbedding(embedding []float32) string {
	if len(embedding) == 0 {
		return ""
	}

	var b strings.Builder
	b.Grow(2 + 16*len(embedding))
	b.WriteByte('[')

	for i, value := range embedding {
		if i > 0 {
			b.WriteByte(',')
		}

		b.WriteString(strconv.FormatFloat(float64(value), 'f', -1, 32))
	}

	b.WriteByte(']')

	return b.String()
}
