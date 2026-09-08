[![Build](https://github.com/botforge-pro/tory/actions/workflows/go.yml/badge.svg)](https://github.com/botforge-pro/tory/actions/workflows/go.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/botforge-pro/tory/v2.svg)](https://pkg.go.dev/github.com/botforge-pro/tory/v2)
## Tory

SQL lives in `.sql` files, Go calls it by name. A thin layer over
[pgx](https://github.com/jackc/pgx) for PostgreSQL: named queries, named arguments, rows scanned
into your own types, transactions that nest. Inspired by [dotsql](https://github.com/qustavo/dotsql).

**queries.sql**

```sql
-- name: get-user-by-id
SELECT id, name FROM users WHERE id = :id;

-- name: list-users-by-city
SELECT id, name FROM users WHERE city = :city ORDER BY name;
```

**main.go**

```go
//go:embed *.sql
var sqlFiles embed.FS

type User struct {
	ID   int
	Name string
}

func main() {
	pool, err := pgxpool.New(context.Background(), "postgres://...")
	if err != nil {
		panic(err)
	}

	t := tory.New(pool)
	if err := t.Load(sqlFiles); err != nil {
		panic(err)
	}

	ctx := context.Background()

	user, err := t.Get[User](ctx, "get-user-by-id", tory.Args{"id": 42})
	if err != nil {
		panic(err)
	}
	if user == nil {
		log.Println("no such user")
	}

	locals, err := t.Select[User](ctx, "list-users-by-city", tory.Args{"city": "Belgrade"})
	if err != nil {
		panic(err)
	}
	log.Println("users:", locals)
}
```

Transactions are a method, and they nest — PostgreSQL runs a nested one as a savepoint, so a batch
the server rejects costs that batch rather than the whole import:

```go
var skipped []Batch

report, err := t.Atomic(ctx, func(tx tory.Tx) (*Report, error) {
	for _, batch := range batches {
		_, err := tx.Atomic(ctx, func(tx tory.Tx) (any, error) {
			return nil, tx.Exec(ctx, "import-batch", tory.Args{"rows": batch.Rows})
		})
		if tory.IsViolationOfCheckConstraint(err) {
			skipped = append(skipped, batch) // the outer transaction is still alive
			continue
		}
		if err != nil {
			return nil, err
		}
	}
	return tx.Get[Report](ctx, "build-report", nil)
})
```

`ApplyPatches` moves the schema forward through numbered patches and remembers where it got to.

Everything else — the full method set, what a savepoint does and does not hold, the error helpers —
is in the [package documentation](https://pkg.go.dev/github.com/botforge-pro/tory/v2). Changes
between releases are in [CHANGELOG.md](CHANGELOG.md).
