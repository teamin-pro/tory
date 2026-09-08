package tory

import (
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

var likeEscape = strings.NewReplacer(
	"%", "\\%",
	"_", "\\_",
	".", "\\.",
	"*", "\\*",
)

// LikeEscape escapes the wildcards in a user-supplied substring so it can be
// spliced into a LIKE pattern without letting the user match more than the
// substring: pattern := "%" + tory.LikeEscape(input) + "%".
func LikeEscape(s string) string {
	return likeEscape.Replace(s)
}

// IsDuplicateKeyValueViolatesUniqueConstraint reports whether err is SQLSTATE
// 23505, the unique violation an INSERT or UPDATE hits when it collides with an
// existing row. Catch it to turn the collision into an error of your own domain.
func IsDuplicateKeyValueViolatesUniqueConstraint(err error) bool { return isPgError(err, "23505") }

// IsViolationOfCheckConstraint reports whether err is SQLSTATE 23514, raised
// when a CHECK constraint rejects the write.
func IsViolationOfCheckConstraint(err error) bool { return isPgError(err, "23514") }

func isPgError(err error, code string) bool {
	if err == nil {
		return false
	}

	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		return pgErr.Code == code
	}

	return false
}
