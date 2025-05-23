package deps_test

import (
	"context"
	"testing"

	"github.com/optiflow-os/tracelens-cli/pkg/deps"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParserVbNet_Parse(t *testing.T) {
	parser := deps.ParserVbNet{}

	dependencies, err := parser.Parse(context.Background(), "testdata/vbnet.vb")
	require.NoError(t, err)

	assert.Equal(t, []string{
		"WakaTime",
		"Math",
		"Proper",
	}, dependencies)
}
