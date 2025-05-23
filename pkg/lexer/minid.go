package lexer

import (
	"github.com/optiflow-os/tracelens-cli/pkg/heartbeat"

	"github.com/alecthomas/chroma/v2"
)

// MiniD lexer.
type MiniD struct{}

// Lexer returns the lexer.
func (l MiniD) Lexer() chroma.Lexer {
	return chroma.MustNewLexer(
		&chroma.Config{
			Name:    l.Name(),
			Aliases: []string{"minid"},
			// Don't lex .md as MiniD, reserve for Markdown.
			Filenames: []string{},
			MimeTypes: []string{"text/x-minidsrc"},
		},
		func() chroma.Rules {
			return chroma.Rules{
				"root": {},
			}
		},
	)
}

// Name returns the name of the lexer.
func (MiniD) Name() string {
	return heartbeat.LanguageMiniD.StringChroma()
}
