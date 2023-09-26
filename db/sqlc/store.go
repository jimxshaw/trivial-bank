package db

import (
	"context"
	"database/sql"
	"fmt"
)

// Store interface that provides functionalities for
// both db queries and transactions.
//
//go:generate mockgen -destination=../mocks/store.go -package=mocks . Store
type Store interface {
	Querier // Auto-generated by sqlc's emit interface.
	TransferTx(ctx context.Context, params TransferTxParams) (TransferTxResult, error)
}

// DBStore provides functionalities for
// both db queries and transactions.
type DBStore struct {
	*Queries
	db *sql.DB
}

func NewStore(db *sql.DB) Store {
	return &DBStore{
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
func (s *DBStore) TransferTx(ctx context.Context, params TransferTxParams) (TransferTxResult, error) {
	var result TransferTxResult

	err := s.execTx(ctx, func(q *Queries) error {
		var err error
		var fromAccount Account

		// Get the accounts and lock them.
		if fromAccount, err = q.GetAccountForUpdate(ctx, params.FromAccountID); err != nil {
			return err
		}

		if fromAccount.Balance < params.Amount {
			return fmt.Errorf("source account has insufficient funds")
		}

		if _, err = q.GetAccountForUpdate(ctx, params.ToAccountID); err != nil {
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
		// Ensures that there's a consistent order in which accounts are locked,
		// regardless of whether they are the source or the destination
		if params.FromAccountID < params.ToAccountID {
			// Lock source first, then destination.
			// Must subtract (-) amount from the source.
			result.FromAccount, result.ToAccount, err = addMoney(ctx, q, params.FromAccountID, -params.Amount, params.ToAccountID, params.Amount)
		} else {
			// Lock destination first, then source.
			// Must subtract (-) amount from the source.
			result.ToAccount, result.FromAccount, err = addMoney(ctx, q, params.ToAccountID, params.Amount, params.FromAccountID, -params.Amount)
		}

		return err
	})

	return result, err
}

// execTx executes the input function within a db transaction.
func (s *DBStore) execTx(ctx context.Context, fn func(*Queries) error) error {
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

func addMoney(
	ctx context.Context,
	q *Queries,
	accountID1 int64,
	amount1 int64,
	accountID2 int64,
	amount2 int64,
) (account1 Account, account2 Account, err error) {
	account1, err = q.AddToAccountBalance(ctx, AddToAccountBalanceParams{
		ID:     accountID1,
		Amount: amount1,
	})
	if err != nil {
		return
	}

	account2, err = q.AddToAccountBalance(ctx, AddToAccountBalanceParams{
		ID:     accountID2,
		Amount: amount2,
	})
	return
}
