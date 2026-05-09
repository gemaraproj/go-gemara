// SPDX-License-Identifier: Apache-2.0

package gemara

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testPolicy(id string, catalogRefs []string, guidanceRefs []string) Policy {
	var mappingRefs []MappingReference
	for _, ref := range catalogRefs {
		mappingRefs = append(mappingRefs, MappingReference{Id: ref, Title: ref, Version: "1.0"})
	}
	for _, ref := range guidanceRefs {
		found := false
		for _, existing := range mappingRefs {
			if existing.Id == ref {
				found = true
				break
			}
		}
		if !found {
			mappingRefs = append(mappingRefs, MappingReference{Id: ref, Title: ref, Version: "1.0"})
		}
	}

	var catImports []CatalogImport
	for _, ref := range catalogRefs {
		catImports = append(catImports, CatalogImport{ReferenceId: ref})
	}

	var guideImports []GuidanceImport
	for _, ref := range guidanceRefs {
		guideImports = append(guideImports, GuidanceImport{ReferenceId: ref})
	}

	return Policy{
		Title: "Test Policy " + id,
		Metadata: Metadata{
			Id:                id,
			MappingReferences: mappingRefs,
		},
		Imports: Imports{
			Catalogs: catImports,
			Guidance: guideImports,
		},
	}
}

// --- Layer 3: ResolveEffectivePolicy integration tests ---

func TestResolveEffectivePolicy_SingleCatalog(t *testing.T) {
	pol := testPolicy("pol-1", []string{"cat-1"}, nil)
	cat := testCatalog("cat-1", testControl("c1", "Control 1"))

	ep, err := ResolveEffectivePolicy(pol, []ControlCatalog{cat}, nil)
	require.NoError(t, err)
	assert.Equal(t, "pol-1", ep.Policy.Metadata.Id)
	require.Len(t, ep.ControlCatalogs, 1)
	assert.Equal(t, "cat-1", ep.ControlCatalogs[0].Metadata.Id)
	assert.Len(t, ep.ControlCatalogs[0].Controls, 1)
	assert.Empty(t, ep.GuidanceCatalogs)
}

func TestResolveEffectivePolicy_SingleGuidance(t *testing.T) {
	pol := testPolicy("pol-1", nil, []string{"gc-1"})
	gc := testGuidanceCatalog("gc-1", testGuideline("g1", "Guideline 1"))

	ep, err := ResolveEffectivePolicy(pol, nil, []GuidanceCatalog{gc})
	require.NoError(t, err)
	assert.Empty(t, ep.ControlCatalogs)
	require.Len(t, ep.GuidanceCatalogs, 1)
	assert.Equal(t, "gc-1", ep.GuidanceCatalogs[0].Metadata.Id)
	assert.Len(t, ep.GuidanceCatalogs[0].Guidelines, 1)
}

func TestResolveEffectivePolicy_MultipleCatalogs(t *testing.T) {
	pol := testPolicy("pol-1", []string{"cat-1", "cat-2"}, nil)
	cat1 := testCatalog("cat-1", testControl("c1", "Control 1"))
	cat2 := testCatalog("cat-2", testControl("c2", "Control 2"), testControl("c3", "Control 3"))

	ep, err := ResolveEffectivePolicy(pol, []ControlCatalog{cat1, cat2}, nil)
	require.NoError(t, err)
	require.Len(t, ep.ControlCatalogs, 2)
	assert.Len(t, ep.ControlCatalogs[0].Controls, 1)
	assert.Len(t, ep.ControlCatalogs[1].Controls, 2)
}

func TestResolveEffectivePolicy_CatalogsAndGuidance(t *testing.T) {
	pol := testPolicy("pol-1", []string{"cat-1"}, []string{"gc-1"})
	cat := testCatalog("cat-1", testControl("c1", "Control"))
	gc := testGuidanceCatalog("gc-1", testGuideline("g1", "Guideline"))

	ep, err := ResolveEffectivePolicy(pol, []ControlCatalog{cat}, []GuidanceCatalog{gc})
	require.NoError(t, err)
	assert.Len(t, ep.ControlCatalogs, 1)
	assert.Len(t, ep.GuidanceCatalogs, 1)
}

