package auth

import "testing"

func TestHashPassword_roundTrip(t *testing.T) {
	hash, err := HashPassword("secret123456")
	if err != nil {
		t.Fatal(err)
	}
	if !PasswordMatches("secret123456", hash) {
		t.Fatal("expected password to match hash")
	}
	if PasswordMatches("other-password", hash) {
		t.Fatal("wrong password should not match")
	}
}
