package gemaraconv

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/gemaraproj/go-gemara"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testDataFileURL returns a file:// URI to ../test-data/<name> relative to the gemaraconv package directory (go test cwd).
func testDataFileURL(t *testing.T, name string) string {
	t.Helper()
	p := filepath.Join("..", "test-data", name)
	return "file://" + filepath.ToSlash(p)
}

func loadControlCatalogFromTestData(t *testing.T, name string) *gemara.ControlCatalog {
	t.Helper()
	c := &gemara.ControlCatalog{}
	err := c.LoadFile(testDataFileURL(t, name))
	require.NoError(t, err, "load %s", name)
	return c
}

func TestCollapseExtraNewlines(t *testing.T) {
	assert.Equal(t, "a\n\nb", collapseExtraNewlines("a\n\n\nb"))
	assert.Equal(t, "a\n\nb", collapseExtraNewlines("a\n\n\n\n\n\n\nb"))
	assert.Equal(t, "a\n\nb", collapseExtraNewlines("a\n\nb"))
}

func TestCatalogToMarkdown_nil(t *testing.T) {
	_, err := CatalogToMarkdown(nil)
	require.Error(t, err)
}

func TestCatalogToMarkdown_goodCCCYAML(t *testing.T) {
	catalog := loadControlCatalogFromTestData(t, "good-ccc.yaml")

	out, err := CatalogToMarkdown(catalog)
	require.NoError(t, err)
	s := string(out)

	require.NotEmpty(t, catalog.Groups)
	group0 := catalog.Groups[0]
	var groupControls []gemara.Control
	for _, c := range catalog.Controls {
		if c.Group == group0.Id && c.State == gemara.LifecycleActive {
			groupControls = append(groupControls, c)
		}
	}
	require.NotEmpty(t, groupControls)
	sort.Slice(groupControls, func(i, j int) bool { return groupControls[i].Id < groupControls[j].Id })
	c0 := groupControls[0]
	require.NotEmpty(t, c0.AssessmentRequirements)
	ars := append([]gemara.AssessmentRequirement(nil), c0.AssessmentRequirements...)
	sort.Slice(ars, func(i, j int) bool { return ars[i].Id < ars[j].Id })
	ar0 := ars[0]

	numARs := 0
	activeControls := 0
	for _, c := range catalog.Controls {
		activeControls++
		numARs += len(c.AssessmentRequirements)
	}

	assert.Contains(t, s, fmt.Sprintf("# %s - %s", catalog.Title, catalog.Metadata.Id))
	assert.Contains(t, s, "_"+catalog.Title+"_ is a Gemara")
	assert.Contains(t, s, "## Table of contents")
	assert.Contains(t, s, fmt.Sprintf("- [%s](#%s)", group0.Title, markdownAnchor(group0.Id)))
	assert.Contains(t, s, fmt.Sprintf("  - [%s: %s](#%s)", c0.Id, c0.Title, markdownAnchor(c0.Id+": "+c0.Title)))
	assert.Contains(t, s, fmt.Sprintf("## %s: %s", group0.Id, group0.Title))
	assert.Contains(t, s, fmt.Sprintf("### %s", c0.Id))
	assert.Contains(t, s, fmt.Sprintf("#### %s", ar0.Id))
	assert.Contains(t, s, "#### Guidelines")
	assert.Contains(t, s, "#### Threats")
	assert.Contains(t, s, fmt.Sprintf("_Summary: %d control(s), %d assessment requirement(s)._", activeControls, numARs))
}

func TestCatalogToMarkdown_goodOSPSYAML(t *testing.T) {
	catalog := loadControlCatalogFromTestData(t, "good-osps.yml")

	out, err := CatalogToMarkdown(catalog, WithTOC(false))
	require.NoError(t, err)
	s := string(out)

	assert.Contains(t, s, "# Open Source Project Security Baseline")
	assert.Contains(t, s, "_Open Source Project Security Baseline_ is a Gemara")
	assert.NotContains(t, s, "## Table of contents")
	assert.Greater(t, len(out), 5000)
	assert.Contains(t, s, "### Description")
	assert.Contains(t, s, "### Mapping References")
}

func TestCatalogToMarkdown_nestedGoodCCCYAML(t *testing.T) {
	c := &gemara.ControlCatalog{}
	err := c.LoadNestedCatalog(testDataFileURL(t, "nested-good-ccc.yaml"), "catalog")
	require.NoError(t, err)

	out, err := CatalogToMarkdown(c)
	require.NoError(t, err)
	s := string(out)

	assert.Contains(t, s, "# FINOS Cloud Control Catalog")
	assert.Contains(t, s, "### CCC.C01")
}

