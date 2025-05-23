package deps_test

import (
	"context"
	"testing"

	"github.com/optiflow-os/tracelens-cli/pkg/deps"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParserPHP_Parse(t *testing.T) {
	parser := deps.ParserPHP{}

	dependencies, err := parser.Parse(context.Background(), "testdata/php.php")
	require.NoError(t, err)

	assert.Equal(t, []string{
		"Interop",
		"'ServiceLocator.php'",
		"'ServiceLocatorTwo.php'",
		"FooBarOne",
		"FooBarTwo",
		"ArrayObject",
		"FooBarThree",
		"FooBarFour",
		"FooBarSeven",
		"FooBarEight",
	}, dependencies)
}
