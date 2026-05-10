package auth

import (
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = bcrypt.DefaultCost

// HashPassword delegates to bcrypt so UserService never chooses a custom KDF (consistent cost, time-memory tuned).
func HashPassword(plaintext string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// PasswordMatches runs constant-time CompareHashAndPassword — never log plaintext on failure.
func PasswordMatches(plaintext, hashed string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plaintext)) == nil
}
