// Package repository is the SQL boundary: returns domain types and sentinel errors, not gin contexts.
package repository

import (
	"context"

	"github.com/google/uuid"

	"mainstory-digital-library-takehome/internal/domain"
)

// AuthCredentials holds fields needed for login; never expose PasswordHash in HTTP JSON.
type AuthCredentials struct {
	UserID uuid.UUID
	Role   string
	// PasswordHash is separate from domain.User so list/get user queries never load bcrypt blobs.
	PasswordHash string
}

// UserStore is implemented by UserRepository. Use the interface in services and tests.
type UserStore interface {
	Create(ctx context.Context, email, role, passwordHash string) (*domain.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetAuthCredentialsByEmail(ctx context.Context, email string) (*AuthCredentials, error)
	GetAuthCredentialsByID(ctx context.Context, id uuid.UUID) (*AuthCredentials, error)
	ListFiltered(ctx context.Context, filter domain.UserListFilter, limit, offset int32) ([]domain.User, error)
	Update(ctx context.Context, id uuid.UUID, email *string, role *string) (*domain.User, error)
	UpdatePasswordHash(ctx context.Context, id uuid.UUID, passwordHash string) error
	Delete(ctx context.Context, id uuid.UUID) error
}
