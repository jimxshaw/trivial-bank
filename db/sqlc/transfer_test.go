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

func TestCreateTransfer(t *testing.T) {
	createRandomTransfer(t)
}

func TestGetTransfer(t *testing.T) {
	transfer1 := createRandomTransfer(t)

	query := `
		SELECT id, from_account_id, to_account_id, amount, created_at 
		FROM transfers
		WHERE id = $1 LIMIT 1
	`

	rows := sqlmock.NewRows([]string{"id", "from_account_id", "to_account_id", "amount", "created_at"}).
		AddRow(transfer1.ID, transfer1.FromAccountID, transfer1.ToAccountID, transfer1.Amount, transfer1.CreatedAt)

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(transfer1.ID).
		WillReturnRows(rows)

	transfer2, err := testQueries.GetTransfer(context.Background(), transfer1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, transfer2)

	require.Equal(t, transfer1.ID, transfer2.ID)
	require.Equal(t, transfer1.FromAccountID, transfer2.FromAccountID)
	require.Equal(t, transfer1.ToAccountID, transfer2.ToAccountID)
	require.Equal(t, transfer1.Amount, transfer2.Amount)
	require.WithinDuration(t, transfer1.CreatedAt, transfer2.CreatedAt, time.Second)
}

func TestListTransfers(t *testing.T) {
	var expectedTransfers []Transfer
	for i := 0; i < 10; i++ {
		transfer := createRandomTransfer(t)
		expectedTransfers = append(expectedTransfers, transfer)
	}

	query := `
		SELECT id, from_account_id, to_account_id, amount, created_at 
		FROM transfers
		ORDER BY id
		LIMIT $1
		OFFSET $2
	`

	params := ListTransfersParams{
		Limit:  5,
		Offset: 5,
	}

	rows := sqlmock.NewRows([]string{"id", "from_account_id", "to_account_id", "amount", "created_at"})
	for _, transfer := range expectedTransfers[5:10] {
		rows.AddRow(transfer.ID, transfer.FromAccountID, transfer.ToAccountID, transfer.Amount, transfer.CreatedAt)
	}

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(params.Limit, params.Offset).
		WillReturnRows(rows)

	transfers, err := testQueries.ListTransfers(context.Background(), params)
	require.NoError(t, err)
	require.Len(t, transfers, 5)

	for _, transfer := range transfers {
		require.NotEmpty(t, transfer)
	}
}

func createRandomTransfer(t *testing.T) Transfer {
	fromAccount := createRandomAccount(t)
	toAccount := createRandomAccount(t)

	params := CreateTransferParams{
		FromAccountID: fromAccount.ID,
		ToAccountID:   toAccount.ID,
		Amount:        util.RandomAmount(),
	}

	query := `
		INSERT INTO transfers (
			from_account_id,
			to_account_id,
			amount
		) VALUES (
			$1, $2, $3
		) RETURNING id, from_account_id, to_account_id, amount, created_at
	`

	rows := sqlmock.NewRows([]string{"id", "from_account_id", "to_account_id", "amount", "created_at"}).
		AddRow(1, params.FromAccountID, params.ToAccountID, params.Amount, time.Now())

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(params.FromAccountID, params.ToAccountID, params.Amount).
		WillReturnRows(rows)

	transfer, err := testQueries.CreateTransfer(context.Background(), params)
	require.NoError(t, err)
	require.NotEmpty(t, transfer)

	require.Equal(t, params.FromAccountID, transfer.FromAccountID)
	require.Equal(t, params.ToAccountID, transfer.ToAccountID)
	require.Equal(t, params.Amount, transfer.Amount)

	require.NotZero(t, transfer.ID)
	require.NotZero(t, transfer.CreatedAt)

	err = mock.ExpectationsWereMet()
	require.NoError(t, err)

	return transfer
}
