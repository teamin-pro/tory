package tory

import (
	"embed"

	"github.com/pkg/errors"
)

var allQueries = make(map[string]parsedQuery)

func LoadQueries(files embed.FS) (int, error) {
	var numFound int

	dir, err := files.ReadDir(".")
	if err != nil {
		return numFound, errors.Wrapf(err, "read sql dir fail")
	}

	for _, f := range dir {
		fileQueries, err := readQueries(files, f.Name())
		if err != nil {
			return numFound, errors.Wrapf(err, "read sql file %s fail:", f.Name())
		}
		for k, v := range fileQueries {
			allQueries[k] = v
			numFound++
		}
	}

	return numFound, nil
}
