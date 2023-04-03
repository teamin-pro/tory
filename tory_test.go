package tory

import (
	"embed"
	"testing"

	"github.com/stretchr/testify/assert"
)

//go:embed test.sql
var testFiles embed.FS

func TestParse(t *testing.T) {
	n, err := LoadQueries(testFiles)
	assert.NoError(t, err)
	assert.Equal(t, 4, n)

	t.Run("remove comments", func(t *testing.T) {
		q := getQuery("test-comments")
		assert.Equal(t, "SELECT 1 + 2", q.Body())
	})

	t.Run("remove comments after semicolon", func(t *testing.T) {
		q := getQuery("test-comments-after-semicolon")
		assert.Equal(t, "SELECT 1 + 2", q.Body())
	})

	t.Run("arguments", func(t *testing.T) {
		q := getQuery("test-arguments")
		assert.Equal(t, "SELECT name FROM users WHERE id = $1 AND name ILIKE $2", q.Body())
		assert.Equal(t, []any{42, "Alice"}, q.Args(Args{"id": 42, "q": "Alice"}))
	})

	t.Run("name", func(t *testing.T) {
		getQuery("test.name-with-dots")
	})
}
