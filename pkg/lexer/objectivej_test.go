package lexer_test

import (
	"os"
	"testing"

	"github.com/optiflow-os/tracelens-cli/pkg/lexer"

	"github.com/stretchr/testify/assert"
)

func TestObjectiveJ_AnalyseText(t *testing.T) {
	data, err := os.ReadFile("testdata/objectivej_import.j")
	assert.NoError(t, err)

	l := lexer.ObjectiveJ{}.Lexer()

	assert.Equal(t, float32(1.0), l.AnalyseText(string(data)))
}
