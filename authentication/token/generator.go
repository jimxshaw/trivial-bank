package token

import "time"

// Generator is a token management interface.
type Generator interface {
	// GenerateToken creates a new token for the specified user.
	GenerateToken(userID int64, duration time.Duration) (string, *Payload, error)

	// ValidateToken validates if the token is proper.
	ValidateToken(token string) (*Payload, error)
}
