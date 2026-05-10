package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"mainstory-digital-library-takehome/internal/auth"
	"mainstory-digital-library-takehome/internal/config"
	"mainstory-digital-library-takehome/internal/domain"
	"mainstory-digital-library-takehome/internal/repository"
)

// fakeStore is an in-memory UserStore for unit tests.
type fakeStore struct {
	byID    map[uuid.UUID]*domain.User
	byEmail map[string]uuid.UUID
	hashes  map[string]string // normalized email -> bcrypt hash
	// onCreate may return a custom error (e.g. conflict).
	onCreate func(email, role, passwordHash string) error
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		byID:    make(map[uuid.UUID]*domain.User),
		byEmail: make(map[string]uuid.UUID),
		hashes:  make(map[string]string),
	}
}

func mustHash(t *testing.T, plain string) string {
	t.Helper()
	h, err := auth.HashPassword(plain)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func (f *fakeStore) Create(ctx context.Context, email, role, passwordHash string) (*domain.User, error) {
	if f.onCreate != nil {
		if err := f.onCreate(email, role, passwordHash); err != nil {
			return nil, err
		}
	}
	email = domain.NormalizeEmail(email)
	if _, exists := f.byEmail[email]; exists {
		return nil, domain.ErrConflict
	}
	u := &domain.User{ID: uuid.New(), Email: email, Role: role}
	f.byID[u.ID] = u
	f.byEmail[email] = u.ID
	f.hashes[email] = passwordHash
	return u, nil
}

func (f *fakeStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return cloneUser(u), nil
}

func (f *fakeStore) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	id, ok := f.byEmail[domain.NormalizeEmail(email)]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return cloneUser(f.byID[id]), nil
}

func (f *fakeStore) GetAuthCredentialsByEmail(ctx context.Context, email string) (*repository.AuthCredentials, error) {
	id, ok := f.byEmail[domain.NormalizeEmail(email)]
	if !ok {
		return nil, domain.ErrNotFound
	}
	u := f.byID[id]
	h := f.hashes[u.Email]
	return &repository.AuthCredentials{UserID: u.ID, Role: u.Role, PasswordHash: h}, nil
}

func (f *fakeStore) List(ctx context.Context, limit, offset int32) ([]domain.User, error) {
	var out []domain.User
	for _, u := range f.byID {
		out = append(out, *cloneUser(u))
	}
	if int(offset) >= len(out) {
		return nil, nil
	}
	end := int(offset) + int(limit)
	if end > len(out) {
		end = len(out)
	}
	return out[int(offset):end], nil
}

func (f *fakeStore) Update(ctx context.Context, id uuid.UUID, email *string, role *string) (*domain.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	if email != nil {
		newE := domain.NormalizeEmail(*email)
		if newE != u.Email {
			if _, taken := f.byEmail[newE]; taken {
				return nil, domain.ErrConflict
			}
			delete(f.byEmail, u.Email)
			h := f.hashes[u.Email]
			delete(f.hashes, u.Email)
			u.Email = newE
			f.byEmail[newE] = id
			f.hashes[newE] = h
		}
	}
	if role != nil {
		u.Role = *role
	}
	return cloneUser(u), nil
}

func (f *fakeStore) Delete(ctx context.Context, id uuid.UUID) error {
	u, ok := f.byID[id]
	if !ok {
		return domain.ErrNotFound
	}
	delete(f.byID, id)
	delete(f.byEmail, u.Email)
	delete(f.hashes, u.Email)
	return nil
}

func cloneUser(u *domain.User) *domain.User {
	c := *u
	return &c
}

func testConfig(t *testing.T) *config.Config {
	t.Helper()
	return &config.Config{
		JWTSecret: []byte("unit-test-secret-at-least-32-characters-long"),
		JWTExpiry: time.Hour,
	}
}

func TestRegister_InvalidEmail(t *testing.T) {
	svc := NewUserService(testConfig(t), newFakeStore())
	_, _, err := svc.Register(context.Background(), "not-an-email", "password123")
	if !errors.Is(err, domain.ErrInvalidEmail) {
		t.Fatalf("want ErrInvalidEmail, got %v", err)
	}
}

func TestRegister_InvalidPassword(t *testing.T) {
	svc := NewUserService(testConfig(t), newFakeStore())
	_, _, err := svc.Register(context.Background(), "ok@example.com", "short")
	if !errors.Is(err, domain.ErrInvalidPassword) {
		t.Fatalf("want ErrInvalidPassword, got %v", err)
	}
}

