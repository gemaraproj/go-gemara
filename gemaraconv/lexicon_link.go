package gemaraconv

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"
)

// lexiconRefLine is one reference for glossary rendering.
type lexiconRefLine struct {
	Citation string
	URL      string
}

// lexiconEntry is normalized lexicon data for autolinking and the glossary.
type lexiconEntry struct {
	Canonical  string
	Definition string
	Synonyms   []string
	Refs       []lexiconRefLine
}

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

func addLexiconLinksForTerm(lexicon []lexiconEntry, text, term string) string {
	escapedTerm := regexp.QuoteMeta(term)
	termRegex := regexp.MustCompile(`(?i)\b` + escapedTerm + `(?:s)?\b`)

	termIdx := slices.IndexFunc(lexicon, func(t lexiconEntry) bool {
		return containsLexiconSynonym(t.Synonyms, t.Canonical, term)
	})
	if termIdx == -1 {
		panic(fmt.Sprintf("gemaraconv: addLexiconLinksForTerm called for unknown term %q", term))
	}
	canonical := lexicon[termIdx].Canonical

	return termRegex.ReplaceAllStringFunc(text, func(matched string) string {
		if lexiconIsWrapped(text, matched) {
			return matched
		}
		return fmt.Sprintf("[%s][%s]", matched, canonical)
	})
}

// addLexiconLinks applies baseline-style reference autolinks for every canonical term and synonym.
func addLexiconLinks(lexicon []lexiconEntry, text string) string {
	for _, entry := range lexicon {
		text = addLexiconLinksForTerm(lexicon, text, entry.Canonical)
		for _, syn := range entry.Synonyms {
			text = addLexiconLinksForTerm(lexicon, text, syn)
		}
	}
	return text
}

func newLexiconLinker(entries []lexiconEntry) func(string) string {
	if len(entries) == 0 {
		return func(s string) string { return s }
	}
	return func(text string) string {
		return addLexiconLinks(entries, text)
	}
}

// lexiconRefSlug matches security-baseline asLink: lowercase, drop '.', other non-alnum → '-'.
func lexiconRefSlug(text string) string {
	return "#" + strings.Map(func(r rune) rune {
		switch {
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			return unicode.ToLower(r)
		case r == '.':
			return -1
		default:
			return '-'
		}
	}, text)
}
