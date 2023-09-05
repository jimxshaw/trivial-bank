package db

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestTransferTx(t *testing.T) {
	store := NewStore(testDB)

	account1 := createRandomAccount(t)
	account2 := createRandomAccount(t)

	amount := int64(500)

	qCreateTransfer := `
		INSERT INTO transfers (
			from_account_id,
			to_account_id,
			amount
		) VALUES (
			$1, $2, $3
		) RETURNING id, from_account_id, to_account_id, amount, created_at
	`

	pCreateTransfer := CreateTransferParams{
		FromAccountID: account1.ID,
		ToAccountID:   account2.ID,
		Amount:        amount,
	}

	qCreateEntry := `
		INSERT INTO entries (
			account_id,
			amount
		) VALUES (
			$1, $2
		) RETURNING id, account_id, amount, created_at
	`

	pCreateFromEntry := CreateEntryParams{
		AccountID: account1.ID,
		Amount:    -amount, // Negative amount for money leaving.
	}

	pCreateToEntry := CreateEntryParams{
		AccountID: account2.ID,
		Amount:    amount,
	}

	// qUpdateAccount := `
	// 	UPDATE accounts
	// 	SET balance = $2
	// 	WHERE id = $1
	// 	RETURNING id, owner, balance, currency, created_at
	// `

	// pUpdateAccount1 := UpdateAccountParams{
	// 	ID:      account1.ID,
	// 	Balance: -amount,
	// }

	// pUpdateAccount2 := UpdateAccountParams{
	// 	ID:      account2.ID,
	// 	Balance: amount,
	// }

	t.Run("Happy Path", func(t *testing.T) {
		// Mock starting a transaction.
		mock.ExpectBegin()

		// Create Transfer expectation.
		rCreateTransfer := sqlmock.NewRows([]string{"id", "from_account_id", "to_account_id", "amount", "created_at"}).
			AddRow(1, pCreateTransfer.FromAccountID, pCreateTransfer.ToAccountID, pCreateTransfer.Amount, time.Now())

		mock.ExpectQuery(regexp.QuoteMeta(qCreateTransfer)).
			WithArgs(pCreateTransfer.FromAccountID, pCreateTransfer.ToAccountID, pCreateTransfer.Amount).
			WillReturnRows(rCreateTransfer)

		// Create entries expectations.
		rCreateFromEntry := sqlmock.NewRows([]string{"id", "account_id", "amount", "created_at"}).
			AddRow(1, pCreateFromEntry.AccountID, pCreateFromEntry.Amount, time.Now())

		rCreateToEntry := sqlmock.NewRows([]string{"id", "account_id", "amount", "created_at"}).
			AddRow(1, pCreateToEntry.AccountID, pCreateToEntry.Amount, time.Now())

		mock.ExpectQuery(regexp.QuoteMeta(qCreateEntry)).
			WithArgs(pCreateFromEntry.AccountID, pCreateFromEntry.Amount).
			WillReturnRows(rCreateFromEntry)

		mock.ExpectQuery(regexp.QuoteMeta(qCreateEntry)).
			WithArgs(pCreateToEntry.AccountID, pCreateToEntry.Amount).
			WillReturnRows(rCreateToEntry)

		// Update accounts expectations.
		// rUpdateAccount1 := sqlmock.NewRows([]string{"id", "owner", "balance", "currency", "created_at"}).
		// 	AddRow(pUpdateAccount1.ID, account1.Owner, pUpdateAccount1.Balance, account1.Currency, account1.CreatedAt)

		// rUpdateAccount2 := sqlmock.NewRows([]string{"id", "owner", "balance", "currency", "created_at"}).
		// 	AddRow(pUpdateAccount2.ID, account2.Owner, pUpdateAccount2.Balance, account2.Currency, account2.CreatedAt)

		// mock.ExpectQuery(regexp.QuoteMeta(qUpdateAccount)).
		// 	WithArgs(pUpdateAccount1.ID, pUpdateAccount1.Balance).
		// 	WillReturnRows(rUpdateAccount1)

		// mock.ExpectQuery(regexp.QuoteMeta(qUpdateAccount)).
		// 	WithArgs(pUpdateAccount2.ID, pUpdateAccount2.Balance).
		// 	WillReturnRows(rUpdateAccount2)

		// Commit the transfer expectation.
		mock.ExpectCommit()

		result, err := store.TransferTx(context.Background(), TransferTxParams{
			FromAccountID: account1.ID,
			ToAccountID:   account2.ID,
			Amount:        amount,
		})
		require.NoError(t, err)

		// Check transfer.
		transfer := result.Transfer
		require.NotEmpty(t, transfer)
		require.Equal(t, account1.ID, transfer.FromAccountID)
		require.Equal(t, account2.ID, transfer.ToAccountID)
		require.Equal(t, amount, transfer.Amount)
		require.NotZero(t, transfer.ID) // auto-incremented
		require.NotZero(t, transfer.CreatedAt)

		// Check entries.
		fromEntry := result.FromEntry
		require.NotEmpty(t, fromEntry)
		require.Equal(t, account1.ID, fromEntry.AccountID)
		require.Equal(t, -amount, fromEntry.Amount)
		require.NotZero(t, fromEntry.ID)
		require.NotZero(t, fromEntry.CreatedAt)

		toEntry := result.ToEntry
		require.NotEmpty(t, fromEntry)
		require.Equal(t, account2.ID, toEntry.AccountID)
		require.Equal(t, amount, toEntry.Amount)
		require.NotZero(t, toEntry.ID)
		require.NotZero(t, toEntry.CreatedAt)

		// Check accounts balances.
		fromAccount := result.FromAccount
		require.NotEmpty(t, fromAccount)
		require.Equal(t, account1.ID, fromAccount.ID)

		toAccount := result.ToAccount
		require.NotEmpty(t, toAccount)
		require.Equal(t, account1.ID, toAccount.ID)

	})

	t.Run("Must Rollback", func(t *testing.T) {
		mock.ExpectBegin()

		rCreateTransfer := sqlmock.NewRows([]string{"id", "from_account_id", "to_account_id", "amount", "created_at"}).
			AddRow(1, pCreateTransfer.FromAccountID, pCreateTransfer.ToAccountID, pCreateTransfer.Amount, time.Now())

		mock.ExpectQuery(regexp.QuoteMeta(qCreateTransfer)).
			WithArgs(pCreateTransfer.FromAccountID, pCreateTransfer.ToAccountID, pCreateTransfer.Amount).
			WillReturnRows(rCreateTransfer)

		// Trigger some error.
		mock.ExpectQuery(regexp.QuoteMeta(qCreateEntry)).
			WithArgs(pCreateFromEntry.AccountID, pCreateFromEntry.Amount).
			WillReturnError(errors.New("some error that triggers rollback"))

		// Must rollback because of the error.
		mock.ExpectRollback()

		_, err := store.TransferTx(context.Background(), TransferTxParams{
			FromAccountID: account1.ID,
			ToAccountID:   account2.ID,
			Amount:        amount,
		})

		require.Error(t, err)
		require.Contains(t, err.Error(), "some error that triggers rollback")
	})
}
