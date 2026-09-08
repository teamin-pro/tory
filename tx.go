package tory

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Atomic runs fn inside one transaction and commits it when fn returns no
// error; any error rolls the whole transaction back.
func (t Tory) Atomic[R any](ctx context.Context, fn func(Tx) (R, error)) (R, error) {
	return atomic(ctx, t, t.pool, fn)
}

// Atomic runs fn inside a nested transaction, which PostgreSQL implements as a
// savepoint: an error from fn rolls the transaction back to the point where the
// nested block began and leaves the outer transaction open. The rollback goes by
// time rather than by handle, so write through the Tx passed to fn and not
// through the outer one, and let fn return its error rather than swallow it: a
// nested block that ends cleanly after a failed statement cannot release its
// savepoint. A cancelled context or a lost connection is not contained by a
// savepoint and ends the outer transaction as well, so tell the errors you
// expect from the data apart from the rest instead of skipping whatever fails.
//
// Nesting costs two round trips per block, and a block that writes also takes a
// subtransaction id, of which a backend caches 64: nest per batch, not per row.
func (tx Tx) Atomic[R any](ctx context.Context, fn func(Tx) (R, error)) (R, error) {
	resp, err := atomic(ctx, tx.db, tx.pgxTx, fn)
	if err != nil {
		return resp, fmt.Errorf("nested %w", err)
	}

	return resp, nil
}

// Deprecated: use [Tory.Atomic], which needs no explicit db argument.
func Atomic[R any](ctx context.Context, db Tory, fn func(Tx) (R, error)) (R, error) {
	return db.Atomic(ctx, fn)
}

type beginner interface {
	Begin(context.Context) (pgx.Tx, error)
}

func atomic[R any](ctx context.Context, db Tory, b beginner, fn func(Tx) (R, error)) (resp R, err error) {
	err = pgx.BeginFunc(ctx, b, func(pgxTx pgx.Tx) error {
		resp, err = fn(Tx{db: db, pgxTx: pgxTx})
		return err
	})
	if err != nil {
		return resp, fmt.Errorf("atomic transaction: %w", err)
	}

	return resp, nil
}

// Tx is a transaction in progress. It carries the same query methods as [Tory],
// plus [Tx.Query] for rows a scan function turns into values itself.
type Tx struct {
	db    Tory
	pgxTx pgx.Tx
}

func (tx Tx) Exec(ctx context.Context, name string, args Args) error {
	_, err := tx.ExecReturning(ctx, name, args)
	return err
}

func (tx Tx) ExecReturning(ctx context.Context, name string, args Args) (*pgconn.CommandTag, error) {
	return execReturning(ctx, tx.db, tx.pgxTx, name, args)
}

func (tx Tx) QueryRow(ctx context.Context, name string, args Args, fields ...any) error {
	return queryRow(ctx, tx.db, tx.pgxTx, name, args, fields...)
}

// Query runs the named query and builds a slice by calling scanRow for each
// row, for results that do not map onto a struct on their own.
func (tx Tx) Query[T any](ctx context.Context, name string, args Args, scanRow func(pgx.Rows) (T, error)) ([]T, error) {
	query, err := tx.db.Query(name)
	if err != nil {
		return nil, err
	}

	rows, err := tx.pgxTx.Query(ctx, query.Body(), query.Args(args)...)
	if err != nil {
		return nil, fmt.Errorf("Query() `%s` fail: %w", name, err)
	}
	defer rows.Close()

	result := make([]T, 0)
	for rows.Next() {
		item, err := scanRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}

	return result, nil
}

func (tx Tx) Select[T any](ctx context.Context, name string, args Args) ([]T, error) {
	return selectRows[T](ctx, tx.db, tx.pgxTx, name, args)
}

func (tx Tx) Get[T any](ctx context.Context, name string, args Args) (*T, error) {
	return getRow[T](ctx, tx.db, tx.pgxTx, name, args)
}

func (tx Tx) Scalar[T any](ctx context.Context, name string, args Args) (T, error) {
	return scalar[T](ctx, tx.db, tx.pgxTx, name, args)
}
