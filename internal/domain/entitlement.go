package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

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

type Entitlement struct {
	ID          uuid.UUID  `json:"id"`
	UserID      uuid.UUID  `json:"user_id"`
	BookID      *uuid.UUID `json:"book_id,omitempty"`
	Type        string     `json:"type"`
	Status      string     `json:"status"`
	EndsAt      *time.Time `json:"ends_at,omitempty"`
	RenewedAt   *time.Time `json:"renewed_at,omitempty"`
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
