package tory

import (
	"context"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pkg/errors"
)

func Atomic[R any, T any](db Tory, fn func(tx Tx[T]) (R, error)) (resp R, err error) {
	conn, err := db.pool.Acquire(context.Background())
	if err != nil {
		return resp, errors.Wrap(err, "get connection fail")
	}
	defer conn.Release()

	tx, err := conn.Begin(context.Background())
	if err != nil {
		return resp, errors.Wrap(err, "transaction fail")
	}

	resp, err = fn(Tx[T]{
		db:    db,
		pgxTx: tx,
	})
	if err != nil {
		_ = tx.Rollback(context.Background())
		return resp, errors.Wrapf(err, "exec fail")
	}

	err = tx.Commit(context.Background())
	if err != nil {
		return resp, errors.Wrap(err, "commit fail")
	}

	return resp, nil
}

type Tx[T any] struct {
	db    Tory
	pgxTx pgx.Tx
}

func (tx Tx[T]) Exec(name string, args Args) error {
	_, err := tx.ExecReturning(name, args)
	return err
}

func (tx Tx[T]) ExecReturning(name string, args Args) (*pgconn.CommandTag, error) {
	query, err := tx.db.Query(name)
	if err != nil {
		return nil, err
	}

	tag, err := tx.pgxTx.Exec(context.Background(), query.Body(), query.Args(args)...)
	if err != nil {
		return nil, errors.Wrapf(err, "exec `%s` fail", name)
	}

	return &tag, nil
}

func (tx Tx[T]) QueryRow(name string, args Args, fields ...any) error {
	query, err := tx.db.Query(name)
	if err != nil {
		return err
	}

	err = tx.pgxTx.QueryRow(context.Background(), query.Body(), query.Args(args)...).Scan(fields...)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return errors.Wrapf(err, "query row fail on `%s`", name)
	}

	return nil
}

func (tx Tx[T]) Query(name string, args Args, scanRow func(rows pgx.Rows) (T, error)) ([]T, error) {
	query, err := tx.db.Query(name)
	if err != nil {
		return nil, err
	}

	rows, err := tx.pgxTx.Query(context.Background(), query.Body(), query.Args(args)...)
	if err != nil {
		return nil, errors.Wrapf(err, "query `%s` fail", name)
	}
	defer rows.Close()

	result := make([]T, 0)
	for rows.Next() {
		item, err := scanRow(rows)
		if err != nil {
			return nil, errors.Wrap(err, "scan row fail")
		}
		result = append(result, item)
	}

	err = rows.Err()
	if err != nil {
		return nil, errors.Wrap(err, "rows fail")
	}

	return result, nil
}

func (tx Tx[T]) Select(name string, args Args) (result []T, err error) {
	query, err := tx.db.Query(name)
	if err != nil {
		return nil, err
	}

	if err := pgxscan.Select(context.Background(), tx.pgxTx, &result, query.Body(), query.Args(args)...); err != nil {
		return nil, errors.Wrapf(err, "select fail on `%s`", name)
	}

	return
}

func (tx Tx[T]) Get(name string, args Args) (*T, error) {
	query, err := tx.db.Query(name)
	if err != nil {
		return nil, err
	}

	var result T
	if err := pgxscan.Get(context.Background(), tx.pgxTx, &result, query.Body(), query.Args(args)...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		return nil, errors.Wrapf(err, "get fail on `%s`", name)
	}

	return &result, nil
}
