package deps_test

import (
	"context"
	"testing"

	"github.com/optiflow-os/tracelens-cli/pkg/deps"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParserPython_Parse(t *testing.T) {
	parser := deps.ParserPython{}

	dependencies, err := parser.Parse(context.Background(), "testdata/python.py")
	require.NoError(t, err)

	assert.Equal(t, []string{
		"first",
		"second",
		"django",
		"app",
		"flask",
		"simplejson",
		"jinja",
		"pygments",
		"sqlalchemy",
		"mock",
		"unittest",
	}, dependencies)
}