func TestCatalogToMarkdown_ungrouped(t *testing.T) {
	catalog := &gemara.ControlCatalog{
		Metadata: gemara.Metadata{
			Id:            "c",
			Type:          gemara.ControlCatalogArtifact,
			GemaraVersion: "1.0",
			Description:   "d",
			Author:        gemara.Actor{Name: "a", Type: gemara.Human},
		},
		Title: "Ungrouped Test",
		Groups: []gemara.Group{
			{Id: "G1", Title: "G1", Description: "g1"},
		},
		Controls: []gemara.Control{
			{
				Id:        "IN-G1",
				Group:     "G1",
				Title:     "In group",
				Objective: "o",
				State:     gemara.LifecycleActive,
			},
			{
				Id:        "ORPHAN",
				Group:     "not-listed",
				Title:     "Orphan",
				Objective: "o2",
				State:     gemara.LifecycleActive,
			},
		},
	}

	out, err := CatalogToMarkdown(catalog)
	require.NoError(t, err)
	s := string(out)

	assert.Contains(t, s, "## Ungrouped")
	assert.Contains(t, s, "### ORPHAN")
	assert.Contains(t, s, "- [Ungrouped](#ungrouped)")
	assert.Contains(t, s, fmt.Sprintf("  - [ORPHAN: Orphan](#%s)", markdownAnchor("ORPHAN: Orphan")))
}

func TestCatalogToMarkdown_extendsImportsReplacedBy(t *testing.T) {
	// test-data catalogs do not combine extends/imports/replaced-by; keep a focused synthetic case.
	catalog := &gemara.ControlCatalog{
		Metadata: gemara.Metadata{
			Id:            "full",
			Type:          gemara.ControlCatalogArtifact,
			GemaraVersion: "1.0",
			Description:   "Full metadata.",
			Author:        gemara.Actor{Name: "Author", Type: gemara.Human},
			MappingReferences: []gemara.MappingReference{
				{Id: "ext", Title: "External", Version: "1", Url: "https://example.com"},
			},
		},
		Title: "Complex",
		Extends: []gemara.ArtifactMapping{
			{ReferenceId: "base", Remarks: "extends base"},
		},
		Imports: []gemara.MultiEntryMapping{
			{
				ReferenceId: "imp",
				Remarks:     "imported",
				Entries: []gemara.ArtifactMapping{
					{ReferenceId: "e1", Remarks: "r1"},
				},
			},
		},
		Groups: []gemara.Group{
			{Id: "G", Title: "Group", Description: "gd"},
		},
		Controls: []gemara.Control{
			{
				Id:        "C1",
				Group:     "G",
				Title:     "Control one",
				Objective: "Obj.",
				State:     gemara.LifecycleActive,
				Guidelines: []gemara.MultiEntryMapping{
					{
						ReferenceId: "GL",
						Entries: []gemara.ArtifactMapping{
							{ReferenceId: "sub", Remarks: "nested"},
						},
					},
				},
				Threats: []gemara.MultiEntryMapping{
					{ReferenceId: "TH", Remarks: "threat note"},
				},
				AssessmentRequirements: []gemara.AssessmentRequirement{
					{
						Id:             "C1.1",
						Text:           "Must do X.",
						Applicability:  []string{"a", "b"},
						Recommendation: "Consider Y.",
						State:          gemara.LifecycleActive,
					},
					{
						Id:             "C1.2",
						Text:           "Retired requirement text only.",
						Applicability:  []string{"retired-app"},
						Recommendation: "Retired recommendation should be hidden.",
						State:          gemara.LifecycleRetired,
					},
				},
			},
			{
				Id:        "C-HIDDEN",
				Group:     "G",
				Title:     "Not exported",
				Objective: "Omit from markdown.",
				State:     gemara.LifecycleDeprecated,
				ReplacedBy: &gemara.EntryMapping{
					EntryId: "C1",
					Remarks: "use C1",
				},
			},
		},
	}

	out, err := CatalogToMarkdown(catalog)
	require.NoError(t, err)
	s := string(out)

	assert.Contains(t, s, "## Extends")
	assert.Contains(t, s, "- base — extends base")
	assert.Contains(t, s, "## Imports")
	assert.Contains(t, s, "**imp**")
	assert.Contains(t, s, "### Mapping References")
	assert.NotContains(t, s, "### C-HIDDEN")
	assert.Contains(t, s, "### C1: Control one")
	assert.Contains(t, s, "#### Guidelines")
	assert.Contains(t, s, "#### Threats")
	assert.Contains(t, s, "**Applicability:** a, b")
	assert.Contains(t, s, "**Recommendation**")
	assert.Contains(t, s, "Consider Y.")
	assert.Contains(t, s, "#### C1.2")
	assert.Contains(t, s, "Retired requirement text only.")
	assert.NotContains(t, s, "retired-app")
	assert.NotContains(t, s, "Retired recommendation should be hidden.")
}

