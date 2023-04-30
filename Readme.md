[![Build](https://github.com/gamarjoba-team/tory/actions/workflows/go.yml/badge.svg)](https://github.com/gamarjoba-team/tory/actions/workflows/go.yml)
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

	"github.com/gamarjoba-team/tory"
	"github.com/jackc/pgx/v5/pgxpool"
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
