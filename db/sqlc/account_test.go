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
		SELECT id, user_id, balance, currency, created_at 
		FROM accounts
		WHERE id = $1 LIMIT 1
	`

	rows := sqlmock.NewRows([]string{"id", "user_id", "balance", "currency", "created_at"}).
		AddRow(account1.ID, account1.UserID, account1.Balance, account1.Currency, account1.CreatedAt)

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(account1.ID).
		WillReturnRows(rows)

	account2, err := testQueries.GetAccount(context.Background(), account1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, account2)

	require.Equal(t, account1.ID, account2.ID)
	require.Equal(t, account1.UserID, account2.UserID)
	require.Equal(t, account1.Balance, account2.Balance)
	require.Equal(t, account1.Currency, account2.Currency)
	require.WithinDuration(t, account1.CreatedAt, account2.CreatedAt, time.Second)
}

func TestListAccounts(t *testing.T) {
	var expectedAccounts []Account
	for i := 0; i < 10; i++ {
		account := createRandomAccount(t)
		expectedAccounts = append(expectedAccounts, account)
	}

	query := `
		SELECT id, user_id, balance, currency, created_at 
		FROM accounts
		WHERE user_id = $1
		ORDER BY id
		LIMIT $2
		OFFSET $3
	`

	params := ListAccountsParams{
		UserID: expectedAccounts[9].UserID,
		Limit:  5,
		Offset: 0,
	}

	rows := sqlmock.NewRows([]string{"id", "user_id", "balance", "currency", "created_at"})
	rows.AddRow(expectedAccounts[9].ID, expectedAccounts[9].UserID, expectedAccounts[9].Balance, expectedAccounts[9].Currency, expectedAccounts[9].CreatedAt)

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(params.UserID, params.Limit, params.Offset).
		WillReturnRows(rows)

	accounts, err := testQueries.ListAccounts(context.Background(), params)
	require.NoError(t, err)
	require.NotEmpty(t, accounts)

	for _, account := range accounts {
		require.NotEmpty(t, account)
		require.Equal(t, expectedAccounts[9].UserID, account.UserID)
	}
}

func TestUpdateAccount(t *testing.T) {
	account1 := createRandomAccount(t)

	query := `
		UPDATE accounts
		SET user_id = $2
		WHERE id = $1
		RETURNING id, user_id, balance, currency, created_at
	`

	params := UpdateAccountParams{
		ID:     account1.ID,
		UserID: 1,
	}

	rows := sqlmock.NewRows([]string{"id", "user_id", "balance", "currency", "created_at"}).
		AddRow(params.ID, params.UserID, account1.Balance, account1.Currency, account1.CreatedAt)

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(params.ID, params.UserID).
		WillReturnRows(rows)

	account2, err := testQueries.UpdateAccount(context.Background(), params)
	require.NoError(t, err)
	require.NotEmpty(t, account2)

	require.Equal(t, account1.ID, account2.ID)
	require.Equal(t, params.UserID, account2.UserID)
	require.Equal(t, account1.Balance, account2.Balance)
	require.Equal(t, account1.Currency, account2.Currency)
	require.WithinDuration(t, account1.CreatedAt, account2.CreatedAt, time.Second)
}

func TestDeleteAccount(t *testing.T) {
	account1 := createRandomAccount(t)

	query := `
		DELETE FROM accounts
		WHERE id = $1
	`

	mock.ExpectExec(regexp.QuoteMeta(query)).
		WithArgs(account1.ID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := testQueries.DeleteAccount(context.Background(), account1.ID)
	require.NoError(t, err)
}

func createRandomAccount(t *testing.T) Account {
	params := CreateAccountParams{
		UserID:   util.RandomInt(1, 1_000_000),
		Balance:  util.RandomAmount(),
		Currency: util.RandomCurrency(),
	}

	query := `
		INSERT INTO accounts (
			user_id,
			balance,
			currency
		) VALUES (
			$1, $2, $3
		) RETURNING id, user_id, balance, currency, created_at
`

	rows := sqlmock.NewRows([]string{"id", "user_id", "balance", "currency", "created_at"}).
		AddRow(1, params.UserID, params.Balance, params.Currency, time.Now())

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(params.UserID, params.Balance, params.Currency).
		WillReturnRows(rows)

	account, err := testQueries.CreateAccount(context.Background(), params)
	require.NoError(t, err)
	require.NotEmpty(t, account)

	require.Equal(t, params.UserID, account.UserID)
	require.Equal(t, params.Balance, account.Balance)
	require.Equal(t, params.Currency, account.Currency)

	require.NotZero(t, account.ID)
	require.NotZero(t, account.CreatedAt)

	err = mock.ExpectationsWereMet()
	require.NoError(t, err)

	return account
}
