// Package tory runs named SQL queries against PostgreSQL. Queries live in .sql
// files next to the code, marked up with `-- name:` lines; Load parses them and
// rewrites named arguments into the positional ones pgx expects, so callers ask
// for a query by name and scan the rows straight into their own types.
package tory

import (
	"embed"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// New creates a new Tory containing the given pgxpool.Pool and queries collection
func New(pool *pgxpool.Pool) Tory {
	return Tory{
		pool:    pool,
		queries: make(map[string]ParsedQuery),
	}
}

type Tory struct {
	pool    *pgxpool.Pool
	queries map[string]ParsedQuery
}

// Load loads all queries from the given embed.FS
func (t Tory) Load(files embed.FS) error {
	dir, err := files.ReadDir(".")
	if err != nil {
		return fmt.Errorf("read SQL directory: %w", err)
	}

	for _, f := range dir {
		fileQueries, err := readQueries(files, f.Name())
		if err != nil {
			return fmt.Errorf("read SQL file %s: %w", f.Name(), err)
		}
		maps.Copy(t.queries, fileQueries)
	}

	return nil
}

// Query returns a query by name
func (t Tory) Query(name string) (ParsedQuery, error) {
	query := t.queries[name]
	if query.rawBody == "" {
		return query, fmt.Errorf("query not found: `%s`", name)
	}
	return query, nil
}

// AllQueries returns all queries in the database, sorted by name
func (t Tory) AllQueries() []ParsedQuery {
	res := make([]ParsedQuery, 0, len(t.queries))
	for _, v := range t.queries {
		res = append(res, v)
	}
	slices.SortFunc(res, func(a, b ParsedQuery) int {
		return strings.Compare(a.name, b.name)
	})
	return res
}
