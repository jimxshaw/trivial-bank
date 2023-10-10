package token

import "time"

// Generator is a token management interface.
type Generator interface {
	// GenerateToken creates a new token for the specified user.
	GenerateToken(userID string, duration time.Duration) (string, error)
}
