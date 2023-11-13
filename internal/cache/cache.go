package cache

import "errors"

var (
	ErrNotFound = errors.New("not found")
	ErrInternal = errors.New("internal") // TODO: wrap internal error
)
