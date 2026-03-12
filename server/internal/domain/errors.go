package domain

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrAlreadyExists      = errors.New("already exists")
	ErrPermissionDenied   = errors.New("permission denied")
	ErrGuildFull          = errors.New("guild full")
	ErrAlreadyMember      = errors.New("already a member")
	ErrNotMember          = errors.New("not a member")
	ErrInvalidTransition  = errors.New("invalid state transition")
	ErrNotClaimer         = errors.New("not the claimer")
	ErrInvalidCredentials = errors.New("invalid credentials")
)
