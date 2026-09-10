package tory

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseFile(t *testing.T, content string) (map[string]ParsedQuery, error) {
	t.Helper()
	files := fstest.MapFS{"queries.sql": {Data: []byte(content)}}
	return readQueries(files, "queries.sql")
}

func TestReadQueries(t *testing.T) {
	t.Run("a query that never ends is an error, not a silent drop", func(t *testing.T) {
		_, err := parseFile(t, "-- name: unterminated\nSELECT 1\n")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unterminated")
	})

	t.Run("a colon inside a literal is not a variable", func(t *testing.T) {
		queries, err := parseFile(t, "-- name: literal-colon\nSELECT 'note:hello' WHERE id = :id;\n")
		require.NoError(t, err)

		q := queries["literal-colon"]
		assert.Equal(t, "SELECT 'note:hello' WHERE id = $1", q.Body())
		assert.Equal(t, []any{7}, q.Args(Args{"id": 7}))
	})

	t.Run("two dashes inside a literal do not start a comment", func(t *testing.T) {
		queries, err := parseFile(t, "-- name: literal-dashes\nSELECT 'a--b';\n")
		require.NoError(t, err)

		assert.Equal(t, "SELECT 'a--b'", queries["literal-dashes"].Body())
	})

	t.Run("a semicolon inside a literal does not end the query", func(t *testing.T) {
		queries, err := parseFile(t, "-- name: literal-semicolon\nSELECT 'a;b', 2;\n")
		require.NoError(t, err)

		assert.Equal(t, "SELECT 'a;b', 2", queries["literal-semicolon"].Body())
	})

	t.Run("an apostrophe inside a comment does not open a literal", func(t *testing.T) {
		queries, err := parseFile(t, "-- name: commented\n-- don't be fooled\nSELECT 1;\n")
		require.NoError(t, err)

		assert.Equal(t, "SELECT 1", queries["commented"].Body())
	})

	t.Run("a doubled quote stays inside the literal", func(t *testing.T) {
		queries, err := parseFile(t, "-- name: escaped-quote\nSELECT 'it''s:fine';\n")
		require.NoError(t, err)

		q := queries["escaped-quote"]
		assert.Equal(t, "SELECT 'it''s:fine'", q.Body())
		assert.Empty(t, q.Args(nil))
	})

	t.Run("a cast is not a variable", func(t *testing.T) {
		queries, err := parseFile(t, "-- name: cast\nSELECT :x::int + 1;\n")
		require.NoError(t, err)

		q := queries["cast"]
		assert.Equal(t, "SELECT $1::int + 1", q.Body())
		assert.Equal(t, []any{3}, q.Args(Args{"x": 3}))
	})

	t.Run("a repeated variable binds once", func(t *testing.T) {
		queries, err := parseFile(t, "-- name: repeated\nSELECT :a, :b, :a;\n")
		require.NoError(t, err)

		q := queries["repeated"]
		assert.Equal(t, "SELECT $1, $2, $1", q.Body())
		assert.Equal(t, []any{1, 2}, q.Args(Args{"a": 1, "b": 2}))
	})

	t.Run("a name line inside a dollar block does not start a query", func(t *testing.T) {
		queries, err := parseFile(t, "-- name: outer\ndo $$ begin\n-- name: inner\nperform 1;\nend $$;\n")
		require.NoError(t, err)

		assert.Len(t, queries, 1)
		assert.Equal(t, "do $$ begin -- name: inner perform 1; end $$", queries["outer"].Body())
	})

	t.Run("a second statement under one name is an error, not a silent drop", func(t *testing.T) {
		_, err := parseFile(t, "-- name: three-alters\n"+
			"alter table users add column a text;\n"+
			"alter table users add column b text;\n"+
			"alter table users add column c text;\n")

		require.Error(t, err, "the statements after the first were dropped without a word")
		assert.Contains(t, err.Error(), "three-alters")
		assert.Contains(t, err.Error(), "3")
	})

	t.Run("the refusal does not wait for the block to take arguments", func(t *testing.T) {
		_, err := parseFile(t, "-- name: two-with-a-var\n"+
			"update users set a = :v;\n"+
			"update users set b = :v;\n")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "two-with-a-var")
	})

	t.Run("a statement ending inside a dollar block is one statement", func(t *testing.T) {
		queries, err := parseFile(t, "-- name: guarded\n"+
			"do $$ begin\nalter table t add column a text;\nalter table t add column b text;\nend $$;\n")
		require.NoError(t, err)

		assert.Equal(t,
			"do $$ begin alter table t add column a text; alter table t add column b text; end $$",
			queries["guarded"].Body())
	})

	t.Run("a trailing semicolon is not a statement of its own", func(t *testing.T) {
		queries, err := parseFile(t, "-- name: trailing\nSELECT 1;\n\n")
		require.NoError(t, err)

		assert.Equal(t, "SELECT 1", queries["trailing"].Body())
	})

	t.Run("a comment after the last statement is not an unterminated one", func(t *testing.T) {
		queries, err := parseFile(t, "-- name: commented-tail\nSELECT 1;\n-- a word about the next one\n"+
			"-- name: after\nSELECT 2;\n")
		require.NoError(t, err)

		assert.Equal(t, "SELECT 1", queries["commented-tail"].Body())
		assert.Equal(t, "SELECT 2", queries["after"].Body())
	})

	t.Run("code left hanging after the last statement is an error", func(t *testing.T) {
		_, err := parseFile(t, "-- name: hanging\nSELECT 1;\nSELECT 2\n-- name: after\nSELECT 3;\n")

		require.Error(t, err, "a statement with no terminator was dropped rather than reported")
		assert.Contains(t, err.Error(), "hanging")
	})

	t.Run("a prefix name is not eaten by a shorter one", func(t *testing.T) {
		queries, err := parseFile(t, "-- name: prefixes\nSELECT :id, :idx;\n")
		require.NoError(t, err)

		q := queries["prefixes"]
		assert.Equal(t, "SELECT $1, $2", q.Body())
		assert.Equal(t, []any{1, 2}, q.Args(Args{"id": 1, "idx": 2}))
	})
}
