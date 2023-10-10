package token

import (
	"fmt"
	"time"

	"github.com/aead/chacha20poly1305"
	"github.com/o1egl/paseto"
)

// https://github.com/o1egl/paseto

// PasetoGenerator struct represents a PASETO token generator.
type PasetoGenerator struct {
	paseto       *paseto.V2
	symmetricKey []byte
}

// NewJWTGenerator creates a new PASETO Generator.
func NewPasetoGenerator(symmetricKey string) (Generator, error) {
	if len(symmetricKey) != chacha20poly1305.KeySize {
		return nil, fmt.Errorf("invalid key size: must have exactly %d characters", chacha20poly1305.KeySize)
	}

	generator := &PasetoGenerator{
		paseto:       paseto.NewV2(),
		symmetricKey: []byte(symmetricKey),
	}

	return generator, nil
}

// GenerateToken creates a new token for the specified user.
func (g *PasetoGenerator) GenerateToken(userID int64, duration time.Duration) (string, error) {
	payload, err := NewPayload(userID, duration)
	if err != nil {
		return "", err
	}

	return g.paseto.Encrypt(g.symmetricKey, payload, nil)
}

// ValidateToken validates if the token is proper.
func (g *PasetoGenerator) ValidateToken(token string) (*Payload, error) {
	payload := &Payload{}

	err := g.paseto.Decrypt(token, g.symmetricKey, payload, nil)
	if err != nil {
		return nil, ErrInvalidToken
	}

	err = payload.Valid()
	if err != nil {
		return nil, err
	}

	return payload, nil
}
