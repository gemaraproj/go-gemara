package gemaraconv

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gemaraproj/go-gemara"
	"github.com/gemaraproj/go-gemara/fetcher"
	"github.com/gemaraproj/go-gemara/gemaraconv/markdown"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadThreatCatalogFromTestData(t *testing.T, name string) *gemara.ThreatCatalog {
	t.Helper()
	c, err := gemara.Load[gemara.ThreatCatalog](context.Background(), &fetcher.File{}, filepath.Join("..", "test-data", name))
	require.NoError(t, err, "load %s", name)
	return c
}

func TestThreatCatalogToMarkdown_nil(t *testing.T) {
	_, err := ThreatCatalogToMarkdown(context.Background(), nil)
	require.Error(t, err)
}

func TestThreatCatalogToMarkdown_goodThreatCatalogYAML(t *testing.T) {
	catalog := loadThreatCatalogFromTestData(t, "good-threat-catalog.yaml")

	out, err := ThreatCatalogToMarkdown(context.Background(), catalog)
	require.NoError(t, err)
	s := string(out)

	assert.Contains(t, s, "# Example Threat Catalog")
	assert.Contains(t, s, "Version: 1.0.0")
	assert.Contains(t, s, "_Example Threat Catalog_ is a Gemara")
	assert.Contains(t, s, "## Table of contents")
	assert.Contains(t, s, "_Summary: 2 threat(s)._")

	// Groups — stride-s (Spoofing) has no threats so it is omitted; stride-t (Tampering) does.
	require.True(t, len(catalog.Groups) >= 2)
	g1 := catalog.Groups[1] // Tampering
	assert.Contains(t, s, fmt.Sprintf("## %s: %s", g1.Id, g1.Title))
	assert.Contains(t, s, fmt.Sprintf("- [%s](#%s)", g1.Title, markdown.Anchor(g1.Id)))

	// Threats
	assert.Contains(t, s, "### THREAT-001: Exploitation of Vulnerable Container Images")
	assert.Contains(t, s, "### THREAT-002: Host System Compromise via Container Escape")
	assert.Contains(t, s, "#### Capabilities")
	assert.Contains(t, s, "#### Vectors")
	assert.Contains(t, s, "#### Actors")
	assert.Contains(t, s, "| External Attacker | Human |")

	// Mapping references
	assert.Contains(t, s, "### Mapping References")
}

func TestThreatCatalogToMarkdown_withoutMetadata(t *testing.T) {
	catalog := loadThreatCatalogFromTestData(t, "good-threat-catalog.yaml")

	out, err := ThreatCatalogToMarkdown(context.Background(), catalog, WithMetadata(false))
	require.NoError(t, err)
	s := string(out)

	assert.NotContains(t, s, "_Example Threat Catalog_ is a Gemara")
	assert.NotContains(t, s, "### Description")
	assert.NotContains(t, s, "### Mapping References")
	assert.Contains(t, s, "### THREAT-001")
}

func TestThreatCatalogToMarkdown_withoutTOC(t *testing.T) {
	catalog := loadThreatCatalogFromTestData(t, "good-threat-catalog.yaml")

	out, err := ThreatCatalogToMarkdown(context.Background(), catalog, WithTOC(false))
	require.NoError(t, err)
	s := string(out)

	assert.NotContains(t, s, "## Table of contents")
	assert.Contains(t, s, "### THREAT-001")
}

