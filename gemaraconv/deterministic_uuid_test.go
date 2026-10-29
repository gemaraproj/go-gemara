package gemaraconv

import (
	"testing"

	"github.com/gemaraproj/go-gemara"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCatalogToOSCAL_DeterministicUUIDs(t *testing.T) {
	catalog := gemara.ControlCatalog{
		Metadata: gemara.Metadata{
			Id:      "test-catalog",
			Version: "1.0.0",
			Author:  gemara.Actor{Name: "Test Author", Type: gemara.Human},
		},
		Title: "Test Catalog",
		Groups: []gemara.Group{
			{Id: "AC", Title: "Access Control", Description: "Access control family"},
		},
		Controls: []gemara.Control{
			{
				Id: "AC-01", Group: "AC", Title: "Access Policy",
				Objective: "Enforce access controls",
				AssessmentRequirements: []gemara.AssessmentRequirement{
					{Id: "AC-01.01", Text: "Verify access"},
				},
			},
		},
	}

	result1, err := CatalogToOSCAL(catalog)
	require.NoError(t, err)

	result2, err := CatalogToOSCAL(catalog)
	require.NoError(t, err)

	assert.Equal(t, result1.UUID, result2.UUID, "catalog UUID should be deterministic")
	assert.NotEmpty(t, result1.UUID)

	// Party UUID should also be deterministic
	require.NotNil(t, result1.Metadata.Parties)
	require.NotNil(t, result2.Metadata.Parties)
	assert.Equal(t,
		(*result1.Metadata.Parties)[0].UUID,
		(*result2.Metadata.Parties)[0].UUID,
		"party UUID should be deterministic",
	)
}

func TestGuidanceToOSCAL_DeterministicUUIDs(t *testing.T) {
	guidance := guidanceWithExternalExtends()

	cat1, prof1, err := GuidanceToOSCAL(guidance, "catalog.json",
		WithOSCALImports(map[string]string{"NIST-800-53": "https://nist.gov/800-53/catalog.json"}))
	require.NoError(t, err)

	cat2, prof2, err := GuidanceToOSCAL(guidance, "catalog.json",
		WithOSCALImports(map[string]string{"NIST-800-53": "https://nist.gov/800-53/catalog.json"}))
	require.NoError(t, err)

	assert.Equal(t, cat1.UUID, cat2.UUID, "guidance catalog UUID should be deterministic")
	assert.Equal(t, prof1.UUID, prof2.UUID, "guidance profile UUID should be deterministic")
	assert.NotEqual(t, cat1.UUID, prof1.UUID, "catalog and profile UUIDs should differ")

	// Backmatter resource UUIDs should be deterministic
	if cat1.BackMatter != nil && cat1.BackMatter.Resources != nil {
		require.NotNil(t, cat2.BackMatter)
		require.NotNil(t, cat2.BackMatter.Resources)
		for i := range *cat1.BackMatter.Resources {
			assert.Equal(t,
				(*cat1.BackMatter.Resources)[i].UUID,
				(*cat2.BackMatter.Resources)[i].UUID,
				"resource UUID should be deterministic",
			)
		}
	}
}

func TestAssessmentResults_DeterministicUUIDs(t *testing.T) {
	log := makeEvaluationLog(gemara.Actor{
		Name: "test-tool",
		Type: gemara.Software,
	}, []*gemara.AssessmentLog{
		makeAssessmentLog("REQ-1", "check first", gemara.Failed, "not met", nil),
		makeAssessmentLog("REQ-2", "check second", gemara.Passed, "", nil),
	})
	log.Metadata.Id = "eval-deterministic"
	log.Metadata.Version = "1.0.0"
	log.Target = gemara.Resource{Id: "sys-1", Name: "Test System", Type: gemara.Software}

	ar1, err := EvaluationLogToOSCALAssessmentResults(log, WithImportApHref("#"))
	require.NoError(t, err)

	ar2, err := EvaluationLogToOSCALAssessmentResults(log, WithImportApHref("#"))
	require.NoError(t, err)

	// Document UUID
	assert.Equal(t, ar1.UUID, ar2.UUID, "assessment results UUID should be deterministic")

	// Result UUID
	assert.Equal(t, ar1.Results[0].UUID, ar2.Results[0].UUID, "result UUID should be deterministic")

	// Finding UUIDs
	require.NotNil(t, ar1.Results[0].Findings)
	require.NotNil(t, ar2.Results[0].Findings)
	assert.Equal(t,
		(*ar1.Results[0].Findings)[0].UUID,
		(*ar2.Results[0].Findings)[0].UUID,
		"finding UUID should be deterministic",
	)

	// Observation UUIDs
	require.NotNil(t, ar1.Results[0].Observations)
	require.NotNil(t, ar2.Results[0].Observations)
	for i := range *ar1.Results[0].Observations {
		assert.Equal(t,
			(*ar1.Results[0].Observations)[i].UUID,
			(*ar2.Results[0].Observations)[i].UUID,
			"observation UUID should be deterministic",
		)
	}

	// Log entry UUIDs
	require.NotNil(t, ar1.Results[0].AssessmentLog)
	require.NotNil(t, ar2.Results[0].AssessmentLog)
	for i := range ar1.Results[0].AssessmentLog.Entries {
		assert.Equal(t,
			ar1.Results[0].AssessmentLog.Entries[i].UUID,
			ar2.Results[0].AssessmentLog.Entries[i].UUID,
			"log entry UUID should be deterministic",
		)
	}

	// Inventory item UUID
	require.NotNil(t, ar1.Results[0].LocalDefinitions)
	require.NotNil(t, ar2.Results[0].LocalDefinitions)
	require.NotNil(t, ar1.Results[0].LocalDefinitions.InventoryItems)
	assert.Equal(t,
		(*ar1.Results[0].LocalDefinitions.InventoryItems)[0].UUID,
		(*ar2.Results[0].LocalDefinitions.InventoryItems)[0].UUID,
		"inventory item UUID should be deterministic",
	)

	// Party UUID
	require.NotNil(t, ar1.Metadata.Parties)
	require.NotNil(t, ar2.Metadata.Parties)
	assert.Equal(t,
		(*ar1.Metadata.Parties)[0].UUID,
		(*ar2.Metadata.Parties)[0].UUID,
		"party UUID should be deterministic",
	)
}

func TestDifferentInputs_DifferentUUIDs(t *testing.T) {
	catalog1 := gemara.ControlCatalog{
		Metadata: gemara.Metadata{Id: "catalog-a", Version: "1.0.0"},
		Title:    "Catalog A",
		Groups:   []gemara.Group{{Id: "G1", Description: "Group"}},
		Controls: []gemara.Control{{
			Id: "C-1", Group: "G1", Title: "Control",
			AssessmentRequirements: []gemara.AssessmentRequirement{{Id: "C-1.1", Text: "Req"}},
		}},
	}
	catalog2 := gemara.ControlCatalog{
		Metadata: gemara.Metadata{Id: "catalog-b", Version: "1.0.0"},
		Title:    "Catalog B",
		Groups:   []gemara.Group{{Id: "G1", Description: "Group"}},
		Controls: []gemara.Control{{
			Id: "C-1", Group: "G1", Title: "Control",
			AssessmentRequirements: []gemara.AssessmentRequirement{{Id: "C-1.1", Text: "Req"}},
		}},
	}

	r1, err := CatalogToOSCAL(catalog1)
	require.NoError(t, err)

	r2, err := CatalogToOSCAL(catalog2)
	require.NoError(t, err)

	assert.NotEqual(t, r1.UUID, r2.UUID, "different catalog IDs should produce different UUIDs")
}

func TestNoCollisionBetweenAssemblyTypes(t *testing.T) {
	log := makeEvaluationLog(gemara.Actor{
		Name: "test-tool",
		Type: gemara.Software,
	}, []*gemara.AssessmentLog{
		makeAssessmentLog("REQ-1", "check", gemara.Passed, "", nil),
	})
	log.Metadata.Id = "eval-collision-test"
	log.Metadata.Version = "1.0.0"
	log.Target = gemara.Resource{Id: "sys-1", Name: "System", Type: gemara.Software}

	ar, err := EvaluationLogToOSCALAssessmentResults(log, WithImportApHref("#"))
	require.NoError(t, err)

	result := ar.Results[0]

	findingUUID := (*result.Findings)[0].UUID
	observationUUID := (*result.Observations)[0].UUID
	logEntryUUID := result.AssessmentLog.Entries[0].UUID
	resultUUID := result.UUID
	inventoryUUID := (*result.LocalDefinitions.InventoryItems)[0].UUID
	documentUUID := ar.UUID

	uuids := map[string]string{
		"finding":     findingUUID,
		"observation": observationUUID,
		"log-entry":   logEntryUUID,
		"result":      resultUUID,
		"inventory":   inventoryUUID,
		"document":    documentUUID,
	}

	seen := make(map[string]string)
	for name, uid := range uuids {
		if prev, exists := seen[uid]; exists {
			t.Errorf("UUID collision: %s and %s both produced %s", prev, name, uid)
		}
		seen[uid] = name
	}
}

func TestEmptyMetadataId_StillProducesDeterministicUUID(t *testing.T) {
	catalog := gemara.ControlCatalog{
		Metadata: gemara.Metadata{Id: "", Version: "1.0.0"},
		Title:    "Empty ID Catalog",
		Groups:   []gemara.Group{{Id: "G1", Description: "Group"}},
		Controls: []gemara.Control{{
			Id: "C-1", Group: "G1", Title: "Control",
			AssessmentRequirements: []gemara.AssessmentRequirement{{Id: "C-1.1", Text: "Req"}},
		}},
	}

	r1, err := CatalogToOSCAL(catalog)
	require.NoError(t, err)

	r2, err := CatalogToOSCAL(catalog)
	require.NoError(t, err)

	assert.Equal(t, r1.UUID, r2.UUID, "empty metadata.Id should still produce deterministic UUID")
	assert.NotEmpty(t, r1.UUID)
}

func TestSpecialCharactersInIds_ProduceDeterministicUUIDs(t *testing.T) {
	catalog := gemara.ControlCatalog{
		Metadata: gemara.Metadata{
			Id:      "catalog/with/slashes",
			Version: "1.0.0",
			Author:  gemara.Actor{Name: "Author with spaces & symbols!", Type: gemara.Human},
		},
		Title: "Special Chars Catalog",
		Groups: []gemara.Group{
			{Id: "G-1", Description: "Group"},
		},
		Controls: []gemara.Control{
			{
				Id: "CTRL-1(a)", Group: "G-1", Title: "Control",
				AssessmentRequirements: []gemara.AssessmentRequirement{
					{Id: "CTRL-1(a).01", Text: "Req"},
				},
			},
		},
	}

	r1, err := CatalogToOSCAL(catalog)
	require.NoError(t, err)

	r2, err := CatalogToOSCAL(catalog)
	require.NoError(t, err)

	assert.Equal(t, r1.UUID, r2.UUID, "special characters in IDs should still produce deterministic UUIDs")
	assert.NotEmpty(t, r1.UUID)
}

func TestSeparatorAmbiguity_DifferentInputsDiffer(t *testing.T) {
	// Verify that "/" in identity fields does not cause collisions.
	// catalog1: id="A/B", version="C" → source "gemara:A%2FB/C/catalog"
	// catalog2: id="A",   version="B/C" → source "gemara:A/B%2FC/catalog"
	// Without escaping, both would produce "gemara:A/B/C/catalog".
	catalog1 := gemara.ControlCatalog{
		Metadata: gemara.Metadata{Id: "A/B", Version: "C"},
		Title:    "Catalog 1",
		Groups:   []gemara.Group{{Id: "G1", Description: "G"}},
		Controls: []gemara.Control{{
			Id: "C-1", Group: "G1", Title: "C",
			AssessmentRequirements: []gemara.AssessmentRequirement{{Id: "C-1.1", Text: "R"}},
		}},
	}
	catalog2 := gemara.ControlCatalog{
		Metadata: gemara.Metadata{Id: "A", Version: "B/C"},
		Title:    "Catalog 2",
		Groups:   []gemara.Group{{Id: "G1", Description: "G"}},
		Controls: []gemara.Control{{
			Id: "C-1", Group: "G1", Title: "C",
			AssessmentRequirements: []gemara.AssessmentRequirement{{Id: "C-1.1", Text: "R"}},
		}},
	}

	r1, err := CatalogToOSCAL(catalog1)
	require.NoError(t, err)
	r2, err := CatalogToOSCAL(catalog2)
	require.NoError(t, err)

	assert.NotEqual(t, r1.UUID, r2.UUID, "id=A/B version=C must differ from id=A version=B/C")
}

func TestPinnedCatalogUUID(t *testing.T) {
	// Pin a known UUID to catch source-string regressions.
	// Source: "gemara:test-catalog/1.0.0/catalog" → UUIDv5(URL, ...).
	catalog := gemara.ControlCatalog{
		Metadata: gemara.Metadata{Id: "test-catalog", Version: "1.0.0"},
		Title:    "Test Catalog",
		Groups:   []gemara.Group{{Id: "G1", Description: "G"}},
		Controls: []gemara.Control{{
			Id: "C-1", Group: "G1", Title: "C",
			AssessmentRequirements: []gemara.AssessmentRequirement{{Id: "C-1.1", Text: "R"}},
		}},
	}

	r, err := CatalogToOSCAL(catalog)
	require.NoError(t, err)

	assert.Equal(t, "7c8fc6e5-7637-5f21-aedc-27d913baf0ae", r.UUID, "catalog UUID must match pinned value")
}

func TestNilEvaluations_DoNotPanic(t *testing.T) {
	log := gemara.EvaluationLog{
		Metadata: gemara.Metadata{
			Id:      "eval-nil-test",
			Version: "1.0.0",
			Author:  gemara.Actor{Name: "tool", Type: gemara.Software},
		},
		Evaluations: []*gemara.ControlEvaluation{nil, nil},
		Target:      gemara.Resource{Id: "sys-1", Name: "System", Type: gemara.Software},
	}

	ar, err := EvaluationLogToOSCALAssessmentResults(log, WithImportApHref("#"))
	require.NoError(t, err)
	assert.NotEmpty(t, ar.UUID)
}

func TestNilAssessmentLogs_DoNotPanic(t *testing.T) {
	log := gemara.EvaluationLog{
		Metadata: gemara.Metadata{
			Id:      "eval-nil-alog",
			Version: "1.0.0",
			Author:  gemara.Actor{Name: "tool", Type: gemara.Software},
		},
		Evaluations: []*gemara.ControlEvaluation{
			{
				Name:           "Test Control",
				Control:        gemara.EntryMapping{EntryId: "CTRL-1"},
				Result:         gemara.Passed,
				AssessmentLogs: []*gemara.AssessmentLog{nil, nil},
			},
		},
		Target: gemara.Resource{Id: "sys-1", Name: "System", Type: gemara.Software},
	}

	ar, err := EvaluationLogToOSCALAssessmentResults(log, WithImportApHref("#"))
	require.NoError(t, err)
	assert.NotEmpty(t, ar.UUID)
	// Should still produce a finding even with nil assessment logs
	require.NotNil(t, ar.Results[0].Findings)
	assert.Len(t, *ar.Results[0].Findings, 1)
}

func TestEmptyAuthorName_PartyNotCreated(t *testing.T) {
	catalog := gemara.ControlCatalog{
		Metadata: gemara.Metadata{
			Id:      "no-author-catalog",
			Version: "1.0.0",
			Author:  gemara.Actor{Name: "", Type: gemara.Human},
		},
		Title:  "No Author",
		Groups: []gemara.Group{{Id: "G1", Description: "G"}},
		Controls: []gemara.Control{{
			Id: "C-1", Group: "G1", Title: "C",
			AssessmentRequirements: []gemara.AssessmentRequirement{{Id: "C-1.1", Text: "R"}},
		}},
	}

	r, err := CatalogToOSCAL(catalog)
	require.NoError(t, err)
	assert.NotEmpty(t, r.UUID)
	// Empty author name means no Party is created (existing behavior)
	assert.Nil(t, r.Metadata.Parties)
}

func TestContentChange_DoesNotAffectUUIDs(t *testing.T) {
	makeLog := func(message string, result gemara.Result) gemara.EvaluationLog {
		return gemara.EvaluationLog{
			Metadata: gemara.Metadata{
				Id:      "eval-content-test",
				Version: "1.0.0",
				Author:  gemara.Actor{Name: "tool", Type: gemara.Software},
			},
			Evaluations: []*gemara.ControlEvaluation{
				{
					Name:    "Test",
					Control: gemara.EntryMapping{EntryId: "CTRL-1"},
					Result:  result,
					Message: message,
					AssessmentLogs: []*gemara.AssessmentLog{
						{
							Requirement: gemara.EntryMapping{EntryId: "REQ-1"},
							Description: "check requirement",
							Result:      result,
							Message:     message,
						},
					},
				},
			},
			Target: gemara.Resource{Id: "sys-1", Name: "System", Type: gemara.Software},
		}
	}

	// Different message, same control
	ar1, err := EvaluationLogToOSCALAssessmentResults(makeLog("passed all checks", gemara.Passed), WithImportApHref("#"))
	require.NoError(t, err)

	ar2, err := EvaluationLogToOSCALAssessmentResults(makeLog("satisfied all requirements", gemara.Passed), WithImportApHref("#"))
	require.NoError(t, err)

	// Different result, same control
	ar3, err := EvaluationLogToOSCALAssessmentResults(makeLog("failed", gemara.Failed), WithImportApHref("#"))
	require.NoError(t, err)

	// Document UUID should be same (same id + version)
	assert.Equal(t, ar1.UUID, ar2.UUID, "message change should not affect document UUID")
	assert.Equal(t, ar1.UUID, ar3.UUID, "result change should not affect document UUID")

	// Finding UUID should be same (same control ID)
	f1 := (*ar1.Results[0].Findings)[0].UUID
	f2 := (*ar2.Results[0].Findings)[0].UUID
	f3 := (*ar3.Results[0].Findings)[0].UUID
	assert.Equal(t, f1, f2, "message change should not affect finding UUID")
	assert.Equal(t, f1, f3, "result change should not affect finding UUID")

	// Observation UUID should be same (same control + requirement)
	o1 := (*ar1.Results[0].Observations)[0].UUID
	o2 := (*ar2.Results[0].Observations)[0].UUID
	o3 := (*ar3.Results[0].Observations)[0].UUID
	assert.Equal(t, o1, o2, "message change should not affect observation UUID")
	assert.Equal(t, o1, o3, "result change should not affect observation UUID")
}

func TestEmptyVersion_DefaultsBeforeUUIDGeneration(t *testing.T) {
	log := makeEvaluationLog(gemara.Actor{
		Name: "test-tool",
		Type: gemara.Software,
	}, []*gemara.AssessmentLog{
		makeAssessmentLog("REQ-1", "check", gemara.Passed, "", nil),
	})
	log.Metadata.Id = "eval-empty-version"
	log.Metadata.Version = ""
	log.Target = gemara.Resource{Id: "sys-1", Name: "System", Type: gemara.Software}

	ar1, err := EvaluationLogToOSCALAssessmentResults(log, WithImportApHref("#"))
	require.NoError(t, err)

	ar2, err := EvaluationLogToOSCALAssessmentResults(log, WithImportApHref("#"))
	require.NoError(t, err)

	// Empty version should still produce deterministic UUIDs
	assert.Equal(t, ar1.UUID, ar2.UUID, "empty version should produce deterministic document UUID")
	assert.NotEmpty(t, ar1.UUID)

	// The UUID should match what an explicit DefaultOSCALVersion would produce
	log.Metadata.Version = "0.0.0"
	ar3, err := EvaluationLogToOSCALAssessmentResults(log, WithImportApHref("#"))
	require.NoError(t, err)

	assert.Equal(t, ar1.UUID, ar3.UUID, "empty version should produce same UUID as explicit default version")
}

func TestVersionBump_ChangesDocumentUUID_KeepsAssemblyUUIDs(t *testing.T) {
	log := makeEvaluationLog(gemara.Actor{
		Name: "test-tool",
		Type: gemara.Software,
	}, []*gemara.AssessmentLog{
		makeAssessmentLog("REQ-1", "check", gemara.Passed, "", nil),
	})
	log.Metadata.Id = "eval-version-test"
	log.Target = gemara.Resource{Id: "sys-1", Name: "System", Type: gemara.Software}

	log.Metadata.Version = "1.0.0"
	ar1, err := EvaluationLogToOSCALAssessmentResults(log, WithImportApHref("#"))
	require.NoError(t, err)

	log.Metadata.Version = "2.0.0"
	ar2, err := EvaluationLogToOSCALAssessmentResults(log, WithImportApHref("#"))
	require.NoError(t, err)

	// Document UUID changes on version bump
	assert.NotEqual(t, ar1.UUID, ar2.UUID, "version bump should change document UUID")

	// Result UUID stays the same (per-subject: same log ID)
	assert.Equal(t, ar1.Results[0].UUID, ar2.Results[0].UUID,
		"result UUID should not change on version bump",
	)

	// Assembly UUIDs stay the same (per-subject: same control, same requirement)
	assert.Equal(t,
		(*ar1.Results[0].Findings)[0].UUID,
		(*ar2.Results[0].Findings)[0].UUID,
		"finding UUID should not change on version bump",
	)
	assert.Equal(t,
		(*ar1.Results[0].Observations)[0].UUID,
		(*ar2.Results[0].Observations)[0].UUID,
		"observation UUID should not change on version bump",
	)
	assert.Equal(t,
		(*ar1.Results[0].LocalDefinitions.InventoryItems)[0].UUID,
		(*ar2.Results[0].LocalDefinitions.InventoryItems)[0].UUID,
		"inventory item UUID should not change on version bump",
	)
}
