package config

import (
	"testing"
)

func TestLoad_MissingDatabaseURL(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-for-config-load-validation-long-enough")
	t.Setenv("DATABASE_URL", "")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error when DATABASE_URL empty")
	}
}

func TestLoad_MissingJWTSecret(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://localhost/dummy")
	t.Setenv("JWT_SECRET", "")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error when JWT_SECRET empty")
	}
}

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://localhost/dummy")
	t.Setenv("JWT_SECRET", "test-secret-for-config-load-validation-long-enough")
	t.Setenv("PORT", "")
	t.Setenv("JWT_EXPIRY_HOURS", "")
	t.Setenv("CORS_ALLOW_ORIGIN", "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != "8080" {
		t.Fatalf("port %q", cfg.Port)
	}
	if cfg.CORSAllowOrigin != "*" {
		t.Fatalf("cors %q", cfg.CORSAllowOrigin)
	}
}
