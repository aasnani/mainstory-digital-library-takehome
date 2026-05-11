package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"mainstory-digital-library-takehome/internal/domain"
	"mainstory-digital-library-takehome/internal/repository"
)

// EntitlementService owns mock checkout rules: creating rows is how subscription/purchase grants access without a payments vendor.
type EntitlementService struct {
	ents repository.EntitlementStore
}

// NewEntitlementService wires the entitlement store used by HTTP handlers.
func NewEntitlementService(ents repository.EntitlementStore) *EntitlementService {
	return &EntitlementService{ents: ents}
}

// List: MEMBER sees own; LIBRARIAN and ADMIN see all (unfiltered slice — use ListStaff for filter browse).
func (s *EntitlementService) List(ctx context.Context, actorID uuid.UUID, role string, limit, offset int32) ([]domain.Entitlement, error) {
	switch role {
	case domain.RoleLibrarian, domain.RoleAdmin:
		return s.ents.ListAll(ctx, limit, offset)
	default:
		return s.ents.ListByUser(ctx, actorID, limit, offset)
	}
}

// ListStaff returns global entitlements with optional filters; caller must enforce librarian/admin only.
func (s *EntitlementService) ListStaff(ctx context.Context, filter domain.EntitlementListFilter, limit, offset int32) ([]domain.Entitlement, error) {
	if err := domain.ValidateEntitlementListFilter(filter); err != nil {
		return nil, err
	}
	return s.ents.ListAllFiltered(ctx, filter, limit, offset)
}

// Get: MEMBER only if own; librarian and admin any.
func (s *EntitlementService) Get(ctx context.Context, actorID uuid.UUID, role string, id uuid.UUID) (*domain.Entitlement, error) {
	e, err := s.ents.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	// WHY: subscription rows can become stale until lazy expiry runs — refresh so GET reflects current status.
	if e.Type == domain.EntitlementSubscription {
		if err := s.ents.ExpireStaleSubscriptionsForUser(ctx, e.UserID); err != nil {
			return nil, err
		}
		e, err = s.ents.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
	}
	switch role {
	case domain.RoleLibrarian, domain.RoleAdmin:
		return e, nil
	default:
		if e.UserID != actorID {
			return nil, domain.ErrForbidden
		}
		return e, nil
	}
}

// CreateEntitlementInput keeps admin “grant to user X” and member “grant to self” in one struct guarded by role checks.
type CreateEntitlementInput struct {
	TargetUserID *uuid.UUID // admin only: whose entitlement to create
	Type         string
	BookID       *uuid.UUID
	Status       string // optional; default ACTIVE
}

// Create inserts a subscription or purchase row after validating shape, book existence, and actor permissions.
func (s *EntitlementService) Create(ctx context.Context, actorID uuid.UUID, role string, in CreateEntitlementInput) (*domain.Entitlement, error) {
	// Librarians read entitlements but cannot mint access (product decision: staff curate catalog, don’t grant paid access).
	if role == domain.RoleLibrarian {
		return nil, domain.ErrForbidden
	}
	if !domain.ValidEntitlementType(in.Type) {
		return nil, domain.ErrInvalidEntitlementType
	}
	status := in.Status
	if status == "" {
		status = domain.EntitlementActive
	}
	if !domain.ValidEntitlementStatus(status) {
		return nil, domain.ErrInvalidEntitlementStatus
	}
	var target uuid.UUID
	switch role {
	case domain.RoleAdmin:
		if in.TargetUserID == nil {
			return nil, domain.ErrInvalidEntitlementRequest
		}
		target = *in.TargetUserID
	default:
		if in.TargetUserID != nil && *in.TargetUserID != actorID {
			return nil, domain.ErrForbidden
		}
		target = actorID
	}

	switch in.Type {
	case domain.EntitlementSubscription:
		if in.BookID != nil {
			return nil, domain.ErrInvalidEntitlementShape
		}
	case domain.EntitlementSinglePurchase:
		if in.BookID == nil {
			return nil, domain.ErrInvalidEntitlementShape
		}
		ok, err := s.ents.BookExists(ctx, *in.BookID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, domain.ErrNotFound
		}
	}

	if err := s.ents.ExpireStaleSubscriptionsForUser(ctx, target); err != nil {
		return nil, err
	}

	var endsAt *time.Time
	var renewedAt *time.Time
	if in.Type == domain.EntitlementSubscription {
		// WHAT: mock renewal anchor is “now”; ends_at is renewed_at + SubscriptionPeriodDays (see domain constant).
		t := time.Now().UTC()
		renewedAt = &t
		end := t.AddDate(0, 0, domain.SubscriptionPeriodDays)
		endsAt = &end
	}

	return s.ents.Create(ctx, target, in.BookID, in.Type, status, endsAt, renewedAt)
}

// Patch updates status/ends_at for admin support flows (handler enforces ADMIN middleware).
func (s *EntitlementService) Patch(ctx context.Context, id uuid.UUID, status *string, endsAt *time.Time) (*domain.Entitlement, error) {
	if status != nil && !domain.ValidEntitlementStatus(*status) {
		return nil, domain.ErrInvalidEntitlementStatus
	}
	return s.ents.Update(ctx, id, status, endsAt)
}

// CancelMySubscription records cancellation at period end: access stays until ends_at (from renewed_at + SubscriptionPeriodDays). Idempotent if already requested.
func (s *EntitlementService) CancelMySubscription(ctx context.Context, userID uuid.UUID) (*domain.Entitlement, error) {
	e, err := s.ents.GetActiveSubscriptionEntitlement(ctx, userID)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, domain.ErrNoActiveSubscription
	}
	if e.CancelledAt != nil {
		return e, nil
	}
	return s.ents.SetSubscriptionCancelledAt(ctx, e.ID, time.Now().UTC())
}
