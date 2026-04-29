package markdown

import (
	"context"
	"strings"
	"testing"

	"github.com/gemaraproj/go-gemara"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func defaultRenderConfig() Config {
	return Config{TOC: true, LineEnding: "\n", Metadata: true}
}

func TestCatalogToMarkdown_nilCatalog(t *testing.T) {
	_, err := CatalogToMarkdown(context.Background(), nil, defaultRenderConfig())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestAnchor(t *testing.T) {
	assert.Equal(t, "section", Anchor(""))
	assert.Equal(t, "hello-world", Anchor("Hello World"))
	assert.Equal(t, "section", Anchor("!!!"))
	assert.Equal(t, "a1b2", Anchor("A1B2"))
}

func TestCatalogToMarkdown_emptyLineEndingUsesLF(t *testing.T) {
	catalog := minimalCatalog(t)
	cfg := defaultRenderConfig()
	cfg.LineEnding = ""
	out, err := CatalogToMarkdown(context.Background(), catalog, cfg)
	require.NoError(t, err)
	assert.NotContains(t, string(out), "\r\n")
}

func TestCatalogToMarkdown_crlf(t *testing.T) {
	catalog := minimalCatalog(t)
	cfg := defaultRenderConfig()
	cfg.LineEnding = "\r\n"
	out, err := CatalogToMarkdown(context.Background(), catalog, cfg)
	require.NoError(t, err)
	assert.Contains(t, string(out), "\r\n")
}

func TestCatalogToMarkdown_metadataBranches(t *testing.T) {
	catalog := &gemara.ControlCatalog{
		Metadata: gemara.Metadata{
			Id:            "mid",
			Type:          gemara.ControlCatalogArtifact,
			Version:       "1.2.3",
			GemaraVersion: "1.0",
			Date:          gemara.Datetime("2024-06-01"),
			Description:   "Desc body.",
			Author:        gemara.Actor{Name: "Pat", Type: gemara.Human, Id: "pat@example.com"},
			Draft:         true,
			Lexicon:       &gemara.ArtifactMapping{ReferenceId: "lex", Remarks: "see refs"},
			ApplicabilityGroups: []gemara.Group{
				{Id: "ag1", Title: "AG Title", Description: "AG desc"},
			},
		},
		Title: "Rich meta",
		Groups: []gemara.Group{
			{Id: "G", Title: "Group", Description: "G desc"},
		},
		Controls: []gemara.Control{
			{Id: "C1", Group: "G", Title: "C", Objective: "O", State: gemara.LifecycleActive},
		},
	}
	out, err := CatalogToMarkdown(context.Background(), catalog, Config{TOC: false, LineEnding: "\n", Metadata: true})
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, "published **2024-06-01**")
	assert.Contains(t, s, "draft")
	assert.Contains(t, s, "### Requirement Applicability Groups")
	assert.Contains(t, s, "**ag1**")
	assert.Contains(t, s, "defined in **lex**")
	assert.Contains(t, s, "see refs")
}

func TestCatalogToMarkdown_retiredAssessmentRequirement(t *testing.T) {
	catalog := &gemara.ControlCatalog{
		Metadata: gemara.Metadata{
			Id: "x", Type: gemara.ControlCatalogArtifact, Description: "d", Author: gemara.Actor{Name: "a", Type: gemara.Human},
		},
		Title:  "T",
		Groups: []gemara.Group{{Id: "G", Title: "G"}},
		Controls: []gemara.Control{
			{
				Id:        "C1",
				Group:     "G",
				Title:     "Ctl",
				Objective: "obj",
				State:     gemara.LifecycleActive,
				AssessmentRequirements: []gemara.AssessmentRequirement{
					{
						Id:             "C1.R",
						Text:           "Retired only.",
						State:          gemara.LifecycleRetired,
						Applicability:  []string{"L1"},
						Recommendation: "Hidden when retired.",
					},
				},
			},
		},
	}
	out, err := CatalogToMarkdown(context.Background(), catalog, Config{TOC: false, Metadata: false})
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, "#### C1.R")
	assert.Contains(t, s, "Retired only.")
	assert.NotContains(t, s, "Hidden when retired.")
	assert.NotContains(t, s, "**Applicability:**")
}

