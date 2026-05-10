package repository

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"

	"mainstory-digital-library-takehome/internal/db"
	"mainstory-digital-library-takehome/internal/domain"
)

// Integration tests require DATABASE_URL and a migrated database (Flyway V1 applied).
// CI skips these when DATABASE_URL is unset.
func skipWithoutDB(t *testing.T) {
	t.Helper()
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("set DATABASE_URL to run integration tests against Postgres")
	}
}

func TestUserRepository_CreateGetUpdateDelete(t *testing.T) {
	skipWithoutDB(t)
	ctx := context.Background()
	pool, err := db.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	repo := NewUserRepository(pool)
	email := "repo-test-" + uuid.New().String() + "@example.com"

	u, err := repo.Create(ctx, email, domain.RoleMember)
	if err != nil {
		t.Fatal(err)
	}
	if u.Email != strings.ToLower(strings.TrimSpace(email)) {
		t.Fatalf("email normalization: got %q want lower(trim)", u.Email)
	}

	got, err := repo.GetByID(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != u.ID {
		t.Fatal("id mismatch")
	}

	byEmail, err := repo.GetByEmail(ctx, strings.ToUpper(email))
	if err != nil {
		t.Fatal(err)
	}
	if byEmail.ID != u.ID {
		t.Fatal("GetByEmail mismatch")
	}

	newMail := "updated-" + uuid.New().String() + "@example.com"
	updated, err := repo.Update(ctx, u.ID, &newMail, nil)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Email != domain.NormalizeEmail(newMail) {
		t.Fatalf("update email %q", updated.Email)
	}

	if err := repo.Delete(ctx, u.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetByID(ctx, u.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("after delete: %v", err)
	}
}

func TestUserRepository_DuplicateEmail(t *testing.T) {
	skipWithoutDB(t)
	ctx := context.Background()
	pool, err := db.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	repo := NewUserRepository(pool)
	email := "dup-" + uuid.New().String() + "@example.com"
	u, err := repo.Create(ctx, email, domain.RoleMember)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = repo.Delete(ctx, u.ID) }()

	_, err = repo.Create(ctx, email, domain.RoleMember)
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("want conflict, got %v", err)
	}
}
