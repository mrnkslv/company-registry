package errs

import "errors"

var (
	ErrNotFound      = errors.New("company not found")
	ErrAlreadyExists = errors.New("company already exists")
	ErrValidation    = errors.New("validation failed")
)
