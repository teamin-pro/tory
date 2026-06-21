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
	"github.com/teamin-pro/tory"
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
	err = tory.QueryRow(t, "get-current-time", nil, &now)
	if err != nil {
		panic(err)
	}
	log.Println("now:", now)

	var user struct{
		Id   int
		Name string
	}
	err = tory.QueryRow(t, "get-current-time", tory.Args{"id": 42}, &user.Id, &user.Name)
	if err != nil {
		panic(err)
	}
	log.Println("user:", user)
}
```

### API

**Setup.** `New(pool)` wraps a `*pgxpool.Pool`; `t.Load(fsys)` registers the named queries from the embedded `*.sql` files.

**Reading.**

- `Select[T](t, name, args)` returns `[]T`.
- `Get[T](t, name, args)` returns `*T`, and `(nil, nil)` when the query matches no row. Check for nil; a missing row is not an error.
- `Scalar[T](t, name, args)` returns a single `T` (one row, one column).
- `QueryRow(t, name, args, &field, ...)` scans one row into the given destinations.

**Writing.**

- `Exec(t, name, args)` runs a statement and returns only `error`.
- `ExecReturning(t, name, args)` also returns the `*pgconn.CommandTag`.

**Transactions.** `Atomic[R, T](t, func(tx Tx[T]) (R, error) { ... })` runs the function in one transaction; return an error to roll back. Inside, use `tx.Exec` / `tx.Get` / `tx.Select` against the same named queries.

**Migrations.** `ApplyPatches(t, opts)` advances the schema through ordered `-- name:` patch blocks and records the version. On a fresh database it baselines to the latest version without running the bodies.

### Error helpers

`tory` re-exports a few `pgconn`-error tests so callers don't reach into the driver themselves. Use these after a write that may hit a constraint:

- `IsDuplicateKeyValueViolatesUniqueConstraint(err)` — SQLSTATE `23505`. Catch this after an `INSERT` / `UPDATE` that may collide with a unique index, and translate to a domain-level error your callers understand.
- `IsViolationOfCheckConstraint(err)` — SQLSTATE `23514`. Catch this when a `CHECK` constraint rejects the write.

```go
_, err := tory.Exec(t, "create-user", tory.Args{"email": email})
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
rows, err := tory.Select[Result](t, "search-by-name", tory.Args{"pattern": pattern})
```

