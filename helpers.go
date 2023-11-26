package tory

import (
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pkg/errors"
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

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == code
	}

	return false
}
