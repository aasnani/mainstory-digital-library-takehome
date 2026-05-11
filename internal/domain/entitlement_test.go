package domain

import "testing"

func TestValidateEntitlementListFilter(t *testing.T) {
	if err := ValidateEntitlementListFilter(EntitlementListFilter{Type: "X"}); err != ErrInvalidEntitlementType {
		t.Fatalf("got %v", err)
	}
	if err := ValidateEntitlementListFilter(EntitlementListFilter{Status: "X"}); err != ErrInvalidEntitlementStatus {
		t.Fatalf("got %v", err)
	}
	if err := ValidateEntitlementListFilter(EntitlementListFilter{
		Type: EntitlementSubscription, Status: EntitlementActive,
	}); err != nil {
		t.Fatal(err)
	}
}
