package lexer

import (
	"github.com/optiflow-os/tracelens-cli/pkg/heartbeat"

	"github.com/alecthomas/chroma/v2"
)

// GosuTemplate lexer.
type GosuTemplate struct{}

// Lexer returns the lexer.
func (l GosuTemplate) Lexer() chroma.Lexer {
	return chroma.MustNewLexer(
		&chroma.Config{
			Name:      l.Name(),
			Aliases:   []string{"gst"},
			Filenames: []string{"*.gst"},
			MimeTypes: []string{"text/x-gosu-template"},
		},
		func() chroma.Rules {
			return chroma.Rules{
				"root": {},
			}
		},
	)
}

// Name returns the name of the lexer.
func (GosuTemplate) Name() string {
	return heartbeat.LanguageGosuTemplate.StringChroma()
}
