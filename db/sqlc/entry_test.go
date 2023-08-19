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

func TestGetEntry(t *testing.T) {
	entry1 := createRandomEntry(t)

	query := `
		SELECT id, account_id, amount, created_at 
		FROM entries
		WHERE id = $1 LIMIT 1
	`

	rows := sqlmock.NewRows([]string{"id", "account_id", "amount", "created_at"}).
		AddRow(entry1.ID, entry1.AccountID, entry1.Amount, entry1.CreatedAt)

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(entry1.ID).
		WillReturnRows(rows)

	entry2, err := testQueries.GetEntry(context.Background(), entry1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, entry2)

	require.Equal(t, entry1.ID, entry2.ID)
	require.Equal(t, entry1.AccountID, entry2.AccountID)
	require.Equal(t, entry1.Amount, entry2.Amount)
	require.WithinDuration(t, entry1.CreatedAt, entry2.CreatedAt, time.Second)
}

func TestListEntries(t *testing.T) {
	var expectedEntries []Entry
	for i := 0; i < 10; i++ {
		entry := createRandomEntry(t)
		expectedEntries = append(expectedEntries, entry)
	}

	query := `
		SELECT id, account_id, amount, created_at 
		FROM entries
		ORDER BY id
		LIMIT $1
		OFFSET $2
	`

	params := ListEntriesParams{
		Limit:  5,
		Offset: 5,
	}

	rows := sqlmock.NewRows([]string{"id", "account_id", "amount", "created_at"})
	for _, entry := range expectedEntries[5:10] {
		rows.AddRow(entry.ID, entry.AccountID, entry.Amount, entry.CreatedAt)
	}

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(params.Limit, params.Offset).
		WillReturnRows(rows)

	entries, err := testQueries.ListEntries(context.Background(), params)
	require.NoError(t, err)
	require.Len(t, entries, 5)

	for _, entry := range entries {
		require.NotEmpty(t, entry)
	}
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
