package service

import (
	"context"
	"unicode/utf8"

	"github.com/google/uuid"

	"mainstory-digital-library-takehome/internal/domain"
	"mainstory-digital-library-takehome/internal/repository"
)

// BookService pairs catalog reads with entitlements because "can read content?" is a joint decision in this MVP.
type BookService struct {
	books repository.BookStore
	ents  repository.EntitlementStore
}

func NewBookService(books repository.BookStore, ents repository.EntitlementStore) *BookService {
	return &BookService{books: books, ents: ents}
}

// ValidateBookListFilter returns an error if search params are too short or price range invalid.
func ValidateBookListFilter(f domain.BookListFilter) error {
	check := func(s string) error {
		if s == "" {
			return nil
		}
		if utf8.RuneCountInString(s) < domain.MinSearchRunes {
			return domain.ErrSearchTermTooShort
		}
		return nil
	}
	if err := check(f.Q); err != nil {
		return err
	}
	if err := check(f.Title); err != nil {
		return err
	}
	if err := check(f.Author); err != nil {
		return err
	}
	if f.MinPriceCents != nil && f.MaxPriceCents != nil && *f.MinPriceCents > *f.MaxPriceCents {
		return domain.ErrInvalidCatalogFilters
	}
	return nil
}

// List returns catalog rows (never includes content). Guests (nil user id, empty role) and members get entitlement-based flags; librarian/admin always accessible when JWT present.
func (s *BookService) List(ctx context.Context, userID uuid.UUID, role string, filter domain.BookListFilter, limit, offset int32) ([]domain.BookListItem, error) {
	if err := ValidateBookListFilter(filter); err != nil {
		return nil, err
	}
	rows, err := s.books.ListCatalog(ctx, filter, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]domain.BookListItem, 0, len(rows))
	for _, b := range rows {
		b.Content = ""
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

// MyLibrary returns active subscription (if any) and per-book purchase rows with catalog metadata (no content).
func (s *BookService) MyLibrary(ctx context.Context, userID uuid.UUID) (*domain.MyLibrary, error) {
	sub, err := s.ents.GetActiveSubscriptionEntitlement(ctx, userID)
	if err != nil {
		return nil, err
	}
	purchases, err := s.ents.ListActivePurchasesByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(purchases))
	for i := range purchases {
		if purchases[i].BookID != nil {
			ids = append(ids, *purchases[i].BookID)
		}
	}
	books, err := s.books.GetCatalogByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[uuid.UUID]domain.Book, len(books))
	for _, b := range books {
		b.Content = ""
		byID[b.ID] = b
	}
	rows := make([]domain.LibraryPurchaseRow, 0, len(purchases))
	for i := range purchases {
		e := purchases[i]
		if e.BookID == nil {
			continue
		}
		bk, ok := byID[*e.BookID]
		if !ok {
			continue
		}
		rows = append(rows, domain.LibraryPurchaseRow{
			Entitlement: e,
			Book: domain.BookListItem{
				Book:         bk,
				IsAccessible: true,
				AccessReason: domain.AccessReasonPurchased,
			},
		})
	}
	out := &domain.MyLibrary{Purchases: rows}
	if sub != nil {
		c := *sub
		out.Subscription = &c
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

// memberBookAccess is the single evaluation order for members: subscription wins; else per-book purchase; else locked.
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
