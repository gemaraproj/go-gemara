package markdown

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"strings"
	"text/template"
	"unicode"

	"github.com/gemaraproj/go-gemara"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// CatalogToMarkdown renders a ControlCatalog as Markdown using embedded templates.
// Only controls whose state is LifecycleActive are included (TOC, body, and summary counts).
func CatalogToMarkdown(ctx context.Context, catalog gemara.ControlCatalog, cfg Config) ([]byte, error) {
	return render(ctx, catalog.Metadata, cfg, "catalog", func(lex []markdownLexiconGlossaryEntry) any {
		return buildMarkdownCatalogView(catalog, cfg, lex)
	})
}

// GuidanceToMarkdown renders a GuidanceCatalog as Markdown using embedded templates.
// Only guidelines whose state is LifecycleActive are included (TOC, body, and summary counts).
func GuidanceToMarkdown(ctx context.Context, guidance gemara.GuidanceCatalog, cfg Config) ([]byte, error) {
	return render(ctx, guidance.Metadata, cfg, "guidance", func(lex []markdownLexiconGlossaryEntry) any {
		return buildMarkdownGuidanceView(guidance, cfg, lex)
	})
}

// render is the shared pipeline: load the lexicon (if configured), build the
// template root through buildView, execute rootTemplate, then normalise blank
// lines and line endings.
func render(ctx context.Context, meta gemara.Metadata, cfg Config, rootTemplate string, buildView func([]markdownLexiconGlossaryEntry) any) ([]byte, error) {
	lineEnding := cfg.LineEnding
	if lineEnding == "" {
		lineEnding = "\n"
	}
	cfg.LineEnding = lineEnding

	var lexEntries []lexiconEntry
	switch {
	case cfg.LexiconAutolink && meta.Lexicon != nil:
		if cfg.Fetcher == nil {
			return nil, fmt.Errorf("lexicon autolink requires a non-nil Fetcher")
		}
		lexiconURI, err := resolveLexiconURL(meta)
		if err != nil {
			return nil, fmt.Errorf("lexicon: resolve URL: %w", err)
		}
		loaded, err := loadLexiconFromURI(ctx, cfg.Fetcher, lexiconURI)
		if err != nil {
			return nil, fmt.Errorf("lexicon: %w", err)
		}
		lexEntries = loaded
	case len(cfg.InlineLexicon) > 0:
		loaded, err := normalizeInlineLexicon(cfg.InlineLexicon)
		if err != nil {
			return nil, fmt.Errorf("lexicon: %w", err)
		}
		lexEntries = loaded
	}

	view := buildView(buildLexiconGlossaryView(lexEntries))

	linker := newLexiconLinker(lexEntries)
	t, err := template.New("").Funcs(markdownFuncMap(linker)).ParseFS(templatesFS, "templates/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parse markdown templates: %w", err)
	}

	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, rootTemplate, view); err != nil {
		return nil, fmt.Errorf("execute markdown template: %w", err)
	}

	// Normalise CRLF first: templates and fixtures may be checked out with CRLF
	// (Windows autocrlf), and the collapse only recognises bare "\n" runs.
	text := strings.ReplaceAll(buf.String(), "\r\n", "\n")
	text = collapseExtraNewlines(text)
	out := []byte(text)
	if lineEnding != "\n" {
		out = []byte(strings.ReplaceAll(string(out), "\n", lineEnding))
	}
	return out, nil
}

// collapseExtraNewlines replaces every run of three or more consecutive newlines
// with exactly two newlines, repeating until stable (one blank line between blocks).
func collapseExtraNewlines(s string) string {
	for strings.Contains(s, "\n\n\n") {
		s = strings.ReplaceAll(s, "\n\n\n", "\n\n")
	}
	return s
}

func markdownFuncMap(lexiconLink func(string) string) template.FuncMap {
	return template.FuncMap{
		"lexiconLink":  lexiconLink,
		"anchor":       Anchor,
		"lifecycle":    func(l gemara.Lifecycle) string { return l.String() },
		"isRetired":    func(l gemara.Lifecycle) bool { return l == gemara.LifecycleRetired },
		"artifactType": func(a gemara.ArtifactType) string { return a.String() },
		"entityType":   func(e gemara.EntityType) string { return e.String() },
		"guidanceType": func(g gemara.GuidanceType) string { return g.String() },
		// indent keeps a multi-line string inside one list item.
		"indent":      func(s string) string { return strings.ReplaceAll(strings.TrimRight(s, "\n"), "\n", "\n  ") },
		"datetime":    func(d gemara.Datetime) string { return string(d) },
		"joinStrings": func(ss []string, sep string) string { return strings.Join(ss, sep) },
		"joinArtifactEntries": func(entries []gemara.ArtifactMapping, sep string) string {
			if len(entries) == 0 {
				return ""
			}
			parts := make([]string, 0, len(entries))
			for _, e := range entries {
				s := e.ReferenceId
				if e.Remarks != "" {
					s += " — " + e.Remarks
				}
				parts = append(parts, s)
			}
			return strings.Join(parts, sep)
		},
		"artifactMapping": func(m gemara.ArtifactMapping) string {
			s := m.ReferenceId
			if m.Remarks != "" {
				s += " — " + m.Remarks
			}
			return s
		},
	}
}

// Anchor returns a GitHub-style fragment id for heading text (lowercase, hyphen-separated).
func Anchor(s string) string {
	if s == "" {
		return "section"
	}
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		if b.Len() > 0 && !prevDash {
			b.WriteByte('-')
			prevDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "section"
	}
	return out
}
