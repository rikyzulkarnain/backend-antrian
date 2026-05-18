package domain

import "errors"

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
	ErrInvalidInput  = errors.New("invalid input")
	ErrConflict      = errors.New("conflict")

	ErrInvalidCredentials = errors.New("invalid credentials")
)