func TestRegister_Success(t *testing.T) {
	svc := NewUserService(testConfig(t), newFakeStore())
	u, tok, err := svc.Register(context.Background(), "User@Example.COM ", "password123")
	if err != nil {
		t.Fatal(err)
	}
	if u.Role != domain.RoleMember {
		t.Fatalf("role %q", u.Role)
	}
	if !strings.Contains(tok, ".") {
		t.Fatal("expected JWT shape")
	}
}

func TestRegister_Conflict(t *testing.T) {
	store := newFakeStore()
	svc := NewUserService(testConfig(t), store)
	if _, _, err := svc.Register(context.Background(), "a@b.com", "password123"); err != nil {
		t.Fatal(err)
	}
	_, _, err := svc.Register(context.Background(), "a@b.com", "password123")
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("want conflict, got %v", err)
	}
}

func TestLogin_UnknownEmail(t *testing.T) {
	svc := NewUserService(testConfig(t), newFakeStore())
	_, _, err := svc.Login(context.Background(), "missing@example.com", "password123")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("want unauthorized, got %v", err)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	store := newFakeStore()
	svc := NewUserService(testConfig(t), store)
	if _, _, err := svc.Register(context.Background(), "ok@example.com", "correct-pass"); err != nil {
		t.Fatal(err)
	}
	_, _, err := svc.Login(context.Background(), "ok@example.com", "wrong-password")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("want unauthorized, got %v", err)
	}
}

func TestLogin_Success(t *testing.T) {
	store := newFakeStore()
	svc := NewUserService(testConfig(t), store)
	if _, _, err := svc.Register(context.Background(), "ok@example.com", "password123"); err != nil {
		t.Fatal(err)
	}
	u, tok, err := svc.Login(context.Background(), "ok@example.com", "password123")
	if err != nil {
		t.Fatal(err)
	}
	if u.Email != domain.NormalizeEmail("ok@example.com") {
		t.Fatalf("email %q", u.Email)
	}
	claims, err := auth.Parse(testConfig(t), tok)
	if err != nil {
		t.Fatal(err)
	}
	id, err := auth.UserID(claims)
	if err != nil || id != u.ID {
		t.Fatalf("token subject mismatch")
	}
}

func TestPatch_ForbiddenCrossUser(t *testing.T) {
	store := newFakeStore()
	u1, _ := store.Create(context.Background(), "one@test.com", domain.RoleMember, mustHash(t, "pw123456"))
	u2, _ := store.Create(context.Background(), "two@test.com", domain.RoleMember, mustHash(t, "pw123456"))
	svc := NewUserService(testConfig(t), store)

	newEmail := "x@test.com"
	_, err := svc.Patch(context.Background(), u1.ID, u2.ID, PatchInput{Email: &newEmail}, false)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("want forbidden, got %v", err)
	}
}

func TestPatch_SelfCannotChangeRole(t *testing.T) {
	store := newFakeStore()
	u, _ := store.Create(context.Background(), "me@test.com", domain.RoleMember, mustHash(t, "pw123456"))
	svc := NewUserService(testConfig(t), store)
	role := domain.RoleAdmin
	_, err := svc.Patch(context.Background(), u.ID, u.ID, PatchInput{Role: &role}, false)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("want forbidden, got %v", err)
	}
}

func TestPatch_AdminCanChangeRole(t *testing.T) {
	store := newFakeStore()
	u, _ := store.Create(context.Background(), "sub@test.com", domain.RoleMember, mustHash(t, "pw123456"))
	svc := NewUserService(testConfig(t), store)
	role := domain.RoleAdmin
	out, err := svc.Patch(context.Background(), u.ID, u.ID, PatchInput{Role: &role}, true)
	if err != nil {
		t.Fatal(err)
	}
	if out.Role != domain.RoleAdmin {
		t.Fatalf("got %q", out.Role)
	}
}

func TestPatch_InvalidRole(t *testing.T) {
	store := newFakeStore()
	u, _ := store.Create(context.Background(), "r@test.com", domain.RoleMember, mustHash(t, "pw123456"))
	svc := NewUserService(testConfig(t), store)
	bad := "SUPERUSER"
	_, err := svc.Patch(context.Background(), u.ID, u.ID, PatchInput{Role: &bad}, true)
	if !errors.Is(err, domain.ErrInvalidRole) {
		t.Fatalf("want invalid role, got %v", err)
	}
}
