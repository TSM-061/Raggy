package serviceerr

import "errors"

var (
	NotFound     = errors.New("resource not found")
	Unauthorized = errors.New("unauthorized")
)
