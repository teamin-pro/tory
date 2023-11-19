package tory

import (
	"context"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pkg/errors"
)

type Args map[string]any

func Exec(db Tory, name string, args Args) error {
	_, err := ExecReturning(db, name, args)
	return err
}

func ExecReturning(db Tory, name string, args Args) (*pgconn.CommandTag, error) {
	query, err := db.Query(name)
	if err != nil {
		return nil, err
	}

	tag, err := db.pool.Exec(context.Background(), query.Body(), query.Args(args)...)
	if err != nil {
		return nil, errors.Wrapf(err, "exec `%s` fail", name)
	}

	return &tag, nil
}

func QueryRow(db Tory, name string, args Args, fields ...any) error {
	query, err := db.Query(name)
	if err != nil {
		return err
	}

	err = db.pool.QueryRow(context.Background(), query.Body(), query.Args(args)...).Scan(fields...)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		return errors.Wrapf(err, "query row fail on `%s`", name)
	}

	return nil
}

func Select[T any](db Tory, name string, args Args) ([]T, error) {
	query, err := db.Query(name)
	if err != nil {
		return nil, err
	}

	result := make([]T, 0)
	if err := pgxscan.Select(context.Background(), db.pool, &result, query.Body(), query.Args(args)...); err != nil {
		return nil, errors.Wrapf(err, "select fail on `%s`", name)
	}

	return result, nil
}

func Get[T any](db Tory, name string, args Args) (*T, error) {
	query, err := db.Query(name)
	if err != nil {
		return nil, err
	}

	var result T
	if err := pgxscan.Get(context.Background(), db.pool, &result, query.Body(), query.Args(args)...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		return nil, errors.Wrapf(err, "get fail on `%s`", name)
	}

	return &result, nil
}
