package deps_test

import (
	"testing"

	"github.com/optiflow-os/tracelens-cli/pkg/deps"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParserHTML_Parse(t *testing.T) {
	ctx := t.Context()

	tests := map[string]struct {
		Filepath string
		Expected []string
	}{
		"html": {
			Filepath: "testdata/html.html",
			Expected: []string{
				`"wakatime.js"`,
				`"../scripts/wakatime.js"`,
				`"https://www.wakatime.com/scripts/my.js"`,
				"\"this is a\n multiline value\"",
			},
		},
		"html django": {
			Filepath: "testdata/html_django.html",
			Expected: []string{`"libs/json2.js"`},
		},
		"html with PHP": {
			Filepath: "testdata/html_with_php.html",
			Expected: []string{`"https://maxcdn.bootstrapcdn.com/bootstrap/3.3.5/js/bootstrap.min.js"`},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			parser := deps.ParserHTML{}

			dependencies, err := parser.Parse(ctx, test.Filepath)
			require.NoError(t, err)

			assert.Equal(t, test.Expected, dependencies)
		})
	}
}