func TestCatalogToMarkdown_guidelinesAndThreatsTables(t *testing.T) {
	catalog := &gemara.ControlCatalog{
		Metadata: gemara.Metadata{
			Id: "x", Type: gemara.ControlCatalogArtifact, Description: "d", Author: gemara.Actor{Name: "a", Type: gemara.Human},
		},
		Title:  "T",
		Groups: []gemara.Group{{Id: "G", Title: "G"}},
		Controls: []gemara.Control{
			{
				Id:        "C1",
				Group:     "G",
				Title:     "Ctl",
				Objective: "obj",
				State:     gemara.LifecycleActive,
				Guidelines: []gemara.MultiEntryMapping{
					{
						ReferenceId: "GL",
						Remarks:     "note",
						Entries: []gemara.ArtifactMapping{
							{ReferenceId: "e1", Remarks: "r1"},
						},
					},
				},
				Threats: []gemara.MultiEntryMapping{
					{ReferenceId: "TH", Remarks: "threat note"},
				},
				AssessmentRequirements: []gemara.AssessmentRequirement{
					{Id: "C1.1", Text: "t", State: gemara.LifecycleActive},
				},
			},
		},
	}
	out, err := CatalogToMarkdown(context.Background(), catalog, Config{TOC: false, Metadata: false})
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, "#### Guidelines")
	assert.Contains(t, s, "e1 — r1")
	assert.Contains(t, s, "#### Threats")
	assert.Contains(t, s, "**TH**")
}

func TestCatalogToMarkdown_applicabilityMatrixImplicitColumns(t *testing.T) {
	catalog := &gemara.ControlCatalog{
		Metadata: gemara.Metadata{
			Id:          "m",
			Type:        gemara.ControlCatalogArtifact,
			Description: "d",
			Author:      gemara.Actor{Name: "a", Type: gemara.Human},
		},
		Title:  "Matrix",
		Groups: []gemara.Group{{Id: "G", Title: "G"}},
		Controls: []gemara.Control{
			{
				Id:        "C",
				Group:     "G",
				Title:     "T",
				Objective: "o",
				State:     gemara.LifecycleActive,
				AssessmentRequirements: []gemara.AssessmentRequirement{
					{Id: "C.1", Text: "t", State: gemara.LifecycleActive, Applicability: []string{"Zebra", "Alpha"}},
				},
			},
		},
	}
	out, err := CatalogToMarkdown(context.Background(), catalog, Config{
		TOC: false, LineEnding: "\n", Metadata: false, ApplicabilityMatrix: true,
	})
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, "## Requirements and Applicability")
	assert.Contains(t, s, "| Alpha |")
	assert.Contains(t, s, "| Zebra |")
}

func TestCatalogToMarkdown_applicabilityMatrix_noNewlineInTableRow(t *testing.T) {
	catalog := &gemara.ControlCatalog{
		Metadata: gemara.Metadata{
			Id: "m", Type: gemara.ControlCatalogArtifact, Description: "d", Author: gemara.Actor{Name: "a", Type: gemara.Human},
			ApplicabilityGroups: []gemara.Group{{Id: "L1", Title: "Level 1"}},
		},
		Title:  "T",
		Groups: []gemara.Group{{Id: "G", Title: "G"}},
		Controls: []gemara.Control{
			{
				Id: "C-1", Group: "G", Title: "Ctl", Objective: "o", State: gemara.LifecycleActive,
				AssessmentRequirements: []gemara.AssessmentRequirement{
					{Id: "C-1.1\n", Text: "t", State: gemara.LifecycleActive, Applicability: []string{"L1"}},
				},
			},
		},
	}
	out, err := CatalogToMarkdown(context.Background(), catalog, Config{TOC: false, Metadata: false, ApplicabilityMatrix: true})
	require.NoError(t, err)
	s := string(out)
	idx := strings.Index(s, "## Requirements and Applicability")
	require.GreaterOrEqual(t, idx, 0)
	section := s[idx:]
	if end := strings.Index(section, "\n\n## "); end > 0 {
		section = section[:end]
	}
	assert.NotContains(t, section, "C-1.1\n |")
	assert.Contains(t, section, "| [**C-1.1**](#c-1-1) |X|")
}

func TestCatalogToMarkdown_ungroupedBucket(t *testing.T) {
	catalog := &gemara.ControlCatalog{
		Metadata: gemara.Metadata{
			Id: "c", Type: gemara.ControlCatalogArtifact, GemaraVersion: "1.0", Description: "d",
			Author: gemara.Actor{Name: "a", Type: gemara.Human},
		},
		Title: "Ungrouped",
		Groups: []gemara.Group{
			{Id: "G1", Title: "G1", Description: "g1"},
		},
		Controls: []gemara.Control{
			{Id: "IN", Group: "G1", Title: "In", Objective: "o", State: gemara.LifecycleActive},
			{Id: "OR", Group: "missing", Title: "Orphan", Objective: "o2", State: gemara.LifecycleActive},
		},
	}
	out, err := CatalogToMarkdown(context.Background(), catalog, defaultRenderConfig())
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, "## Ungrouped")
	assert.Contains(t, s, "### OR")
}

