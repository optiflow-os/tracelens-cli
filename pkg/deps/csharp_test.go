package deps_test

import (
	"testing"

	"github.com/optiflow-os/tracelens-cli/pkg/deps"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParserCSharp_Parse(t *testing.T) {
	parser := deps.ParserCSharp{}

	dependencies, err := parser.Parse(t.Context(), "testdata/csharp.cs")
	require.NoError(t, err)

	assert.Equal(t, []string{
		"WakaTime",
		"Math",
		"Fart",
		"Proper",
	}, dependencies)
}
