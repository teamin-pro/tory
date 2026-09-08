package tory

import (
	"fmt"
	"io/fs"
	"log"
	"strings"
)

type ParsedQuery struct {
	name       string
	rawBody    string
	parsedBody string
	varsList   []string
}

func (q ParsedQuery) String() string { return q.Name() }

func (q ParsedQuery) Name() string { return q.name }

func (q ParsedQuery) Body() string { return q.parsedBody }

func (q ParsedQuery) Args(args Args) []any {
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

func readQueries(files fs.ReadFileFS, fname string) (map[string]ParsedQuery, error) {
	content, err := files.ReadFile(fname)
	if err != nil {
		return nil, err
	}

	src := string(content)
	spans := classify(src)
	queries := make(map[string]ParsedQuery)

	var name string
	var body strings.Builder

	for i := 0; i < len(src); {
		if spans[i] == spanComment {
			end := lineEnd(src, i)
			if next, ok := queryName(src[i:end]); ok {
				if name != "" {
					return nil, unterminated(fname, name)
				}
				name, body = next, strings.Builder{}
			}
			i = end
			continue
		}

		if name == "" {
			i++
			continue
		}

		if src[i] == ';' && spans[i] == spanCode {
			queries[name] = newParsedQuery(name, body.String())
			name = ""
			i++
			continue
		}

		body.WriteByte(src[i])
		i++
	}

	if name != "" {
		return nil, unterminated(fname, name)
	}

	return queries, nil
}

func unterminated(fname, name string) error {
	return fmt.Errorf("query `%s` in %s has no terminating `;`", name, fname)
}

func newParsedQuery(name, rawBody string) ParsedQuery {
	body := normalizeSQL(rawBody)
	parsedBody, varsList := bindVars(body)

	return ParsedQuery{
		name:       name,
		rawBody:    body,
		parsedBody: parsedBody,
		varsList:   varsList,
	}
}

// bindVars rewrites the `:name` placeholders that sit in SQL code into the
// positional `$1` form pgx takes, numbering them in the order they appear and
// reusing the number of a name that repeats. Placeholders inside literals are
// left alone, and `::` keeps meaning a cast.
func bindVars(body string) (string, []string) {
	spans := classify(body)
	numbers := make(map[string]int)
	varsList := make([]string, 0)

	var parsed strings.Builder
	for i := 0; i < len(body); {
		name, end := varAt(body, spans, i)
		if name == "" {
			parsed.WriteByte(body[i])
			i++
			continue
		}

		number, seen := numbers[name]
		if !seen {
			varsList = append(varsList, name)
			number = len(varsList)
			numbers[name] = number
		}

		fmt.Fprintf(&parsed, "$%d", number)
		i = end
	}

	return parsed.String(), varsList
}

// varAt returns the variable name starting at i, and the index just past it.
func varAt(body string, spans []spanKind, i int) (string, int) {
	if spans[i] != spanCode || body[i] != ':' {
		return "", i
	}
	if i > 0 && body[i-1] == ':' {
		return "", i
	}

	end := i + 1
	for end < len(body) && isVarByte(body[end], end == i+1) {
		end++
	}
	if end == i+1 {
		return "", i
	}

	return body[i+1 : end], end
}

func isVarByte(c byte, first bool) bool {
	letter := c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
	return letter || !first && c == '_'
}

func queryName(comment string) (string, bool) {
	bits := strings.Fields(comment)
	if len(bits) != 3 || bits[0] != "--" || bits[1] != "name:" {
		return "", false
	}

	return bits[2], true
}

func lineEnd(s string, i int) int {
	if next := strings.IndexByte(s[i:], '\n'); next != -1 {
		return i + next
	}

	return len(s)
}

func normalizeSQL(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

type spanKind uint8

const (
	spanCode spanKind = iota
	spanComment
	spanLiteral
)

// classify walks s once and says what each byte belongs to: SQL code, a `--`
// comment, or a quoted run. Quoting covers 'strings' with their doubled-quote
// escape, "identifiers" and $$-delimited blocks. PostgreSQL also allows tagged
// dollar quotes ($tag$ ... $tag$), which are deliberately not tracked: plain $$
// covers the DDL fragments embedded queries use, and a tag would need a parser
// rather than a scanner.
func classify(s string) []spanKind {
	spans := make([]spanKind, len(s))

	for i := 0; i < len(s); {
		switch {
		case strings.HasPrefix(s[i:], "--"):
			end := lineEnd(s, i)
			fill(spans, i, end, spanComment)
			i = end

		case s[i] == '\'' || s[i] == '"':
			end := quoteEnd(s, i)
			fill(spans, i, end, spanLiteral)
			i = end

		case strings.HasPrefix(s[i:], "$$"):
			end := len(s)
			if next := strings.Index(s[i+2:], "$$"); next != -1 {
				end = i + 2 + next + 2
			}
			fill(spans, i, end, spanLiteral)
			i = end

		default:
			spans[i] = spanCode
			i++
		}
	}

	return spans
}

// quoteEnd returns the index just past the run opened by the quote at i.
func quoteEnd(s string, i int) int {
	quote := s[i]
	for end := i + 1; end < len(s); end++ {
		if s[end] != quote {
			continue
		}
		if end+1 < len(s) && s[end+1] == quote {
			end++
			continue
		}

		return end + 1
	}

	return len(s)
}

func fill(spans []spanKind, from, to int, kind spanKind) {
	for i := from; i < to; i++ {
		spans[i] = kind
	}
}
