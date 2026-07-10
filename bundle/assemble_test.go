// SPDX-License-Identifier: Apache-2.0

package bundle

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/gemaraproj/go-gemara"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mapFetcher is a test double that serves content from a static map.
type mapFetcher map[string][]byte

func (m mapFetcher) Fetch(_ context.Context, source string) (io.ReadCloser, error) {
	data, ok := m[source]
	if !ok {
		return nil, fmt.Errorf("not found: %s", source)
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	data, err := yaml.Marshal(v)
	require.NoError(t, err)
	return data
}

var testAuthor = gemara.Actor{
	Id:   "test-author",
	Name: "Test Author",
	Type: gemara.Human,
}

func testControlCatalog(id string, refs []gemara.MappingReference, extends []gemara.ArtifactMapping, imports []gemara.MultiEntryMapping) gemara.Catalog {
	return testControlCatalogWithVersion(id, "", refs, extends, imports)
}

func testControlCatalogWithVersion(id, version string, refs []gemara.MappingReference, extends []gemara.ArtifactMapping, imports []gemara.MultiEntryMapping) gemara.Catalog {
	return gemara.Catalog{
		Title: "Test Catalog",
		Metadata: gemara.Metadata{
			Id:                id,
			Type:              gemara.ControlCatalogArtifact,
			GemaraVersion:     "1.0.0",
			Version:           version,
			Description:       "test catalog",
			Author:            testAuthor,
			MappingReferences: refs,
		},
		Extends: extends,
		Imports: imports,
	}
}

func testGuidanceCatalog(id string) gemara.GuidanceCatalog {
	return gemara.GuidanceCatalog{
		Title: "External Guidance",
		Metadata: gemara.Metadata{
			Id:            id,
			Type:          gemara.GuidanceCatalogArtifact,
			GemaraVersion: "1.0.0",
			Description:   "imported guidance",
			Author:        testAuthor,
		},
	}
}

func TestAssembler_Assemble(t *testing.T) {
	guidance := testGuidanceCatalog("ext-guidance")

	tests := []struct {
		name    string
		fetcher mapFetcher
		source  File
		wantErr string
		check   func(t *testing.T, b *Bundle)
	}{
		{
			name: "resolves imports with manifest artifacts",
			fetcher: mapFetcher{
				"https://example.com/guidance.yaml": mustMarshal(t, guidance),
			},
			source: File{
				Name: "controls.yaml",
				Data: mustMarshal(t, testControlCatalog("test-cat",
					[]gemara.MappingReference{
						{Id: "EXT-GUIDE", Title: "External Guidance", Version: "1.0", Url: "https://example.com/guidance.yaml"},
						{Id: "LOCAL-REF", Title: "Local Only", Version: "1.0"},
					},
					nil,
					[]gemara.MultiEntryMapping{
						{ReferenceId: "EXT-GUIDE", Entries: []gemara.ArtifactMapping{{ReferenceId: "G1"}}},
					},
				)),
			},
			check: func(t *testing.T, b *Bundle) {
				assert.Equal(t, "1", b.Manifest.BundleVersion)
				assert.Equal(t, "v1.0.0", b.Manifest.GemaraVersion)
				assert.Equal(t, "controls.yaml", b.Source.Name)
				require.Len(t, b.Imports, 1)
				assert.Equal(t, "guidance.yaml", b.Imports[0].Name)

				require.Len(t, b.Manifest.Artifacts, 2)
				assert.Equal(t, Artifact{Name: "controls.yaml", Type: "ControlCatalog", ID: "test-cat", Role: roleArtifact, Dependencies: []string{"guidance.yaml"}}, b.Manifest.Artifacts[0])
				assert.Equal(t, Artifact{Name: "guidance.yaml", Type: "GuidanceCatalog", ID: "ext-guidance", Role: roleImport}, b.Manifest.Artifacts[1])
			},
		},
		{
			name:    "skips mapping ref without URL",
			fetcher: mapFetcher{},
			source: File{
				Name: "a.yaml",
				Data: mustMarshal(t, testControlCatalog("test-cat",
					[]gemara.MappingReference{{Id: "NO-URL", Title: "No URL", Version: "1.0"}},
					nil,
					[]gemara.MultiEntryMapping{
						{ReferenceId: "NO-URL", Entries: []gemara.ArtifactMapping{{ReferenceId: "x"}}},
					},
				)),
			},
			check: func(t *testing.T, b *Bundle) {
				assert.Nil(t, b.Imports)
			},
		},
		{
			name: "resolves extends references",
			fetcher: mapFetcher{
				"https://example.com/base.yaml": mustMarshal(t, testControlCatalog("base", nil, nil, nil)),
			},
			source: File{
				Name: "child.yaml",
				Data: mustMarshal(t, testControlCatalog("child",
					[]gemara.MappingReference{{Id: "BASE", Title: "Base Catalog", Version: "1.0", Url: "https://example.com/base.yaml"}},
					[]gemara.ArtifactMapping{{ReferenceId: "BASE", Remarks: "builds upon base"}},
					nil,
				)),
			},
			check: func(t *testing.T, b *Bundle) {
				require.Len(t, b.Imports, 1)
				assert.Equal(t, "base.yaml", b.Imports[0].Name)
			},
		},
		{
			name: "resolves both extends and imports",
			fetcher: mapFetcher{
				"https://example.com/base.yaml":  mustMarshal(t, testControlCatalog("base", nil, nil, nil)),
				"https://example.com/guide.yaml": mustMarshal(t, testGuidanceCatalog("guide")),
			},
			source: File{
				Name: "c.yaml",
				Data: mustMarshal(t, testControlCatalog("combined",
					[]gemara.MappingReference{
						{Id: "BASE", Title: "Base", Version: "1.0", Url: "https://example.com/base.yaml"},
						{Id: "GUIDE", Title: "Guide", Version: "1.0", Url: "https://example.com/guide.yaml"},
					},
					[]gemara.ArtifactMapping{{ReferenceId: "BASE"}},
					[]gemara.MultiEntryMapping{
						{ReferenceId: "GUIDE", Entries: []gemara.ArtifactMapping{{ReferenceId: "G1"}}},
					},
				)),
			},
			check: func(t *testing.T, b *Bundle) {
				require.Len(t, b.Imports, 2)
				names := importNames(b)
				assert.True(t, names["base.yaml"], "extends dependency should be assembled")
				assert.True(t, names["guide.yaml"], "import dependency should be assembled")
			},
		},
		{
			name: "policy imports catalogs and guidance",
			fetcher: mapFetcher{
				"https://example.com/controls.yaml": mustMarshal(t, testControlCatalog("ctrl", nil, nil, nil)),
				"https://example.com/guidance.yaml": mustMarshal(t, testGuidanceCatalog("guide")),
			},
			source: File{
				Name: "policy.yaml",
				Data: mustMarshal(t, gemara.Policy{
					Title: "Org Policy",
					Metadata: gemara.Metadata{
						Id: "org-policy", Type: gemara.PolicyArtifact, GemaraVersion: "1.0.0",
						Description: "org-wide policy", Author: testAuthor,
						MappingReferences: []gemara.MappingReference{
							{Id: "CTRL", Title: "Control Catalog", Version: "1.0", Url: "https://example.com/controls.yaml"},
							{Id: "GUIDE", Title: "Guidance Doc", Version: "1.0", Url: "https://example.com/guidance.yaml"},
						},
					},
					Contacts: gemara.RACI{
						Responsible: []gemara.Contact{{Name: "R"}},
						Accountable: []gemara.Contact{{Name: "A"}},
					},
					Scope: gemara.Scope{In: gemara.Dimensions{Technologies: []string{"cloud"}}},
					Imports: gemara.Imports{
						Catalogs: []gemara.CatalogImport{{ReferenceId: "CTRL"}},
						Guidance: []gemara.GuidanceImport{{ReferenceId: "GUIDE"}},
					},
					Adherence: gemara.Adherence{
						EvaluationMethods: []gemara.AcceptedMethod{
							{Id: "em1", Type: gemara.MethodGate, Mode: gemara.ModeAutomated, Required: true},
						},
					},
				}),
			},
			check: func(t *testing.T, b *Bundle) {
				require.Len(t, b.Imports, 2)
				names := importNames(b)
				assert.True(t, names["controls.yaml"])
				assert.True(t, names["guidance.yaml"])
			},
		},
		{
			name:    "no imports produces single artifact",
			fetcher: mapFetcher{},
			source:  File{Name: "e.yaml", Data: mustMarshal(t, testControlCatalog("e", nil, nil, nil))},
			check: func(t *testing.T, b *Bundle) {
				assert.Nil(t, b.Imports)
				assert.Equal(t, "e.yaml", b.Source.Name)
				require.Len(t, b.Manifest.Artifacts, 1)
				assert.Equal(t, roleArtifact, b.Manifest.Artifacts[0].Role)
				assert.Nil(t, b.Manifest.Artifacts[0].Dependencies)
			},
		},
		{
			name:    "empty source returns error",
			fetcher: mapFetcher{},
			source:  File{},
			wantErr: "source file is required",
		},
		{
			name:    "fetch failure propagates error",
			fetcher: mapFetcher{},
			source: File{
				Name: "c.yaml",
				Data: mustMarshal(t, testControlCatalog("test-cat",
					[]gemara.MappingReference{{Id: "EXT-GUIDE", Title: "External Guidance", Version: "1.0", Url: "https://example.com/guidance.yaml"}},
					nil,
					[]gemara.MultiEntryMapping{
						{ReferenceId: "EXT-GUIDE", Entries: []gemara.ArtifactMapping{{ReferenceId: "G1"}}},
					},
				)),
			},
			wantErr: "fetching dependency",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asm := NewAssembler(tt.fetcher)
			m := Manifest{BundleVersion: "1", GemaraVersion: "v1.0.0"}

			b, err := asm.Assemble(context.Background(), m, tt.source, WithContinueOnError())
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			tt.check(t, b)
		})
	}
}

