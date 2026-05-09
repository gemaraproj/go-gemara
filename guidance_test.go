// SPDX-License-Identifier: Apache-2.0

package gemara

import (
	"testing"

	"github.com/gemaraproj/go-gemara/internal/codec"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Test helpers ---

func testGuideline(id, title string) Guideline {
	return Guideline{
		Id:        id,
		Title:     title,
		Objective: "objective for " + id,
		Group:     "g1",
		State:     LifecycleActive,
	}
}

func testGuidanceCatalog(id string, guidelines ...Guideline) GuidanceCatalog {
	return GuidanceCatalog{
		Title:      "Guidance " + id,
		Metadata:   Metadata{Id: id},
		Guidelines: guidelines,
	}
}

func mustGuidancePool(t *testing.T, cats []GuidanceCatalog) map[string]GuidanceCatalog {
	t.Helper()
	idx, err := poolIndex(cats, guidanceAccessor.metadataID, guidanceAccessor.typeName)
	require.NoError(t, err)
	return idx
}

// --- Sugar: SGuidanceCatalog ---

func TestSGuidanceCatalog_RoundTrip(t *testing.T) {
	original := GuidanceCatalog{
		Title: "Test Guidance",
		Metadata: Metadata{
			Id:          "test-gc",
			Type:        GuidanceCatalogArtifact,
			Description: "test",
			Author:      Actor{Id: "test", Name: "Test", Type: Human},
		},
		GuidanceType: GuidanceFramework,
		Groups:       []Group{{Id: "g1", Title: "Group 1"}},
		Guidelines: []Guideline{
			{Id: "gl-1", Title: "GL1", Objective: "obj1", Group: "g1", State: LifecycleActive},
			{Id: "gl-2", Title: "GL2", Objective: "obj2", Group: "g1", State: LifecycleActive},
		},
	}
	sg := original.Sugar()

	yamlBytes, err := codec.MarshalYAML(sg)
	require.NoError(t, err)

	var roundTripped SGuidanceCatalog
	require.NoError(t, codec.UnmarshalYAML(yamlBytes, &roundTripped))

	assert.Equal(t, original.Title, roundTripped.Title)
	assert.Equal(t, original.Metadata.Id, roundTripped.Metadata.Id)
	assert.Equal(t, len(original.Groups), len(roundTripped.Groups))
	assert.Equal(t, len(original.Guidelines), len(roundTripped.Guidelines))

	if diff := cmp.Diff(original.Guidelines, roundTripped.Guidelines); diff != "" {
		t.Errorf("guidelines mismatch (-original +roundtripped):\n%s", diff)
	}
}

func TestSGuidanceCatalog_CacheResetOnUnmarshal(t *testing.T) {
	original := GuidanceCatalog{
		Title: "Test Guidance",
		Metadata: Metadata{
			Id:          "test-gc",
			Type:        GuidanceCatalogArtifact,
			Description: "test",
			Author:      Actor{Id: "test", Name: "Test", Type: Human},
		},
		GuidanceType: GuidanceFramework,
		Groups:       []Group{{Id: "g1", Title: "Group 1"}},
		Guidelines: []Guideline{
			{Id: "gl-1", Title: "GL1", Objective: "obj1", Group: "g1", State: LifecycleActive},
		},
	}
	sg := original.Sugar()

	_ = sg.GetGroupNames()
	require.NotEmpty(t, sg.GetGroupNames(), "cache should be populated")

	yamlBytes, err := codec.MarshalYAML(sg)
	require.NoError(t, err)
	require.NoError(t, codec.UnmarshalYAML(yamlBytes, sg))

	groups := sg.GetGroupNames()
	require.NotEmpty(t, groups, "cache should repopulate after unmarshal")
}

func TestSGuidanceCatalog_GetGroupNames(t *testing.T) {
	gc := GuidanceCatalog{
		Title:    "Test",
		Metadata: Metadata{Id: "test"},
		Groups: []Group{
			{Id: "g1", Title: "Group One"},
			{Id: "g2", Title: "Group Two"},
		},
	}
	sg := gc.Sugar()

	names := sg.GetGroupNames()
	assert.Equal(t, []string{"Group One", "Group Two"}, names)

	names2 := sg.GetGroupNames()
	assert.Equal(t, names, names2, "cached result should be identical")
}

func TestSGuidanceCatalog_GetGuidelinesForGroup(t *testing.T) {
	gc := GuidanceCatalog{
		Title:    "Test",
		Metadata: Metadata{Id: "test"},
		Guidelines: []Guideline{
			{Id: "gl-1", Title: "GL1", Group: "g1", State: LifecycleActive},
			{Id: "gl-2", Title: "GL2", Group: "g2", State: LifecycleActive},
			{Id: "gl-3", Title: "GL3", Group: "g1", State: LifecycleActive},
		},
	}
	sg := gc.Sugar()

	g1 := sg.GetGuidelinesForGroup("g1")
	require.Len(t, g1, 2)
	assert.Equal(t, "gl-1", g1[0].Id)
	assert.Equal(t, "gl-3", g1[1].Id)

	g2 := sg.GetGuidelinesForGroup("g2")
	require.Len(t, g2, 1)
	assert.Equal(t, "gl-2", g2[0].Id)

	assert.Nil(t, sg.GetGuidelinesForGroup("nonexistent"))
}

func TestSGuidanceCatalog_ToBaseFromBase(t *testing.T) {
	gc := GuidanceCatalog{
		Title:    "Original",
		Metadata: Metadata{Id: "orig"},
		Guidelines: []Guideline{
			{Id: "gl-1", Title: "GL1", Group: "g1", State: LifecycleActive},
		},
	}
	sg := gc.Sugar()

	_ = sg.GetGroupNames()
	base := sg.ToBase()
	assert.Equal(t, "Original", base.Title)

	updated := GuidanceCatalog{
		Title:    "Updated",
		Metadata: Metadata{Id: "updated"},
		Groups:   []Group{{Id: "new-g", Title: "New Group"}},
	}
	sg.FromBase(&updated)
	assert.Equal(t, "Updated", sg.Title)
	assert.Equal(t, []string{"New Group"}, sg.GetGroupNames(), "cache should reset after FromBase")
}

// --- Layer 1: ResolveGuidanceCatalog (standalone, no Policy) ---

func TestResolveGuidanceCatalog_Basic(t *testing.T) {
	gc := testGuidanceCatalog("gc-1", testGuideline("g1", "Guideline 1"), testGuideline("g2", "Guideline 2"))

	resolved, err := ResolveGuidanceCatalog(gc, nil)
	require.NoError(t, err)
	assert.Equal(t, "gc-1", resolved.Metadata.Id)
	assert.Equal(t, "Guidance gc-1", resolved.Title)
	assert.Len(t, resolved.Guidelines, 2)
}

func TestResolveGuidanceCatalog_EmptyMetadataID(t *testing.T) {
	gc := GuidanceCatalog{Title: "No ID"}

	_, err := ResolveGuidanceCatalog(gc, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no metadata.id")
}

func TestResolveGuidanceCatalog_EmptyGuidelines(t *testing.T) {
	gc := testGuidanceCatalog("gc-empty")

	resolved, err := ResolveGuidanceCatalog(gc, nil)
	require.NoError(t, err)
	assert.Empty(t, resolved.Guidelines)
}

func TestResolveGuidanceCatalog_DuplicatePoolIDs(t *testing.T) {
	gc := testGuidanceCatalog("gc-1", testGuideline("g1", "Guideline"))
	dup := testGuidanceCatalog("gc-1", testGuideline("g2", "Duplicate"))

	_, err := ResolveGuidanceCatalog(gc, []GuidanceCatalog{gc, dup})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate guidance catalog metadata.id")
}

func TestResolveGuidanceCatalog_WithExtends(t *testing.T) {
	base := testGuidanceCatalog("base", testGuideline("bg1", "Base Guideline"))
	child := GuidanceCatalog{
		Title:      "Child",
		Metadata:   Metadata{Id: "child"},
		Guidelines: []Guideline{testGuideline("cg1", "Child Guideline")},
		Extends:    []ArtifactMapping{{ReferenceId: "base"}},
	}

	resolved, err := ResolveGuidanceCatalog(child, []GuidanceCatalog{base, child})
	require.NoError(t, err)
	require.Len(t, resolved.Guidelines, 2)
	assert.Equal(t, "cg1", resolved.Guidelines[0].Id)
	assert.Equal(t, "bg1", resolved.Guidelines[1].Id)
}

func TestResolveGuidanceCatalog_TransitiveExtends(t *testing.T) {
	grandparent := testGuidanceCatalog("gp", testGuideline("gp1", "Grandparent"))
	parent := GuidanceCatalog{
		Title:      "Parent",
		Metadata:   Metadata{Id: "parent"},
		Guidelines: []Guideline{testGuideline("p1", "Parent")},
		Extends:    []ArtifactMapping{{ReferenceId: "gp"}},
	}
	child := GuidanceCatalog{
		Title:      "Child",
		Metadata:   Metadata{Id: "child"},
		Guidelines: []Guideline{testGuideline("c1", "Child")},
		Extends:    []ArtifactMapping{{ReferenceId: "parent"}},
	}

	resolved, err := ResolveGuidanceCatalog(child, []GuidanceCatalog{grandparent, parent, child})
	require.NoError(t, err)
	require.Len(t, resolved.Guidelines, 3)
	assert.Equal(t, "c1", resolved.Guidelines[0].Id)
	assert.Equal(t, "p1", resolved.Guidelines[1].Id)
	assert.Equal(t, "gp1", resolved.Guidelines[2].Id)
}

func TestResolveGuidanceCatalog_CycleDetection(t *testing.T) {
	a := GuidanceCatalog{
		Title:      "A",
		Metadata:   Metadata{Id: "a"},
		Guidelines: []Guideline{testGuideline("a1", "A")},
		Extends:    []ArtifactMapping{{ReferenceId: "b"}},
	}
	b := GuidanceCatalog{
		Title:      "B",
		Metadata:   Metadata{Id: "b"},
		Guidelines: []Guideline{testGuideline("b1", "B")},
		Extends:    []ArtifactMapping{{ReferenceId: "a"}},
	}

	resolved, err := ResolveGuidanceCatalog(a, []GuidanceCatalog{a, b})
	require.NoError(t, err)
	assert.Len(t, resolved.Guidelines, 2, "cycle should be broken; a1 + b1 only")
	assert.Equal(t, "a1", resolved.Guidelines[0].Id)
	assert.Equal(t, "b1", resolved.Guidelines[1].Id)
}

func TestResolveGuidanceCatalog_StrictExtends_MissingTarget(t *testing.T) {
	child := GuidanceCatalog{
		Title:        "Child",
		Metadata:     Metadata{Id: "child"},
		GuidanceType: GuidanceBestPractice,
		Guidelines:   []Guideline{testGuideline("g1", "Child")},
		Extends:      []ArtifactMapping{{ReferenceId: "gone"}},
	}
	_, err := ResolveGuidanceCatalog(child, []GuidanceCatalog{child})
	require.NoError(t, err)

	_, err = ResolveGuidanceCatalogWithOpts(child, []GuidanceCatalog{child}, ResolveCatalogOpts{StrictExtends: true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unresolved extends")
}

func TestResolveGuidanceCatalog_PreservesMetadata(t *testing.T) {
	gc := GuidanceCatalog{
		Title:        "My Guidance",
		Metadata:     Metadata{Id: "gc-1"},
		Groups:       []Group{{Id: "g1", Title: "Group 1"}},
		GuidanceType: GuidanceBestPractice,
		FrontMatter:  "intro text",
		Guidelines:   []Guideline{testGuideline("g1", "Guideline")},
	}

	resolved, err := ResolveGuidanceCatalog(gc, nil)
	require.NoError(t, err)
	assert.Equal(t, "My Guidance", resolved.Title)
	assert.Equal(t, "gc-1", resolved.Metadata.Id)
	assert.Len(t, resolved.Groups, 1)
	assert.Equal(t, GuidanceBestPractice, resolved.GuidanceType)
	assert.Equal(t, "intro text", resolved.FrontMatter)
}

func TestResolveGuidanceCatalog_DeepCopy(t *testing.T) {
	original := testGuidanceCatalog("gc-1", Guideline{
		Id: "g1", Title: "Guideline", Objective: "obj", Group: "g1",
		State: LifecycleActive, Recommendations: []string{"rec1"},
	})

	resolved, err := ResolveGuidanceCatalog(original, nil)
	require.NoError(t, err)
	require.Len(t, resolved.Guidelines, 1)

	resolved.Guidelines[0].Recommendations[0] = "mutated"
	assert.Equal(t, "rec1", original.Guidelines[0].Recommendations[0],
		"mutating resolved catalog must not affect original")
}

func TestResolveGuidanceCatalog_PreservesImports(t *testing.T) {
	gc := GuidanceCatalog{
		Title:        "With Imports",
		Metadata:     Metadata{Id: "gc-1"},
		GuidanceType: GuidanceBestPractice,
		Guidelines:   []Guideline{testGuideline("g1", "Guideline")},
		Imports: []MultiEntryMapping{
			{ReferenceId: "ref-1", Entries: []ArtifactMapping{{ReferenceId: "e1"}}},
		},
	}

	resolved, err := ResolveGuidanceCatalog(gc, nil)
	require.NoError(t, err)
	require.Len(t, resolved.Imports, 1)
	assert.Equal(t, "ref-1", resolved.Imports[0].ReferenceId)

	resolved.Imports[0].ReferenceId = "mutated"
	assert.Equal(t, "ref-1", gc.Imports[0].ReferenceId,
		"mutating resolved imports must not affect original")
}

// --- Layer 2: ApplyGuidanceOverlays ---

func TestApplyGuidanceOverlays_NoOverlays(t *testing.T) {
	gc := testGuidanceCatalog("gc-1", testGuideline("g1", "Guideline 1"), testGuideline("g2", "Guideline 2"))
	imp := GuidanceImport{ReferenceId: "gc-1"}

	eg := ApplyGuidanceOverlays(gc, imp)
	assert.Equal(t, "gc-1", eg.Metadata.Id)
	assert.Len(t, eg.Guidelines, 2)
}

func TestApplyGuidanceOverlays_WithExclusions(t *testing.T) {
	gc := testGuidanceCatalog("gc-1",
		testGuideline("g1", "Keep"),
		testGuideline("g2", "Exclude"),
		testGuideline("g3", "Keep"),
	)
	imp := GuidanceImport{
		ReferenceId: "gc-1",
		Exclusions:  []string{"g2"},
	}

	eg := ApplyGuidanceOverlays(gc, imp)
	assert.Len(t, eg.Guidelines, 2)
	assert.Equal(t, "g1", eg.Guidelines[0].Id)
	assert.Equal(t, "g3", eg.Guidelines[1].Id)
}

func TestApplyGuidanceOverlays_DeepCopy(t *testing.T) {
	gc := testGuidanceCatalog("gc-1", Guideline{
		Id: "g1", Title: "Guideline", Objective: "obj", Group: "g1",
		State: LifecycleActive, Recommendations: []string{"rec1"},
	})
	imp := GuidanceImport{ReferenceId: "gc-1"}

	eg := ApplyGuidanceOverlays(gc, imp)
	eg.Guidelines[0].Recommendations[0] = "mutated"
	assert.Equal(t, "rec1", gc.Guidelines[0].Recommendations[0],
		"mutating overlay result must not affect input catalog")
}

// --- Internal helper tests ---

func TestFlattenEntries_Guideline_NoExtends(t *testing.T) {
	gc := testGuidanceCatalog("gc-1", testGuideline("g1", "Guideline 1"))
	pool := mustGuidancePool(t, nil)
	result := flattenEntries(gc, pool, guidanceAccessor)
	assert.Len(t, result, 1)
	assert.Equal(t, "g1", result[0].Id)
}

func TestFlattenEntries_Guideline_WithExtends(t *testing.T) {
	base := testGuidanceCatalog("base", testGuideline("bg1", "Base Guideline"))
	child := GuidanceCatalog{
		Title:      "Child",
		Metadata:   Metadata{Id: "child"},
		Guidelines: []Guideline{testGuideline("cg1", "Child Guideline")},
		Extends:    []ArtifactMapping{{ReferenceId: "base"}},
	}
	pool := mustGuidancePool(t, []GuidanceCatalog{base, child})

	result := flattenEntries(child, pool, guidanceAccessor)
	assert.Len(t, result, 2)
	assert.Equal(t, "cg1", result[0].Id)
	assert.Equal(t, "bg1", result[1].Id)
}

func TestFlattenEntries_Guideline_TransitiveExtends(t *testing.T) {
	grandparent := testGuidanceCatalog("gp", testGuideline("gp1", "Grandparent"))
	parent := GuidanceCatalog{
		Title:      "Parent",
		Metadata:   Metadata{Id: "parent"},
		Guidelines: []Guideline{testGuideline("p1", "Parent")},
		Extends:    []ArtifactMapping{{ReferenceId: "gp"}},
	}
	child := GuidanceCatalog{
		Title:      "Child",
		Metadata:   Metadata{Id: "child"},
		Guidelines: []Guideline{testGuideline("c1", "Child")},
		Extends:    []ArtifactMapping{{ReferenceId: "parent"}},
	}
	pool := mustGuidancePool(t, []GuidanceCatalog{grandparent, parent, child})

	result := flattenEntries(child, pool, guidanceAccessor)
	require.Len(t, result, 3)
	assert.Equal(t, "c1", result[0].Id)
	assert.Equal(t, "p1", result[1].Id)
	assert.Equal(t, "gp1", result[2].Id)
}

func TestFlattenEntries_Guideline_CycleDetection(t *testing.T) {
	a := GuidanceCatalog{
		Title:      "A",
		Metadata:   Metadata{Id: "a"},
		Guidelines: []Guideline{testGuideline("a1", "A")},
		Extends:    []ArtifactMapping{{ReferenceId: "b"}},
	}
	b := GuidanceCatalog{
		Title:      "B",
		Metadata:   Metadata{Id: "b"},
		Guidelines: []Guideline{testGuideline("b1", "B")},
		Extends:    []ArtifactMapping{{ReferenceId: "a"}},
	}
	pool := mustGuidancePool(t, []GuidanceCatalog{a, b})

	result := flattenEntries(a, pool, guidanceAccessor)
	assert.Len(t, result, 2, "cycle should be broken; a1 + b1 only")
	assert.Equal(t, "a1", result[0].Id)
	assert.Equal(t, "b1", result[1].Id)
}

func TestFlattenEntries_Guideline_MultipleExtends_DeterministicOrder(t *testing.T) {
	ext1 := testGuidanceCatalog("ext1", testGuideline("e1", "Ext1"))
	ext2 := testGuidanceCatalog("ext2", testGuideline("e2", "Ext2"))
	ext3 := testGuidanceCatalog("ext3", testGuideline("e3", "Ext3"))
	child := GuidanceCatalog{
		Title:      "Child",
		Metadata:   Metadata{Id: "child"},
		Guidelines: []Guideline{testGuideline("c1", "Child")},
		Extends: []ArtifactMapping{
			{ReferenceId: "ext1"},
			{ReferenceId: "ext2"},
			{ReferenceId: "ext3"},
		},
	}
	pool := mustGuidancePool(t, []GuidanceCatalog{ext1, ext2, ext3, child})

	for i := 0; i < 20; i++ {
		result := flattenEntries(child, pool, guidanceAccessor)
		require.Len(t, result, 4)
		assert.Equal(t, "c1", result[0].Id)
		assert.Equal(t, "e1", result[1].Id, "run %d: extends order must be deterministic", i)
		assert.Equal(t, "e2", result[2].Id, "run %d: extends order must be deterministic", i)
		assert.Equal(t, "e3", result[3].Id, "run %d: extends order must be deterministic", i)
	}
}

func TestFlattenEntries_Guideline_SkipsSelfReference(t *testing.T) {
	gc := GuidanceCatalog{
		Title:      "Self Ref",
		Metadata:   Metadata{Id: "self"},
		Guidelines: []Guideline{testGuideline("g1", "Guideline")},
		Extends:    []ArtifactMapping{{ReferenceId: "self"}},
	}
	pool := mustGuidancePool(t, []GuidanceCatalog{gc})

	result := flattenEntries(gc, pool, guidanceAccessor)
	assert.Len(t, result, 1)
}

func TestFlattenEntries_Guideline_MissingExtendedCatalog(t *testing.T) {
	gc := GuidanceCatalog{
		Title:      "Missing",
		Metadata:   Metadata{Id: "child"},
		Guidelines: []Guideline{testGuideline("g1", "Guideline")},
		Extends:    []ArtifactMapping{{ReferenceId: "nonexistent"}},
	}
	pool := mustGuidancePool(t, nil)

	result := flattenEntries(gc, pool, guidanceAccessor)
	assert.Len(t, result, 1)
}

func TestFlattenEntries_Guideline_DeepCopy(t *testing.T) {
	original := testGuidanceCatalog("gc-1", Guideline{
		Id: "g1", Title: "Guideline", Objective: "obj", Group: "g1",
		State: LifecycleActive, Recommendations: []string{"rec1"},
	})
	pool := mustGuidancePool(t, nil)

	result := flattenEntries(original, pool, guidanceAccessor)
	require.Len(t, result, 1)

	result[0].Recommendations[0] = "mutated"
	assert.Equal(t, "rec1", original.Guidelines[0].Recommendations[0],
		"mutating result must not affect original")
}

func TestApplyEntryExclusions_RemovesMatchingGuidelines(t *testing.T) {
	guidelines := []Guideline{
		testGuideline("g1", "Keep"),
		testGuideline("g2", "Remove"),
		testGuideline("g3", "Keep"),
	}
	result := applyEntryExclusions(guidelines, []string{"g2"}, guidanceAccessor.entryID)
	assert.Len(t, result, 2)
	assert.Equal(t, "g1", result[0].Id)
	assert.Equal(t, "g3", result[1].Id)
}

func TestApplyEntryExclusions_EmptyGuidelineList(t *testing.T) {
	guidelines := []Guideline{testGuideline("g1", "Keep")}
	result := applyEntryExclusions(guidelines, nil, guidanceAccessor.entryID)
	assert.Len(t, result, 1)
}

func TestApplyEntryExclusions_AllGuidelinesExcluded(t *testing.T) {
	guidelines := []Guideline{testGuideline("g1", "X"), testGuideline("g2", "X")}
	result := applyEntryExclusions(guidelines, []string{"g1", "g2"}, guidanceAccessor.entryID)
	assert.Empty(t, result)
}

// --- Sugar: SGuidanceCatalog.Resolve ---

func TestSGuidanceCatalog_Resolve_Basic(t *testing.T) {
	gc := testGuidanceCatalog("gc-1", testGuideline("g1", "Guideline 1"), testGuideline("g2", "Guideline 2"))
	s := gc.Sugar()

	resolved, err := s.Resolve(nil)
	require.NoError(t, err)
	assert.Equal(t, "gc-1", resolved.Metadata.Id)
	assert.Len(t, resolved.Guidelines, 2)
}

func TestSGuidanceCatalog_Resolve_WithExtends(t *testing.T) {
	base := testGuidanceCatalog("base", testGuideline("bg1", "Base"))
	child := GuidanceCatalog{
		Title:      "Child",
		Metadata:   Metadata{Id: "child"},
		Guidelines: []Guideline{testGuideline("cg1", "Child")},
		Extends:    []ArtifactMapping{{ReferenceId: "base"}},
	}
	s := child.Sugar()

	resolved, err := s.Resolve([]GuidanceCatalog{base, child})
	require.NoError(t, err)
	require.Len(t, resolved.Guidelines, 2)
	assert.Equal(t, "cg1", resolved.Guidelines[0].Id)
	assert.Equal(t, "bg1", resolved.Guidelines[1].Id)
}

func TestSGuidanceCatalog_Resolve_CachesReady(t *testing.T) {
	gc := GuidanceCatalog{
		Title:      "GC",
		Metadata:   Metadata{Id: "gc-1"},
		Groups:     []Group{{Id: "g1", Title: "Group 1"}},
		Guidelines: []Guideline{testGuideline("gl1", "Guideline")},
	}
	s := gc.Sugar()

	resolved, err := s.Resolve(nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"Group 1"}, resolved.GetGroupNames())
	assert.Len(t, resolved.GetGuidelinesForGroup("g1"), 1)
}

func TestSGuidanceCatalog_Resolve_Error(t *testing.T) {
	gc := GuidanceCatalog{Title: "No ID"}
	s := gc.Sugar()

	_, err := s.Resolve(nil)
	require.Error(t, err)
}

func TestPoolIndex_GuidanceCatalog_EmptyMetadataIDSkipped(t *testing.T) {
	pool := []GuidanceCatalog{
		{Title: "No ID"},
		testGuidanceCatalog("gc-1", testGuideline("g1", "Guideline")),
	}
	idx, err := poolIndex(pool, guidanceAccessor.metadataID, guidanceAccessor.typeName)
	require.NoError(t, err)
	assert.Len(t, idx, 1)
	assert.Contains(t, idx, "gc-1")
}