func TestResolveEffectivePolicy_UnresolvableRef(t *testing.T) {
	pol := testPolicy("pol-1", []string{"missing-cat"}, nil)

	_, err := ResolveEffectivePolicy(pol, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no imports could be resolved")
}

func TestResolveEffectivePolicy_PartialResolution(t *testing.T) {
	pol := testPolicy("pol-1", []string{"cat-1", "missing-cat"}, nil)
	cat := testCatalog("cat-1", testControl("c1", "Control"))

	ep, err := ResolveEffectivePolicy(pol, []ControlCatalog{cat}, nil)
	require.NoError(t, err)
	assert.Len(t, ep.ControlCatalogs, 1, "should resolve the available catalog and skip the missing one")
}

func TestResolveEffectivePolicy_NoImports(t *testing.T) {
	pol := Policy{
		Title:    "Empty Imports",
		Metadata: Metadata{Id: "pol-empty"},
	}

	ep, err := ResolveEffectivePolicy(pol, nil, nil)
	require.NoError(t, err)
	assert.Empty(t, ep.ControlCatalogs)
	assert.Empty(t, ep.GuidanceCatalogs)
}

func TestResolveEffectivePolicy_NoMappingRefForImport(t *testing.T) {
	pol := Policy{
		Title:    "No Mapping Ref",
		Metadata: Metadata{Id: "pol-1"},
		Imports: Imports{
			Catalogs: []CatalogImport{{ReferenceId: "unmapped"}},
		},
	}

	_, err := ResolveEffectivePolicy(pol, []ControlCatalog{testCatalog("unmapped", testControl("c1", "C"))}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no imports could be resolved")
}

func TestResolveEffectivePolicy_WithExclusions(t *testing.T) {
	pol := Policy{
		Title: "With Exclusions",
		Metadata: Metadata{
			Id:                "pol-1",
			MappingReferences: []MappingReference{{Id: "cat-1", Title: "Cat", Version: "1.0"}},
		},
		Imports: Imports{
			Catalogs: []CatalogImport{
				{ReferenceId: "cat-1", Exclusions: []string{"c2"}},
			},
		},
	}
	cat := testCatalog("cat-1", testControl("c1", "Keep"), testControl("c2", "Exclude"))

	ep, err := ResolveEffectivePolicy(pol, []ControlCatalog{cat}, nil)
	require.NoError(t, err)
	require.Len(t, ep.ControlCatalogs, 1)
	assert.Len(t, ep.ControlCatalogs[0].Controls, 1)
	assert.Equal(t, "c1", ep.ControlCatalogs[0].Controls[0].Id)
}

func TestResolveEffectivePolicy_WithARMods(t *testing.T) {
	pol := Policy{
		Title: "With AR Mods",
		Metadata: Metadata{
			Id:                "pol-1",
			MappingReferences: []MappingReference{{Id: "cat-1", Title: "Cat", Version: "1.0"}},
		},
		Imports: Imports{
			Catalogs: []CatalogImport{
				{
					ReferenceId: "cat-1",
					AssessmentRequirementModifications: []AssessmentRequirementModifier{
						{Id: "mod-1", TargetId: "c1.1", ModificationType: ModReplace,
							ModificationRationale: "updated", Text: "replaced text"},
					},
				},
			},
		},
	}
	cat := testCatalog("cat-1", testControl("c1", "Control"))

	ep, err := ResolveEffectivePolicy(pol, []ControlCatalog{cat}, nil)
	require.NoError(t, err)
	ar := ep.ControlCatalogs[0].Controls[0].AssessmentRequirements[0]
	assert.Equal(t, "replaced text", ar.Text)
}

func TestResolveEffectivePolicy_CatalogWithExtends(t *testing.T) {
	base := testCatalog("base", testControl("b1", "Base Control"))
	child := ControlCatalog{
		Title:    "Child",
		Metadata: Metadata{Id: "child"},
		Controls: []Control{testControl("c1", "Child Control")},
		Extends:  []ArtifactMapping{{ReferenceId: "base"}},
	}
	pol := testPolicy("pol-1", []string{"child"}, nil)

	ep, err := ResolveEffectivePolicy(pol, []ControlCatalog{base, child}, nil)
	require.NoError(t, err)
	require.Len(t, ep.ControlCatalogs, 1)
	assert.Len(t, ep.ControlCatalogs[0].Controls, 2, "extends should be flattened via Layer 1")
	assert.Equal(t, "c1", ep.ControlCatalogs[0].Controls[0].Id)
	assert.Equal(t, "b1", ep.ControlCatalogs[0].Controls[1].Id)
}

func TestResolveEffectivePolicy_ExtendsAndExclusions(t *testing.T) {
	base := testCatalog("base", testControl("b1", "Keep"), testControl("b2", "Exclude"))
	child := ControlCatalog{
		Title:    "Child",
		Metadata: Metadata{Id: "child"},
		Controls: []Control{testControl("c1", "Child")},
		Extends:  []ArtifactMapping{{ReferenceId: "base"}},
	}
	pol := Policy{
		Title: "Extends + Exclusions",
		Metadata: Metadata{
			Id:                "pol-1",
			MappingReferences: []MappingReference{{Id: "child", Title: "Child", Version: "1.0"}},
		},
		Imports: Imports{
			Catalogs: []CatalogImport{
				{ReferenceId: "child", Exclusions: []string{"b2"}},
			},
		},
	}

	ep, err := ResolveEffectivePolicy(pol, []ControlCatalog{base, child}, nil)
	require.NoError(t, err)
	require.Len(t, ep.ControlCatalogs, 1)
	assert.Len(t, ep.ControlCatalogs[0].Controls, 2, "c1 + b1 (b2 excluded)")
	assert.Equal(t, "c1", ep.ControlCatalogs[0].Controls[0].Id)
	assert.Equal(t, "b1", ep.ControlCatalogs[0].Controls[1].Id)
}

func TestResolveEffectivePolicy_DuplicateCatalogPoolIDs(t *testing.T) {
	pol := testPolicy("pol-1", []string{"cat-1"}, nil)
	cat := testCatalog("cat-1", testControl("c1", "Control"))
	dup := testCatalog("cat-1", testControl("c2", "Duplicate"))

	_, err := ResolveEffectivePolicy(pol, []ControlCatalog{cat, dup}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate catalog metadata.id")
}

func TestResolveEffectivePolicy_DuplicateGuidancePoolIDs(t *testing.T) {
	pol := testPolicy("pol-1", nil, []string{"gc-1"})
	gc := testGuidanceCatalog("gc-1", testGuideline("g1", "Guideline"))
	dup := testGuidanceCatalog("gc-1", testGuideline("g2", "Duplicate"))

	_, err := ResolveEffectivePolicy(pol, nil, []GuidanceCatalog{gc, dup})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate guidance catalog metadata.id")
}

func TestResolveEffectivePolicy_PreservesOriginalPolicy(t *testing.T) {
	pol := testPolicy("pol-1", []string{"cat-1"}, nil)
	pol.Title = "Original Title"
	pol.Scope = Scope{In: Dimensions{Technologies: []string{"linux"}}}
	cat := testCatalog("cat-1", testControl("c1", "Control"))

	ep, err := ResolveEffectivePolicy(pol, []ControlCatalog{cat}, nil)
	require.NoError(t, err)
	assert.Equal(t, "Original Title", ep.Policy.Title)
	assert.Equal(t, []string{"linux"}, ep.Policy.Scope.In.Technologies)
}

// --- Sugar: SPolicy ---

func TestSPolicy_Sugar_RoundTrip(t *testing.T) {
	pol := testPolicy("pol-1", []string{"cat-1"}, nil)
	s := pol.Sugar()
	assert.Equal(t, "pol-1", s.Metadata.Id)
	assert.Equal(t, pol, s.ToBase())
}

func TestSPolicy_FromBase(t *testing.T) {
	pol := testPolicy("pol-1", []string{"cat-1"}, nil)
	s := pol.Sugar()

	pol2 := testPolicy("pol-2", []string{"cat-2"}, nil)
	s.FromBase(&pol2)
	assert.Equal(t, "pol-2", s.Metadata.Id)
}

func TestSPolicy_Resolve(t *testing.T) {
	pol := testPolicy("pol-1", []string{"cat-1"}, []string{"gc-1"})
	cat := testCatalog("cat-1", testControl("c1", "Control"))
	gc := testGuidanceCatalog("gc-1", testGuideline("g1", "Guideline"))

	sep, err := pol.Sugar().Resolve([]ControlCatalog{cat}, []GuidanceCatalog{gc})
	require.NoError(t, err)
	assert.Len(t, sep.ControlCatalogs, 1)
	assert.Len(t, sep.GuidanceCatalogs, 1)
}

func TestSPolicy_Resolve_Error(t *testing.T) {
	pol := testPolicy("pol-1", []string{"missing"}, nil)

	_, err := pol.Sugar().Resolve(nil, nil)
	require.Error(t, err)
}

// --- EffectivePolicy lookup methods ---

func TestEffectivePolicy_GetControlCatalog(t *testing.T) {
	pol := testPolicy("pol-1", []string{"cat-1", "cat-2"}, nil)
	cat1 := testCatalog("cat-1", testControl("c1", "Control 1"))
	cat2 := testCatalog("cat-2", testControl("c2", "Control 2"))

	sep, err := pol.Sugar().Resolve([]ControlCatalog{cat1, cat2}, nil)
	require.NoError(t, err)

	ec := sep.GetControlCatalog("cat-1")
	require.NotNil(t, ec)
	assert.Equal(t, "cat-1", ec.Metadata.Id)
	assert.Len(t, ec.Controls, 1)

	ec2 := sep.GetControlCatalog("cat-2")
	require.NotNil(t, ec2)
	assert.Equal(t, "cat-2", ec2.Metadata.Id)

	assert.Nil(t, sep.GetControlCatalog("nonexistent"))
}

func TestEffectivePolicy_GetGuidanceCatalog(t *testing.T) {
	pol := testPolicy("pol-1", nil, []string{"gc-1", "gc-2"})
	gc1 := testGuidanceCatalog("gc-1", testGuideline("g1", "Guideline 1"))
	gc2 := testGuidanceCatalog("gc-2", testGuideline("g2", "Guideline 2"))

	sep, err := pol.Sugar().Resolve(nil, []GuidanceCatalog{gc1, gc2})
	require.NoError(t, err)

	eg := sep.GetGuidanceCatalog("gc-1")
	require.NotNil(t, eg)
	assert.Equal(t, "gc-1", eg.Metadata.Id)
	assert.Len(t, eg.Guidelines, 1)

	eg2 := sep.GetGuidanceCatalog("gc-2")
	require.NotNil(t, eg2)
	assert.Equal(t, "gc-2", eg2.Metadata.Id)

	assert.Nil(t, sep.GetGuidanceCatalog("nonexistent"))
}

func TestEffectivePolicy_ControlCatalogIDs(t *testing.T) {
	pol := testPolicy("pol-1", []string{"cat-1", "cat-2"}, nil)
	cat1 := testCatalog("cat-1", testControl("c1", "C1"))
	cat2 := testCatalog("cat-2", testControl("c2", "C2"))

	sep, err := pol.Sugar().Resolve([]ControlCatalog{cat1, cat2}, nil)
	require.NoError(t, err)

	ids := sep.ControlCatalogIDs()
	assert.Equal(t, []string{"cat-1", "cat-2"}, ids)
}

func TestEffectivePolicy_GuidanceCatalogIDs(t *testing.T) {
	pol := testPolicy("pol-1", nil, []string{"gc-1", "gc-2"})
	gc1 := testGuidanceCatalog("gc-1", testGuideline("g1", "G1"))
	gc2 := testGuidanceCatalog("gc-2", testGuideline("g2", "G2"))

	sep, err := pol.Sugar().Resolve(nil, []GuidanceCatalog{gc1, gc2})
	require.NoError(t, err)

	ids := sep.GuidanceCatalogIDs()
	assert.Equal(t, []string{"gc-1", "gc-2"}, ids)
}

func TestEffectivePolicy_EmptyPolicy(t *testing.T) {
	pol := Policy{
		Title:    "Empty",
		Metadata: Metadata{Id: "pol-empty"},
	}

	sep, err := pol.Sugar().Resolve(nil, nil)
	require.NoError(t, err)
	assert.Empty(t, sep.ControlCatalogIDs())
	assert.Empty(t, sep.GuidanceCatalogIDs())
	assert.Nil(t, sep.GetControlCatalog("anything"))
	assert.Nil(t, sep.GetGuidanceCatalog("anything"))
}

func TestEffectivePolicy_LookupsFromResolve(t *testing.T) {
	pol := testPolicy("pol-1", []string{"cat-1"}, nil)
	cat := testCatalog("cat-1", testControl("c1", "Control"))

	ep, err := ResolveEffectivePolicy(pol, []ControlCatalog{cat}, nil)
	require.NoError(t, err)

	assert.Len(t, ep.ControlCatalogIDs(), 1)
	assert.Equal(t, "cat-1", ep.GetControlCatalog("cat-1").Metadata.Id)
}

func TestBuildRefIndex(t *testing.T) {
	refs := []MappingReference{
		{Id: "ref-1", Title: "Ref 1", Version: "1.0"},
		{Id: "ref-2", Title: "Ref 2", Version: "2.0"},
	}
	idx := buildRefIndex(refs)
	assert.Len(t, idx, 2)
	assert.Equal(t, "ref-1", idx["ref-1"])
	assert.Equal(t, "ref-2", idx["ref-2"])
}

func TestBuildRefIndex_Empty(t *testing.T) {
	idx := buildRefIndex(nil)
	assert.Empty(t, idx)
}
