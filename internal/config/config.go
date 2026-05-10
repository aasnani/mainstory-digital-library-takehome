// Package config centralizes env parsing so main and tests share one definition of required secrets and defaults.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds process-wide settings: kept as plain fields (not a map) so misuse is a compile error at call sites.
type Config struct {
	Port        string
	DatabaseURL string
	// JWTSecret is []byte because jwt and HMAC APIs expect byte keys; string would invite accidental logging.
	JWTSecret       []byte
	JWTExpiry       time.Duration
	CORSAllowOrigin string
}

func Load() (*Config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	// Default 24h matches a typical “session day” for SPAs; override in hours to keep env values human-readable integers.
	expiry := 24 * time.Hour
	if v := os.Getenv("JWT_EXPIRY_HOURS"); v != "" {
		h, err := strconv.Atoi(v)
		if err != nil || h < 1 {
			return nil, fmt.Errorf("JWT_EXPIRY_HOURS must be a positive integer")
		}
		expiry = time.Duration(h) * time.Hour
	}

	// "*" keeps local Lovable/proxy setups frictionless; production should pin a single origin to avoid credentialed wildcard pitfalls.
	cors := os.Getenv("CORS_ALLOW_ORIGIN")
	if cors == "" {
		cors = "*"
	}

	return &Config{
		Port:            port,
		DatabaseURL:     dbURL,
		JWTSecret:       []byte(secret),
		JWTExpiry:       expiry,
		CORSAllowOrigin: cors,
	}, nil
}
