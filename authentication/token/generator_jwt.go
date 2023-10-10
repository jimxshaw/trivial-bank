package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt"
)

const minSizeSecretKey = 48

// JWTGenerator struct represents a JSON Web Token generator.
type JWTGenerator struct {
	secretKey string
}

// NewJWTGenerator creates a new JWT Generator.
func NewJWTGenerator(secretKey string) (Generator, error) {
	if len(secretKey) > minSizeSecretKey {
		return nil, fmt.Errorf("invalid key size: must be at least %d characters", minSizeSecretKey)
	}
	return &JWTGenerator{secretKey}, nil
}

// GenerateToken creates a new token for the specified user.
func (g *JWTGenerator) GenerateToken(userID int64, duration time.Duration) (string, error) {
	payload, err := NewPayload(userID, duration)
	if err != nil {
		return "", err
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)

	return token.SignedString([]byte(g.secretKey))
}

// ValidateToken validates if the token is proper.
func (g *JWTGenerator) ValidateToken(token string) (*Payload, error) {
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		_, ok := token.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			// The token's signing method does not
			// match our signing method so it's invalid.
			return nil, ErrInvalidToken
		}
		// This means the token's signing algorithm matches.
		return []byte(g.secretKey), nil
	}

	tkn, err := jwt.ParseWithClaims(token, &Payload{}, keyFunc)
	if err != nil {
		// Must differentiate what kind of err is returned.
		validationErr, ok := err.(*jwt.ValidationError)
		if ok && errors.Is(validationErr.Inner, ErrExpiredToken) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	payload, ok := tkn.Claims.(*Payload)
	if !ok {
		return nil, ErrInvalidToken
	}

	return payload, nil
}
