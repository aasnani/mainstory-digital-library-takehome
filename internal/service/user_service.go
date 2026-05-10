package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"mainstory-digital-library-takehome/internal/auth"
	"mainstory-digital-library-takehome/internal/config"
	"mainstory-digital-library-takehome/internal/domain"
	"mainstory-digital-library-takehome/internal/repository"
)

type UserService struct {
	cfg  *config.Config
	repo repository.UserStore
}

func NewUserService(cfg *config.Config, repo repository.UserStore) *UserService {
	return &UserService{cfg: cfg, repo: repo}
}

func (s *UserService) IssueToken(u *domain.User) (string, error) {
	return auth.Sign(s.cfg, u.ID, u.Role)
}

func (s *UserService) Register(ctx context.Context, email string) (*domain.User, string, error) {
	if err := domain.ValidateEmail(email); err != nil {
		return nil, "", err
	}
	email = domain.NormalizeEmail(email)
	u, err := s.repo.Create(ctx, email, domain.RoleMember)
	if err != nil {
		return nil, "", err
	}
	tok, err := s.IssueToken(u)
	if err != nil {
		return nil, "", err
	}
	return u, tok, nil
}

func (s *UserService) Login(ctx context.Context, email string) (*domain.User, string, error) {
	if err := domain.ValidateEmail(email); err != nil {
		return nil, "", err
	}
	email = domain.NormalizeEmail(email)
	u, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, "", domain.ErrUnauthorized
		}
		return nil, "", err
	}
	tok, err := s.IssueToken(u)
	if err != nil {
		return nil, "", err
	}
	return u, tok, nil
}

func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *UserService) List(ctx context.Context, limit, offset int32) ([]domain.User, error) {
	return s.repo.List(ctx, limit, offset)
}

type PatchInput struct {
	Email *string
	Role  *string
}

func (s *UserService) Patch(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID, in PatchInput, isAdmin bool) (*domain.User, error) {
	if _, err := s.repo.GetByID(ctx, targetID); err != nil {
		return nil, err
	}

	if !isAdmin && actorID != targetID {
		return nil, domain.ErrForbidden
	}

	var email *string
	var role *string

	if in.Email != nil {
		if err := domain.ValidateEmail(*in.Email); err != nil {
			return nil, err
		}
		n := domain.NormalizeEmail(*in.Email)
		email = &n
	}

	if in.Role != nil {
		if !domain.ValidRole(*in.Role) {
			return nil, domain.ErrInvalidRole
		}
		if !isAdmin {
			return nil, domain.ErrForbidden
		}
		role = in.Role
	}

	return s.repo.Update(ctx, targetID, email, role)
}

func (s *UserService) Delete(ctx context.Context, id uuid.UUID) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		return err
	}
	return nil
}
