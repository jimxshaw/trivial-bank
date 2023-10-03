package util

import "golang.org/x/crypto/bcrypt"

// HashPassword turns input password string into a hash.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}
