package session

import (
	"context"
	"errors"
	"fmt"

	"github.com/TSM-061/Raggy/shared/serviceerr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type PostgresSessionRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) Repo {
	return &PostgresSessionRepo{
		pool: pool,
	}
}

func (r *PostgresSessionRepo) Create(ctx context.Context, s *Session) error {
	query := `
		INSERT INTO sessions (validator_hash, user_id)
		VALUES ($1, $2)
		RETURNING selector, created_at, updated_at;
	`

	err := r.pool.QueryRow(ctx, query, s.ValidatorHash, s.UserID).
		Scan(&s.Selector, &s.CreatedAt, &s.UpdatedAt)

	if err != nil {
		return fmt.Errorf(
			"session creation failed (user=%s): %w",
			s.UserID,
			serviceerr.WrapPostgresError(err))
	}

	return nil
}

func (r *PostgresSessionRepo) GetBySelector(
	ctx context.Context, selector uuid.UUID) (*Session, error) {

	query := `
		SELECT selector, validator_hash, user_id, created_at, updated_at
		FROM sessions
		WHERE selector = $1;
	`

	session := &Session{}

	err := r.pool.QueryRow(ctx, query, selector).
		Scan(
			&session.Selector,
			&session.ValidatorHash,
			&session.UserID,
			&session.CreatedAt,
			&session.UpdatedAt,
		)

	if err == nil {
		return session, nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("session not found: %w", serviceerr.NotFound)
	}

	return nil, fmt.Errorf(
		"session retrieval failed (selector=%s): %w",
		selector,
		serviceerr.WrapPostgresError(err))
}

func (r *PostgresSessionRepo) Update(
	ctx context.Context, s *Session) error {

	query := `
		UPDATE sessions
		SET selector = uuidv7(),
				validator_hash = $2,
				updated_at = NOW()
		WHERE selector = $1
		RETURNING selector;
	`

	err := r.pool.QueryRow(ctx, query,
		s.Selector,
		s.ValidatorHash,
	).Scan(&s.Selector)

	if err != nil {
		return fmt.Errorf(
			"session refresh failed (selector=%s): %w",
			s.Selector,
			serviceerr.WrapPostgresError(err),
		)
	}

	return nil
}

func (r *PostgresSessionRepo) Delete(ctx context.Context, s *Session) error {
	query := `
		DELETE FROM sessions
		WHERE selector = $1;
	`

	result, err := r.pool.Exec(ctx, query, s.Selector)
	if err != nil {
		return fmt.Errorf(
			"session deletion failed (selector=%s): %w",
			s.Selector,
			serviceerr.WrapPostgresError(err),
		)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("session not found (selector=%s): %w", s.Selector, serviceerr.NotFound)
	}

	return nil
}
