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

// BookStore is implemented by BookRepository.
type BookStore interface {
	Create(ctx context.Context, title, description, author, genre string, isFiction bool, publishedDate interface{}, language string, priceCents int32, content string) (*domain.Book, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Book, error)
	List(ctx context.Context, limit, offset int32) ([]domain.Book, error)
	Update(ctx context.Context, id uuid.UUID, title, description, author, genre string, isFiction bool, publishedDate interface{}, language string, priceCents int32, content string) (*domain.Book, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type BookRepository struct {
	pool *pgxpool.Pool
}

func NewBookRepository(pool *pgxpool.Pool) *BookRepository {
	return &BookRepository{pool: pool}
}

func (r *BookRepository) Create(ctx context.Context, title, description, author, genre string, isFiction bool, publishedDate interface{}, language string, priceCents int32, content string) (*domain.Book, error) {
	const q = `
		INSERT INTO books (title, description, author, genre, is_fiction, published_date, language, price_cents, content)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, title, description, author, genre, is_fiction, published_date, added_at, language, price_cents, content`
	row := r.pool.QueryRow(ctx, q, title, description, author, genre, isFiction, publishedDate, language, priceCents, content)
	return scanBook(row)
}

func (r *BookRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Book, error) {
	const q = `
		SELECT id, title, description, author, genre, is_fiction, published_date, added_at, language, price_cents, content
		FROM books WHERE id = $1`
	row := r.pool.QueryRow(ctx, q, id)
	b, err := scanBook(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return b, err
}

func (r *BookRepository) List(ctx context.Context, limit, offset int32) ([]domain.Book, error) {
	const q = `
		SELECT id, title, description, author, genre, is_fiction, published_date, added_at, language, price_cents, ''
		FROM books
		ORDER BY added_at DESC
		LIMIT $1 OFFSET $2`
	rows, err := r.pool.Query(ctx, q, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Book
	for rows.Next() {
		var b domain.Book
		if err := rows.Scan(&b.ID, &b.Title, &b.Description, &b.Author, &b.Genre, &b.IsFiction, &b.PublishedDate, &b.AddedAt, &b.Language, &b.PriceCents, &b.Content); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *BookRepository) Update(ctx context.Context, id uuid.UUID, title, description, author, genre string, isFiction bool, publishedDate interface{}, language string, priceCents int32, content string) (*domain.Book, error) {
	const q = `
		UPDATE books SET
			title = $2, description = $3, author = $4, genre = $5, is_fiction = $6,
			published_date = $7, language = $8, price_cents = $9, content = $10
		WHERE id = $1
		RETURNING id, title, description, author, genre, is_fiction, published_date, added_at, language, price_cents, content`
	row := r.pool.QueryRow(ctx, q, id, title, description, author, genre, isFiction, publishedDate, language, priceCents, content)
	b, err := scanBook(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return b, err
}

func (r *BookRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM books WHERE id = $1`, id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return domain.ErrConflict
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func scanBook(row pgx.Row) (*domain.Book, error) {
	var b domain.Book
	err := row.Scan(&b.ID, &b.Title, &b.Description, &b.Author, &b.Genre, &b.IsFiction, &b.PublishedDate, &b.AddedAt, &b.Language, &b.PriceCents, &b.Content)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

var _ BookStore = (*BookRepository)(nil)