func TestControlCatalogConverter_ToMarkdown(t *testing.T) {
	catalog := loadControlCatalogFromTestData(t, "good-ccc.yaml")
	out, err := ControlCatalog(catalog).ToMarkdown()
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(out), "# FINOS Cloud Control Catalog"))
}

func TestCatalogToMarkdown_lineEnding(t *testing.T) {
	catalog := loadControlCatalogFromTestData(t, "good-ccc.yaml")
	out, err := CatalogToMarkdown(catalog, WithLineEnding("\r\n"))
	require.NoError(t, err)
	assert.Contains(t, string(out), "\r\n")
}

func TestCatalogToMarkdown_withoutMetadata(t *testing.T) {
	catalog := loadControlCatalogFromTestData(t, "good-ccc.yaml")
	out, err := CatalogToMarkdown(catalog, WithMetadata(false))
	require.NoError(t, err)
	s := string(out)

	assert.NotContains(t, s, "_"+catalog.Title+"_ is a Gemara")
	assert.NotContains(t, s, "### Description")
	assert.NotContains(t, s, "### Mapping References")
	assert.Contains(t, s, "## Table of contents")
	assert.Contains(t, s, "### CCC.C01:")
}

func TestCatalogToMarkdown_applicabilityMatrix(t *testing.T) {
	catalog := &gemara.ControlCatalog{
		Metadata: gemara.Metadata{
			Id:          "m",
			Type:        gemara.ControlCatalogArtifact,
			Description: "d",
			Author:      gemara.Actor{Name: "a", Type: gemara.Human},
			ApplicabilityGroups: []gemara.Group{
				{Id: "L1", Title: "Level 1"},
				{Id: "L2", Title: "Level 2"},
			},
		},
		Title: "Matrix Test",
		Groups: []gemara.Group{
			{Id: "G", Title: "Group"},
		},
		Controls: []gemara.Control{
			{
				Id:        "C-A",
				Group:     "G",
				Title:     "Alpha",
				Objective: "o1",
				State:     gemara.LifecycleActive,
				AssessmentRequirements: []gemara.AssessmentRequirement{
					{Id: "C-A.1", Text: "t", State: gemara.LifecycleActive, Applicability: []string{"L1"}},
				},
			},
			{
				Id:        "C-B",
				Group:     "G",
				Title:     "Beta",
				Objective: "o2",
				State:     gemara.LifecycleActive,
				AssessmentRequirements: []gemara.AssessmentRequirement{
					{Id: "C-B.1", Text: "t", State: gemara.LifecycleActive, Applicability: []string{"L1", "L2"}},
				},
			},
		},
	}

	out, err := CatalogToMarkdown(catalog, WithApplicabilityMatrix(true), WithMetadata(false), WithTOC(false))
	require.NoError(t, err)
	s := string(out)

	assert.Contains(t, s, "## Applicability coverage")
	assert.Contains(t, s, "| Control | Level 1 | Level 2 |")
	assert.Contains(t, s, "| **C-A**: Alpha |X||")
	assert.Contains(t, s, "| **C-B**: Beta |X|X|")
}

func TestCatalogToMarkdown_applicabilityMatrix_offByDefault(t *testing.T) {
	catalog := loadControlCatalogFromTestData(t, "good-ccc.yaml")
	out, err := CatalogToMarkdown(catalog)
	require.NoError(t, err)
	assert.NotContains(t, string(out), "## Applicability coverage")
}