func TestCatalogToMarkdown_inlineLexiconPipeline(t *testing.T) {
	catalog := &gemara.ControlCatalog{
		Metadata: gemara.Metadata{
			Id: "m", Type: gemara.ControlCatalogArtifact, Description: "d", Author: gemara.Actor{Name: "a", Type: gemara.Human},
		},
		Title:  "T",
		Groups: []gemara.Group{{Id: "G", Title: "G"}},
		Controls: []gemara.Control{
			{Id: "C", Group: "G", Title: "About widgets", Objective: "widgets rock", State: gemara.LifecycleActive},
		},
	}
	cfg := Config{TOC: false, LineEnding: "\n", Metadata: false, InlineLexicon: []InlineLexiconTerm{
		{Term: "widgets", Definition: "Small gadgets."},
	}}
	out, err := CatalogToMarkdown(context.Background(), catalog, cfg)
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, "## Lexicon")
	assert.Contains(t, s, "### widgets")
	assert.Contains(t, s, "[widgets][widgets]")
}

func TestCatalogToMarkdown_lexiconAutolinkFromFile(t *testing.T) {
	catalog := &gemara.ControlCatalog{
		Metadata: gemara.Metadata{
			Id: "m", Type: gemara.ControlCatalogArtifact, Description: "d", Author: gemara.Actor{Name: "a", Type: gemara.Human},
			Lexicon: &gemara.ArtifactMapping{ReferenceId: "lex"},
			MappingReferences: []gemara.MappingReference{
				{Id: "lex", Title: "L", Version: "1", Url: lexiconTestdataAbsPath(t, "lexicon_good.yaml")},
			},
		},
		Title:  "Lex",
		Groups: []gemara.Group{{Id: "G", Title: "G"}},
		Controls: []gemara.Control{
			{Id: "C", Group: "G", Title: "Example Term in title", Objective: "text", State: gemara.LifecycleActive},
		},
	}
	out, err := CatalogToMarkdown(context.Background(), catalog, Config{TOC: false, LexiconAutolink: true})
	require.NoError(t, err)
	assert.Contains(t, string(out), "## Lexicon")
}

func TestCatalogToMarkdown_lexiconResolveError(t *testing.T) {
	catalog := &gemara.ControlCatalog{
		Metadata: gemara.Metadata{
			Id: "m", Type: gemara.ControlCatalogArtifact, Description: "d", Author: gemara.Actor{Name: "a", Type: gemara.Human},
			Lexicon: &gemara.ArtifactMapping{ReferenceId: "nope"},
		},
		Title:    "x",
		Groups:   []gemara.Group{{Id: "G", Title: "G"}},
		Controls: []gemara.Control{{Id: "C", Group: "G", Title: "T", Objective: "o", State: gemara.LifecycleActive}},
	}
	_, err := CatalogToMarkdown(context.Background(), catalog, Config{LexiconAutolink: true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "lexicon")
}

func TestCatalogToMarkdown_inlineLexiconNormalizeError(t *testing.T) {
	catalog := &gemara.ControlCatalog{
		Metadata: gemara.Metadata{
			Id: "m", Type: gemara.ControlCatalogArtifact, Description: "d", Author: gemara.Actor{Name: "a", Type: gemara.Human},
		},
		Title:    "T",
		Groups:   []gemara.Group{{Id: "G", Title: "G"}},
		Controls: []gemara.Control{{Id: "C", Group: "G", Title: "T", Objective: "o", State: gemara.LifecycleActive}},
	}
	_, err := CatalogToMarkdown(context.Background(), catalog, Config{InlineLexicon: []InlineLexiconTerm{{Term: "", Definition: "d"}}})
	require.Error(t, err)
}

func TestMarkdownFuncMap_joinArtifactEntriesEmpty(t *testing.T) {
	fn := markdownFuncMap(func(s string) string { return s })
	join := fn["joinArtifactEntries"].(func([]gemara.ArtifactMapping, string) string)
	assert.Equal(t, "", join(nil, " · "))
	assert.Equal(t, "", join([]gemara.ArtifactMapping{}, " · "))
}

func minimalCatalog(t *testing.T) *gemara.ControlCatalog {
	t.Helper()
	return &gemara.ControlCatalog{
		Metadata: gemara.Metadata{
			Id: "id1", Type: gemara.ControlCatalogArtifact, Version: "v1", Description: "One line.",
			Author: gemara.Actor{Name: "Author", Type: gemara.Human},
		},
		Title: "Minimal",
		Groups: []gemara.Group{
			{Id: "Grp", Title: "The Group", Description: "Group text."},
		},
		Controls: []gemara.Control{
			{
				Id:        "C-1",
				Group:     "Grp",
				Title:     "First control",
				Objective: "Do the thing.",
				State:     gemara.LifecycleActive,
				AssessmentRequirements: []gemara.AssessmentRequirement{
					{
						Id:             "C-1.1",
						Text:           "Requirement text.",
						State:          gemara.LifecycleActive,
						Applicability:  []string{"env-a", "env-b"},
						Recommendation: "Maybe do more.",
					},
				},
			},
		},
	}
}