func TestThreatCatalogToMarkdown_synthetic(t *testing.T) {
	catalog := &gemara.ThreatCatalog{
		Metadata: gemara.Metadata{
			Id:            "synth",
			Type:          gemara.ThreatCatalogArtifact,
			GemaraVersion: "1.0",
			Description:   "Synthetic threat catalog for test.",
			Author:        gemara.Actor{Name: "Tester", Type: gemara.Human},
			MappingReferences: []gemara.MappingReference{
				{Id: "cap-ref", Title: "Caps", Version: "1", Url: "https://example.com/caps"},
				{Id: "vec-ref", Title: "Vecs", Version: "1"},
				{Id: "imp-ref", Title: "Imported Threats", Version: "2", Url: "https://example.com/imported"},
			},
		},
		Title: "Synthetic Threats",
		Extends: []gemara.ArtifactMapping{
			{ReferenceId: "base-threats", Remarks: "base catalog"},
		},
		Imports: []gemara.MultiEntryMapping{
			{
				ReferenceId: "imp-ref",
				Remarks:     "external threats",
				Entries: []gemara.ArtifactMapping{
					{ReferenceId: "EXT-T1", Remarks: "first imported"},
				},
			},
		},
		Groups: []gemara.Group{
			{Id: "G1", Title: "Group One", Description: "first group"},
		},
		Threats: []gemara.Threat{
			{
				Id:          "T1",
				Title:       "Threat Alpha",
				Description: "Alpha description.",
				Group:       "G1",
				Capabilities: []gemara.MultiEntryMapping{
					{
						ReferenceId: "cap-ref",
						Entries:     []gemara.ArtifactMapping{{ReferenceId: "CAP-1"}},
					},
				},
				Vectors: []gemara.MultiEntryMapping{
					{
						ReferenceId: "vec-ref",
						Entries:     []gemara.ArtifactMapping{{ReferenceId: "VEC-1", Remarks: "primary vector"}},
					},
				},
				Actors: []gemara.Actor{
					{Id: "attacker", Name: "Attacker", Type: gemara.Human},
					{Id: "bot", Name: "Automated Bot", Type: gemara.Software},
				},
			},
			{
				Id:          "T-ORPHAN",
				Title:       "Orphan Threat",
				Description: "No matching group.",
				Group:       "missing-group",
				Capabilities: []gemara.MultiEntryMapping{
					{ReferenceId: "cap-ref", Entries: []gemara.ArtifactMapping{{ReferenceId: "CAP-2"}}},
				},
			},
		},
	}

	out, err := ThreatCatalogToMarkdown(context.Background(), catalog)
	require.NoError(t, err)
	s := string(out)

	// Title and metadata
	assert.Contains(t, s, "# Synthetic Threats")
	assert.Contains(t, s, "_Synthetic Threats_ is a Gemara")
	assert.Contains(t, s, "_Summary: 2 threat(s)._")

	// Extends
	assert.Contains(t, s, "## Extends")
	assert.Contains(t, s, "- base-threats — base catalog")

	// Imports (new header style)
	assert.Contains(t, s, "## Imports")
	assert.Contains(t, s, "### imp-ref: Imported Threats")
	assert.Contains(t, s, "external threats")
	assert.Contains(t, s, "**Source:** [https://example.com/imported](https://example.com/imported)")
	assert.Contains(t, s, "#### EXT-T1 — first imported")

	// Group
	assert.Contains(t, s, "## G1: Group One")
	assert.Contains(t, s, "first group")

	// Threat in group
	assert.Contains(t, s, "### T1: Threat Alpha")
	assert.Contains(t, s, "Alpha description.")
	assert.Contains(t, s, "#### Capabilities")
	assert.Contains(t, s, "**cap-ref**")
	assert.Contains(t, s, "CAP-1")
	assert.Contains(t, s, "#### Vectors")
	assert.Contains(t, s, "VEC-1 — primary vector")
	assert.Contains(t, s, "#### Actors")
	assert.Contains(t, s, "| Attacker | Human |")
	assert.Contains(t, s, "| Automated Bot | Software |")

	// Ungrouped
	assert.Contains(t, s, "## Ungrouped")
	assert.Contains(t, s, "### T-ORPHAN: Orphan Threat")

	// Mapping references
	assert.Contains(t, s, "### Mapping References")
	assert.Contains(t, s, "**cap-ref**")

	// TOC
	assert.Contains(t, s, "## Table of contents")
	assert.Contains(t, s, fmt.Sprintf("- [Group One](#%s)", markdown.Anchor("G1")))
	assert.Contains(t, s, fmt.Sprintf("  - [T1: Threat Alpha](#%s)", markdown.Anchor("T1: Threat Alpha")))
}

func TestThreatCatalogToMarkdown_lineEnding(t *testing.T) {
	catalog := loadThreatCatalogFromTestData(t, "good-threat-catalog.yaml")
	out, err := ThreatCatalogToMarkdown(context.Background(), catalog, WithLineEnding("\r\n"))
	require.NoError(t, err)
	assert.Contains(t, string(out), "\r\n")
}

func TestThreatCatalogConverter_ToMarkdown(t *testing.T) {
	catalog := loadThreatCatalogFromTestData(t, "good-threat-catalog.yaml")
	out, err := ThreatCatalog(catalog).ToMarkdown(context.Background())
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(out), "# Example Threat Catalog"))
}
