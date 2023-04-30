package tory

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseVersion(t *testing.T) {
	assert.Equal(t, -1, parseVersion("0"))
	assert.Equal(t, 1, parseVersion("0001-aaa"))
	assert.Equal(t, 333, parseVersion(strings.TrimPrefix("prefix-0333-aaa", "prefix-")))
}
