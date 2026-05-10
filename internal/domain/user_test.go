package domain

import "testing"

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		in    string
		valid bool
	}{
		{"", false},
		{"a", false},
		{"user@example.com", true},
		{"  User@Example.Com  ", true},
	}
	for _, tt := range tests {
		err := ValidateEmail(tt.in)
		if tt.valid && err != nil {
			t.Errorf("%q: want valid, got %v", tt.in, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("%q: want invalid", tt.in)
		}
	}
}

func TestNormalizeEmail(t *testing.T) {
	got := NormalizeEmail("  Foo@BAR.com ")
	want := "foo@bar.com"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestValidRole(t *testing.T) {
	if !ValidRole(RoleMember) || !ValidRole(RoleAdmin) {
		t.Fatal("known roles should be valid")
	}
	if ValidRole("nope") {
		t.Fatal("invalid role")
	}
}
