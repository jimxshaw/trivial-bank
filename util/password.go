package util

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword turns input password string into a hash.
func HashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashed), nil
}
