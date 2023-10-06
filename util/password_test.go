package util

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword(t *testing.T) {
	pwd := RandomString(10)
	hashedPwd1, err := HashPassword(pwd)
	require.NoError(t, err)
	require.NotEmpty(t, hashedPwd1)

	t.Run("passwords match", func(t *testing.T) {
		err = ComparePasswords(pwd, hashedPwd1)
		require.NoError(t, err)
	})

	t.Run("passwords mismatch", func(t *testing.T) {
		wrongPwd := RandomString(10)
		err = ComparePasswords(wrongPwd, hashedPwd1)
		require.EqualError(t, err, bcrypt.ErrMismatchedHashAndPassword.Error())
	})

	// Hashing the same password again should produce a different hash value.
	t.Run("hashing password again creates different hash value", func(t *testing.T) {
		hashedPwd2, err := HashPassword(pwd)
		require.NoError(t, err)
		require.NotEmpty(t, hashedPwd2)
		require.NotEqual(t, hashedPwd1, hashedPwd2)
	})
}
