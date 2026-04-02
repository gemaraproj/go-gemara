package gemaraconv

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddLexiconLinks_basic(t *testing.T) {
	lex := []lexiconEntry{
		{
			Canonical:  "Example Term",
			Definition: "d",
			Synonyms:   []string{"ET"},
			Refs:       nil,
		},
	}
	out := addLexiconLinks(lex, "Use Example Term and ET in prose.")
	assert.Contains(t, out, "[Example Term][Example Term]")
	assert.Contains(t, out, "[ET][Example Term]")
}

func TestAddLexiconLinks_pluralAndCase(t *testing.T) {
	lex := []lexiconEntry{{Canonical: "Widget", Definition: "d"}}
	out := addLexiconLinks(lex, "Many widgets here.")
	assert.Contains(t, out, "[widgets][Widget]")
}

func TestAddLexiconLinks_skipsInsideBrackets(t *testing.T) {
	lex := []lexiconEntry{{Canonical: "Term", Definition: "d"}}
	out := addLexiconLinks(lex, "already [Term] linked")
	assert.Equal(t, "already [Term] linked", out)
}

func TestLexiconRefSlug(t *testing.T) {
	assert.Equal(t, "#example-term", lexiconRefSlug("Example Term"))
	assert.Equal(t, "#ab", lexiconRefSlug("a.b"))
}

func TestNewLexiconLinker_noop(t *testing.T) {
	f := newLexiconLinker(nil)
	require.Equal(t, "plain", f("plain"))
}
