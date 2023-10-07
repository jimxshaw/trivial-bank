package util

import (
	"fmt"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

const MinPasswordLength = 8

var PasswordValidationMessage = fmt.Sprintf("Password should be at least %d characters with at least 1 uppercase, 1 lowercase and 1 special character.", MinPasswordLength)

// ValidatePassword checks if password is valid according to the requirements.
func IsValidPassword(password string) bool {
	var (
		hasMinLen  = false
		hasUpper   = false
		hasLower   = false
		hasNumber  = false
		hasSpecial = false
	)

	if len(password) >= MinPasswordLength {
		hasMinLen = true
	}

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}
	return hasMinLen && hasUpper && hasLower && hasNumber && hasSpecial
}

// HashPassword turns input password string into a hash.
func HashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashed), nil
}

// ComparePasswords checks if the input password and hashed password are the same or not.
func ComparePasswords(password string, hashedPassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
