package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"mainstory-digital-library-takehome/internal/domain"
)

func TestEntitlementService_CreateSubscription_SetsPeriod(t *testing.T) {
	_, er := newFakeCatalog()
	u := uuid.New()
	svc := NewEntitlementService(er)
	e, err := svc.Create(context.Background(), u, domain.RoleMember, CreateEntitlementInput{
		Type: domain.EntitlementSubscription,
	})
	if err != nil {
		t.Fatal(err)
	}
	if e.RenewedAt == nil || e.EndsAt == nil {
		t.Fatalf("expected renewed_at and ends_at: %+v", e)
	}
	if !e.EndsAt.After(*e.RenewedAt) {
		t.Fatalf("ends_at should be after renewed_at: %+v", e)
	}
	want := e.RenewedAt.AddDate(0, 0, domain.SubscriptionPeriodDays)
	if !e.EndsAt.Equal(want) {
		t.Fatalf("ends_at = %v want %v", e.EndsAt, want)
	}
	if e.Status != domain.EntitlementActive {
		t.Fatalf("status %q", e.Status)
	}
}

func TestEntitlementService_CancelMySubscription_SetsCancelledAtKeepsAccess(t *testing.T) {
	_, er := newFakeCatalog()
	u := uuid.New()
	svc := NewEntitlementService(er)
	sub, err := svc.Create(context.Background(), u, domain.RoleMember, CreateEntitlementInput{
		Type: domain.EntitlementSubscription,
	})
	if err != nil {
		t.Fatal(err)
	}

	out, err := svc.CancelMySubscription(context.Background(), u)
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != sub.ID || out.Status != domain.EntitlementActive {
		t.Fatalf("got %+v", out)
	}
	if out.CancelledAt == nil {
		t.Fatal("expected cancelled_at")
	}
	ok, err := er.HasActiveSubscription(context.Background(), u)
	if err != nil || !ok {
		t.Fatalf("subscription should stay active until ends_at: ok=%v err=%v", ok, err)
	}

	again, err := svc.CancelMySubscription(context.Background(), u)
	if err != nil {
		t.Fatal(err)
	}
	if again.CancelledAt == nil || again.ID != out.ID {
		t.Fatalf("idempotent cancel: %+v", again)
	}
}

func TestEntitlementService_CancelMySubscription_NoActiveAfterExpiry(t *testing.T) {
	_, er := newFakeCatalog()
	u := uuid.New()
	rn := time.Now().Add(-60 * 24 * time.Hour)
	en := rn.AddDate(0, 0, domain.SubscriptionPeriodDays)
	_, err := er.Create(context.Background(), u, nil, domain.EntitlementSubscription, domain.EntitlementActive, &en, &rn)
	if err != nil {
		t.Fatal(err)
	}

	svc := NewEntitlementService(er)
	_, err = svc.CancelMySubscription(context.Background(), u)
	if !errors.Is(err, domain.ErrNoActiveSubscription) {
		t.Fatalf("got %v", err)
	}
}

func TestEntitlementService_CancelMySubscription_NoActive(t *testing.T) {
	_, er := newFakeCatalog()
	u := uuid.New()
	svc := NewEntitlementService(er)
	_, err := svc.CancelMySubscription(context.Background(), u)
	if !errors.Is(err, domain.ErrNoActiveSubscription) {
		t.Fatalf("got %v", err)
	}
}

func TestEntitlementService_CancelMySubscription_IgnoresPurchaseRows(t *testing.T) {
	_, er := newFakeCatalog()
	u := uuid.New()
	bid := uuid.New()
	er.books[bid] = true
	_, err := er.Create(context.Background(), u, &bid, domain.EntitlementSinglePurchase, domain.EntitlementActive, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	svc := NewEntitlementService(er)
	_, err = svc.CancelMySubscription(context.Background(), u)
	if !errors.Is(err, domain.ErrNoActiveSubscription) {
		t.Fatalf("got %v", err)
	}
}
