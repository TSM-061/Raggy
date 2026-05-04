package serviceerr

import "errors"

var (
	Internal     = errors.New("internal error")
	InvalidInput = errors.New("invalid input")
	NotFound     = errors.New("resource not found")
	Unauthorized = errors.New("unauthorized")
	Conflict     = errors.New("resource already exists")
)
