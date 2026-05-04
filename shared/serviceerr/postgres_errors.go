package serviceerr

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

func WrapPostgresError(err error) error {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	if !ok {
		return err
	}

	switch pgErr.Code {
	case "23502":
		// not_null_violation
		return fmt.Errorf("%w: %w", InvalidInput, err)
	case "23503":
		// foreign_key_violation
		return fmt.Errorf("%w: %w", NotFound, err)
	case "23505":
		// unique_violation
		return fmt.Errorf("%w: %w", Conflict, err)
	default:
		return fmt.Errorf("%w: %w", Internal, err)
	}
}
