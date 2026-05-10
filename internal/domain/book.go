package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidPrice          = errors.New("invalid price")
	ErrInvalidBook           = errors.New("invalid book")
	ErrSearchTermTooShort    = errors.New("search term too short")
	ErrInvalidCatalogFilters = errors.New("invalid catalog filters")
)

// MinSearchRunes is the minimum length for q/title/author when those filters are used (reduces noisy autocomplete traffic).
const MinSearchRunes = 2

// BookListFilter drives GET /books catalog queries. All string filters are optional; combine with AND.
// Q matches title, author, or genre (case-insensitive substring).
type BookListFilter struct {
	Q             string
	Title         string
	Author        string
	Genre         string
	Language      string
	IsFiction     *bool
	MinPriceCents *int32
	MaxPriceCents *int32
}

type Book struct {
	ID            uuid.UUID  `json:"id"`
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	Author        string     `json:"author"`
	Genre         string     `json:"genre"`
	IsFiction     bool       `json:"is_fiction"`
	PublishedDate *time.Time `json:"published_date,omitempty"`
	AddedAt       time.Time  `json:"added_at"`
	Language      string     `json:"language"`
	PriceCents    int32      `json:"price_cents"`
	Content       string     `json:"content,omitempty"`
}

// BookListItem is a catalog row with access hints for members.
type BookListItem struct {
	Book
	IsAccessible bool   `json:"is_accessible"`
	AccessReason string `json:"access_reason,omitempty"`
}

// MyLibrary is a single response for "my purchases / subscription" (no full book content).
type MyLibrary struct {
	Subscription *Entitlement         `json:"subscription,omitempty"`
	Purchases    []LibraryPurchaseRow `json:"purchases"`
}

// LibraryPurchaseRow ties an active purchase entitlement to catalog metadata (no content).
type LibraryPurchaseRow struct {
	Entitlement Entitlement  `json:"entitlement"`
	Book        BookListItem `json:"book"`
}
