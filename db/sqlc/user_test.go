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

func TestGetUser(t *testing.T) {
	user1 := createRandomUser(t)

	query := `
		SELECT id, first_name, last_name, email, username, password, password_changed_at, created_at 
		FROM users
		WHERE id = $1 LIMIT 1
	`

	rows := sqlmock.NewRows([]string{"id", "first_name", "last_name", "email", "username", "password", "password_changed_at", "created_at"}).
		AddRow(1, user1.FirstName, user1.LastName, user1.Email, user1.Username, user1.Password, time.Now(), time.Now())

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(user1.ID).
		WillReturnRows(rows)

	user2, err := testQueries.GetUser(context.Background(), user1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, user2)

	require.Equal(t, user1.ID, user2.ID)
	require.Equal(t, user1.FirstName, user2.FirstName)
	require.Equal(t, user1.LastName, user2.LastName)
	require.Equal(t, user1.Email, user2.Email)
	require.Equal(t, user1.Username, user2.Username)
	require.Equal(t, user1.Password, user2.Password)
	require.WithinDuration(t, user1.PasswordChangedAt, user2.PasswordChangedAt, time.Second)
	require.WithinDuration(t, user1.CreatedAt, user2.CreatedAt, time.Second)
}

func createRandomUser(t *testing.T) User {
	randPwd, err := util.HashPassword(util.RandomString(10))
	require.NoError(t, err)

	params := CreateUserParams{
		FirstName: util.RandomString(10),
		LastName:  util.RandomString(10),
		Email:     util.RandomEmail(),
		Username:  util.RandomUsername(),
		Password:  randPwd,
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
