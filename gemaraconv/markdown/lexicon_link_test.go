package markdown

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddLexiconLinks_basic(t *testing.T) {
	sampleLexicon := []lexiconEntry{
		{
			Canonical:  "Example Term",
			Definition: "d",
			Synonyms:   []string{"ET"},
			Refs:       nil,
		},
	}
	out, err := addLexiconLinks(sampleLexicon, "Use Example Term and ET in prose.")
	require.NoError(t, err)
	assert.Contains(t, out, "[Example Term][Example Term]")
	assert.Contains(t, out, "[ET][Example Term]")
}

func TestAddLexiconLinks_pluralAndCase(t *testing.T) {
	sampleLexicon := []lexiconEntry{{Canonical: "Widget", Definition: "d"}}
	out, err := addLexiconLinks(sampleLexicon, "Many widgets here.")
	require.NoError(t, err)
	assert.Contains(t, out, "[widgets][Widget]")
}

func TestAddLexiconLinks_skipsInsideBrackets(t *testing.T) {
	sampleLexicon := []lexiconEntry{{Canonical: "Term", Definition: "d"}}
	out, err := addLexiconLinks(sampleLexicon, "already [Term] linked")
	require.NoError(t, err)
	assert.Equal(t, "already [Term] linked", out)
}

func TestAddLexiconLinksForTerm_unknownTerm(t *testing.T) {
	_, err := addLexiconLinksForTerm(nil, "plain", "missing")
	require.EqualError(t, err, `markdown: cannot link unknown lexicon term "missing"`)
}

func TestLexiconRefSlug(t *testing.T) {
	assert.Equal(t, "#example-term", lexiconRefSlug("Example Term"))
	assert.Equal(t, "#ab", lexiconRefSlug("a.b"))
}

func TestNewLexiconLinker_noop(t *testing.T) {
	linkFunc := newLexiconLinker(nil)
	out, err := linkFunc("plain")
	require.NoError(t, err)
	require.Equal(t, "plain", out)
}
