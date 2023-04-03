package tory

import (
	"embed"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed patches.sql
var sqlFiles embed.FS

func init() {
	_, err := LoadQueries(sqlFiles)
	if err != nil {
		panic(err)
	}
}

type DbVersion struct {
	Version   int       `db:"version"`
	CreatedAt time.Time `db:"created_at"`
}

type Patch struct {
	Version int
	Name    string
}

func ApplyPatches(pool *pgxpool.Pool, version int, patches []Patch) (DbVersion, error) {
	sort.Slice(patches, func(i, j int) bool {
		return patches[i].Version < patches[j].Version
	})

	return Atomic(pool, func(tx Tx[DbVersion]) (DbVersion, error) {
		if err := Exec(pool, "tory.create-table-db-version", nil); err != nil {
			return DbVersion{}, err
		}

		current, err := Get[DbVersion](pool, "tory.upsert-db-version", Args{
			"version": version,
		})
		if err != nil {
			return current, err
		}

		for _, patch := range patches {
			if patch.Version <= current.Version {
				continue
			}

			if err := Exec(pool, patch.Name, nil); err != nil {
				return current, err
			}

			current.Version = patch.Version
			if err := Exec(pool, "tory.update-db-version", Args{"version": current.Version}); err != nil {
				return current, err
			}
		}

		return current, nil
	})
}
