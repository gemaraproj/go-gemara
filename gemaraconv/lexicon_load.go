package gemaraconv

import (
	"fmt"
	"strings"

	"github.com/gemaraproj/go-gemara"
	"github.com/gemaraproj/go-gemara/internal/loaders"
)

// resolveLexiconURL returns the https:// or file:// URI for the lexicon artifact.
// Precedence: metadata.mapping-references entry whose id matches metadata.lexicon.reference-id;
// else metadata.lexicon.remarks if it is a fetchable URL.
func resolveLexiconURL(md gemara.Metadata) (string, error) {
	if md.Lexicon == nil {
		return "", fmt.Errorf("lexicon mapping is nil")
	}
	refID := strings.TrimSpace(md.Lexicon.ReferenceId)
	for _, m := range md.MappingReferences {
		if m.Id == refID && refID != "" {
			u := strings.TrimSpace(m.Url)
			if u == "" {
				return "", fmt.Errorf("mapping-references entry %q has empty url", refID)
			}
			return u, nil
		}
	}
	remarks := strings.TrimSpace(md.Lexicon.Remarks)
	if isLexiconFetchURL(remarks) {
		return remarks, nil
	}
	if refID == "" {
		return "", fmt.Errorf("metadata.lexicon has empty reference-id and remarks is not a fetchable URL")
	}
	return "", fmt.Errorf("no mapping-references entry with id %q for metadata.lexicon", refID)
}

// loadLexiconFromURI fetches YAML and returns normalized entries, or an error.
func loadLexiconFromURI(uri string) ([]lexiconEntry, error) {
	var doc gemara.Lexicon
	if err := loaders.LoadYAML(uri, &doc); err != nil {
		return nil, fmt.Errorf("load lexicon YAML: %w", err)
	}
	return parseLexiconDocument(&doc)
}

func parseLexiconDocument(doc *gemara.Lexicon) ([]lexiconEntry, error) {
	if err := validateLexicon(doc); err != nil {
		return nil, err
	}
	return normalizeLexicon(doc)
}

// parseLexiconYAML decodes bytes as a single Gemara Lexicon document and returns normalized entries.
func parseLexiconYAML(data []byte) ([]lexiconEntry, error) {
	var doc gemara.Lexicon
	if err := loaders.UnmarshalYAML(data, &doc); err != nil {
		return nil, fmt.Errorf("decode lexicon YAML: %w", err)
	}
	return parseLexiconDocument(&doc)
}

func validateLexicon(l *gemara.Lexicon) error {
	if l == nil {
		return fmt.Errorf("lexicon is nil")
	}
	if len(l.Terms) == 0 {
		return fmt.Errorf("lexicon has no terms")
	}
	for i, t := range l.Terms {
		if strings.TrimSpace(t.Title) == "" && strings.TrimSpace(t.Id) == "" {
			return fmt.Errorf("lexicon terms[%d]: title and id are both empty", i)
		}
		if strings.TrimSpace(t.Definition) == "" {
			return fmt.Errorf("lexicon terms[%d]: definition is empty", i)
		}
		for j, r := range t.References {
			if strings.TrimSpace(r.Citation) == "" {
				return fmt.Errorf("lexicon terms[%d].references[%d]: citation is empty", i, j)
			}
		}
	}
	return nil
}

func normalizeLexicon(l *gemara.Lexicon) ([]lexiconEntry, error) {
	seen := make(map[string]struct{})
	out := make([]lexiconEntry, 0, len(l.Terms))
	for i, t := range l.Terms {
		canonical := strings.TrimSpace(t.Title)
		if canonical == "" {
			canonical = strings.TrimSpace(t.Id)
		}
		if err := markGemaraCanonicalSeen(seen, canonical); err != nil {
			return nil, err
		}

		syns, err := trimSynonyms(t.Synonyms, i, "lexicon terms")
		if err != nil {
			return nil, err
		}

		out = append(out, lexiconEntry{
			Canonical:  canonical,
			Definition: strings.TrimSpace(t.Definition),
			Synonyms:   syns,
			Refs:       refLinesFromGemara(t.References),
		})
	}
	return out, nil
}
