package gemaraconv

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeInlineLexicon_basic(t *testing.T) {
	entries, err := normalizeInlineLexicon([]InlineLexiconTerm{
		{Term: "Alpha", Definition: "def a", Synonyms: []string{"A"}, References: []string{"https://example.com"}},
	})
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "Alpha", entries[0].Canonical)
	assert.Equal(t, "https://example.com", entries[0].Refs[0].URL)
}

func TestNormalizeInlineLexicon_skipsEmptyRefs(t *testing.T) {
	entries, err := normalizeInlineLexicon([]InlineLexiconTerm{
		{Term: "T", Definition: "d", References: []string{"", "  ", "note"}},
	})
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Len(t, entries[0].Refs, 1)
	assert.Equal(t, "note", entries[0].Refs[0].Citation)
}

func TestNormalizeInlineLexicon_dupTerm(t *testing.T) {
	_, err := normalizeInlineLexicon([]InlineLexiconTerm{
		{Term: "Same", Definition: "a"},
		{Term: "same", Definition: "b"},
	})
	require.Error(t, err)
}
