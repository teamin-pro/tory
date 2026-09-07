package tory

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func Atomic[R any](ctx context.Context, db Tory, fn func(Tx) (R, error)) (resp R, err error) {
	err = pgx.BeginFunc(ctx, db.pool, func(pgxTx pgx.Tx) error {
		resp, err = fn(Tx{db: db, pgxTx: pgxTx})
		return err
	})
	if err != nil {
		return resp, fmt.Errorf("atomic transaction: %w", err)
	}

	return resp, nil
}

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
