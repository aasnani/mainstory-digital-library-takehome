package domain

import (
	"errors"
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	RoleMember    = "MEMBER"
	RoleLibrarian = "LIBRARIAN"
	RoleAdmin     = "ADMIN"
)

// Sentinel errors for users; compared with == in api.WriteErrorFromDomain (wrap with %w if you extend callers).
var (
	ErrInvalidEmail     = errors.New("invalid email")
	ErrInvalidRole      = errors.New("invalid role")
	ErrNotFound         = errors.New("not found")
	ErrConflict         = errors.New("conflict")
	ErrForbidden        = errors.New("forbidden")
	ErrUnauthorized     = errors.New("unauthorized")
	ErrCannotDeleteUser = errors.New("cannot delete user: existing entitlements")
)

// User is the public projection: no password_hash here so json.Marshal on handlers can’t leak secrets by accident.
type User struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Role  string    `json:"role"`
}

// UserListFilter drives staff-only GET /users query params; empty filter lists all rows (subject to limit/offset).
type UserListFilter struct {
	UserID *uuid.UUID
	Q      string // substring on email (case-insensitive); if set, must satisfy MinSearchRunes like catalog search
	Role   string // exact MEMBER | LIBRARIAN | ADMIN when non-empty
}

// ValidateUserListFilter enforces the same minimum length as book search for q to reduce noisy queries.
func ValidateUserListFilter(f UserListFilter) error {
	if f.Q != "" && utf8.RuneCountInString(f.Q) < MinSearchRunes {
		return ErrSearchTermTooShort
	}
	if f.Role != "" && !ValidRole(f.Role) {
		return ErrInvalidRole
	}
	return nil
}

// ValidateEmail exists because we reject garbage before hitting Postgres unique constraints (clearer errors than 23505).
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

// NormalizeEmail keeps login/register case-insensitive and stable for UNIQUE(lower(email)) style lookups in SQL.
func NormalizeEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}

// ValidRole mirrors the database CHECK so services and migrations can’t drift on allowed role strings.
func ValidRole(r string) bool {
	switch r {
	case RoleMember, RoleLibrarian, RoleAdmin:
		return true
	default:
		return false
	}
}
