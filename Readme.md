[![Build](https://github.com/teamin-pro/tory/actions/workflows/go.yml/badge.svg)](https://github.com/teamin-pro/tory/actions/workflows/go.yml)
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
	"github.com/teamin-pro/tory/v2"
)

//go:embed *.sql
var sqlFiles embed.FS

func main() {
	pool, err := pgxpool.Connect(context.Background(), "...")
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
- `t.Scalar[T](ctx, name, args)` returns a single `T` (one row, one column).
- `t.QueryRow(ctx, name, args, &field, ...)` scans one row into the given destinations.

**Writing.**

- `t.Exec(ctx, name, args)` runs a statement and returns only `error`.
- `t.ExecReturning(ctx, name, args)` also returns the `*pgconn.CommandTag`.

**Transactions.** `Atomic(ctx, t, func(tx Tx) (R, error) { ... })` runs the function in one transaction; return an error to roll back. `Tx` has the same generic query methods as `Tory`:

```go
user, err := tory.Atomic(ctx, t, func(tx tory.Tx) (*User, error) {
    return tx.Get[User](ctx, "get-user-by-id", tory.Args{"id": 42})
})
```

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
