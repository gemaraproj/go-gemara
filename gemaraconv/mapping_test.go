// SPDX-License-Identifier: Apache-2.0

package gemaraconv

import (
	"testing"

	oscal12 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-2-2"
	"github.com/gemaraproj/go-gemara"
	"github.com/stretchr/testify/assert"
)

func TestMappingOptionDefaults(t *testing.T) {
	opts := defaultMappingOpts()
	assert.Empty(t, opts.methodOverride)
	assert.Nil(t, opts.reverseDoc)
}

func TestMappingOptionOverrides(t *testing.T) {
	opts := defaultMappingOpts()
	WithAutomatedMethod()(&opts)
	reverseDoc := gemara.MappingDocument{Title: "reverse"}
	WithReverseMappingDocument(reverseDoc)(&opts)

	assert.Equal(t, "automated", opts.methodOverride)
	assert.NotNil(t, opts.reverseDoc)
	assert.Equal(t, "reverse", opts.reverseDoc.Title)
}

func TestMappingOptionOverrides_AllMethods(t *testing.T) {
	tests := []struct {
		name string
		opt  MappingOption
		want string
	}{
		{name: "human", opt: WithHumanMethod(), want: "human"},
		{name: "automated", opt: WithAutomatedMethod(), want: "automated"},
		{name: "machine-assisted", opt: WithMachineAssistedMethod(), want: "machine-assisted"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := defaultMappingOpts()
			tt.opt(&opts)
			assert.Equal(t, tt.want, opts.methodOverride)
		})
	}
}

func newTestMappingDocument() gemara.MappingDocument {
	return gemara.MappingDocument{
		Title: "Test Mapping",
		Metadata: gemara.Metadata{
			Id:      "test-mapping",
			Version: "1.0.0",
			Author:  gemara.Actor{Name: "Test Author"},
			MappingReferences: []gemara.MappingReference{
				{
					Id:    "source-ref",
					Title: "Source Catalog",
					Url:   "https://example.com/source",
				},
				{
					Id:    "target-ref",
					Title: "Target Catalog",
					Url:   "https://example.com/target",
				},
			},
		},
		SourceReference: gemara.TypedMapping{
			EntryType:   gemara.EntryTypeControl,
			ReferenceId: "source-ref",
		},
		TargetReference: gemara.TypedMapping{
			EntryType:   gemara.EntryTypeControl,
			ReferenceId: "target-ref",
		},
		Mappings: []gemara.Mapping{
			{
				Id:           "map-1",
				Source:       "AC-01",
				Relationship: gemara.RelEquivalent,
				Targets: []gemara.MappingTarget{
					{EntryId: "CC-01", Rationale: "Same control objective"},
				},
			},
		},
		Remarks: "Test mapping remarks",
	}
}

