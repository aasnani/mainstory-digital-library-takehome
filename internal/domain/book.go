// Package domain holds types and validation errors shared by HTTP, services, and repositories without import cycles.
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
	Q        string
	Title    string
	Author   string
	Genre    string
	Language string
	// IsFiction is *bool so SQL can distinguish “no filter” from “false only” (three-valued logic for queries).
	IsFiction *bool
	// Price bounds use *int32 to mirror optional query params and keep JSON/SQL numeric width consistent with price_cents.
	MinPriceCents *int32
	MaxPriceCents *int32
}

// Book is the persistence + JSON shape for catalog and full text; list endpoints never load Content from SQL to avoid huge rows.
type Book struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Author      string    `json:"author"`
	Genre       string    `json:"genre"`
	IsFiction   bool      `json:"is_fiction"`
	// PublishedDate is optional: many MVP rows won’t have a real on-sale date; pointer omits JSON when unset.
	PublishedDate *time.Time `json:"published_date,omitempty"`
	AddedAt       time.Time  `json:"added_at"`
	Language      string     `json:"language"`
	// PriceCents is int32 to match Postgres INT and keep money as integer cents (avoid float rounding in a storefront).
	PriceCents int32  `json:"price_cents"`
	Content    string `json:"content,omitempty"`
}

// BookListItem embeds Book so JSON stays flat for clients while adding entitlement UX fields the DB does not store.
type BookListItem struct {
	Book
	// IsAccessible tells SPAs whether to show read buttons without a second round-trip.
	IsAccessible bool `json:"is_accessible"`
	// AccessReason explains why (subscription vs purchase vs locked) for badges and upsell copy.
	AccessReason string `json:"access_reason,omitempty"`
}

// MyLibrary batches “account state” for a library page without streaming full book blobs.
type MyLibrary struct {
	Subscription *Entitlement         `json:"subscription,omitempty"`
	Purchases    []LibraryPurchaseRow `json:"purchases"`
}

// LibraryPurchaseRow ties an active purchase entitlement to catalog metadata (no content).
type LibraryPurchaseRow struct {
	Entitlement Entitlement  `json:"entitlement"`
	Book        BookListItem `json:"book"`
}
