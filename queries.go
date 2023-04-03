package tory

import (
	"context"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
)

type Args map[string]any

func Exec(pool *pgxpool.Pool, name string, args Args) error {
	_, err := ExecReturning(pool, name, args)
	return err
}

func ExecReturning(pool *pgxpool.Pool, name string, args Args) (*pgconn.CommandTag, error) {
	conn, err := pool.Acquire(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "get connection fail")
	}
	defer conn.Release()

	query := getQuery(name)
	tag, err := conn.Exec(context.Background(), query.Body(), query.Args(args)...)
	if err != nil {
		return nil, errors.Wrapf(err, "exec `%s` fail", name)
	}

	return &tag, nil
}

func QueryRow(pool *pgxpool.Pool, name string, args Args, fields ...any) error {
	conn, err := pool.Acquire(context.Background())
	if err != nil {
		return errors.Wrap(err, "get connection fail")
	}
	defer conn.Release()

	query := getQuery(name)
	err = conn.QueryRow(context.Background(), query.Body(), query.Args(args)...).Scan(fields...)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		return errors.Wrapf(err, "query row fail on `%s`", name)
	}

	return nil
}

func Select[T any](pool *pgxpool.Pool, name string, args Args) (result []T, err error) {
	query := getQuery(name)
	if err := pgxscan.Select(context.Background(), pool, &result, query.Body(), query.Args(args)...); err != nil {
		return nil, errors.Wrapf(err, "select fail on `%s`", name)
	}
	return
}

func Get[T any](pool *pgxpool.Pool, name string, args Args) (result T, err error) {
	query := getQuery(name)
	if err := pgxscan.Get(context.Background(), pool, &result, query.Body(), query.Args(args)...); err != nil {
		return *new(T), errors.Wrapf(err, "get fail on `%s`", name)
	}
	return
}

func Atomic[R any, T any](pool *pgxpool.Pool, fn func(tx Tx[T]) (R, error)) (resp R, err error) {
	conn, err := pool.Acquire(context.Background())
	if err != nil {
		return resp, errors.Wrap(err, "get connection fail")
	}
	defer conn.Release()

	tx, err := conn.Begin(context.Background())
	if err != nil {
		return resp, errors.Wrap(err, "transaction fail")
	}

	resp, err = fn(Tx[T]{pgxTx: tx})
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
	pgxTx pgx.Tx
}

func (tx Tx[T]) Exec(name string, args Args) error {
	_, err := tx.ExecReturning(name, args)
	return err
}

func (tx Tx[T]) ExecReturning(name string, args Args) (*pgconn.CommandTag, error) {
	query := getQuery(name)
	tag, err := tx.pgxTx.Exec(context.Background(), query.Body(), query.Args(args)...)
	if err != nil {
		return nil, errors.Wrapf(err, "exec `%s` fail", name)
	}
	return &tag, nil
}

func (tx Tx[T]) QueryRow(name string, args Args, fields ...any) error {
	query := getQuery(name)
	err := tx.pgxTx.QueryRow(context.Background(), query.Body(), query.Args(args)...).Scan(fields...)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		return errors.Wrapf(err, "query row fail on `%s`", name)
	}
	return nil
}

func (tx Tx[T]) Query(name string, args Args, scanRow func(rows pgx.Rows) (T, error)) (result []T, err error) {
	query := getQuery(name)
	rows, err := tx.pgxTx.Query(context.Background(), query.Body(), query.Args(args)...)
	if err != nil {
		return result, errors.Wrapf(err, "query `%s` fail", name)
	}
	defer rows.Close()

	for rows.Next() {
		item, err := scanRow(rows)
		if err != nil {
			return result, errors.Wrap(err, "scan row fail")
		}
		result = append(result, item)
	}

	err = rows.Err()
	if err != nil {
		return result, errors.Wrap(err, "rows fail")
	}

	return
}

func (tx Tx[T]) Select(name string, args Args) (result []T, err error) {
	query := getQuery(name)
	if err := pgxscan.Select(context.Background(), tx.pgxTx, &result, query.Body(), query.Args(args)...); err != nil {
		return nil, errors.Wrapf(err, "select fail on `%s`", name)
	}
	return
}

func (tx Tx[T]) Get(name string, args Args) (result T, err error) {
	query := getQuery(name)
	if err := pgxscan.Get(context.Background(), tx.pgxTx, &result, query.Body(), query.Args(args)...); err != nil {
		return *new(T), errors.Wrapf(err, "get fail on `%s`", name)
	}
	return
}
