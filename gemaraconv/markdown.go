package gemaraconv

import (
	"bytes"
	"embed"
	"fmt"
	"strings"
	"text/template"
	"unicode"

	"github.com/gemaraproj/go-gemara"
)

//go:embed templates/*.tmpl
var markdownTemplates embed.FS

// CatalogToMarkdown renders a ControlCatalog as Markdown using embedded templates.
func CatalogToMarkdown(catalog *gemara.ControlCatalog, opts ...MarkdownOption) ([]byte, error) {
	if catalog == nil {
		return nil, fmt.Errorf("catalog is nil")
	}

	o := defaultMarkdownOpts()
	o.apply(opts...)

	view := buildMarkdownCatalogView(catalog, o)

	t, err := template.New("").Funcs(markdownFuncMap()).ParseFS(markdownTemplates, "templates/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parse markdown templates: %w", err)
	}

	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "catalog", view); err != nil {
		return nil, fmt.Errorf("execute markdown template: %w", err)
	}

	out := buf.Bytes()
	if o.lineEnding != "\n" {
		out = []byte(strings.ReplaceAll(string(out), "\n", o.lineEnding))
	}
	return out, nil
}

func markdownFuncMap() template.FuncMap {
	return template.FuncMap{
		"anchor":       markdownAnchor,
		"lifecycle":    func(l gemara.Lifecycle) string { return l.String() },
		"artifactType": func(a gemara.ArtifactType) string { return a.String() },
		"entityType":   func(e gemara.EntityType) string { return e.String() },
		"datetime":     func(d gemara.Datetime) string { return string(d) },
		"joinStrings":  func(ss []string, sep string) string { return strings.Join(ss, sep) },
		"artifactMapping": func(m gemara.ArtifactMapping) string {
			s := m.ReferenceId
			if m.Remarks != "" {
				s += " — " + m.Remarks
			}
			return s
		},
	}
}

// markdownAnchor returns a GitHub-style fragment id for heading text (lowercase, hyphen-separated).
func markdownAnchor(s string) string {
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
