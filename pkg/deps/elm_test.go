package deps_test

import (
	"testing"

	"github.com/optiflow-os/tracelens-cli/pkg/deps"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParserElm_Parse(t *testing.T) {
	parser := deps.ParserElm{}

	dependencies, err := parser.Parse(t.Context(), "testdata/elm.elm")
	require.NoError(t, err)

	assert.Equal(t, []string{
		"Color",
		"Dict",
		"TempFontAwesome",
		"Html",
		"Html",
		"Markdown",
		"String",
	}, dependencies)
}
