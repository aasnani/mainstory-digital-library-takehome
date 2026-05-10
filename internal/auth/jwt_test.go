package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"mainstory-digital-library-takehome/internal/config"
	"mainstory-digital-library-takehome/internal/domain"
)

func TestSignParseRoundTrip(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: []byte("test-secret-at-least-32-bytes-long!!"),
		JWTExpiry: time.Hour,
	}
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	tok, err := Sign(cfg, id, domain.RoleMember)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := Parse(cfg, tok)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := UserID(claims)
	if err != nil || parsed != id {
		t.Fatalf("got id %v err %v", parsed, err)
	}
	if claims.Role != domain.RoleMember {
		t.Fatalf("role %q", claims.Role)
	}
}
