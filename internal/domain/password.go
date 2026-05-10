package domain

import (
	"errors"
	"unicode/utf8"
)

const (
	PasswordMinLen = 8
	PasswordMaxLen = 72 // bcrypt operates on at most 72 bytes; keep UX predictable
)

var ErrInvalidPassword = errors.New("invalid password")

// ValidatePassword checks length bounds suitable for bcrypt-backed storage.
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
