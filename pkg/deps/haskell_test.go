package deps_test

import (
	"testing"

	"github.com/optiflow-os/tracelens-cli/pkg/deps"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParserHaskell_Parse(t *testing.T) {
	parser := deps.ParserHaskell{}

	dependencies, err := parser.Parse(t.Context(), "testdata/haskell.hs")
	require.NoError(t, err)

	assert.Equal(t, []string{
		"Control",
		"Data",
		"Network",
		"System",
	}, dependencies)
}
