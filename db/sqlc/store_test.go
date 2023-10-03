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

	account1 := Account{
		ID:        1,
		UserID:    1,
		Balance:   int64(1000),
		Currency:  "USD",
		CreatedAt: time.Now(),
	}

	account2 := Account{
		ID:        2,
		UserID:    2,
		Balance:   int64(1000),
		Currency:  "USD",
		CreatedAt: time.Now(),
	}

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

	qGetAccountForUpdate := `
		SELECT id, owner, balance, currency, created_at
		FROM accounts
		WHERE id = $1 LIMIT 1 FOR NO KEY UPDATE
	`

	qAddToAccountBalance := `
		UPDATE accounts
		SET balance = balance + $1
		WHERE id = $2
		RETURNING id, owner, balance, currency, created_at
	`

	pAddtoAccountBalance1 := AddToAccountBalanceParams{
		ID:     account1.ID,
		Amount: -amount,
	}

	pAddtoAccountBalance2 := AddToAccountBalanceParams{
		ID:     account2.ID,
		Amount: amount,
	}

	t.Run("Happy Path", func(t *testing.T) {
		// Mock starting a transaction.
		mock.ExpectBegin()

		// Get Accounts for Updates expectations.
		rFromAccount := sqlmock.NewRows([]string{"id", "user_id", "balance", "currency", "created_at"}).
			AddRow(account1.ID, account1.UserID, account1.Balance, account1.Currency, account1.CreatedAt)

		rToAccount := sqlmock.NewRows([]string{"id", "user_id", "balance", "currency", "created_at"}).
			AddRow(account2.ID, account2.UserID, account2.Balance, account2.Currency, account2.CreatedAt)

		mock.ExpectQuery(regexp.QuoteMeta(qGetAccountForUpdate)).
			WithArgs(account1.ID).
			WillReturnRows(rFromAccount)

		mock.ExpectQuery(regexp.QuoteMeta(qGetAccountForUpdate)).
			WithArgs(account2.ID).
			WillReturnRows(rToAccount)

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
		rUpdateAccount1 := sqlmock.NewRows([]string{"id", "user_id", "balance", "currency", "created_at"}).
			AddRow(account1.ID, account1.UserID, account1.Balance-amount, account1.Currency, account1.CreatedAt)

		rUpdateAccount2 := sqlmock.NewRows([]string{"id", "user_id", "balance", "currency", "created_at"}).
			AddRow(account2.ID, account2.UserID, account2.Balance+amount, account2.Currency, account2.CreatedAt)

		mock.ExpectQuery(regexp.QuoteMeta(qAddToAccountBalance)).
			WithArgs(pAddtoAccountBalance1.Amount, pAddtoAccountBalance1.ID).
			WillReturnRows(rUpdateAccount1)

		mock.ExpectQuery(regexp.QuoteMeta(qAddToAccountBalance)).
			WithArgs(pAddtoAccountBalance2.Amount, pAddtoAccountBalance2.ID).
			WillReturnRows(rUpdateAccount2)

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
		require.Equal(t, account1.Balance-amount, fromAccount.Balance)

		toAccount := result.ToAccount
		require.NotEmpty(t, toAccount)
		require.Equal(t, account2.ID, toAccount.ID)
		require.Equal(t, account2.Balance+amount, toAccount.Balance)

	})

	t.Run("Must Rollback", func(t *testing.T) {
		mock.ExpectBegin()

		rFromAccount := sqlmock.NewRows([]string{"id", "user_id", "balance", "currency", "created_at"}).
			AddRow(account1.ID, account1.UserID, account1.Balance, account1.Currency, account1.CreatedAt)

		rToAccount := sqlmock.NewRows([]string{"id", "user_id", "balance", "currency", "created_at"}).
			AddRow(account2.ID, account2.UserID, account2.Balance, account2.Currency, account2.CreatedAt)

		mock.ExpectQuery(regexp.QuoteMeta(qGetAccountForUpdate)).
			WithArgs(account1.ID).
			WillReturnRows(rFromAccount)

		mock.ExpectQuery(regexp.QuoteMeta(qGetAccountForUpdate)).
			WithArgs(account2.ID).
			WillReturnRows(rToAccount)

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
