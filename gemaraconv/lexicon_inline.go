package gemaraconv

import (
	"fmt"
	"strings"
)

// InlineLexiconTerm carries list-shaped lexicon rows (e.g. OSPS baseline/lexicon.yaml:
// term, definition, synonyms, string references) for Markdown autolink + glossary without
// fetching a Gemara Lexicon YAML document.
type InlineLexiconTerm struct {
	Term       string
	Definition string
	Synonyms   []string
	References []string
}

func normalizeInlineLexicon(terms []InlineLexiconTerm) ([]lexiconEntry, error) {
	if len(terms) == 0 {
		return nil, nil
	}
	seen := make(map[string]struct{})
	out := make([]lexiconEntry, 0, len(terms))
	for i, t := range terms {
		canonical := strings.TrimSpace(t.Term)
		if canonical == "" {
			return nil, fmt.Errorf("inline lexicon[%d]: empty term", i)
		}
		def := strings.TrimSpace(t.Definition)
		if def == "" {
			return nil, fmt.Errorf("inline lexicon[%d]: empty definition", i)
		}
		if err := markInlineLexiconTermSeen(seen, canonical); err != nil {
			return nil, err
		}

		syns, err := trimSynonyms(t.Synonyms, i, "inline lexicon")
		if err != nil {
			return nil, err
		}

		out = append(out, lexiconEntry{
			Canonical:  canonical,
			Definition: def,
			Synonyms:   syns,
			Refs:       refLinesFromStrings(t.References),
		})
	}
	return out, nil
}
