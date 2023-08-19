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

func TestCreateAccount(t *testing.T) {
	createRandomAccount(t)
}

func TestGetAccount(t *testing.T) {
	account1 := createRandomAccount(t)

	query := `
SELECT id, owner, balance, currency, created_at 
FROM accounts
WHERE id = $1 LIMIT 1
`

	rows := sqlmock.NewRows([]string{"id", "owner", "balance", "currency", "created_at"}).
		AddRow(account1.ID, account1.Owner, account1.Balance, account1.Currency, account1.CreatedAt)

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(account1.ID).
		WillReturnRows(rows)

	account2, err := testQueries.GetAccount(context.Background(), account1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, account2)

	require.Equal(t, account1.ID, account2.ID)
	require.Equal(t, account1.Owner, account2.Owner)
	require.Equal(t, account1.Balance, account2.Balance)
	require.Equal(t, account1.Currency, account2.Currency)
	require.WithinDuration(t, account1.CreatedAt, account2.CreatedAt, time.Second)
}

func TestUpdateAccount(t *testing.T) {
	account1 := createRandomAccount(t)

	query := `
UPDATE accounts
SET balance = $2
WHERE id = $1
RETURNING id, owner, balance, currency, created_at
`

	params := UpdateAccountParams{
		ID:      account1.ID,
		Balance: util.RandomAmount(),
	}

	rows := sqlmock.NewRows([]string{"id", "owner", "balance", "currency", "created_at"}).
		AddRow(params.ID, account1.Owner, params.Balance, account1.Currency, account1.CreatedAt)

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(params.ID, params.Balance).
		WillReturnRows(rows)

	account2, err := testQueries.UpdateAccount(context.Background(), params)
	require.NoError(t, err)
	require.NotEmpty(t, account2)

	require.Equal(t, account1.ID, account2.ID)
	require.Equal(t, account1.Owner, account2.Owner)
	require.Equal(t, params.Balance, account2.Balance)
	require.Equal(t, account1.Currency, account2.Currency)
	require.WithinDuration(t, account1.CreatedAt, account2.CreatedAt, time.Second)
}

func createRandomAccount(t *testing.T) Account {
	params := CreateAccountParams{
		Owner:    util.RandomOwner(),
		Balance:  util.RandomAmount(),
		Currency: util.RandomCurrency(),
	}

	query := `
INSERT INTO accounts (
  owner,
  balance,
  currency
) VALUES (
  $1, $2, $3
) RETURNING id, owner, balance, currency, created_at
`

	rows := sqlmock.NewRows([]string{"id", "owner", "balance", "currency", "created_at"}).
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

	return account
}