func TestGemaraRelationshipToOSCAL(t *testing.T) {
	tests := []struct {
		name    string
		rel     gemara.RelationshipType
		want    string
		wantErr bool
	}{
		{name: "equivalent", rel: gemara.RelEquivalent, want: "equivalent-to"},
		{name: "subsumes", rel: gemara.RelSubsumes, want: "superset-of"},
		{name: "relates-to", rel: gemara.RelRelatesTo, want: "intersects-with"},
		{name: "no-match", rel: gemara.RelNoMatch, want: "no-relationship"},
		{name: "implements unsupported", rel: gemara.RelImplements, wantErr: true},
		{name: "supports unsupported", rel: gemara.RelSupports, wantErr: true},
		{name: "invalid", rel: gemara.InvalidRelationshipType, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := gemaraRelationshipToOSCAL(tt.rel)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMappingDocumentToOSCAL_UnsupportedRelationship(t *testing.T) {
	doc := newTestMappingDocument()
	doc.Mappings[0].Relationship = gemara.RelImplements

	_, err := MappingDocumentToOSCAL(doc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "relationship")
}

func TestMappingDocumentToOSCAL_MultipleTargets(t *testing.T) {
	doc := newTestMappingDocument()
	doc.Mappings = []gemara.Mapping{
		{
			Id:           "map-1",
			Source:       "AC-01",
			Relationship: gemara.RelSubsumes,
			Targets: []gemara.MappingTarget{
				{EntryId: "CC-01", Rationale: "Covers access control"},
				{EntryId: "CC-02", Rationale: "Covers identity management"},
			},
			Remarks: "Broad coverage",
		},
		{
			Id:           "map-2",
			Source:       "AC-02",
			Relationship: gemara.RelEquivalent,
			Targets: []gemara.MappingTarget{
				{EntryId: "CC-03"},
			},
		},
	}

	result, err := MappingDocumentToOSCAL(doc)
	assert.NoError(t, err)
	assert.Len(t, result.Mappings.Maps, 2)

	m1 := result.Mappings.Maps[0]
	assert.Len(t, m1.Targets, 2)
	assert.Equal(t, "Covers access control\nCovers identity management\nBroad coverage", m1.Remarks)

	m2 := result.Mappings.Maps[1]
	assert.Len(t, m2.Targets, 1)
	assert.Empty(t, m2.Remarks)
}

func TestMappingDocumentToOSCAL_SourceGap(t *testing.T) {
	doc := newTestMappingDocument()
	doc.Mappings = []gemara.Mapping{
		{
			Id:           "map-1",
			Source:       "AC-01",
			Relationship: gemara.RelEquivalent,
			Targets:      []gemara.MappingTarget{{EntryId: "CC-01"}},
		},
		{
			Id:           "map-2",
			Source:       "AC-02",
			Relationship: gemara.RelNoMatch,
		},
		{
			Id:           "map-3",
			Source:       "AC-03",
			Relationship: gemara.RelNoMatch,
		},
	}

	result, err := MappingDocumentToOSCAL(doc)
	assert.NoError(t, err)

	// no-match entries excluded from maps
	assert.Len(t, result.Mappings.Maps, 1)

	// source-gap populated
	assert.NotNil(t, result.Mappings.SourceGapSummary)
	assert.NotEmpty(t, result.Mappings.SourceGapSummary.UUID)
	ids := *result.Mappings.SourceGapSummary.UnmappedControls[0].WithIds
	assert.Equal(t, []string{"AC-02", "AC-03"}, ids)

	// no target-gap without reverse mapping
	assert.Nil(t, result.Mappings.TargetGapSummary)
}

func TestMappingDocumentToOSCAL_TargetGap(t *testing.T) {
	doc := newTestMappingDocument()

	reverseDoc := gemara.MappingDocument{
		Title: "Reverse Mapping",
		Metadata: gemara.Metadata{
			Id: "reverse-mapping",
			MappingReferences: []gemara.MappingReference{
				{Id: "target-ref", Title: "Target Catalog", Url: "https://example.com/target"},
				{Id: "source-ref", Title: "Source Catalog", Url: "https://example.com/source"},
			},
		},
		SourceReference: gemara.TypedMapping{
			EntryType:   gemara.EntryTypeControl,
			ReferenceId: "target-ref",
		},
		TargetReference: gemara.TypedMapping{
			EntryType:   gemara.EntryTypeControl,
			ReferenceId: "source-ref",
		},
		Mappings: []gemara.Mapping{
			{
				Id:           "rev-1",
				Source:       "CC-01",
				Relationship: gemara.RelEquivalent,
				Targets:      []gemara.MappingTarget{{EntryId: "AC-01"}},
			},
			{
				Id:           "rev-2",
				Source:       "CC-99",
				Relationship: gemara.RelNoMatch,
			},
		},
	}

	result, err := MappingDocumentToOSCAL(doc, WithReverseMappingDocument(reverseDoc))
	assert.NoError(t, err)

	assert.NotNil(t, result.Mappings.TargetGapSummary)
	ids := *result.Mappings.TargetGapSummary.UnmappedControls[0].WithIds
	assert.Equal(t, []string{"CC-99"}, ids)
}

func TestMappingDocumentToOSCAL_ProvenanceMethodOverride(t *testing.T) {
	doc := newTestMappingDocument()
	doc.Metadata.Author.Type = gemara.Human
	result, err := MappingDocumentToOSCAL(doc, WithAutomatedMethod())

	assert.NoError(t, err)
	assert.Equal(t, "automated", result.Provenance.Method)
	assert.Equal(t, "draft", result.Provenance.Status)
}

func TestMappingDocumentToOSCAL_ProvenanceMethodInference(t *testing.T) {
	tests := []struct {
		name       string
		authorType gemara.EntityType
		want       string
	}{
		{name: "human author", authorType: gemara.Human, want: "human"},
		{name: "software author", authorType: gemara.Software, want: "automated"},
		{name: "software-assisted author", authorType: gemara.SoftwareAssisted, want: "machine-assisted"},
		{name: "unset author type", authorType: gemara.InvalidEntityType, want: "human"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := newTestMappingDocument()
			doc.Metadata.Author.Type = tt.authorType

			result, err := MappingDocumentToOSCAL(doc)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, result.Provenance.Method)
			assert.Equal(t, "draft", result.Provenance.Status)
		})
	}
}

func TestMappingDocumentToOSCAL_Errors(t *testing.T) {
	tests := []struct {
		name    string
		doc     gemara.MappingDocument
		opts    []MappingOption
		wantErr string
	}{
		{
			name: "empty mappings",
			doc: gemara.MappingDocument{
				Title:    "Empty",
				Mappings: nil,
			},
			wantErr: "no mappings",
		},
		{
			name: "missing source reference",
			doc: gemara.MappingDocument{
				Title: "Bad Source",
				Metadata: gemara.Metadata{
					MappingReferences: []gemara.MappingReference{
						{Id: "target-ref", Url: "https://example.com/target"},
					},
				},
				SourceReference: gemara.TypedMapping{ReferenceId: "nonexistent"},
				TargetReference: gemara.TypedMapping{ReferenceId: "target-ref"},
				Mappings:        []gemara.Mapping{{Id: "m1", Source: "A", Relationship: gemara.RelEquivalent}},
			},
			wantErr: "source reference",
		},
		{
			name: "missing target reference",
			doc: gemara.MappingDocument{
				Title: "Bad Target",
				Metadata: gemara.Metadata{
					MappingReferences: []gemara.MappingReference{
						{Id: "source-ref", Url: "https://example.com/source"},
					},
				},
				SourceReference: gemara.TypedMapping{ReferenceId: "source-ref"},
				TargetReference: gemara.TypedMapping{ReferenceId: "nonexistent"},
				Mappings:        []gemara.Mapping{{Id: "m1", Source: "A", Relationship: gemara.RelEquivalent}},
			},
			wantErr: "target reference",
		},
		{
			name: "reverse mapping direction mismatch",
			doc:  newTestMappingDocument(),
			opts: []MappingOption{
				WithReverseMappingDocument(gemara.MappingDocument{
					SourceReference: gemara.TypedMapping{ReferenceId: "wrong-ref"},
					TargetReference: gemara.TypedMapping{ReferenceId: "source-ref"},
				}),
			},
			wantErr: "reverse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := MappingDocumentToOSCAL(tt.doc, tt.opts...)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestMappingDocumentConverterWrapper(t *testing.T) {
	doc := newTestMappingDocument()
	converter := MappingDocument(doc)

	result, err := converter.ToOSCAL()
	assert.NoError(t, err)
	assert.Equal(t, "Test Mapping", result.Metadata.Title)
}

func TestMappingDocumentToOSCAL_Basic(t *testing.T) {
	doc := newTestMappingDocument()
	result, err := MappingDocumentToOSCAL(doc)

	assert.NoError(t, err)
	assert.NotEmpty(t, result.UUID)
	assert.Equal(t, "Test Mapping", result.Metadata.Title)
	assert.Equal(t, "1.0.0", result.Metadata.Version)
	assert.Equal(t, oscal12.Version, result.Metadata.OscalVersion)

	// Provenance defaults (Author.Type unset -> inferred as human)
	assert.Equal(t, "human", result.Provenance.Method)
	assert.Equal(t, "draft", result.Provenance.Status)
	assert.Equal(t, "functional", result.Provenance.MatchingRationale)
	assert.Equal(t, "Test mapping remarks", result.Provenance.MappingDescription)

	// Source/target resources
	assert.Equal(t, "https://example.com/source", result.Mappings.SourceResource.Href)
	assert.Equal(t, "control", result.Mappings.SourceResource.Type)
	assert.Equal(t, "https://example.com/target", result.Mappings.TargetResource.Href)
	assert.Equal(t, "control", result.Mappings.TargetResource.Type)

	// Maps
	assert.Len(t, result.Mappings.Maps, 1)
	m := result.Mappings.Maps[0]
	assert.NotEmpty(t, m.UUID)
	assert.Equal(t, "equivalent-to", m.Relationship)
	assert.Len(t, m.Sources, 1)
	assert.Equal(t, "AC-01", m.Sources[0].IDRef)
	assert.Equal(t, "control", m.Sources[0].Type)
	assert.Len(t, m.Targets, 1)
	assert.Equal(t, "CC-01", m.Targets[0].IDRef)
	assert.Equal(t, "control", m.Targets[0].Type)

	// BackMatter
	assert.NotNil(t, result.BackMatter)
	assert.Len(t, *result.BackMatter.Resources, 2)
}
