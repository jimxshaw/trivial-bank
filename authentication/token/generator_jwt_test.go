package token

import (
	"testing"
	"time"

	"github.com/jimxshaw/trivial-bank/util"
	"github.com/stretchr/testify/require"
)

func TestJWTGenerator(t *testing.T) {
	generator, err := NewJWTGenerator(util.RandomString(48))
	require.NoError(t, err)

	t.Run("happy path", func(t *testing.T) {
		userID := util.RandomInt(1, 1000)
		duration := time.Minute

		issuedAt := time.Now()
		expiredAt := issuedAt.Add(duration)

		// Generate the token.
		token, err := generator.GenerateToken(userID, duration)
		require.NoError(t, err)
		require.NotEmpty(t, token)

		// Validate the token.
		payload, err := generator.ValidateToken(token)
		require.NoError(t, err)
		require.NotEmpty(t, payload)

		require.NotZero(t, payload.ID)
		require.Equal(t, userID, payload.UserID)
		require.WithinDuration(t, issuedAt, payload.IssuedAt, time.Second)
		require.WithinDuration(t, expiredAt, payload.ExpiredAt, time.Second)
	})

	t.Run("expired token", func(t *testing.T) {
		token, err := generator.GenerateToken(util.RandomInt(1, 1000), -time.Minute)
		require.NoError(t, err)
		require.NotEmpty(t, token)

		payload, err := generator.ValidateToken(token)
		require.Error(t, err)
		require.EqualError(t, err, ErrExpiredToken.Error())
		require.Nil(t, payload)
	})
}
