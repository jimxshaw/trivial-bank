package db

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestCreateAccount(t *testing.T) {
	params := CreateAccountParams{
		Owner:    "James",
		Balance:  1000,
		Currency: "USD",
	}

	query := `-- name: CreateAccount :one
INSERT INTO accounts (
  owner,
  balance,
  currency
) VALUES (
  $1, $2, $3
) RETURNING id, owner, balance, currency, created_at
`

	rows := sqlmock.NewRows([]string{"id", "owner", "balance", "current", "created_at"}).
		AddRow(1, params.Owner, params.Balance, params.Currency, time.Now())

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(params.Owner, params.Balance, params.Currency).
		WillReturnRows(rows)

	account, err := testQueries.CreateAccount(context.Background(), params)

	require.NoError(t, err)
	require.NotEmpty(t, account)

	require.Equal(t, params.Owner, account.Owner)
	require.Equal(t, params.Balance, account.Balance)
	require.Equal(t, params.Currency, account.Currency)

	require.NotZero(t, account.ID)
	require.NotZero(t, account.CreatedAt)

	err = mock.ExpectationsWereMet()
	require.NoError(t, err)
}
