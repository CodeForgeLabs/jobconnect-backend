package domain

import "errors"

var (
	ErrForbidden    = errors.New("forbidden")
	ErrConflict     = errors.New("conflict")
	ErrInvalidState = errors.New("invalid state")
)
