package repository

import (
	"context"

	"github.com/google/uuid"

	"mainstory-digital-library-takehome/internal/domain"
)

// UserStore is implemented by UserRepository. Use the interface in services and tests.
type UserStore interface {
	Create(ctx context.Context, email, role string) (*domain.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	List(ctx context.Context, limit, offset int32) ([]domain.User, error)
	Update(ctx context.Context, id uuid.UUID, email *string, role *string) (*domain.User, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
