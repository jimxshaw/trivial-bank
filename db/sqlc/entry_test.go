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

func TestCreateEntry(t *testing.T) {
	createRandomEntry(t)
}

func createRandomEntry(t *testing.T) Entry {
	account := createRandomAccount(t)

	params := CreateEntryParams{
		AccountID: account.ID,
		Amount:    util.RandomAmount(),
	}

	query := `
		INSERT INTO entries (
			account_id,
			amount
		) VALUES (
			$1, $2
		) RETURNING id, account_id, amount, created_at
	`

	rows := sqlmock.NewRows([]string{"id", "account_id", "amount", "created_at"}).
		AddRow(1, params.AccountID, params.Amount, time.Now())

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(params.AccountID, params.Amount).
		WillReturnRows(rows)

	entry, err := testQueries.CreateEntry(context.Background(), params)
	require.NoError(t, err)
	require.NotEmpty(t, entry)

	require.Equal(t, params.AccountID, entry.AccountID)
	require.Equal(t, params.Amount, entry.Amount)

	require.NotZero(t, entry.ID)
	require.NotZero(t, entry.CreatedAt)

	err = mock.ExpectationsWereMet()
	require.NoError(t, err)

	return entry
}
