package lexer

import (
	"github.com/optiflow-os/tracelens-cli/pkg/heartbeat"

	"github.com/alecthomas/chroma/v2"
)

// ElixirIexSsession lexer.
type ElixirIexSsession struct{}

// Lexer returns the lexer.
func (l ElixirIexSsession) Lexer() chroma.Lexer {
	return chroma.MustNewLexer(
		&chroma.Config{
			Name:      l.Name(),
			Aliases:   []string{"iex"},
			MimeTypes: []string{"text/x-elixir-shellsession"},
		},
		func() chroma.Rules {
			return chroma.Rules{
				"root": {},
			}
		},
	)
}

// Name returns the name of the lexer.
func (ElixirIexSsession) Name() string {
	return heartbeat.LanguageElixirIexSession.StringChroma()
}
