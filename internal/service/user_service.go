// Package service contains use-cases orchestrating repositories and auth helpers (domain rules live here, not in Gin).
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

// Register always creates MEMBER rows: privileged roles are DB/ops concerns, not self-serve signup.
func (s *UserService) Register(ctx context.Context, email, password string) (*domain.User, string, error) {
	if err := domain.ValidateEmail(email); err != nil {
		return nil, "", err
	}
	if err := domain.ValidatePassword(password); err != nil {
		return nil, "", err
	}
	email = domain.NormalizeEmail(email)
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, "", err
	}
	u, err := s.repo.Create(ctx, email, domain.RoleMember, hash)
	if err != nil {
		return nil, "", err
	}
	tok, err := s.IssueToken(u)
	if err != nil {
		return nil, "", err
	}
	return u, tok, nil
}

// Login returns ErrUnauthorized on missing users or bad passwords to avoid account enumeration via different errors.
func (s *UserService) Login(ctx context.Context, email, password string) (*domain.User, string, error) {
	if err := domain.ValidateEmail(email); err != nil {
		return nil, "", err
	}
	if password == "" {
		return nil, "", domain.ErrUnauthorized
	}
	email = domain.NormalizeEmail(email)
	creds, err := s.repo.GetAuthCredentialsByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, "", domain.ErrUnauthorized
		}
		return nil, "", err
	}
	if !auth.PasswordMatches(password, creds.PasswordHash) {
		return nil, "", domain.ErrUnauthorized
	}
	u, err := s.repo.GetByID(ctx, creds.UserID)
	if err != nil {
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

// PatchInput separates admin mutations (email/role) from self-service password changes to prevent mixed semantics in one struct without rules.
type PatchInput struct {
	Email           *string
	Role            *string
	CurrentPassword *string
	NewPassword     *string
}

// Patch encodes RBAC: admins can’t set others’ passwords via PATCH; members can’t escalate role/email on self.
func (s *UserService) Patch(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID, in PatchInput, isAdmin bool) (*domain.User, error) {
	if _, err := s.repo.GetByID(ctx, targetID); err != nil {
		return nil, err
	}

	if !isAdmin && actorID != targetID {
		return nil, domain.ErrForbidden
	}

	if isAdmin && actorID != targetID {
		if in.CurrentPassword != nil || in.NewPassword != nil {
			return nil, domain.ErrCannotPatchOtherUserPassword
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
			role = in.Role
		}
		return s.repo.Update(ctx, targetID, email, role)
	}

	if in.Email != nil || in.Role != nil {
		return nil, domain.ErrForbidden
	}

	hasCurrent := in.CurrentPassword != nil
	hasNew := in.NewPassword != nil
	if hasCurrent || hasNew {
		if !hasCurrent || !hasNew {
			return nil, domain.ErrInvalidPasswordChange
		}
		if err := domain.ValidatePassword(*in.NewPassword); err != nil {
			return nil, err
		}
		creds, err := s.repo.GetAuthCredentialsByID(ctx, targetID)
		if err != nil {
			return nil, err
		}
		if !auth.PasswordMatches(*in.CurrentPassword, creds.PasswordHash) {
			return nil, domain.ErrUnauthorized
		}
		hash, err := auth.HashPassword(*in.NewPassword)
		if err != nil {
			return nil, err
		}
		if err := s.repo.UpdatePasswordHash(ctx, targetID, hash); err != nil {
			return nil, err
		}
		return s.repo.GetByID(ctx, targetID)
	}

	return nil, domain.ErrEmptyPatch
}

func (s *UserService) Delete(ctx context.Context, id uuid.UUID) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		return err
	}
	return nil
}
