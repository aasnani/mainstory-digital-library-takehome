package service

import (
	"context"

	"github.com/google/uuid"

	"mainstory-digital-library-takehome/internal/domain"
	"mainstory-digital-library-takehome/internal/repository"
)

type BookService struct {
	books repository.BookStore
	ents  repository.EntitlementStore
}

func NewBookService(books repository.BookStore, ents repository.EntitlementStore) *BookService {
	return &BookService{books: books, ents: ents}
}

// List returns catalog rows with per-book access for MEMBER; staff roles see full catalog access flags.
func (s *BookService) List(ctx context.Context, userID uuid.UUID, role string, limit, offset int32) ([]domain.BookListItem, error) {
	rows, err := s.books.List(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]domain.BookListItem, 0, len(rows))
	for _, b := range rows {
		item := domain.BookListItem{Book: b}
		switch role {
		case domain.RoleLibrarian, domain.RoleAdmin:
			item.IsAccessible = true
			item.AccessReason = ""
		default:
			ok, reason, err := s.memberBookAccess(ctx, userID, b.ID)
			if err != nil {
				return nil, err
			}
			item.IsAccessible = ok
			item.AccessReason = reason
		}
		out = append(out, item)
	}
	return out, nil
}

// Get returns book detail; members get content only when entitled. Librarian and admin always see content.
func (s *BookService) Get(ctx context.Context, userID uuid.UUID, role string, bookID uuid.UUID) (*domain.BookListItem, error) {
	b, err := s.books.GetByID(ctx, bookID)
	if err != nil {
		return nil, err
	}
	item := &domain.BookListItem{Book: *b}
	switch role {
	case domain.RoleLibrarian, domain.RoleAdmin:
		item.IsAccessible = true
		item.AccessReason = ""
		return item, nil
	default:
		ok, reason, err := s.memberBookAccess(ctx, userID, bookID)
		if err != nil {
			return nil, err
		}
		item.IsAccessible = ok
		item.AccessReason = reason
		if !ok {
			item.Content = ""
		}
		return item, nil
	}
}

func (s *BookService) memberBookAccess(ctx context.Context, userID, bookID uuid.UUID) (bool, string, error) {
	sub, err := s.ents.HasActiveSubscription(ctx, userID)
	if err != nil {
		return false, "", err
	}
	if sub {
		return true, domain.AccessReasonSubscription, nil
	}
	pur, err := s.ents.HasActivePurchase(ctx, userID, bookID)
	if err != nil {
		return false, "", err
	}
	if pur {
		return true, domain.AccessReasonPurchased, nil
	}
	return false, domain.AccessReasonLocked, nil
}

type BookCreateInput struct {
	Title         string
	Description   string
	Author        string
	Genre         string
	IsFiction     bool
	PublishedDate interface{}
	Language      string
	PriceCents    int32
	Content       string
}

func (s *BookService) Create(ctx context.Context, in BookCreateInput) (*domain.Book, error) {
	if in.Title == "" {
		return nil, domain.ErrInvalidBook
	}
	if in.PriceCents < 0 {
		return nil, domain.ErrInvalidPrice
	}
	return s.books.Create(ctx, in.Title, in.Description, in.Author, in.Genre, in.IsFiction, in.PublishedDate, in.Language, in.PriceCents, in.Content)
}

type BookUpdateInput struct {
	Title         string
	Description   string
	Author        string
	Genre         string
	IsFiction     bool
	PublishedDate interface{}
	Language      string
	PriceCents    int32
	Content       string
}

func (s *BookService) Update(ctx context.Context, id uuid.UUID, in BookUpdateInput) (*domain.Book, error) {
	if in.Title == "" {
		return nil, domain.ErrInvalidBook
	}
	if in.PriceCents < 0 {
		return nil, domain.ErrInvalidPrice
	}
	return s.books.Update(ctx, id, in.Title, in.Description, in.Author, in.Genre, in.IsFiction, in.PublishedDate, in.Language, in.PriceCents, in.Content)
}

func (s *BookService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.books.Delete(ctx, id)
}
