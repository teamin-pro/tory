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

