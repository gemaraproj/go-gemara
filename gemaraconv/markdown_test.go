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
	groupControls := append([]gemara.Control(nil), catalog.GetControlsForGroup(group0.Id)...)
	require.NotEmpty(t, groupControls)
	sort.Slice(groupControls, func(i, j int) bool { return groupControls[i].Id < groupControls[j].Id })
	c0 := groupControls[0]
	require.NotEmpty(t, c0.AssessmentRequirements)
	ars := append([]gemara.AssessmentRequirement(nil), c0.AssessmentRequirements...)
	sort.Slice(ars, func(i, j int) bool { return ars[i].Id < ars[j].Id })
	ar0 := ars[0]

	numARs := 0
	for _, c := range catalog.Controls {
		numARs += len(c.AssessmentRequirements)
	}

	assert.Contains(t, s, fmt.Sprintf("# %s", catalog.Title))
	assert.Contains(t, s, fmt.Sprintf("| **ID** | %s |", catalog.Metadata.Id))
	assert.Contains(t, s, "## Table of contents")
	assert.Contains(t, s, fmt.Sprintf("- [%s](#%s)", group0.Title, markdownAnchor(group0.Id)))
	assert.Contains(t, s, fmt.Sprintf("  - [%s — %s](#%s)", c0.Id, c0.Title, markdownAnchor(c0.Id)))
	assert.Contains(t, s, fmt.Sprintf("## %s", group0.Id))
	assert.Contains(t, s, fmt.Sprintf("**%s**", group0.Title))
	assert.Contains(t, s, fmt.Sprintf("### %s", c0.Id))
	assert.Contains(t, s, fmt.Sprintf("#### %s", ar0.Id))
	assert.Contains(t, s, "#### Guidelines")
	assert.Contains(t, s, "#### Threats")
	assert.Contains(t, s, fmt.Sprintf("_Summary: %d control(s), %d assessment requirement(s)._", len(catalog.Controls), numARs))
}

func TestCatalogToMarkdown_goodOSPSYAML(t *testing.T) {
	catalog := loadControlCatalogFromTestData(t, "good-osps.yml")

	out, err := CatalogToMarkdown(catalog, WithTOC(false))
	require.NoError(t, err)
	s := string(out)

	assert.Contains(t, s, "# Open Source Project Security Baseline")
	assert.Contains(t, s, "| **ID** | OSPS-B |")
	assert.NotContains(t, s, "## Table of contents")
	assert.Greater(t, len(out), 5000)
	assert.Contains(t, s, "## Metadata")
	assert.Contains(t, s, "### Mapping references")
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
	assert.Contains(t, s, "  - [ORPHAN — Orphan](#orphan)")
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
				State:     gemara.LifecycleDeprecated,
				ReplacedBy: &gemara.EntryMapping{
					EntryId: "C2",
					Remarks: "use C2",
				},
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
	assert.Contains(t, s, "### Mapping references")
	assert.Contains(t, s, "**State:** Deprecated")
	assert.Contains(t, s, "**Replaced by:** `C2`")
	assert.Contains(t, s, "#### Guidelines")
	assert.Contains(t, s, "#### Threats")
	assert.Contains(t, s, "**Applicability:** a, b")
	assert.Contains(t, s, "**Recommendation**")
	assert.Contains(t, s, "Consider Y.")
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