func TestCatalogToMarkdown_lexiconAutolink(t *testing.T) {
	catalog := &gemara.ControlCatalog{
		Metadata: gemara.Metadata{
			Id:            "m",
			Type:          gemara.ControlCatalogArtifact,
			Description:   "d",
			Author:        gemara.Actor{Name: "Author", Type: gemara.Human},
			GemaraVersion: "1.0",
			Lexicon:       &gemara.ArtifactMapping{ReferenceId: "lex"},
			MappingReferences: []gemara.MappingReference{
				{Id: "lex", Title: "Lex", Version: "1", Url: lexiconFileURL(t, "lexicon_golden.yaml")},
			},
		},
		Title:  "Lex test",
		Groups: []gemara.Group{{Id: "G", Title: "Group"}},
		Controls: []gemara.Control{
			{
				Id:        "C1",
				Group:     "G",
				Title:     "Uses Example Term in title",
				Objective: "Objective mentions sample term and Second Term.",
				State:     gemara.LifecycleActive,
				Guidelines: []gemara.MultiEntryMapping{
					{ReferenceId: "Example Term", Entries: []gemara.ArtifactMapping{{ReferenceId: "x"}}},
				},
				AssessmentRequirements: []gemara.AssessmentRequirement{
					{Id: "C1.1", Text: "See Second Term.", Recommendation: "Read Example Term.", State: gemara.LifecycleActive},
				},
			},
		},
	}

	out, err := CatalogToMarkdown(catalog, WithLexiconAutolink(true), WithTOC(false))
	require.NoError(t, err)
	s := string(out)

	assert.Contains(t, s, "## Lexicon")
	assert.Contains(t, s, "### Example Term")
	assert.Contains(t, s, "### Second Term")
	assert.Contains(t, s, "[Example Term][Example Term]")
	assert.Contains(t, s, "[Second Term][Second Term]")
	assert.Contains(t, s, "[sample term][Example Term]")
	assert.Contains(t, s, "[Example Term]: #example-term")
}

func TestCatalogToMarkdown_lexiconAutolink_offByDefault(t *testing.T) {
	catalog := &gemara.ControlCatalog{
		Metadata: gemara.Metadata{
			Id:          "m",
			Type:        gemara.ControlCatalogArtifact,
			Description: "d",
			Author:      gemara.Actor{Name: "a", Type: gemara.Human},
			Lexicon:     &gemara.ArtifactMapping{ReferenceId: "lex"},
			MappingReferences: []gemara.MappingReference{
				{Id: "lex", Title: "L", Version: "1", Url: lexiconFileURL(t, "lexicon_golden.yaml")},
			},
		},
		Title:  "x",
		Groups: []gemara.Group{{Id: "G", Title: "G"}},
		Controls: []gemara.Control{
			{Id: "C", Group: "G", Title: "T", Objective: "Example Term", State: gemara.LifecycleActive},
		},
	}
	out, err := CatalogToMarkdown(catalog, WithTOC(false))
	require.NoError(t, err)
	assert.NotContains(t, string(out), "## Lexicon")
}

func TestCatalogToMarkdown_inlineLexicon(t *testing.T) {
	catalog := &gemara.ControlCatalog{
		Metadata: gemara.Metadata{
			Id:            "m",
			Type:          gemara.ControlCatalogArtifact,
			Description:   "d",
			Author:        gemara.Actor{Name: "a", Type: gemara.Human},
			GemaraVersion: "1.0",
		},
		Title:  "T",
		Groups: []gemara.Group{{Id: "G", Title: "G"}},
		Controls: []gemara.Control{
			{Id: "C", Group: "G", Title: "Widget talk", Objective: "widgets", State: gemara.LifecycleActive},
		},
	}
	out, err := CatalogToMarkdown(catalog, WithTOC(false), WithInlineLexicon([]InlineLexiconTerm{
		{Term: "Widget", Definition: "A widget."},
	}))
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, "## Lexicon")
	assert.Contains(t, s, "### Widget")
	assert.Contains(t, s, "[widgets][Widget]")
}

func TestCatalogToMarkdown_lexiconAutolink_resolveError(t *testing.T) {
	catalog := &gemara.ControlCatalog{
		Metadata: gemara.Metadata{
			Id:          "m",
			Type:        gemara.ControlCatalogArtifact,
			Description: "d",
			Author:      gemara.Actor{Name: "a", Type: gemara.Human},
			Lexicon:     &gemara.ArtifactMapping{ReferenceId: "missing"},
		},
		Title:    "x",
		Groups:   []gemara.Group{{Id: "G", Title: "G"}},
		Controls: []gemara.Control{{Id: "C", Group: "G", Title: "T", Objective: "o", State: gemara.LifecycleActive}},
	}
	_, err := CatalogToMarkdown(catalog, WithLexiconAutolink(true))
	require.Error(t, err)
}
