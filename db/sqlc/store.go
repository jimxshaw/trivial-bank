package db

import (
	"context"
	"database/sql"
	"fmt"
)

// Store provides functionalities for
// both db queries and transactions.
type Store struct {
	*Queries
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{
		db:      db,
		Queries: New(db),
	}
}

// TransferTxParams has parameters for a transfer transaction.
type TransferTxParams struct {
	FromAccountID int64 `json:"from_account_id"`
	ToAccountID   int64 `json:"to_account_id"`
	Amount        int64 `json:"amount"`
}

// TransferTxResult is the result struct for a transfer transaction.
type TransferTxResult struct {
	Transfer    Transfer `json:"transfer"`
	FromAccount Account  `json:"from_account"`
	ToAccount   Account  `json:"to_account"`
	FromEntry   Entry    `json:"from_entry"`
	ToEntry     Entry    `json:"to_entry"`
}

// TransferTx executes a money transfer from one account to another.
// It updates accounts balances, creates a transfer record and entry records
// in a single db transaction.
func (s *Store) TransferTx(ctx context.Context, params TransferTxParams) (TransferTxResult, error) {
	var result TransferTxResult

	err := s.execTx(ctx, func(q *Queries) error {
		var err error

		// Get the accounts and lock them.
		if result.FromAccount, err = q.GetAccountForUpdate(ctx, params.FromAccountID); err != nil {
			return err
		}

		if result.FromAccount.Balance < params.Amount {
			return fmt.Errorf("source account has insufficient funds")
		}

		if result.ToAccount, err = q.GetAccountForUpdate(ctx, params.ToAccountID); err != nil {
			return err
		}

		// Create transfer and entry records as an audit trail.
		if result.Transfer, err = q.CreateTransfer(ctx, CreateTransferParams(params)); err != nil {
			return err
		}

		result.FromEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: params.FromAccountID,
			Amount:    -params.Amount,
		})
		if err != nil {
			return err
		}

		result.ToEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: params.ToAccountID,
			Amount:    params.Amount,
		})
		if err != nil {
			return err
		}

		// Update the accounts balances.
		_, err = q.AddToAccountBalance(ctx, AddToAccountBalanceParams{
			ID:     params.FromAccountID,
			Amount: -params.Amount, // Must subtract (-) from the source.
		})
		if err != nil {
			return err
		}

		_, err = q.AddToAccountBalance(ctx, AddToAccountBalanceParams{
			ID:     params.ToAccountID,
			Amount: params.Amount, // Must add (+) to the destination.
		})
		if err != nil {
			return err
		}

		return nil
	})

	return result, err
}

// execTx executes the input function within a db transaction.
func (s *Store) execTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// Get the queries from the transaction.
	q := s.Queries.WithTx(tx)

	err = fn(q)
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("transaction error: %w, rollback error: %w", err, rollbackErr)
		}

		// If rollback is successful then just return the err.
		return err
	}

	// If all transaction operations are successful then commit.
	// If the commit has an error then return it to the caller.
	return tx.Commit()
}
