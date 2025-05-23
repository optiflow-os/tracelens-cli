package lexer

import (
	"github.com/optiflow-os/tracelens-cli/pkg/heartbeat"

	"github.com/alecthomas/chroma/v2"
)

// PythonConsole lexer.
type PythonConsole struct{}

// Lexer returns the lexer.
func (l PythonConsole) Lexer() chroma.Lexer {
	return chroma.MustNewLexer(
		&chroma.Config{
			Name:      l.Name(),
			Aliases:   []string{"pycon"},
			MimeTypes: []string{"text/x-python-doctest"},
		},
		func() chroma.Rules {
			return chroma.Rules{
				"root": {},
			}
		},
	)
}

// Name returns the name of the lexer.
func (PythonConsole) Name() string {
	return heartbeat.LanguagePythonConsole.StringChroma()
}
