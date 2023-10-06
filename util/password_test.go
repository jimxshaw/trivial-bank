package util

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword(t *testing.T) {
	pwd := RandomString(10)
	hashedPwd, err := HashPassword(pwd)
	require.NoError(t, err)
	require.NotEmpty(t, hashedPwd)

	t.Run("passwords match", func(t *testing.T) {
		err = ComparePasswords(pwd, hashedPwd)
		require.NoError(t, err)
	})

	t.Run("passwords mismatch", func(t *testing.T) {
		wrongPwd := RandomString(10)
		err = ComparePasswords(wrongPwd, hashedPwd)
		require.EqualError(t, err, bcrypt.ErrMismatchedHashAndPassword.Error())
	})
}
