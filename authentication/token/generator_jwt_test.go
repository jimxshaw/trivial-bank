package token

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt"
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
		token, payload, err := generator.GenerateToken(userID, duration)
		require.NoError(t, err)
		require.NotEmpty(t, token)
		require.NotEmpty(t, payload)

		// Validate the token.
		payload, err = generator.ValidateToken(token)
		require.NoError(t, err)
		require.NotEmpty(t, payload)

		require.NotZero(t, payload.ID)
		require.Equal(t, userID, payload.UserID)
		require.WithinDuration(t, issuedAt, payload.IssuedAt, time.Second)
		require.WithinDuration(t, expiredAt, payload.ExpiredAt, time.Second)
	})

	t.Run("expired token", func(t *testing.T) {
		token, payload, err := generator.GenerateToken(util.RandomInt(1, 1000), -time.Minute)
		require.NoError(t, err)
		require.NotEmpty(t, token)
		require.NotEmpty(t, payload)

		payload, err = generator.ValidateToken(token)
		require.Error(t, err)
		require.EqualError(t, err, ErrExpiredToken.Error())
		require.Nil(t, payload)
	})

	t.Run("invalid token, signing algorithm", func(t *testing.T) {
		payload, err := NewPayload(util.RandomInt(1, 1000), time.Minute)
		require.NoError(t, err)

		token := jwt.NewWithClaims(jwt.SigningMethodNone, payload)
		tkn, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
		require.NoError(t, err)

		payload, err = generator.ValidateToken(tkn)
		require.Error(t, err)
		require.EqualError(t, err, ErrInvalidToken.Error())
		require.Nil(t, payload)
	})

	t.Run("invalid secret key", func(t *testing.T) {
		generator, err := NewJWTGenerator(util.RandomString(10))
		require.Error(t, err)
		require.Nil(t, generator)
	})
}
