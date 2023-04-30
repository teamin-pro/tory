package tory

import (
	"embed"
	"sort"
	"time"
)

//go:embed patches.sql
var sqlFiles embed.FS

type DbVersion struct {
	Version   int       `db:"version"`
	CreatedAt time.Time `db:"created_at"`
}

type Patch struct {
	Version int
	Name    string
}

type ApplyPatchesOptions struct {
	OnSkip   func(Patch)
	OnStart  func(Patch)
	OnFinish func(Patch)
}

func ApplyPatches(db DB, version int, patches []Patch, opts ApplyPatchesOptions) (DbVersion, error) {
	_, err := db.LoadQueries(sqlFiles)
	if err != nil {
		return DbVersion{}, err
	}

	sort.Slice(patches, func(i, j int) bool {
		return patches[i].Version < patches[j].Version
	})

	return Atomic(db, func(tx Tx[DbVersion]) (DbVersion, error) {
		if err := Exec(db, "tory.create-table-db-version", nil); err != nil {
			return DbVersion{}, err
		}

		current, err := Get[DbVersion](db, "tory.upsert-db-version", Args{
			"version": version,
		})
		if err != nil {
			return current, err
		}

		for _, patch := range patches {
			if patch.Version <= current.Version {
				if opts.OnSkip != nil {
					opts.OnSkip(patch)
				}
				continue
			}

			if opts.OnStart != nil {
				opts.OnStart(patch)
			}

			if err := Exec(db, patch.Name, nil); err != nil {
				return current, err
			}

			current.Version = patch.Version
			if err := Exec(db, "tory.update-db-version", Args{"version": current.Version}); err != nil {
				return current, err
			}

			if opts.OnFinish != nil {
				opts.OnFinish(patch)
			}
		}

		return current, nil
	})
}
