package token

import (
	"time"

	"github.com/google/uuid"
)

// Payload is the token's payload data.
type Payload struct {
	ID        uuid.UUID `json:"id"`
	UserID    string    `json:"user_id"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiredAt time.Time `json:"expired_at"`
}
