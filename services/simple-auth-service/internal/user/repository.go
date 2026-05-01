package user

import (
	"context"
)

type Repo interface {
	Create(ctx context.Context, user *User) error
	Exists(ctx context.Context, username string) (bool, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
}
