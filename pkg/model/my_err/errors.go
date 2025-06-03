package my_err

import "errors"

var (
	ErrEmpty           = errors.New("empty")
	ErrNotFound        = errors.New("notFound")
	ErrNotInUse        = errors.New("notInUse")
	ErrOutOfRange      = errors.New("id out of range")
	ErrReleaseMismatch = errors.New("release mismatch")
)
