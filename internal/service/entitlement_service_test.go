package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"mainstory-digital-library-takehome/internal/domain"
)

func TestEntitlementService_CancelMySubscription_Success(t *testing.T) {
	_, er := newFakeCatalog()
	u := uuid.New()
	sub, err := er.Create(context.Background(), u, nil, domain.EntitlementSubscription, domain.EntitlementActive, nil)
	if err != nil {
		t.Fatal(err)
	}

	svc := NewEntitlementService(er)
	out, err := svc.CancelMySubscription(context.Background(), u)
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != sub.ID || out.Status != domain.EntitlementCancelled {
		t.Fatalf("got %+v", out)
	}
	if er.subs[u] {
		t.Fatal("expected subs map cleared after cancel")
	}
	_, err = svc.CancelMySubscription(context.Background(), u)
	if !errors.Is(err, domain.ErrNoActiveSubscription) {
		t.Fatalf("second cancel: got %v", err)
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
	_, err := er.Create(context.Background(), u, &bid, domain.EntitlementSinglePurchase, domain.EntitlementActive, nil)
	if err != nil {
		t.Fatal(err)
	}

	svc := NewEntitlementService(er)
	_, err = svc.CancelMySubscription(context.Background(), u)
	if !errors.Is(err, domain.ErrNoActiveSubscription) {
		t.Fatalf("got %v", err)
	}
}
