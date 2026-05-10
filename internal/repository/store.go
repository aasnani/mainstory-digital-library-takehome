package repository

import (
	"context"

	"github.com/google/uuid"

	"mainstory-digital-library-takehome/internal/domain"
)

// AuthCredentials holds fields needed for login; never expose PasswordHash in HTTP JSON.
type AuthCredentials struct {
	UserID       uuid.UUID
	Role         string
	PasswordHash string
}

// UserStore is implemented by UserRepository. Use the interface in services and tests.
type UserStore interface {
	Create(ctx context.Context, email, role, passwordHash string) (*domain.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetAuthCredentialsByEmail(ctx context.Context, email string) (*AuthCredentials, error)
	List(ctx context.Context, limit, offset int32) ([]domain.User, error)
	Update(ctx context.Context, id uuid.UUID, email *string, role *string) (*domain.User, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
