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

func LikeEscape(s string) string {
	return likeEscape.Replace(s)
}

func IsDuplicateKeyValueViolatesUniqueConstraint(err error) bool { return isPgError(err, "23505") }

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
