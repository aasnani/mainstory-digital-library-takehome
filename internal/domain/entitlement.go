package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Entitlement type/status strings mirror Flyway CHECK constraints so the app cannot invent states the DB rejects.
const (
	EntitlementSinglePurchase = "SINGLE_PURCHASE"
	EntitlementSubscription   = "SUBSCRIPTION"

	EntitlementActive    = "ACTIVE"
	EntitlementCancelled = "CANCELLED"
	EntitlementPastDue   = "PAST_DUE"

	// SubscriptionPeriodDays is the access window from renewed_at (or subscription start) until ends_at.
	SubscriptionPeriodDays = 30

	AccessReasonSubscription = "SUBSCRIPTION"
	AccessReasonPurchased    = "PURCHASED"
	AccessReasonLocked       = "LOCKED"
)

var (
	ErrInvalidEntitlementType    = errors.New("invalid entitlement type")
	ErrInvalidEntitlementStatus  = errors.New("invalid entitlement status")
	ErrInvalidEntitlementShape   = errors.New("invalid entitlement: SINGLE_PURCHASE requires book_id; SUBSCRIPTION must omit book_id")
	ErrInvalidEntitlementRequest = errors.New("invalid entitlement request")
	ErrNoActiveSubscription      = errors.New("no active subscription")
)

// Entitlement is a row in the ledger: subscription rows omit BookID; purchase rows require it (enforced in SQL + service).
type Entitlement struct {
	ID     uuid.UUID `json:"id"`
	UserID uuid.UUID `json:"user_id"`
	// BookID is nil for SUBSCRIPTION because access is all books for the user until ends_at.
	BookID *uuid.UUID `json:"book_id,omitempty"`
	Type   string     `json:"type"`
	Status string     `json:"status"`
	// EndsAt defines the paid window for subscriptions; purchases use ACTIVE without an end in MVP.
	EndsAt *time.Time `json:"ends_at,omitempty"`
	// RenewedAt anchors billing periods (mock renewals bump this and recompute ends_at).
	RenewedAt *time.Time `json:"renewed_at,omitempty"`
	// CancelledAt is set when the user opts out of renewal but should keep access until EndsAt.
	CancelledAt *time.Time `json:"cancelled_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

func ValidEntitlementType(t string) bool {
	switch t {
	case EntitlementSinglePurchase, EntitlementSubscription:
		return true
	default:
		return false
	}
}

func ValidEntitlementStatus(s string) bool {
	switch s {
	case EntitlementActive, EntitlementCancelled, EntitlementPastDue:
		return true
	default:
		return false
	}
}

// EntitlementListFilter drives staff-only GET /entitlements/staff; all fields optional and combined with AND.
type EntitlementListFilter struct {
	UserID *uuid.UUID
	BookID *uuid.UUID
	Type   string
	Status string
}

// ValidateEntitlementListFilter rejects unknown type/status strings before SQL.
func ValidateEntitlementListFilter(f EntitlementListFilter) error {
	if f.Type != "" && !ValidEntitlementType(f.Type) {
		return ErrInvalidEntitlementType
	}
	if f.Status != "" && !ValidEntitlementStatus(f.Status) {
		return ErrInvalidEntitlementStatus
	}
	return nil
}
