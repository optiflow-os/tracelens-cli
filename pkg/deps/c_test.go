package deps_test

import (
	"testing"

	"github.com/optiflow-os/tracelens-cli/pkg/deps"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParserC_Parse(t *testing.T) {
	parser := deps.ParserC{}

	dependencies, err := parser.Parse(t.Context(), "testdata/c.c")
	require.NoError(t, err)

	assert.Equal(t, []string{
		"math",
		"openssl",
	}, dependencies)
}
