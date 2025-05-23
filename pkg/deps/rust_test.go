package deps_test

import (
	"context"
	"testing"

	"github.com/optiflow-os/tracelens-cli/pkg/deps"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParserRust_Parse(t *testing.T) {
	parser := deps.ParserRust{}

	dependencies, err := parser.Parse(context.Background(), "testdata/rust.rs")
	require.NoError(t, err)

	assert.Equal(t, []string{
		"proc_macro",
		"phrases",
		"syn",
		"quote",
	}, dependencies)
}
