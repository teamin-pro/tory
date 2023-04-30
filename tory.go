package tory

import (
	"embed"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
)

// NewDB creates a new DB containing the given pgxpool.Pool and queries collection
func NewDB(pool *pgxpool.Pool) DB {
	return DB{
		pool:    pool,
		queries: make(map[string]ParsedQuery),
	}
}

type DB struct {
	pool    *pgxpool.Pool
	queries map[string]ParsedQuery
}

// Load loads all queries from the given embed.FS
func (db DB) Load(files embed.FS) error {
	dir, err := files.ReadDir(".")
	if err != nil {
		return errors.Wrapf(err, "read sql dir fail")
	}

	for _, f := range dir {
		fileQueries, err := readQueries(files, f.Name())
		if err != nil {
			return errors.Wrapf(err, "read sql file %s fail:", f.Name())
		}
		for k, v := range fileQueries {
			db.queries[k] = v
		}
	}

	return nil
}

// Query returns a query by name
func (db DB) Query(name string) (ParsedQuery, error) {
	query := db.queries[name]
	if query.rawBody == "" {
		return query, errors.Errorf("query not found: `%s`", name)
	}
	return query, nil
}

// AllQueries returns all queries in the database, sorted by name
func (db DB) AllQueries() []ParsedQuery {
	res := make([]ParsedQuery, 0, len(db.queries))
	for _, v := range db.queries {
		res = append(res, v)
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].name < res[j].name
	})
	return res
}
