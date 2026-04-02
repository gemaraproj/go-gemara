package gemaraconv

import (
	"fmt"
	"strings"

	"github.com/gemaraproj/go-gemara"
)

// isLexiconFetchURL reports whether s uses a scheme loaders accept for lexicon YAML (https, http, file).
func isLexiconFetchURL(s string) bool {
	return strings.HasPrefix(s, "https://") ||
		strings.HasPrefix(s, "http://") ||
		strings.HasPrefix(s, "file://")
}

func refLinesFromGemara(refs []gemara.LexiconReference) []lexiconRefLine {
	out := make([]lexiconRefLine, len(refs))
	for i, r := range refs {
		out[i] = lexiconRefLine{
			Citation: strings.TrimSpace(r.Citation),
			URL:      strings.TrimSpace(r.Url),
		}
	}
	return out
}

func refLinesFromStrings(refs []string) []lexiconRefLine {
	var out []lexiconRefLine
	for _, r := range refs {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		if strings.HasPrefix(r, "http://") || strings.HasPrefix(r, "https://") {
			out = append(out, lexiconRefLine{Citation: r, URL: r})
		} else {
			out = append(out, lexiconRefLine{Citation: r})
		}
	}
	return out
}

// trimSynonyms returns trimmed non-empty synonyms or an error.
// scope is the message prefix, e.g. "lexicon terms" or "inline lexicon".
func trimSynonyms(synonyms []string, termIndex int, scope string) ([]string, error) {
	out := make([]string, 0, len(synonyms))
	for _, s := range synonyms {
		s = strings.TrimSpace(s)
		if s == "" {
			return nil, fmt.Errorf("%s[%d]: empty synonym", scope, termIndex)
		}
		out = append(out, s)
	}
	return out, nil
}

func markGemaraCanonicalSeen(seen map[string]struct{}, canonical string) error {
	key := strings.ToLower(canonical)
	if _, dup := seen[key]; dup {
		return fmt.Errorf("duplicate lexicon canonical %q", canonical)
	}
	seen[key] = struct{}{}
	return nil
}

func markInlineLexiconTermSeen(seen map[string]struct{}, canonical string) error {
	key := strings.ToLower(canonical)
	if _, dup := seen[key]; dup {
		return fmt.Errorf("duplicate inline lexicon term %q", canonical)
	}
	seen[key] = struct{}{}
	return nil
}
