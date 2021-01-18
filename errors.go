package rest

import "errors"

var (
	ErrUnmarshal            = errors.New("UNMARSHAL_ERROR")
	ErrMissingRequiredField = errors.New("MISSING_FIELD")
	ErrNotFound             = errors.New("NOT_FOUND")
)
