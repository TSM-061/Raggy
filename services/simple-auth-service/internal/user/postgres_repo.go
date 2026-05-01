package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/TSM-061/Raggy/shared/serviceerr"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) Repo {
	return &PostgresRepo{
		pool: pool,
	}
}

func (r *PostgresRepo) Create(ctx context.Context, user *User) error {
	query := `
		INSERT INTO users (username, password_hash)
		VALUES ($1, $2)
		RETURNING id, created_at, updated_at
	`

	err := r.pool.QueryRow(ctx, query, user.Username, user.PasswordHash).
		Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return fmt.Errorf("user creation failed: %w", serviceerr.WrapPostgresError(err))
	}

	return nil
}

func (r *PostgresRepo) Exists(ctx context.Context, username string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM users
			WHERE username = $1
		)
	`

	var exists bool
	if err := r.pool.QueryRow(ctx, query, username).Scan(&exists); err != nil {
		return false, fmt.Errorf(
			"failed to check user existence: %w",
			serviceerr.WrapPostgresError(err),
		)
	}

	return exists, nil
}

func (r *PostgresRepo) GetByUsername(
	ctx context.Context, username string) (*User, error) {

	query := `
		SELECT id, username, password_hash, created_at, updated_at
		FROM users
		WHERE username = $1
	`

	user := &User{}

	err := r.pool.QueryRow(ctx, query, username).
		Scan(
			&user.ID,
			&user.Username,
			&user.PasswordHash,
			&user.CreatedAt,
			&user.UpdatedAt,
		)

	if err == nil {
		return user, nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("user not found: %s %w", username, serviceerr.NotFound)
	}

	return nil, fmt.Errorf(
		"database error retrieving user by username: %w",
		serviceerr.WrapPostgresError(err),
	)
}
