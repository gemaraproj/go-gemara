package gemaraconv

import (
	"context"
	"fmt"
	"testing"

	"github.com/gemaraproj/go-gemara"
	"github.com/gemaraproj/go-gemara/gemaraconv/markdown"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGuidanceToMarkdown_goodAIGFYAML(t *testing.T) {
	guidance, err := goodAIGFExample()
	require.NoError(t, err)

	out, err := GuidanceToMarkdown(context.Background(), guidance)
	require.NoError(t, err)
	s := string(out)

	require.NotEmpty(t, guidance.Groups)
	group0 := guidance.Groups[0]
	var g0 gemara.Guideline
	for _, g := range guidance.Guidelines {
		if g.Group == group0.Id && (g0.Id == "" || g.Id < g0.Id) {
			g0 = g
		}
	}
	require.NotEmpty(t, g0.Id)
	numStatements := 0
	for _, g := range guidance.Guidelines {
		numStatements += len(g.Statements)
	}

	assert.Contains(t, s, fmt.Sprintf("# %s\n\nVersion: %s", guidance.Title, guidance.Metadata.Version))
	assert.Contains(t, s, "_"+guidance.Title+"_ is a Gemara")
	assert.Contains(t, s, fmt.Sprintf("_Summary: %s with %d guideline(s), %d statement(s)._", guidance.GuidanceType, len(guidance.Guidelines), numStatements))
	assert.Contains(t, s, guidance.FrontMatter)
	assert.Contains(t, s, "## Table of contents")
	assert.Contains(t, s, fmt.Sprintf("- [%s](#%s)", group0.Title, markdown.Anchor(group0.Id)))
	assert.Contains(t, s, fmt.Sprintf("  - [%s: %s](#%s)", g0.Id, g0.Title, markdown.Anchor(g0.Id+": "+g0.Title)))
	assert.Contains(t, s, fmt.Sprintf("## %s: %s", group0.Id, group0.Title))
	assert.Contains(t, s, fmt.Sprintf("### %s: %s", g0.Id, g0.Title))
	assert.Contains(t, s, "**Objective**\n\n"+g0.Objective)

	// The fixture's first guideline carries statements with multi-line
	// recommendations, a principles mapping, a rationale and see-also ids.
	var rich gemara.Guideline
	for _, g := range guidance.Guidelines {
		if len(g.Statements) > 0 {
			rich = g
			break
		}
	}
	require.NotEmpty(t, rich.Id)
	st := rich.Statements[0]
	assert.Contains(t, s, fmt.Sprintf("#### %s: %s\n\n%s", st.Id, st.Title, st.Text))
	assert.Contains(t, s, "**Recommendations**")
	assert.NotContains(t, s, "\nObjectives:", "a multi-line recommendation must stay inside its list item")
	assert.Contains(t, s, "  Objectives:")
	assert.Contains(t, s, "**Rationale**\n\n"+rich.Rationale.Importance)
	assert.Contains(t, s, "#### Principles")
	for _, id := range rich.SeeAlso {
		assert.Contains(t, s, fmt.Sprintf("[%s](#", id), "see-also %s should link to its heading", id)
	}
	assert.NotContains(t, s, "\n\n\n")
}

func TestGuidanceToMarkdown_options(t *testing.T) {
	guidance := gemara.GuidanceCatalog{
		Title: "G",
		Metadata: gemara.Metadata{
			Version:           "1",
			MappingReferences: []gemara.MappingReference{{Id: "imp", Title: "Imported Guidance", Version: "2", Url: "https://example.com/imported"}},
		},
		GuidanceType: gemara.GuidanceStandard,
		// CRLF in source content must not leave triple newlines behind (Windows checkouts).
		FrontMatter: "Intro\r\n\r\n",
		Extends:     []gemara.ArtifactMapping{{ReferenceId: "base", Remarks: "extends base"}},
		Imports:     []gemara.MultiEntryMapping{{ReferenceId: "imp", Remarks: "imported", Entries: []gemara.ArtifactMapping{{ReferenceId: "G-9", Remarks: "from imp"}}}},
		Guidelines: []gemara.Guideline{
			{Id: "G-2", Title: "Second", Objective: "o", SeeAlso: []string{"G-1", "NOPE"}},
			{Id: "G-1", Title: "First", Objective: "o", State: gemara.LifecycleRetired},
		},
		Exemptions: []gemara.Exemption{{Description: "Labs", Reason: "No prod data"}},
	}
	out, err := GuidanceToMarkdown(context.Background(), guidance, WithTOC(false), WithMetadata(false), WithLineEnding("\r\n"))
	require.NoError(t, err)
	s := string(out)
	assert.NotContains(t, s, "Table of contents")
	assert.NotContains(t, s, "_Summary:")
	assert.Contains(t, s, "\r\n")
	assert.NotContains(t, s, "\r\n\r\n\r\n")
	assert.Contains(t, s, "## Extends\r\n\r\nThis document builds upon all of the guidelines and applicability groups detailed in:\r\n\r\n- base — extends base")
	assert.Contains(t, s, "## Imports\r\n\r\nThe following guidelines are imported from external catalogs. All relevant details for these guidelines can be found in the respective catalog's source.")
	assert.Contains(t, s, "### imp: Imported Guidance\r\n\r\nimported\r\n\r\n**Source:** [https://example.com/imported](https://example.com/imported)\r\n\r\n#### G-9 — from imp")
	assert.Contains(t, s, "## Ungrouped")
	assert.NotContains(t, s, "### G-1", "retired guidelines are omitted")
	assert.Contains(t, s, "**See also:** G-1, NOPE", "see-also to an absent heading stays plain text")
	assert.Contains(t, s, "## Exemptions\r\n\r\n- **Labs** — No prod data")

	viaConverter, err := GuidanceCatalog(guidance).ToMarkdown(context.Background(), WithTOC(false), WithMetadata(false), WithLineEnding("\r\n"))
	require.NoError(t, err)
	assert.Equal(t, out, viaConverter)
}
