[![Build](https://github.com/botforge-pro/tory/actions/workflows/go.yml/badge.svg)](https://github.com/botforge-pro/tory/actions/workflows/go.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/botforge-pro/tory/v2.svg)](https://pkg.go.dev/github.com/botforge-pro/tory/v2)
## Tory

Database wrapper and simple migration tool.

Inspired by [dotsql](https://github.com/qustavo/dotsql).

### Usage

**queries.sql**
```sql
-- name: get-user-by-id
SELECT id, name FROM users WHERE id = :id;

-- name: get-current-time
SELECT NOW();
```

**main.go**

```go
package main

import (
	"context"
	"embed"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/botforge-pro/tory/v2"
)

//go:embed *.sql
var sqlFiles embed.FS

func main() {
	pool, err := pgxpool.New(context.Background(), "...")
	if err != nil {
		panic(err)
	}

	t := tory.New(pool)
	
	err = t.Load(sqlFiles)
	if err != nil {
		panic(err)
	}
	
	var now time.Time
	err = t.QueryRow(context.Background(), "get-current-time", nil, &now)
	if err != nil {
		panic(err)
	}
	log.Println("now:", now)

	var user struct{
		Id   int
		Name string
	}
	err = t.QueryRow(context.Background(), "get-user-by-id", tory.Args{"id": 42}, &user.Id, &user.Name)
	if err != nil {
		panic(err)
	}
	log.Println("user:", user)
}
```

### API

**Setup.** `New(pool)` wraps a `*pgxpool.Pool`; `t.Load(fsys)` registers the named queries from the embedded `*.sql` files.

All database operations accept a `context.Context` and are methods on either `Tory` or `Tx`.

**Reading.**

- `t.Select[T](ctx, name, args)` returns `[]T`.
- `t.Get[T](ctx, name, args)` returns `*T`, and `(nil, nil)` when the query matches no row. Check for nil; a missing row is not an error.
- `t.Scalar[T](ctx, name, args)` returns a single `T` (one row, one column). Unlike `Get`, it treats
  no rows as an error.
- `t.QueryRow(ctx, name, args, &field, ...)` scans one row into the given destinations.

**Writing.**

- `t.Exec(ctx, name, args)` runs a statement and returns only `error`.
- `t.ExecReturning(ctx, name, args)` also returns the `*pgconn.CommandTag`.

**Transactions.** `t.Atomic(ctx, func(tx Tx) (R, error) { ... })` runs the function in one transaction; return an error to roll back. `Tx` has the same generic query methods as `Tory`:

```go
user, err := t.Atomic(ctx, func(tx tory.Tx) (*User, error) {
    return tx.Get[User](ctx, "get-user-by-id", tory.Args{"id": 42})
})
```

`tx.Atomic(ctx, ...)` nests: PostgreSQL runs it as a savepoint, so a failing inner block rolls back only its own work and leaves the outer transaction usable. The rollback goes by time, not by handle — it undoes everything the transaction did since the nested block began — so do the nested work through the `Tx` the nested function is handed, not the outer one.

Four things to know before building recovery on it:

- **Return the error from the nested function; do not swallow it.** After a statement fails, its subtransaction is aborted, and a nested function that ends without an error makes tory issue `release savepoint`, which then fails with `current transaction is aborted`. Only an error gets you the `rollback to savepoint` that clears the state.
- **A savepoint holds what the server reports about a statement** — a violated constraint, a bad cast — and nothing else. A dead connection or a cancelled context takes the outer transaction with it, and every later block fails too. Recognise the errors you expect from the data instead of treating every error as one of them.
- **A panic is not an error return.** It unwinds through the rollback, so the savepoint itself is undone, but it keeps going: unless it is recovered inside the outer function, it leaves the outer `Atomic` as a panic, the transaction is rolled back on the way out, and no `err` ever comes back to be checked.
- **Nesting is not free.** Each block costs two round trips, and each block that writes also takes a subtransaction id, of which a backend caches 64; a savepoint per row over a large import is slow and pushes the whole server onto `pg_subtrans` lookups. Nest per batch rather than per row.

```go
var skipped []Batch

report, err := t.Atomic(ctx, func(tx tory.Tx) (*Report, error) {
    for _, batch := range batches {
        _, err := tx.Atomic(ctx, func(tx tory.Tx) (any, error) {
            return nil, tx.Exec(ctx, "import-batch", tory.Args{"rows": batch.Rows})
        })
        switch {
        case err == nil:
        case tory.IsViolationOfCheckConstraint(err),
            tory.IsDuplicateKeyValueViolatesUniqueConstraint(err):
            skipped = append(skipped, batch) // bad data: the outer transaction is still alive
        default:
            return nil, err // may be the transaction itself, not this batch
        }
    }
    return tx.Get[Report](ctx, "build-report", nil)
})
```

The package-level `tory.Atomic(ctx, t, fn)` still works and is now a thin wrapper over `t.Atomic`; it is deprecated.

**Migrations.** `ApplyPatches(ctx, t, opts)` advances the schema through ordered `-- name:` patch blocks and records the version. On a fresh database it baselines to the latest version without running the bodies.

### Error helpers

`tory` re-exports a few `pgconn`-error tests so callers don't reach into the driver themselves. Use these after a write that may hit a constraint:

- `IsDuplicateKeyValueViolatesUniqueConstraint(err)` — SQLSTATE `23505`. Catch this after an `INSERT` / `UPDATE` that may collide with a unique index, and translate to a domain-level error your callers understand.
- `IsViolationOfCheckConstraint(err)` — SQLSTATE `23514`. Catch this when a `CHECK` constraint rejects the write.

```go
_, err := t.ExecReturning(ctx, "create-user", tory.Args{"email": email})
if tory.IsDuplicateKeyValueViolatesUniqueConstraint(err) {
    return ErrEmailTaken
}
if err != nil {
    return err
}
```

### LIKE-pattern helper

`LikeEscape(s)` escapes `%`, `_`, `.`, `*` in a user-supplied substring so it can be spliced into a `LIKE` pattern without letting the user inject wildcards.

```go
pattern := "%" + tory.LikeEscape(userInput) + "%"
rows, err := t.Select[Result](ctx, "search-by-name", tory.Args{"pattern": pattern})
```
