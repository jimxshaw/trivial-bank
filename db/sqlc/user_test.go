package db

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jimxshaw/trivial-bank/util"
	"github.com/stretchr/testify/require"
)

func TestCreateUser(t *testing.T) {
	createRandomUser(t)
}



func createRandomUser(t *testing.T) User {
	params := CreateUserParams{
		FirstName: util.RandomString(10),
		LastName:  util.RandomString(10),
		Email:     util.RandomEmail(),
		Username:  util.RandomUsername(),
		Password:  util.RandomPassword(),
	}

	query := `
		INSERT INTO users (
			first_name,
			last_name,
			email,
			username,
			password
		) VALUES (
			$1, $2, $3, $4, $5
		) RETURNING id, first_name, last_name, email, username, password, password_changed_at, created_at
`

	rows := sqlmock.NewRows([]string{"id", "first_name", "last_name", "email", "username", "password", "password_changed_at", "created_at"}).
		AddRow(1, params.FirstName, params.LastName, params.Email, params.Username, params.Password, time.Now(), time.Now())

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(params.FirstName, params.LastName, params.Email, params.Username, params.Password).
		WillReturnRows(rows)

	user, err := testQueries.CreateUser(context.Background(), params)
	require.NoError(t, err)
	require.NotEmpty(t, user)

	require.Equal(t, params.FirstName, user.FirstName)
	require.Equal(t, params.LastName, user.LastName)
	require.Equal(t, params.Email, user.Email)
	require.Equal(t, params.Username, user.Username)
	require.Equal(t, params.Password, user.Password)

	require.NotZero(t, user.ID)
	require.NotZero(t, user.CreatedAt)

	err = mock.ExpectationsWereMet()
	require.NoError(t, err)

	return user
}
