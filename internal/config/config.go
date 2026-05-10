package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port            string
	DatabaseURL     string
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

	expiry := 24 * time.Hour
	if v := os.Getenv("JWT_EXPIRY_HOURS"); v != "" {
		h, err := strconv.Atoi(v)
		if err != nil || h < 1 {
			return nil, fmt.Errorf("JWT_EXPIRY_HOURS must be a positive integer")
		}
		expiry = time.Duration(h) * time.Hour
	}

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
