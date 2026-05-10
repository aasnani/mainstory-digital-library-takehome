package domain

import (
	"errors"
	"unicode/utf8"
)

const (
	PasswordMinLen = 8
	PasswordMaxLen = 72 // bcrypt operates on at most 72 bytes; keep UX predictable
)

var (
	ErrInvalidPassword              = errors.New("invalid password")
	ErrInvalidPasswordChange        = errors.New("invalid password change")
	ErrEmptyPatch                   = errors.New("empty patch")
	ErrCannotPatchOtherUserPassword = errors.New("cannot change another user's password via this API")
)

// ValidatePassword enforces UX minimum by rune count (user-visible length) and bcrypt’s 72-byte ceiling with len(password) (bcrypt counts bytes).
func ValidatePassword(password string) error {
	n := utf8.RuneCountInString(password)
	if n < PasswordMinLen {
		return ErrInvalidPassword
	}
	if len(password) > PasswordMaxLen {
		return ErrInvalidPassword
	}
	return nil
}
