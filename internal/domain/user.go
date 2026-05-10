package domain

import (
	"errors"
	"net/mail"
	"strings"

	"github.com/google/uuid"
)

const (
	RoleMember    = "MEMBER"
	RoleLibrarian = "LIBRARIAN"
	RoleAdmin     = "ADMIN"
)

var (
	ErrInvalidEmail     = errors.New("invalid email")
	ErrInvalidRole      = errors.New("invalid role")
	ErrNotFound         = errors.New("not found")
	ErrConflict         = errors.New("conflict")
	ErrForbidden        = errors.New("forbidden")
	ErrUnauthorized     = errors.New("unauthorized")
	ErrCannotDeleteUser = errors.New("cannot delete user: existing entitlements")
)

type User struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Role  string    `json:"role"`
}

func ValidateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" || len(email) > 255 {
		return ErrInvalidEmail
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return ErrInvalidEmail
	}
	return nil
}

func NormalizeEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}

func ValidRole(r string) bool {
	switch r {
	case RoleMember, RoleLibrarian, RoleAdmin:
		return true
	default:
		return false
	}
}
