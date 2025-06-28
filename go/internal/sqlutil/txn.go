package sqlutil

import (
	"context"
	"database/sql"
)

// Run executes fn inside a *sql.Tx.
// If fn returns an error the tx rolls back, else it commits.
func Run[T any](
	ctx context.Context,
	db *sql.DB,
	newQueries func(*sql.Tx) *T,
	fn func(q *T) error,
) error {
	tx, err := db.BeginTx(ctx, nil) // BEGIN
	if err != nil {
		return err
	}
	q := newQueries(tx) // bind sqlc Queries to this tx
	if err := fn(q); err != nil {
		_ = tx.Rollback() // ROLLBACK
		return err
	}
	return tx.Commit() // COMMIT
}
