package tory

import (
	"fmt"
	"io/fs"
	"log"
	"regexp"
	"sort"
	"strings"
)

type parsedQuery struct {
	name       string
	rawBody    string
	parsedBody string
	varsList   []string
}

func (q parsedQuery) Body() string {
	return q.parsedBody
}

func (q parsedQuery) Args(args Args) []any {
	res := make([]any, len(q.varsList))
	for i, name := range q.varsList {
		value, found := args[name]
		if !found {
			log.Panicf("query `%s` fail: var `%s` not found in %v", q.name, name, args)
		}
		res[i] = value
	}
	return res
}

func getQuery(name string) parsedQuery {
	query := allQueries[name]
	if query.rawBody == "" {
		log.Panicf("query not found: `%s` (from %d)", name, len(allQueries))
	}
	return query
}

func readQueries(files fs.ReadFileFS, fname string) (map[string]parsedQuery, error) {
	content, err := files.ReadFile(fname)
	if err != nil {
		return nil, err
	}

	queries := make(map[string]parsedQuery)

	var q parsedQuery
	for _, line := range strings.Split(string(content), "\n") {
		bits := strings.Fields(line)
		if len(bits) == 3 && bits[0] == "--" && bits[1] == "name:" {
			q = parsedQuery{
				name: bits[2],
			}
			continue
		}

		if q.name == "" {
			continue
		}

		q.rawBody += " " + removeComments(line)

		if strings.HasSuffix(strings.TrimSpace(q.rawBody), ";") {
			q.rawBody = strings.TrimSuffix(normalizeSQL(q.rawBody), ";")
			q.varsList = parseVars(q.rawBody)

			q.parsedBody = q.rawBody
			for i, name := range q.varsList {
				q.parsedBody = strings.ReplaceAll(q.parsedBody, ":"+name, fmt.Sprintf("$%d", i+1))
			}

			queries[q.name] = q
		}
	}

	return queries, nil
}

func removeComments(s string) string {
	return strings.SplitN(s, "--", 2)[0]
}

func normalizeSQL(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

var varsRegex = regexp.MustCompile(`:[a-zA-Z][a-zA-Z_]*`)

func parseVars(s string) []string {
	names := make(map[string]struct{})
	for _, name := range varsRegex.FindAllString(s, -1) {
		names[name[1:]] = struct{}{}
	}

	lst := make([]string, 0, len(names))
	for name := range names {
		lst = append(lst, name)
	}

	sort.Slice(lst, func(i, j int) bool {
		if len(lst[i]) == len(lst[j]) {
			return lst[i] < lst[j]
		}
		return len(lst[i]) > len(lst[j]) // largest first
	})

	return lst
}
