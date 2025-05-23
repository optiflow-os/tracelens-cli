package lexer_test

import (
	"os"
	"testing"

	"github.com/optiflow-os/tracelens-cli/pkg/lexer"

	"github.com/stretchr/testify/assert"
)

func TestStan_AnalyseText(t *testing.T) {
	data, err := os.ReadFile("testdata/stan_basic.stan")
	assert.NoError(t, err)

	l := lexer.Stan{}.Lexer()

	assert.Equal(t, float32(1.0), l.AnalyseText(string(data)))
}