func TestAssembler_Assemble_TransitiveDeps(t *testing.T) {
	tests := []struct {
		name    string
		fetcher mapFetcher
		source  File
		check   func(t *testing.T, b *Bundle)
	}{
		{
			name: "resolves chain A -> B -> C",
			fetcher: mapFetcher{
				"https://example.com/catalog-b.yaml": mustMarshal(t, testControlCatalog("cat-b",
					[]gemara.MappingReference{{Id: "CAT-C", Title: "Catalog C", Version: "1.0", Url: "https://example.com/catalog-c.yaml"}},
					nil,
					[]gemara.MultiEntryMapping{{ReferenceId: "CAT-C", Entries: []gemara.ArtifactMapping{{ReferenceId: "x"}}}},
				)),
				"https://example.com/catalog-c.yaml": mustMarshal(t, testControlCatalog("cat-c", nil, nil, nil)),
			},
			source: File{
				Name: "catalog-a.yaml",
				Data: mustMarshal(t, testControlCatalog("cat-a",
					[]gemara.MappingReference{{Id: "CAT-B", Title: "Catalog B", Version: "1.0", Url: "https://example.com/catalog-b.yaml"}},
					nil,
					[]gemara.MultiEntryMapping{{ReferenceId: "CAT-B", Entries: []gemara.ArtifactMapping{{ReferenceId: "y"}}}},
				)),
			},
			check: func(t *testing.T, b *Bundle) {
				assert.Equal(t, "catalog-a.yaml", b.Source.Name)
				require.Len(t, b.Imports, 2, "both B and C should be assembled transitively")
				names := importNames(b)
				assert.True(t, names["catalog-b.yaml"], "direct dependency B")
				assert.True(t, names["catalog-c.yaml"], "transitive dependency C")

				artByName := artifactsByName(b)
				assert.Equal(t, []string{"catalog-b.yaml"}, artByName["catalog-a.yaml"].Dependencies)
				assert.Equal(t, []string{"catalog-c.yaml"}, artByName["catalog-b.yaml"].Dependencies)
				assert.Nil(t, artByName["catalog-c.yaml"].Dependencies)
			},
		},
		{
			name: "terminates on cycle A <-> B",
			fetcher: mapFetcher{
				"https://example.com/catalog-a.yaml": mustMarshal(t, testControlCatalog("cat-a",
					[]gemara.MappingReference{{Id: "CAT-B", Title: "Catalog B", Version: "1.0", Url: "https://example.com/catalog-b.yaml"}},
					nil,
					[]gemara.MultiEntryMapping{{ReferenceId: "CAT-B", Entries: []gemara.ArtifactMapping{{ReferenceId: "y"}}}},
				)),
				"https://example.com/catalog-b.yaml": mustMarshal(t, testControlCatalog("cat-b",
					[]gemara.MappingReference{{Id: "CAT-A", Title: "Catalog A", Version: "1.0", Url: "https://example.com/catalog-a.yaml"}},
					nil,
					[]gemara.MultiEntryMapping{{ReferenceId: "CAT-A", Entries: []gemara.ArtifactMapping{{ReferenceId: "x"}}}},
				)),
			},
			source: File{
				Name: "catalog-a.yaml",
				Data: mustMarshal(t, testControlCatalog("cat-a",
					[]gemara.MappingReference{{Id: "CAT-B", Title: "Catalog B", Version: "1.0", Url: "https://example.com/catalog-b.yaml"}},
					nil,
					[]gemara.MultiEntryMapping{{ReferenceId: "CAT-B", Entries: []gemara.ArtifactMapping{{ReferenceId: "y"}}}},
				)),
			},
			check: func(t *testing.T, b *Bundle) {
				require.Len(t, b.Imports, 2, "terminates despite cycle")
				names := importNames(b)
				assert.True(t, names["catalog-b.yaml"])
				assert.True(t, names["catalog-a.yaml"])
			},
		},
		{
			name: "diamond dependency deduplicates shared leaf",
			fetcher: mapFetcher{
				"https://example.com/catalog-b.yaml": mustMarshal(t, testControlCatalog("cat-b",
					[]gemara.MappingReference{{Id: "CAT-D", Title: "Catalog D", Version: "1.0", Url: "https://example.com/catalog-d.yaml"}},
					nil,
					[]gemara.MultiEntryMapping{{ReferenceId: "CAT-D", Entries: []gemara.ArtifactMapping{{ReferenceId: "x"}}}},
				)),
				"https://example.com/catalog-c.yaml": mustMarshal(t, testControlCatalog("cat-c",
					[]gemara.MappingReference{{Id: "CAT-D", Title: "Catalog D", Version: "1.0", Url: "https://example.com/catalog-d.yaml"}},
					nil,
					[]gemara.MultiEntryMapping{{ReferenceId: "CAT-D", Entries: []gemara.ArtifactMapping{{ReferenceId: "y"}}}},
				)),
				"https://example.com/catalog-d.yaml": mustMarshal(t, testControlCatalog("cat-d", nil, nil, nil)),
			},
			source: File{
				Name: "catalog-a.yaml",
				Data: mustMarshal(t, testControlCatalog("cat-a",
					[]gemara.MappingReference{
						{Id: "CAT-B", Title: "Catalog B", Version: "1.0", Url: "https://example.com/catalog-b.yaml"},
						{Id: "CAT-C", Title: "Catalog C", Version: "1.0", Url: "https://example.com/catalog-c.yaml"},
					},
					nil,
					[]gemara.MultiEntryMapping{
						{ReferenceId: "CAT-B", Entries: []gemara.ArtifactMapping{{ReferenceId: "x"}}},
						{ReferenceId: "CAT-C", Entries: []gemara.ArtifactMapping{{ReferenceId: "y"}}},
					},
				)),
			},
			check: func(t *testing.T, b *Bundle) {
				require.Len(t, b.Imports, 3, "B, C, D all assembled; D only once")
				names := importNames(b)
				assert.True(t, names["catalog-b.yaml"])
				assert.True(t, names["catalog-c.yaml"])
				assert.True(t, names["catalog-d.yaml"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asm := NewAssembler(tt.fetcher)
			b, err := asm.Assemble(context.Background(), Manifest{}, tt.source)
			require.NoError(t, err)
			tt.check(t, b)
		})
	}
}

func TestImportFileName(t *testing.T) {
	tests := []struct {
		refID string
		url   string
		want  string
	}{
		{"EXT", "https://example.com/guidance.yaml", "guidance.yaml"},
		{"EXT", "https://example.com/", "EXT.yaml"},
		{"EXT", "not a url ://", "EXT.yaml"},
		{"MY-REF", "file:///tmp/data.yaml", "data.yaml"},
	}
	for _, tt := range tests {
		t.Run(tt.refID+"_"+tt.url, func(t *testing.T) {
			assert.Equal(t, tt.want, importFileName(tt.refID, tt.url))
		})
	}
}

func TestAssembler_Assemble_BundleVersionDefault(t *testing.T) {
	tests := []struct {
		name          string
		manifest      Manifest
		sourceVersion string
		wantBundleVer string
	}{
		{
			name:          "defaults BundleVersion from source artifact",
			manifest:      Manifest{GemaraVersion: "v1.0.0"},
			sourceVersion: "2.3.0",
			wantBundleVer: "2.3.0",
		},
		{
			name:          "preserves explicit BundleVersion",
			manifest:      Manifest{BundleVersion: "explicit-1.0", GemaraVersion: "v1.0.0"},
			sourceVersion: "2.3.0",
			wantBundleVer: "explicit-1.0",
		},
		{
			name:          "empty artifact version produces empty BundleVersion",
			manifest:      Manifest{GemaraVersion: "v1.0.0"},
			sourceVersion: "",
			wantBundleVer: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asm := NewAssembler(mapFetcher{})
			source := File{
				Name: "controls.yaml",
				Data: mustMarshal(t, testControlCatalogWithVersion("cat-1", tt.sourceVersion, nil, nil, nil)),
			}
			b, err := asm.Assemble(context.Background(), tt.manifest, source)
			require.NoError(t, err)
			assert.Equal(t, tt.wantBundleVer, b.Manifest.BundleVersion)
			assert.Equal(t, tt.wantBundleVer, b.Version())
		})
	}
}

func importNames(b *Bundle) map[string]bool {
	names := make(map[string]bool, len(b.Imports))
	for _, imp := range b.Imports {
		names[imp.Name] = true
	}
	return names
}

func artifactsByName(b *Bundle) map[string]Artifact {
	m := make(map[string]Artifact, len(b.Manifest.Artifacts))
	for _, a := range b.Manifest.Artifacts {
		m[a.Name] = a
	}
	return m
}

func TestAssemble_MappingRefErrorByDefault(t *testing.T) {
	tests := []struct {
		name    string
		fetcher mapFetcher
		source  File
		wantErr string
	}{
		{
			name:    "url-less ref matching a known metadata.id succeeds",
			fetcher: mapFetcher{},
			source: File{
				Name: "cat.yaml",
				Data: mustMarshal(t, testControlCatalog("self-ref",
					[]gemara.MappingReference{{Id: "self-ref", Title: "Self", Version: "1.0"}},
					nil, nil,
				)),
			},
		},
		{
			name:    "url-less ref not matching any metadata.id returns error",
			fetcher: mapFetcher{},
			source: File{
				Name: "cat.yaml",
				Data: mustMarshal(t, testControlCatalog("my-cat",
					[]gemara.MappingReference{{Id: "unknown-ref", Title: "Unknown", Version: "1.0"}},
					nil, nil,
				)),
			},
			wantErr: "unmatched mapping-reference",
		},
		{
			name:    "ref with URL is not validated against metadata.ids",
			fetcher: mapFetcher{},
			source: File{
				Name: "cat.yaml",
				Data: mustMarshal(t, testControlCatalog("my-cat",
					[]gemara.MappingReference{{Id: "remote", Title: "Remote", Version: "1.0", Url: "https://example.com/remote.yaml"}},
					nil, nil,
				)),
			},
		},
		{
			name: "multiple mismatches across source and imports returns error",
			fetcher: mapFetcher{
				"https://example.com/controls.yaml": mustMarshal(t, testControlCatalog("imported-cat",
					[]gemara.MappingReference{{Id: "phantom-import", Title: "Phantom Import", Version: "1.0"}},
					nil, nil,
				)),
			},
			source: File{
				Name: "policy.yaml",
				Data: mustMarshal(t, testControlCatalog("my-policy",
					[]gemara.MappingReference{
						{Id: "CTRL", Title: "Controls", Version: "1.0", Url: "https://example.com/controls.yaml"},
						{Id: "phantom-source", Title: "Phantom Source", Version: "1.0"},
					},
					nil,
					[]gemara.MultiEntryMapping{
						{ReferenceId: "CTRL", Entries: []gemara.ArtifactMapping{{ReferenceId: "C1"}}},
					},
				)),
			},
			wantErr: "2 unmatched mapping-references",
		},
		{
			name:    "mixed URL and non-URL refs errors only for non-URL",
			fetcher: mapFetcher{},
			source: File{
				Name: "catalog.yaml",
				Data: mustMarshal(t, testControlCatalog("my-cat",
					[]gemara.MappingReference{
						{Id: "HAS-URL", Title: "Has URL", Version: "1.0", Url: "https://example.com/x.yaml"},
						{Id: "NO-URL", Title: "No URL", Version: "1.0"},
					}, nil, nil,
				)),
			},
			wantErr: "unmatched mapping-reference",
		},
		{
			name: "url-less ref resolved via import metadata.id succeeds",
			fetcher: mapFetcher{
				"https://example.com/guidance.yaml": mustMarshal(t, testGuidanceCatalog("ext-guidance")),
			},
			source: File{
				Name: "cat.yaml",
				Data: mustMarshal(t, testControlCatalog("my-cat",
					[]gemara.MappingReference{
						{Id: "GUIDE", Title: "Guide", Version: "1.0", Url: "https://example.com/guidance.yaml"},
						{Id: "ext-guidance", Title: "Ext Guidance Local", Version: "1.0"},
					},
					nil,
					[]gemara.MultiEntryMapping{
						{ReferenceId: "GUIDE", Entries: []gemara.ArtifactMapping{{ReferenceId: "G1"}}},
					},
				)),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asm := NewAssembler(tt.fetcher)
			b, err := asm.Assemble(context.Background(), Manifest{}, tt.source)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, b)
				return
			}
			require.NoError(t, err)
			assert.Empty(t, b.Warnings)
		})
	}
}

