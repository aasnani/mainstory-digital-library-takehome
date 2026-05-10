package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"mainstory-digital-library-takehome/internal/domain"
)

// EntitlementStore is implemented by EntitlementRepository.
type EntitlementStore interface {
	Create(ctx context.Context, userID uuid.UUID, bookID *uuid.UUID, typ, status string, endsAt *time.Time) (*domain.Entitlement, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Entitlement, error)
	ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]domain.Entitlement, error)
	ListAll(ctx context.Context, limit, offset int32) ([]domain.Entitlement, error)
	Update(ctx context.Context, id uuid.UUID, status *string, endsAt *time.Time) (*domain.Entitlement, error)
	HasActiveSubscription(ctx context.Context, userID uuid.UUID) (bool, error)
	HasActivePurchase(ctx context.Context, userID, bookID uuid.UUID) (bool, error)
	GetActiveSubscriptionEntitlement(ctx context.Context, userID uuid.UUID) (*domain.Entitlement, error)
	ListActivePurchasesByUser(ctx context.Context, userID uuid.UUID) ([]domain.Entitlement, error)
	BookExists(ctx context.Context, bookID uuid.UUID) (bool, error)
}

type EntitlementRepository struct {
	pool *pgxpool.Pool
}

func NewEntitlementRepository(pool *pgxpool.Pool) *EntitlementRepository {
	return &EntitlementRepository{pool: pool}
}

func (r *EntitlementRepository) Create(ctx context.Context, userID uuid.UUID, bookID *uuid.UUID, typ, status string, endsAt *time.Time) (*domain.Entitlement, error) {
	const q = `
		INSERT INTO entitlements (user_id, book_id, type, status, ends_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, book_id, type, status, ends_at, created_at`
	row := r.pool.QueryRow(ctx, q, userID, bookID, typ, status, endsAt)
	e, err := scanEntitlement(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23505":
				return nil, domain.ErrConflict
			case "23503":
				return nil, domain.ErrNotFound
			}
		}
		return nil, err
	}
	return e, nil
}

func (r *EntitlementRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Entitlement, error) {
	const q = `
		SELECT id, user_id, book_id, type, status, ends_at, created_at
		FROM entitlements WHERE id = $1`
	row := r.pool.QueryRow(ctx, q, id)
	e, err := scanEntitlement(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return e, err
}

func (r *EntitlementRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]domain.Entitlement, error) {
	const q = `
		SELECT id, user_id, book_id, type, status, ends_at, created_at
		FROM entitlements WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`
	return r.scanList(ctx, q, userID, limit, offset)
}

func (r *EntitlementRepository) ListAll(ctx context.Context, limit, offset int32) ([]domain.Entitlement, error) {
	const q = `
		SELECT id, user_id, book_id, type, status, ends_at, created_at
		FROM entitlements
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`
	rows, err := r.pool.Query(ctx, q, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEntitlementRows(rows)
}

func (r *EntitlementRepository) scanList(ctx context.Context, q string, args ...interface{}) ([]domain.Entitlement, error) {
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEntitlementRows(rows)
}

func scanEntitlementRows(rows pgx.Rows) ([]domain.Entitlement, error) {
	var out []domain.Entitlement
	for rows.Next() {
		var e domain.Entitlement
		if err := rows.Scan(&e.ID, &e.UserID, &e.BookID, &e.Type, &e.Status, &e.EndsAt, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *EntitlementRepository) Update(ctx context.Context, id uuid.UUID, status *string, endsAt *time.Time) (*domain.Entitlement, error) {
	cur, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	newStatus := cur.Status
	newEnds := cur.EndsAt
	if status != nil {
		newStatus = *status
	}
	if endsAt != nil {
		newEnds = endsAt
	}
	const q = `
		UPDATE entitlements SET status = $2, ends_at = $3 WHERE id = $1
		RETURNING id, user_id, book_id, type, status, ends_at, created_at`
	row := r.pool.QueryRow(ctx, q, id, newStatus, newEnds)
	return scanEntitlement(row)
}

func (r *EntitlementRepository) HasActiveSubscription(ctx context.Context, userID uuid.UUID) (bool, error) {
	const q = `
		SELECT EXISTS(
			SELECT 1 FROM entitlements
			WHERE user_id = $1 AND type = $2 AND status = $3
		)`
	var ok bool
	err := r.pool.QueryRow(ctx, q, userID, domain.EntitlementSubscription, domain.EntitlementActive).Scan(&ok)
	return ok, err
}

func (r *EntitlementRepository) HasActivePurchase(ctx context.Context, userID, bookID uuid.UUID) (bool, error) {
	const q = `
		SELECT EXISTS(
			SELECT 1 FROM entitlements
			WHERE user_id = $1 AND book_id = $2 AND type = $3 AND status = $4
		)`
	var ok bool
	err := r.pool.QueryRow(ctx, q, userID, bookID, domain.EntitlementSinglePurchase, domain.EntitlementActive).Scan(&ok)
	return ok, err
}

func (r *EntitlementRepository) GetActiveSubscriptionEntitlement(ctx context.Context, userID uuid.UUID) (*domain.Entitlement, error) {
	const q = `
		SELECT id, user_id, book_id, type, status, ends_at, created_at
		FROM entitlements
		WHERE user_id = $1 AND type = $2 AND status = $3
		LIMIT 1`
	row := r.pool.QueryRow(ctx, q, userID, domain.EntitlementSubscription, domain.EntitlementActive)
	e, err := scanEntitlement(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (r *EntitlementRepository) ListActivePurchasesByUser(ctx context.Context, userID uuid.UUID) ([]domain.Entitlement, error) {
	const q = `
		SELECT id, user_id, book_id, type, status, ends_at, created_at
		FROM entitlements
		WHERE user_id = $1 AND type = $2 AND status = $3 AND book_id IS NOT NULL
		ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, q, userID, domain.EntitlementSinglePurchase, domain.EntitlementActive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEntitlementRows(rows)
}

func (r *EntitlementRepository) BookExists(ctx context.Context, bookID uuid.UUID) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM books WHERE id = $1)`
	var ok bool
	err := r.pool.QueryRow(ctx, q, bookID).Scan(&ok)
	return ok, err
}

func scanEntitlement(row pgx.Row) (*domain.Entitlement, error) {
	var e domain.Entitlement
	err := row.Scan(&e.ID, &e.UserID, &e.BookID, &e.Type, &e.Status, &e.EndsAt, &e.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// Compile-time check EntitlementRepository implements EntitlementStore.
var _ EntitlementStore = (*EntitlementRepository)(nil)
