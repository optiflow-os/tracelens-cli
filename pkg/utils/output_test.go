package utils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func outputTests() map[string]Output {
	return map[string]Output{
		"text":     TextOutput,
		"json":     JSONOutput,
		"raw-json": RawJSONOutput,
	}
}

func TestParseOutput(t *testing.T) {
	for value, out := range outputTests() {
		t.Run(value, func(t *testing.T) {
			parsed, err := Parse(value)
			require.NoError(t, err)

			assert.Equal(t, out, parsed)
		})
	}
}

func TestParseOutput_Invalid(t *testing.T) {
	_, err := Parse("invalid")
	require.Error(t, err)
}

func TestOutput_String(t *testing.T) {
	for value, out := range outputTests() {
		t.Run(value, func(t *testing.T) {
			assert.Equal(t, value, out.String())
		})
	}
}
