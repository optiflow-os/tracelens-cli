package deps_test

import (
	"testing"

	"github.com/optiflow-os/tracelens-cli/pkg/deps"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParserHaxe_Parse(t *testing.T) {
	parser := deps.ParserHaxe{}

	dependencies, err := parser.Parse(t.Context(), "testdata/haxe.hx")
	require.NoError(t, err)

	assert.Equal(t, []string{
		"alpha",
		"bravo",
		"Math",
		"charlie",
		"delta",
	}, dependencies)
}
