package tory

import (
	"context"
	"errors"
	"fmt"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Args map[string]any

type runner interface {
	pgxscan.Querier
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// Exec runs the named statement and reports only whether it failed. Use
// [Tory.ExecReturning] when the command tag matters.
func (t Tory) Exec(ctx context.Context, name string, args Args) error {
	_, err := execReturning(ctx, t, t.pool, name, args)
	return err
}

// ExecReturning runs the named statement and hands back the command tag, which
// carries the number of rows the statement touched.
func (t Tory) ExecReturning(ctx context.Context, name string, args Args) (*pgconn.CommandTag, error) {
	return execReturning(ctx, t, t.pool, name, args)
}

// QueryRow scans one row into the given destinations, in the order the query
// selects them.
func (t Tory) QueryRow(ctx context.Context, name string, args Args, fields ...any) error {
	return queryRow(ctx, t, t.pool, name, args, fields...)
}

// Select returns every row the named query produces, scanned into T. No rows
// gives an empty slice, not an error.
func (t Tory) Select[T any](ctx context.Context, name string, args Args) ([]T, error) {
	return selectRows[T](ctx, t, t.pool, name, args)
}

// Get returns the single row the named query produces, scanned into T, and
// (nil, nil) when it matches none: a missing row is not an error here, so check
// the pointer.
func (t Tory) Get[T any](ctx context.Context, name string, args Args) (*T, error) {
	return getRow[T](ctx, t, t.pool, name, args)
}

// Scalar returns one row of one column. Unlike [Tory.Get] it treats no rows as
// an error.
func (t Tory) Scalar[T any](ctx context.Context, name string, args Args) (T, error) {
	return scalar[T](ctx, t, t.pool, name, args)
}

func execReturning(ctx context.Context, db Tory, r runner, name string, args Args) (*pgconn.CommandTag, error) {
	query, err := db.Query(name)
	if err != nil {
		return nil, err
	}

	tag, err := r.Exec(ctx, query.Body(), query.Args(args)...)
	if err != nil {
		return nil, fmt.Errorf("ExecReturning() `%s` fail: %w", name, err)
	}

	return &tag, nil
}

func queryRow(ctx context.Context, db Tory, r runner, name string, args Args, fields ...any) error {
	query, err := db.Query(name)
	if err != nil {
		return err
	}

	if err := r.QueryRow(ctx, query.Body(), query.Args(args)...).Scan(fields...); err != nil {
		return fmt.Errorf("QueryRow() fail on `%s`: %w", name, err)
	}

	return nil
}

func selectRows[T any](ctx context.Context, db Tory, r runner, name string, args Args) ([]T, error) {
	query, err := db.Query(name)
	if err != nil {
		return nil, err
	}

	result := make([]T, 0)
	if err := pgxscan.Select(ctx, r, &result, query.Body(), query.Args(args)...); err != nil {
		return nil, fmt.Errorf("Select() fail on `%s`: %w", name, err)
	}

	return result, nil
}

func getRow[T any](ctx context.Context, db Tory, r runner, name string, args Args) (*T, error) {
	query, err := db.Query(name)
	if err != nil {
		return nil, err
	}

	var result T
	if err := pgxscan.Get(ctx, r, &result, query.Body(), query.Args(args)...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("Get() fail on `%s`: %w", name, err)
	}

	return &result, nil
}

func scalar[T any](ctx context.Context, db Tory, r runner, name string, args Args) (T, error) {
	query, err := db.Query(name)
	if err != nil {
		return *new(T), err
	}

	var result T
	if err := pgxscan.Get(ctx, r, &result, query.Body(), query.Args(args)...); err != nil {
		return *new(T), fmt.Errorf("Scalar() fail on `%s`: %w", name, err)
	}

	return result, nil
}
