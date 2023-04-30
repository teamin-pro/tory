package tory

import (
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"
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
	Prefix   string
	OnSkip   func(Patch)
	OnStart  func(Patch)
	OnFinish func(Patch)
}

func ApplyPatches(db Tory, opts ApplyPatchesOptions) (DbVersion, error) {
	patches := make([]Patch, 0)
	versions := make(map[int]struct{})

	for _, q := range db.AllQueries() {
		if !strings.HasPrefix(q.Name(), opts.Prefix) {
			continue
		}

		version := parseVersion(strings.TrimPrefix(q.Name(), opts.Prefix))
		if version == -1 {
			return DbVersion{}, fmt.Errorf("invalid patch name: `%s`", q.Name())
		}

		if _, ok := versions[version]; ok {
			return DbVersion{}, fmt.Errorf("duplicate patch version: `%s`", q.Name())
		}
		versions[version] = struct{}{}

		patches = append(patches, Patch{
			Version: version,
			Name:    q.Name(),
		})
	}

	if len(patches) == 0 {
		return DbVersion{}, fmt.Errorf("no patches found")
	}

	sort.Slice(patches, func(i, j int) bool {
		return patches[i].Version < patches[j].Version
	})

	latestVersion := patches[len(patches)-1].Version

	if err := db.Load(sqlFiles); err != nil {
		return DbVersion{}, err
	}

	return Atomic(db, func(tx Tx[DbVersion]) (DbVersion, error) {
		if err := Exec(db, "tory.create-table-db-version", nil); err != nil {
			return DbVersion{}, err
		}

		currentVersion, err := Get[DbVersion](db, "tory.upsert-db-version", Args{
			"version": latestVersion,
		})
		if err != nil {
			return currentVersion, err
		}

		for _, patch := range patches {
			if patch.Version <= currentVersion.Version {
				if opts.OnSkip != nil {
					opts.OnSkip(patch)
				}
				continue
			}

			if opts.OnStart != nil {
				opts.OnStart(patch)
			}

			if err := Exec(db, patch.Name, nil); err != nil {
				return currentVersion, err
			}

			currentVersion.Version = patch.Version
			if err := Exec(db, "tory.update-db-version", Args{"version": currentVersion.Version}); err != nil {
				return currentVersion, err
			}

			if opts.OnFinish != nil {
				opts.OnFinish(patch)
			}
		}

		return currentVersion, nil
	})
}

func parseVersion(str string) int {
	bits := strings.SplitN(str, "-", 2)
	if len(bits) != 2 {
		return -1
	}

	version, err := strconv.ParseInt(strings.TrimLeft(bits[0], "0"), 10, 64)
	if err != nil {
		return -1
	}

	return int(version)
}
