package tory

import "strings"

var likeEscape = strings.NewReplacer(
	"%", "\\%",
	"_", "\\_",
	".", "\\.",
	"*", "\\*",
)

func LikeEscape(s string) string {
	return likeEscape.Replace(s)
}
