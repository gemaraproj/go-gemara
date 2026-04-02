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
		key := strings.ToLower(canonical)
		if _, dup := seen[key]; dup {
			return nil, fmt.Errorf("duplicate inline lexicon term %q", canonical)
		}
		seen[key] = struct{}{}

		var syns []string
		for _, s := range t.Synonyms {
			s = strings.TrimSpace(s)
			if s == "" {
				return nil, fmt.Errorf("inline lexicon[%d]: empty synonym", i)
			}
			syns = append(syns, s)
		}

		var refs []lexiconRefLine
		for _, r := range t.References {
			r = strings.TrimSpace(r)
			if r == "" {
				continue
			}
			if strings.HasPrefix(r, "http://") || strings.HasPrefix(r, "https://") {
				refs = append(refs, lexiconRefLine{Citation: r, URL: r})
			} else {
				refs = append(refs, lexiconRefLine{Citation: r})
			}
		}

		out = append(out, lexiconEntry{
			Canonical:  canonical,
			Definition: def,
			Synonyms:   syns,
			Refs:       refs,
		})
	}
	return out, nil
}
