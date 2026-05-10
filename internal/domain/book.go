package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidPrice = errors.New("invalid price")
	ErrInvalidBook  = errors.New("invalid book")
)

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
