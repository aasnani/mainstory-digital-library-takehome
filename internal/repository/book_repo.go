package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
	// ListCatalog returns catalog columns only (never loads content TEXT).
	ListCatalog(ctx context.Context, filter domain.BookListFilter, limit, offset int32) ([]domain.Book, error)
	// ListRecentCatalogTop5 returns up to five rows by added_at descending for home-page “new arrivals”; no content column.
	ListRecentCatalogTop5(ctx context.Context) ([]domain.Book, error)
	GetCatalogByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.Book, error)
	Update(ctx context.Context, id uuid.UUID, title, description, author, genre string, isFiction bool, publishedDate interface{}, language string, priceCents int32, content string) (*domain.Book, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type BookRepository struct {
	pool *pgxpool.Pool
}

func NewBookRepository(pool *pgxpool.Pool) *BookRepository {
	return &BookRepository{pool: pool}
}

// Create inserts a full book row including content (admin/librarian flows).
func (r *BookRepository) Create(ctx context.Context, title, description, author, genre string, isFiction bool, publishedDate interface{}, language string, priceCents int32, content string) (*domain.Book, error) {
	const q = `
		INSERT INTO books (title, description, author, genre, is_fiction, published_date, language, price_cents, content)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, title, description, author, genre, is_fiction, published_date, added_at, language, price_cents, content`
	row := r.pool.QueryRow(ctx, q, title, description, author, genre, isFiction, publishedDate, language, priceCents, content)
	return scanBook(row)
}

// GetByID selects all columns including content — used for entitled reads and staff preview.
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

// ListCatalog deliberately omits the content column so pagination stays cheap for large ebooks.
func (r *BookRepository) ListCatalog(ctx context.Context, filter domain.BookListFilter, limit, offset int32) ([]domain.Book, error) {
	var b strings.Builder
	b.WriteString(`SELECT id, title, description, author, genre, is_fiction, published_date, added_at, language, price_cents
		FROM books WHERE 1=1`)
	args := make([]interface{}, 0, 16)
	n := 1
	if filter.Q != "" {
		pat := "%" + filter.Q + "%"
		fmt.Fprintf(&b, ` AND (title ILIKE $%d OR author ILIKE $%d OR genre ILIKE $%d)`, n, n+1, n+2)
		args = append(args, pat, pat, pat)
		n += 3
	}
	if filter.Title != "" {
		fmt.Fprintf(&b, ` AND title ILIKE $%d`, n)
		args = append(args, "%"+filter.Title+"%")
		n++
	}
	if filter.Author != "" {
		fmt.Fprintf(&b, ` AND author ILIKE $%d`, n)
		args = append(args, "%"+filter.Author+"%")
		n++
	}
	if filter.Genre != "" {
		fmt.Fprintf(&b, ` AND genre ILIKE $%d`, n)
		args = append(args, "%"+filter.Genre+"%")
		n++
	}
	if filter.Language != "" {
		fmt.Fprintf(&b, ` AND lower(language) = lower($%d)`, n)
		args = append(args, filter.Language)
		n++
	}
	if filter.IsFiction != nil {
		fmt.Fprintf(&b, ` AND is_fiction = $%d`, n)
		args = append(args, *filter.IsFiction)
		n++
	}
	if filter.MinPriceCents != nil {
		fmt.Fprintf(&b, ` AND price_cents >= $%d`, n)
		args = append(args, *filter.MinPriceCents)
		n++
	}
	if filter.MaxPriceCents != nil {
		fmt.Fprintf(&b, ` AND price_cents <= $%d`, n)
		args = append(args, *filter.MaxPriceCents)
		n++
	}
	fmt.Fprintf(&b, ` ORDER BY added_at DESC LIMIT $%d OFFSET $%d`, n, n+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, b.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Book
	for rows.Next() {
		var bk domain.Book
		if err := rows.Scan(&bk.ID, &bk.Title, &bk.Description, &bk.Author, &bk.Genre, &bk.IsFiction, &bk.PublishedDate, &bk.AddedAt, &bk.Language, &bk.PriceCents); err != nil {
			return nil, err
		}
		out = append(out, bk)
	}
	return out, rows.Err()
}

// ListRecentCatalogTop5 orders by catalog ingestion time (added_at), newest first; capped at five rows.
func (r *BookRepository) ListRecentCatalogTop5(ctx context.Context) ([]domain.Book, error) {
	const q = `
		SELECT id, title, description, author, genre, is_fiction, published_date, added_at, language, price_cents
		FROM books
		ORDER BY added_at DESC
		LIMIT 5`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Book
	for rows.Next() {
		var bk domain.Book
		if err := rows.Scan(&bk.ID, &bk.Title, &bk.Description, &bk.Author, &bk.Genre, &bk.IsFiction, &bk.PublishedDate, &bk.AddedAt, &bk.Language, &bk.PriceCents); err != nil {
			return nil, err
		}
		out = append(out, bk)
	}
	return out, rows.Err()
}

// GetCatalogByIDs powers “my library” joins without N+1 selects for book metadata.
func (r *BookRepository) GetCatalogByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.Book, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	const q = `
		SELECT id, title, description, author, genre, is_fiction, published_date, added_at, language, price_cents
		FROM books WHERE id = ANY($1::uuid[])`
	rows, err := r.pool.Query(ctx, q, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Book
	for rows.Next() {
		var bk domain.Book
		if err := rows.Scan(&bk.ID, &bk.Title, &bk.Description, &bk.Author, &bk.Genre, &bk.IsFiction, &bk.PublishedDate, &bk.AddedAt, &bk.Language, &bk.PriceCents); err != nil {
			return nil, err
		}
		out = append(out, bk)
	}
	return out, rows.Err()
}

// Update overwrites catalog + content fields for the id; 404 when id missing.
func (r *BookRepository) Update(ctx context.Context, id uuid.UUID, title, description, author, genre string, isFiction bool, publishedDate interface{}, language string, priceCents int32, content string) (*domain.Book, error) {
	const q = `
		UPDATE books SET
			title = $2, description = $3, author = $4, genre = $5, is_fiction = $6,
			published_date = $7, language = $8, price_cents = $9, content = $10
		WHERE id = $1
		RETURNING id, title, description, author, genre, is_fiction, published_date, added_at, language, price_cents, content`
	row := r.pool.QueryRow(ctx, q, id, title, description, author, genre, isFiction, publishedDate, language, priceCents, content)
	bk, err := scanBook(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return bk, err
}

// Delete removes a book; FK from entitlements → 23503 mapped to ErrConflict.
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

// scanBook centralizes column order for all RETURNING/SELECT full-book queries.
func scanBook(row pgx.Row) (*domain.Book, error) {
	var bk domain.Book
	err := row.Scan(&bk.ID, &bk.Title, &bk.Description, &bk.Author, &bk.Genre, &bk.IsFiction, &bk.PublishedDate, &bk.AddedAt, &bk.Language, &bk.PriceCents, &bk.Content)
	if err != nil {
		return nil, err
	}
	return &bk, nil
}

var _ BookStore = (*BookRepository)(nil)
