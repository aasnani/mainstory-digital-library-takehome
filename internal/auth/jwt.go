// Package auth wraps JWT signing/parsing so services don’t import jwt library details everywhere.
package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"mainstory-digital-library-takehome/internal/config"
)

// Claims carries only what the API needs in middleware (role) plus registered claims for subject/expiry validation.
type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// Sign mints HS256 tokens: symmetric signing keeps the MVP simple (one secret) vs asymmetric JWKS.
func Sign(cfg *config.Config, userID uuid.UUID, role string) (string, error) {
	now := time.Now()
	claims := Claims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(cfg.JWTExpiry)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(cfg.JWTSecret)
}

// UserID centralizes parsing JWT sub into uuid.UUID so handlers don’t repeat parse rules.
func UserID(c *Claims) (uuid.UUID, error) {
	return uuid.Parse(c.Subject)
}

// Parse rejects wrong algorithms early to prevent "alg:none" style confusion attacks on HMAC verifiers.
func Parse(cfg *config.Config, tokenString string) (*Claims, error) {
	t, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return cfg.JWTSecret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := t.Claims.(*Claims)
	if !ok || !t.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	if _, err := UserID(claims); err != nil {
		return nil, fmt.Errorf("invalid sub")
	}
	return claims, nil
}
