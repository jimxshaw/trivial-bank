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
