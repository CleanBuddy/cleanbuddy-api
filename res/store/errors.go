package store

import "errors"

var (
	ErrUniqueViolation = errors.New("store: duplicate key value violates unique constraint")
	ErrInvalidInput    = errors.New("store: invalid input")

	// Invitation code errors
	ErrInvitationCodeNotFound        = errors.New("store: invitation code not found")
	ErrInvitationCodeAlreadyRedeemed = errors.New("store: invitation code has already been redeemed")
)
