// SPDX-License-Identifier: Apache-2.0

package gemara

import (
	"context"
	"testing"

	"github.com/gemaraproj/go-gemara/internal/codec"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Test helpers ---

func testControl(id, title string) Control {
	return Control{
		Id:        id,
		Title:     title,
		Objective: "objective for " + id,
		Group:     "g1",
		State:     LifecycleActive,
		AssessmentRequirements: []AssessmentRequirement{
			{Id: id + ".1", Text: "requirement for " + id, Applicability: []string{"all"}, State: LifecycleActive},
		},
	}
}

func testCatalog(id string, controls ...Control) ControlCatalog {
	return ControlCatalog{
		Title:    "Catalog " + id,
		Metadata: Metadata{Id: id},
		Controls: controls,
	}
}

func mustCatalogPool(t *testing.T, cats []ControlCatalog) map[string]ControlCatalog {
	t.Helper()
	idx, err := poolIndex(cats, controlAccessor.metadataID, controlAccessor.typeName)
	require.NoError(t, err)
	return idx
}

// --- Sugar: SControlCatalog ---

func TestSControlCatalog_RoundTrip(t *testing.T) {
	original, err := Load[ControlCatalog](context.Background(), fileFetcher, "test-data/good-ccc.yaml")
	require.NoError(t, err)

	sc := original.Sugar()

	yamlBytes, err := codec.MarshalYAML(sc)
	require.NoError(t, err)

	var roundTripped SControlCatalog
	require.NoError(t, codec.UnmarshalYAML(yamlBytes, &roundTripped))

	assert.Equal(t, original.Title, roundTripped.Title)
	assert.Equal(t, original.Metadata.Id, roundTripped.Metadata.Id)
	assert.Equal(t, len(original.Groups), len(roundTripped.Groups))
	assert.Equal(t, len(original.Controls), len(roundTripped.Controls))

	if diff := cmp.Diff(original.Controls, roundTripped.Controls); diff != "" {
		t.Errorf("controls mismatch (-original +roundtripped):\n%s", diff)
	}
}

func TestSControlCatalog_CacheResetOnUnmarshal(t *testing.T) {
	original, err := Load[ControlCatalog](context.Background(), fileFetcher, "test-data/good-ccc.yaml")
	require.NoError(t, err)
	sc := original.Sugar()

	_ = sc.GetGroupNames()
	require.NotEmpty(t, sc.GetGroupNames(), "cache should be populated")

	yamlBytes, err := codec.MarshalYAML(sc)
	require.NoError(t, err)
	require.NoError(t, codec.UnmarshalYAML(yamlBytes, sc))

	groups := sc.GetGroupNames()
	require.NotEmpty(t, groups, "cache should repopulate after unmarshal")
}

func TestResolveControlCatalog_Basic(t *testing.T) {
	cat := testCatalog("cat-1", testControl("c1", "Control 1"), testControl("c2", "Control 2"))

	resolved, err := ResolveControlCatalog(cat, nil)
	require.NoError(t, err)
	assert.Equal(t, "cat-1", resolved.Metadata.Id)
	assert.Equal(t, "Catalog cat-1", resolved.Title)
	assert.Len(t, resolved.Controls, 2)
}

func TestResolveControlCatalog_EmptyMetadataID(t *testing.T) {
	cat := ControlCatalog{Title: "No ID"}

	_, err := ResolveControlCatalog(cat, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no metadata.id")
}

func TestResolveControlCatalog_EmptyControls(t *testing.T) {
	cat := testCatalog("cat-empty")

	resolved, err := ResolveControlCatalog(cat, nil)
	require.NoError(t, err)
	assert.Empty(t, resolved.Controls)
}

func TestResolveControlCatalog_DuplicatePoolIDs(t *testing.T) {
	cat := testCatalog("cat-1", testControl("c1", "Control"))
	dup := testCatalog("cat-1", testControl("c2", "Duplicate"))

	_, err := ResolveControlCatalog(cat, []ControlCatalog{cat, dup})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate catalog metadata.id")
}

func TestResolveControlCatalog_WithExtends(t *testing.T) {
	base := testCatalog("base", testControl("b1", "Base Control"))
	child := ControlCatalog{
		Title:    "Child",
		Metadata: Metadata{Id: "child"},
		Controls: []Control{testControl("c1", "Child Control")},
		Extends:  []ArtifactMapping{{ReferenceId: "base"}},
	}

	resolved, err := ResolveControlCatalog(child, []ControlCatalog{base, child})
	require.NoError(t, err)
	require.Len(t, resolved.Controls, 2)
	assert.Equal(t, "c1", resolved.Controls[0].Id)
	assert.Equal(t, "b1", resolved.Controls[1].Id)
}

func TestResolveControlCatalog_TransitiveExtends(t *testing.T) {
	grandparent := testCatalog("gp", testControl("gp1", "Grandparent"))
	parent := ControlCatalog{
		Title:    "Parent",
		Metadata: Metadata{Id: "parent"},
		Controls: []Control{testControl("p1", "Parent")},
		Extends:  []ArtifactMapping{{ReferenceId: "gp"}},
	}
	child := ControlCatalog{
		Title:    "Child",
		Metadata: Metadata{Id: "child"},
		Controls: []Control{testControl("c1", "Child")},
		Extends:  []ArtifactMapping{{ReferenceId: "parent"}},
	}

	resolved, err := ResolveControlCatalog(child, []ControlCatalog{grandparent, parent, child})
	require.NoError(t, err)
	require.Len(t, resolved.Controls, 3)
	assert.Equal(t, "c1", resolved.Controls[0].Id)
	assert.Equal(t, "p1", resolved.Controls[1].Id)
	assert.Equal(t, "gp1", resolved.Controls[2].Id)
}

func TestResolveControlCatalog_CycleDetection(t *testing.T) {
	a := ControlCatalog{
		Title:    "A",
		Metadata: Metadata{Id: "a"},
		Controls: []Control{testControl("a1", "A")},
		Extends:  []ArtifactMapping{{ReferenceId: "b"}},
	}
	b := ControlCatalog{
		Title:    "B",
		Metadata: Metadata{Id: "b"},
		Controls: []Control{testControl("b1", "B")},
		Extends:  []ArtifactMapping{{ReferenceId: "a"}},
	}

	resolved, err := ResolveControlCatalog(a, []ControlCatalog{a, b})
	require.NoError(t, err)
	assert.Len(t, resolved.Controls, 2, "cycle should be broken; a1 + b1 only")
	assert.Equal(t, "a1", resolved.Controls[0].Id)
	assert.Equal(t, "b1", resolved.Controls[1].Id)
}

func TestResolveControlCatalog_StrictExtends_MissingTarget(t *testing.T) {
	child := ControlCatalog{
		Title:    "Child",
		Metadata: Metadata{Id: "child"},
		Controls: []Control{testControl("c1", "Child")},
		Extends:  []ArtifactMapping{{ReferenceId: "gone"}},
	}
	_, err := ResolveControlCatalog(child, []ControlCatalog{child})
	require.NoError(t, err)

	_, err = ResolveControlCatalogWithOpts(child, []ControlCatalog{child}, ResolveCatalogOpts{StrictExtends: true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unresolved extends")
}

func TestResolveControlCatalog_PreservesMetadata(t *testing.T) {
	cat := ControlCatalog{
		Title:    "My Catalog",
		Metadata: Metadata{Id: "cat-1"},
		Groups:   []Group{{Id: "g1", Title: "Group 1"}},
		Controls: []Control{testControl("c1", "Control")},
	}

	resolved, err := ResolveControlCatalog(cat, nil)
	require.NoError(t, err)
	assert.Equal(t, "My Catalog", resolved.Title)
	assert.Equal(t, "cat-1", resolved.Metadata.Id)
	assert.Len(t, resolved.Groups, 1)
}

func TestResolveControlCatalog_DeepCopy(t *testing.T) {
	original := testCatalog("cat-1", testControl("c1", "Control"))

	resolved, err := ResolveControlCatalog(original, nil)
	require.NoError(t, err)
	require.Len(t, resolved.Controls, 1)

	resolved.Controls[0].AssessmentRequirements[0].Text = "mutated"
	assert.Equal(t, "requirement for c1", original.Controls[0].AssessmentRequirements[0].Text,
		"mutating resolved catalog must not affect original")
}

func TestResolveControlCatalog_PreservesImports(t *testing.T) {
	cat := ControlCatalog{
		Title:    "With Imports",
		Metadata: Metadata{Id: "cat-1"},
		Controls: []Control{testControl("c1", "Control")},
		Imports: []MultiEntryMapping{
			{ReferenceId: "threat-1", Entries: []ArtifactMapping{{ReferenceId: "t1"}}},
		},
	}

	resolved, err := ResolveControlCatalog(cat, nil)
	require.NoError(t, err)
	require.Len(t, resolved.Imports, 1)
	assert.Equal(t, "threat-1", resolved.Imports[0].ReferenceId)

	resolved.Imports[0].ReferenceId = "mutated"
	assert.Equal(t, "threat-1", cat.Imports[0].ReferenceId,
		"mutating resolved imports must not affect original")
}

// --- Layer 2: ApplyCatalogOverlays ---

func TestApplyCatalogOverlays_NoOverlays(t *testing.T) {
	cat := testCatalog("cat-1", testControl("c1", "Control 1"), testControl("c2", "Control 2"))
	imp := CatalogImport{ReferenceId: "cat-1"}

	ec := ApplyCatalogOverlays(cat, imp)
	assert.Equal(t, "cat-1", ec.Metadata.Id)
	assert.Len(t, ec.Controls, 2)
}

func TestApplyCatalogOverlays_Exclusions(t *testing.T) {
	cat := testCatalog("cat-1",
		testControl("c1", "Keep"),
		testControl("c2", "Exclude"),
		testControl("c3", "Keep"),
	)
	imp := CatalogImport{
		ReferenceId: "cat-1",
		Exclusions:  []string{"c2"},
	}

	ec := ApplyCatalogOverlays(cat, imp)
	assert.Len(t, ec.Controls, 2)
	assert.Equal(t, "c1", ec.Controls[0].Id)
	assert.Equal(t, "c3", ec.Controls[1].Id)
}

func TestApplyCatalogOverlays_ARModifications(t *testing.T) {
	cat := testCatalog("cat-1", testControl("c1", "Control"))
	imp := CatalogImport{
		ReferenceId: "cat-1",
		AssessmentRequirementModifications: []AssessmentRequirementModifier{
			{Id: "mod-1", TargetId: "c1.1", ModificationType: ModReplace,
				ModificationRationale: "updated", Text: "replaced text"},
		},
	}

	ec := ApplyCatalogOverlays(cat, imp)
	require.Len(t, ec.Controls, 1)
	assert.Equal(t, "replaced text", ec.Controls[0].AssessmentRequirements[0].Text)
}

func TestApplyCatalogOverlays_ExclusionsAndMods(t *testing.T) {
	cat := testCatalog("cat-1",
		testControl("c1", "Keep and Modify"),
		testControl("c2", "Exclude"),
		testControl("c3", "Keep"),
	)
	imp := CatalogImport{
		ReferenceId: "cat-1",
		Exclusions:  []string{"c2"},
		AssessmentRequirementModifications: []AssessmentRequirementModifier{
			{Id: "mod-1", TargetId: "c1.1", ModificationType: ModReplace,
				ModificationRationale: "updated", Text: "updated requirement"},
		},
	}

	ec := ApplyCatalogOverlays(cat, imp)
	assert.Len(t, ec.Controls, 2)
	assert.Equal(t, "c1", ec.Controls[0].Id)
	assert.Equal(t, "updated requirement", ec.Controls[0].AssessmentRequirements[0].Text)
	assert.Equal(t, "c3", ec.Controls[1].Id)
}

func TestApplyCatalogOverlays_DeepCopy(t *testing.T) {
	cat := testCatalog("cat-1", testControl("c1", "Control"))
	imp := CatalogImport{ReferenceId: "cat-1"}

	ec := ApplyCatalogOverlays(cat, imp)
	ec.Controls[0].AssessmentRequirements[0].Text = "mutated"
	assert.Equal(t, "requirement for c1", cat.Controls[0].AssessmentRequirements[0].Text,
		"mutating overlay result must not affect input catalog")
}

// --- Internal helper tests ---

func TestFlattenEntries_Control_NoExtends(t *testing.T) {
	cat := testCatalog("cat-1", testControl("c1", "Control 1"))
	pool := mustCatalogPool(t, nil)
	result := flattenEntries(cat, pool, controlAccessor)
	assert.Len(t, result, 1)
	assert.Equal(t, "c1", result[0].Id)
}

func TestFlattenEntries_Control_WithExtends(t *testing.T) {
	base := testCatalog("base", testControl("b1", "Base Control"))
	child := ControlCatalog{
		Title:    "Child",
		Metadata: Metadata{Id: "child"},
		Controls: []Control{testControl("c1", "Child Control")},
		Extends:  []ArtifactMapping{{ReferenceId: "base"}},
	}
	pool := mustCatalogPool(t, []ControlCatalog{base, child})

	result := flattenEntries(child, pool, controlAccessor)
	assert.Len(t, result, 2)
	assert.Equal(t, "c1", result[0].Id)
	assert.Equal(t, "b1", result[1].Id)
}

func TestFlattenEntries_Control_TransitiveExtends(t *testing.T) {
	grandparent := testCatalog("gp", testControl("gp1", "Grandparent"))
	parent := ControlCatalog{
		Title:    "Parent",
		Metadata: Metadata{Id: "parent"},
		Controls: []Control{testControl("p1", "Parent")},
		Extends:  []ArtifactMapping{{ReferenceId: "gp"}},
	}
	child := ControlCatalog{
		Title:    "Child",
		Metadata: Metadata{Id: "child"},
		Controls: []Control{testControl("c1", "Child")},
		Extends:  []ArtifactMapping{{ReferenceId: "parent"}},
	}
	pool := mustCatalogPool(t, []ControlCatalog{grandparent, parent, child})

	result := flattenEntries(child, pool, controlAccessor)
	require.Len(t, result, 3)
	assert.Equal(t, "c1", result[0].Id)
	assert.Equal(t, "p1", result[1].Id)
	assert.Equal(t, "gp1", result[2].Id)
}

func TestFlattenEntries_Control_CycleDetection(t *testing.T) {
	a := ControlCatalog{
		Title:    "A",
		Metadata: Metadata{Id: "a"},
		Controls: []Control{testControl("a1", "A")},
		Extends:  []ArtifactMapping{{ReferenceId: "b"}},
	}
	b := ControlCatalog{
		Title:    "B",
		Metadata: Metadata{Id: "b"},
		Controls: []Control{testControl("b1", "B")},
		Extends:  []ArtifactMapping{{ReferenceId: "a"}},
	}
	pool := mustCatalogPool(t, []ControlCatalog{a, b})

	result := flattenEntries(a, pool, controlAccessor)
	assert.Len(t, result, 2, "cycle should be broken; a1 + b1 only")
	assert.Equal(t, "a1", result[0].Id)
	assert.Equal(t, "b1", result[1].Id)
}

func TestFlattenEntries_Control_MultipleExtends_DeterministicOrder(t *testing.T) {
	ext1 := testCatalog("ext1", testControl("e1", "Ext1"))
	ext2 := testCatalog("ext2", testControl("e2", "Ext2"))
	ext3 := testCatalog("ext3", testControl("e3", "Ext3"))
	child := ControlCatalog{
		Title:    "Child",
		Metadata: Metadata{Id: "child"},
		Controls: []Control{testControl("c1", "Child")},
		Extends: []ArtifactMapping{
			{ReferenceId: "ext1"},
			{ReferenceId: "ext2"},
			{ReferenceId: "ext3"},
		},
	}
	pool := mustCatalogPool(t, []ControlCatalog{ext1, ext2, ext3, child})

	for i := 0; i < 20; i++ {
		result := flattenEntries(child, pool, controlAccessor)
		require.Len(t, result, 4)
		assert.Equal(t, "c1", result[0].Id)
		assert.Equal(t, "e1", result[1].Id, "run %d: extends order must be deterministic", i)
		assert.Equal(t, "e2", result[2].Id, "run %d: extends order must be deterministic", i)
		assert.Equal(t, "e3", result[3].Id, "run %d: extends order must be deterministic", i)
	}
}

func TestFlattenEntries_Control_SkipsSelfReference(t *testing.T) {
	cat := ControlCatalog{
		Title:    "Self Ref",
		Metadata: Metadata{Id: "self"},
		Controls: []Control{testControl("c1", "Control")},
		Extends:  []ArtifactMapping{{ReferenceId: "self"}},
	}
	pool := mustCatalogPool(t, []ControlCatalog{cat})

	result := flattenEntries(cat, pool, controlAccessor)
	assert.Len(t, result, 1)
}

func TestFlattenEntries_Control_MissingExtendedCatalog(t *testing.T) {
	cat := ControlCatalog{
		Title:    "Missing Extend",
		Metadata: Metadata{Id: "child"},
		Controls: []Control{testControl("c1", "Control")},
		Extends:  []ArtifactMapping{{ReferenceId: "nonexistent"}},
	}
	pool := mustCatalogPool(t, nil)

	result := flattenEntries(cat, pool, controlAccessor)
	assert.Len(t, result, 1)
}

func TestFlattenEntries_Control_DeepCopy(t *testing.T) {
	original := testCatalog("cat-1", testControl("c1", "Control"))
	pool := mustCatalogPool(t, nil)

	result := flattenEntries(original, pool, controlAccessor)
	require.Len(t, result, 1)

	result[0].AssessmentRequirements[0].Text = "mutated"
	assert.Equal(t, "requirement for c1", original.Controls[0].AssessmentRequirements[0].Text,
		"mutating result must not affect original")
}

func TestApplyEntryExclusions_RemovesMatchingControls(t *testing.T) {
	controls := []Control{testControl("c1", "Keep"), testControl("c2", "Remove"), testControl("c3", "Keep")}
	result := applyEntryExclusions(controls, []string{"c2"}, controlAccessor.entryID)
	assert.Len(t, result, 2)
	assert.Equal(t, "c1", result[0].Id)
	assert.Equal(t, "c3", result[1].Id)
}

func TestApplyEntryExclusions_EmptyList(t *testing.T) {
	controls := []Control{testControl("c1", "Keep")}
	result := applyEntryExclusions(controls, nil, controlAccessor.entryID)
	assert.Len(t, result, 1)
}

func TestApplyEntryExclusions_AllExcluded(t *testing.T) {
	controls := []Control{testControl("c1", "X"), testControl("c2", "X")}
	result := applyEntryExclusions(controls, []string{"c1", "c2"}, controlAccessor.entryID)
	assert.Empty(t, result)
}

func TestApplyEntryExclusions_NonMatchingIDs(t *testing.T) {
	controls := []Control{testControl("c1", "Keep")}
	result := applyEntryExclusions(controls, []string{"nonexistent"}, controlAccessor.entryID)
	assert.Len(t, result, 1)
}

func TestApplyARModifications_Remove(t *testing.T) {
	ctrl := testControl("c1", "Control")
	mods := []AssessmentRequirementModifier{
		{Id: "mod-1", TargetId: "c1.1", ModificationType: ModRemove, ModificationRationale: "not applicable"},
	}
	result := applyARModifications([]Control{ctrl}, mods)
	assert.Empty(t, result[0].AssessmentRequirements)
}

func TestApplyARModifications_Replace(t *testing.T) {
	ctrl := testControl("c1", "Control")
	mods := []AssessmentRequirementModifier{
		{Id: "mod-1", TargetId: "c1.1", ModificationType: ModReplace, ModificationRationale: "updated",
			Text: "replaced text", Applicability: []string{"cloud"}, Recommendation: "use TLS"},
	}
	result := applyARModifications([]Control{ctrl}, mods)
	require.Len(t, result[0].AssessmentRequirements, 1)
	ar := result[0].AssessmentRequirements[0]
	assert.Equal(t, "replaced text", ar.Text)
	assert.Equal(t, []string{"cloud"}, ar.Applicability)
	assert.Equal(t, "use TLS", ar.Recommendation)
}

func TestApplyARModifications_Replace_NoAliasing(t *testing.T) {
	ctrl := testControl("c1", "Control")
	modApplicability := []string{"cloud"}
	mods := []AssessmentRequirementModifier{
		{Id: "mod-1", TargetId: "c1.1", ModificationType: ModReplace, ModificationRationale: "updated",
			Applicability: modApplicability},
	}
	result := applyARModifications([]Control{ctrl}, mods)

	result[0].AssessmentRequirements[0].Applicability[0] = "mutated"
	assert.Equal(t, "cloud", modApplicability[0],
		"mutating result must not affect modifier's applicability slice")
}

func TestApplyARModifications_Override(t *testing.T) {
	ctrl := testControl("c1", "Control")
	mods := []AssessmentRequirementModifier{
		{Id: "mod-1", TargetId: "c1.1", ModificationType: ModOverride, ModificationRationale: "override",
			Text: "overridden"},
	}
	result := applyARModifications([]Control{ctrl}, mods)
	assert.Equal(t, "overridden", result[0].AssessmentRequirements[0].Text)
}

func TestApplyARModifications_Modify(t *testing.T) {
	ctrl := testControl("c1", "Control")
	mods := []AssessmentRequirementModifier{
		{Id: "mod-1", TargetId: "c1.1", ModificationType: ModModify, ModificationRationale: "clarify",
			Recommendation: "quarterly review"},
	}
	result := applyARModifications([]Control{ctrl}, mods)
	ar := result[0].AssessmentRequirements[0]
	assert.Equal(t, "requirement for c1", ar.Text, "text should be unchanged")
	assert.Equal(t, "quarterly review", ar.Recommendation)
}

func TestApplyARModifications_Add(t *testing.T) {
	ctrl := testControl("c1", "Control")
	mods := []AssessmentRequirementModifier{
		{Id: "c1.2", TargetId: "c1.1", ModificationType: ModAdd, ModificationRationale: "additional",
			Text: "new requirement", Applicability: []string{"prod"}},
	}
	result := applyARModifications([]Control{ctrl}, mods)
	require.Len(t, result[0].AssessmentRequirements, 2)
	assert.Equal(t, "c1.1", result[0].AssessmentRequirements[0].Id)
	assert.Equal(t, "c1.2", result[0].AssessmentRequirements[1].Id)
	assert.Equal(t, "new requirement", result[0].AssessmentRequirements[1].Text)
	assert.Equal(t, LifecycleActive, result[0].AssessmentRequirements[1].State)
}

func TestApplyARModifications_NoMods(t *testing.T) {
	ctrl := testControl("c1", "Control")
	result := applyARModifications([]Control{ctrl}, nil)
	assert.Len(t, result[0].AssessmentRequirements, 1)
}

func TestApplyARModifications_NonMatchingTarget(t *testing.T) {
	ctrl := testControl("c1", "Control")
	mods := []AssessmentRequirementModifier{
		{Id: "mod-1", TargetId: "nonexistent", ModificationType: ModRemove, ModificationRationale: "n/a"},
	}
	result := applyARModifications([]Control{ctrl}, mods)
	assert.Len(t, result[0].AssessmentRequirements, 1)
}

func TestApplyARModifications_PartialFieldUpdate(t *testing.T) {
	ctrl := testControl("c1", "Control")
	mods := []AssessmentRequirementModifier{
		{Id: "mod-1", TargetId: "c1.1", ModificationType: ModReplace, ModificationRationale: "partial",
			Text: "new text"},
	}
	result := applyARModifications([]Control{ctrl}, mods)
	ar := result[0].AssessmentRequirements[0]
	assert.Equal(t, "new text", ar.Text)
	assert.Equal(t, []string{"all"}, ar.Applicability, "applicability unchanged when not provided")
}

func TestApplyARModifications_UnknownModType(t *testing.T) {
	ctrl := testControl("c1", "Control")
	mods := []AssessmentRequirementModifier{
		{Id: "mod-1", TargetId: "c1.1", ModificationType: ModType(99), ModificationRationale: "unknown"},
	}
	result := applyARModifications([]Control{ctrl}, mods)
	assert.Len(t, result[0].AssessmentRequirements, 1, "unknown mod type should be skipped")
	assert.Equal(t, "requirement for c1", result[0].AssessmentRequirements[0].Text)
}

// --- Sugar: SControlCatalog.Resolve ---

func TestSControlCatalog_Resolve_Basic(t *testing.T) {
	cat := testCatalog("cat-1", testControl("c1", "Control 1"), testControl("c2", "Control 2"))
	s := cat.Sugar()

	resolved, err := s.Resolve(nil)
	require.NoError(t, err)
	assert.Equal(t, "cat-1", resolved.Metadata.Id)
	assert.Len(t, resolved.Controls, 2)
}

func TestSControlCatalog_Resolve_WithExtends(t *testing.T) {
	base := testCatalog("base", testControl("b1", "Base"))
	child := ControlCatalog{
		Title:    "Child",
		Metadata: Metadata{Id: "child"},
		Controls: []Control{testControl("c1", "Child")},
		Extends:  []ArtifactMapping{{ReferenceId: "base"}},
	}
	s := child.Sugar()

	resolved, err := s.Resolve([]ControlCatalog{base, child})
	require.NoError(t, err)
	require.Len(t, resolved.Controls, 2)
	assert.Equal(t, "c1", resolved.Controls[0].Id)
	assert.Equal(t, "b1", resolved.Controls[1].Id)
}

func TestSControlCatalog_Resolve_CachesReady(t *testing.T) {
	cat := ControlCatalog{
		Title:    "Cat",
		Metadata: Metadata{Id: "cat-1"},
		Groups:   []Group{{Id: "g1", Title: "Group 1"}},
		Controls: []Control{testControl("c1", "Control")},
	}
	s := cat.Sugar()

	resolved, err := s.Resolve(nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"Group 1"}, resolved.GetGroupNames())
	assert.Len(t, resolved.GetControlsForGroup("g1"), 1)
}

func TestSControlCatalog_Resolve_Error(t *testing.T) {
	cat := ControlCatalog{Title: "No ID"}
	s := cat.Sugar()

	_, err := s.Resolve(nil)
	require.Error(t, err)
}

func TestPoolIndex_ControlCatalog_EmptyMetadataIDSkipped(t *testing.T) {
	pool := []ControlCatalog{
		{Title: "No ID"},
		testCatalog("cat-1", testControl("c1", "Control")),
	}
	idx, err := poolIndex(pool, controlAccessor.metadataID, controlAccessor.typeName)
	require.NoError(t, err)
	assert.Len(t, idx, 1)
	assert.Contains(t, idx, "cat-1")
}
