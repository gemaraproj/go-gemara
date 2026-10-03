package markdown

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"
)

func containsLexiconSynonym(synonyms []string, entryCanonical, term string) bool {
	if strings.EqualFold(entryCanonical, term) {
		return true
	}
	for _, item := range synonyms {
		if strings.EqualFold(strings.TrimSpace(item), term) {
			return true
		}
	}
	return false
}

// lexiconIsWrapped reports whether the match is already inside an unclosed Markdown link label.
func lexiconIsWrapped(text, matched string) bool {
	beforeIndex := strings.Index(text, matched)
	if beforeIndex == -1 {
		return true
	}
	substrBeforeTerm := text[:beforeIndex]
	openBrackets := strings.Count(substrBeforeTerm, "[")
	closeBrackets := strings.Count(substrBeforeTerm, "]")
	return openBrackets > closeBrackets
}

func addLexiconLinksForTerm(lexicon []lexiconEntry, text, term string) (string, error) {
	escapedTerm := regexp.QuoteMeta(term)
	termRegex := regexp.MustCompile(`(?i)\b` + escapedTerm + `(?:s)?\b`)

	termIdx := slices.IndexFunc(lexicon, func(entry lexiconEntry) bool {
		return containsLexiconSynonym(entry.Synonyms, entry.Canonical, term)
	})
	if termIdx == -1 {
		return "", fmt.Errorf("markdown: cannot link unknown lexicon term %q", term)
	}
	canonical := lexicon[termIdx].Canonical

	return termRegex.ReplaceAllStringFunc(text, func(matched string) string {
		if lexiconIsWrapped(text, matched) {
			return matched
		}
		return fmt.Sprintf("[%s][%s]", matched, canonical)
	}), nil
}

// addLexiconLinks applies baseline-style reference autolinks for every canonical term and synonym.
func addLexiconLinks(lexicon []lexiconEntry, text string) (string, error) {
	for _, entry := range lexicon {
		var err error
		text, err = addLexiconLinksForTerm(lexicon, text, entry.Canonical)
		if err != nil {
			return "", err
		}
		for _, syn := range entry.Synonyms {
			text, err = addLexiconLinksForTerm(lexicon, text, syn)
			if err != nil {
				return "", err
			}
		}
	}
	return text, nil
}

func newLexiconLinker(entries []lexiconEntry) func(string) (string, error) {
	if len(entries) == 0 {
		return func(plain string) (string, error) { return plain, nil }
	}
	return func(text string) (string, error) {
		return addLexiconLinks(entries, text)
	}
}

// lexiconRefSlug matches security-baseline asLink: lowercase, drop '.', other non-alnum → '-'.
func lexiconRefSlug(text string) string {
	return "#" + strings.Map(func(ch rune) rune {
		switch {
		case unicode.IsLetter(ch) || unicode.IsNumber(ch):
			return unicode.ToLower(ch)
		case ch == '.':
			return -1
		default:
			return '-'
		}
	}, text)
}
