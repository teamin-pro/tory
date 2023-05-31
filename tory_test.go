package tory

import (
	"context"
	"embed"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed test.sql
var testFiles embed.FS

func TestParse(t *testing.T) {
	db := New(newPool(t))

	require.NoError(t, db.Load(testFiles))
	assert.Len(t, db.AllQueries(), 5)

	t.Run("remove comments", func(t *testing.T) {
		q, err := db.Query("test-comments")
		require.NoError(t, err)
		assert.Equal(t, "SELECT 1 + 2", q.Body())
	})

	t.Run("remove comments after semicolon", func(t *testing.T) {
		q, err := db.Query("test-comments-after-semicolon")
		require.NoError(t, err)
		assert.Equal(t, "SELECT 1 + 2", q.Body())
	})

	t.Run("arguments", func(t *testing.T) {
		q, err := db.Query("test-arguments")
		require.NoError(t, err)
		assert.Equal(t, "SELECT name FROM users WHERE id = $1 AND name ILIKE $2", q.Body())
		assert.Equal(t, []any{42, "Alice"}, q.Args(Args{"id": 42, "q": "Alice"}))
	})

	t.Run("name", func(t *testing.T) {
		_, err := db.Query("test.name-with-dots")
		require.NoError(t, err)
	})

	t.Run("exec", func(t *testing.T) {
		var res int
		err := QueryRow(db, "test-sum", Args{"x": 1, "y": 2}, &res)
		require.NoError(t, err)
		assert.Equal(t, 3, res)
	})
}

// BenchmarkExec-10    	   34465	     33082 ns/op	     736 B/op	       9 allocs/op
func BenchmarkExec(b *testing.B) {
	db := New(newPool(b))
	require.NoError(b, db.Load(testFiles))

	var res int

	// warmup
	err := QueryRow(db, "test-sum", Args{"x": 1, "y": 2}, &res)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := QueryRow(db, "test-sum", Args{"x": 1, "y": 2}, &res)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func newPool(t testing.TB) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), "postgres://tory:tory@localhost:5432")
	require.NoError(t, err)
	return pool
}
