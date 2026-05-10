package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"mainstory-digital-library-takehome/internal/domain"
)

// Compile-time check that UserRepository implements UserStore.
var _ UserStore = (*UserRepository)(nil)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, email, role, passwordHash string) (*domain.User, error) {
	const q = `
		INSERT INTO users (email, role, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, email, role`
	row := r.pool.QueryRow(ctx, q, email, role, passwordHash)
	var u domain.User
	if err := row.Scan(&u.ID, &u.Email, &u.Role); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, domain.ErrConflict
		}
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	const q = `SELECT id, email, role FROM users WHERE id = $1`
	var u domain.User
	err := r.pool.QueryRow(ctx, q, id).Scan(&u.ID, &u.Email, &u.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	const q = `SELECT id, email, role FROM users WHERE lower(email) = lower($1)`
	var u domain.User
	err := r.pool.QueryRow(ctx, q, email).Scan(&u.ID, &u.Email, &u.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetAuthCredentialsByEmail(ctx context.Context, email string) (*AuthCredentials, error) {
	const q = `SELECT id, role, password_hash FROM users WHERE lower(email) = lower($1)`
	var c AuthCredentials
	err := r.pool.QueryRow(ctx, q, email).Scan(&c.UserID, &c.Role, &c.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *UserRepository) List(ctx context.Context, limit, offset int32) ([]domain.User, error) {
	const q = `
		SELECT id, email, role FROM users
		ORDER BY email ASC
		LIMIT $1 OFFSET $2`
	rows, err := r.pool.Query(ctx, q, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Role); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *UserRepository) Update(ctx context.Context, id uuid.UUID, email *string, role *string) (*domain.User, error) {
	u, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	newEmail := u.Email
	newRole := u.Role
	if email != nil {
		newEmail = *email
	}
	if role != nil {
		newRole = *role
	}
	const q = `UPDATE users SET email = $2, role = $3 WHERE id = $1 RETURNING id, email, role`
	row := r.pool.QueryRow(ctx, q, id, newEmail, newRole)
	var out domain.User
	if err := row.Scan(&out.ID, &out.Email, &out.Role); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, domain.ErrConflict
		}
		return nil, err
	}
	return &out, nil
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return domain.ErrCannotDeleteUser
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