func TestAssemble_MappingWarningsWithContinueOnError(t *testing.T) {
	tests := []struct {
		name         string
		fetcher      mapFetcher
		source       File
		wantWarnings []MappingWarning
	}{
		{
			name:    "url-less ref not matching any metadata.id produces warning",
			fetcher: mapFetcher{},
			source: File{
				Name: "cat.yaml",
				Data: mustMarshal(t, testControlCatalog("my-cat",
					[]gemara.MappingReference{{Id: "unknown-ref", Title: "Unknown", Version: "1.0"}},
					nil, nil,
				)),
			},
			wantWarnings: []MappingWarning{
				{File: "cat.yaml", ArtifactID: "my-cat", ReferenceID: "unknown-ref"},
			},
		},
		{
			name: "multiple warnings across source and imports",
			fetcher: mapFetcher{
				"https://example.com/controls.yaml": mustMarshal(t, testControlCatalog("imported-cat",
					[]gemara.MappingReference{{Id: "phantom-import", Title: "Phantom Import", Version: "1.0"}},
					nil, nil,
				)),
			},
			source: File{
				Name: "policy.yaml",
				Data: mustMarshal(t, testControlCatalog("my-policy",
					[]gemara.MappingReference{
						{Id: "CTRL", Title: "Controls", Version: "1.0", Url: "https://example.com/controls.yaml"},
						{Id: "phantom-source", Title: "Phantom Source", Version: "1.0"},
					},
					nil,
					[]gemara.MultiEntryMapping{
						{ReferenceId: "CTRL", Entries: []gemara.ArtifactMapping{{ReferenceId: "C1"}}},
					},
				)),
			},
			wantWarnings: []MappingWarning{
				{File: "policy.yaml", ArtifactID: "my-policy", ReferenceID: "phantom-source"},
				{File: "controls.yaml", ArtifactID: "imported-cat", ReferenceID: "phantom-import"},
			},
		},
		{
			name:    "matching ref produces no warning with continue-on-error",
			fetcher: mapFetcher{},
			source: File{
				Name: "cat.yaml",
				Data: mustMarshal(t, testControlCatalog("self-ref",
					[]gemara.MappingReference{{Id: "self-ref", Title: "Self", Version: "1.0"}},
					nil, nil,
				)),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asm := NewAssembler(tt.fetcher)
			b, err := asm.Assemble(context.Background(), Manifest{}, tt.source, WithContinueOnError())
			require.NoError(t, err)
			if tt.wantWarnings == nil {
				assert.Empty(t, b.Warnings)
			} else {
				assert.Equal(t, tt.wantWarnings, b.Warnings)
			}
		})
	}
}

func TestMappingWarning_String(t *testing.T) {
	w := MappingWarning{
		File:        "policy.yaml",
		ArtifactID:  "org-policy",
		ReferenceID: "missing-ref",
	}
	assert.Contains(t, w.String(), "policy.yaml")
	assert.Contains(t, w.String(), "org-policy")
	assert.Contains(t, w.String(), "missing-ref")
}

