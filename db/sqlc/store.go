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

// execTx executes the input function within a db transaction.
func (s *Store) execTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// Get the queries from the transaction.
	q := New(tx)

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
