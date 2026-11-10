package gemaraconv

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/defenseunicorns/go-oscal/src/pkg/uuid"
	"github.com/gemaraproj/go-gemara"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pinConversionClock fixes nowFunc for the duration of the test so
// conversions that must agree on the date-segment fallback cannot straddle
// UTC midnight, and restores the real clock afterwards.
func pinConversionClock(t *testing.T, fixed time.Time) {
	t.Helper()
	orig := nowFunc
	nowFunc = func() time.Time { return fixed }
	t.Cleanup(func() { nowFunc = orig })
}

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
	require.NotNil(t, cat1.BackMatter, "guidance catalog should have back matter")
	require.NotNil(t, cat1.BackMatter.Resources, "guidance catalog should have back-matter resources")
	require.NotNil(t, cat2.BackMatter)
	require.NotNil(t, cat2.BackMatter.Resources)
	require.Len(t, *cat2.BackMatter.Resources, len(*cat1.BackMatter.Resources))
	for i := range *cat1.BackMatter.Resources {
		assert.Equal(t,
			(*cat1.BackMatter.Resources)[i].UUID,
			(*cat2.BackMatter.Resources)[i].UUID,
			"resource UUID should be deterministic",
		)
	}
}

func TestAssessmentResults_DeterministicUUIDs(t *testing.T) {
	// Pin the conversion clock: the document and Result UUIDs key on the
	// date-segment fallback, and two conversions straddling UTC midnight
	// would otherwise legitimately differ.
	pinConversionClock(t, time.Date(2025, 8, 22, 12, 0, 0, 0, time.UTC))

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
	// Pin the conversion clock so all three conversions share the date
	// segment; only the content under test may vary.
	pinConversionClock(t, time.Date(2025, 8, 22, 12, 0, 0, 0, time.UTC))

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
	// Pin the conversion clock: the date-segment fallback must not vary
	// between the conversions compared here.
	pinConversionClock(t, time.Date(2025, 8, 22, 12, 0, 0, 0, time.UTC))

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
	assert.Equal(t, "0.0.0", ar1.Metadata.Version,
		"the defaulted version must also be stamped into metadata.version")

	// The UUID should match what an explicit DefaultOSCALVersion would produce
	log.Metadata.Version = "0.0.0"
	ar3, err := EvaluationLogToOSCALAssessmentResults(log, WithImportApHref("#"))
	require.NoError(t, err)

	assert.Equal(t, ar1.UUID, ar3.UUID, "empty version should produce same UUID as explicit default version")
}

func TestVersionBump_ChangesDocumentUUID_KeepsAssemblyUUIDs(t *testing.T) {
	pinConversionClock(t, time.Date(2025, 8, 22, 12, 0, 0, 0, time.UTC))
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

func TestSameRequirement_MultipleLogs_DistinctUUIDs(t *testing.T) {
	// A requirement can be evaluated in more than one way, so one control
	// can carry several assessment logs for the same requirement. Their
	// observation and log-entry UUIDs must stay distinct within the document.
	log := makeEvaluationLog(gemara.Actor{
		Name: "test-tool",
		Type: gemara.Software,
	}, []*gemara.AssessmentLog{
		makeAssessmentLog("REQ-1", "examine artifact", gemara.Passed, "", nil),
		makeAssessmentLog("REQ-1", "test configuration", gemara.Failed, "", nil),
	})
	log.Metadata.Id = "eval-same-requirement"
	log.Metadata.Version = "1.0.0"
	log.Target = gemara.Resource{Id: "sys-1", Name: "System", Type: gemara.Software}

	ar, err := EvaluationLogToOSCALAssessmentResults(log, WithImportApHref("#"))
	require.NoError(t, err)

	require.NotNil(t, ar.Results[0].Observations)
	observations := *ar.Results[0].Observations
	require.Len(t, observations, 2)
	assert.NotEqual(t, observations[0].UUID, observations[1].UUID,
		"two logs for the same requirement must produce distinct observation UUIDs",
	)

	require.NotNil(t, ar.Results[0].AssessmentLog)
	entries := ar.Results[0].AssessmentLog.Entries
	require.Len(t, entries, 2)
	assert.NotEqual(t, entries[0].UUID, entries[1].UUID,
		"two logs for the same requirement must produce distinct log-entry UUIDs",
	)
}

func TestDateChange_ChangesRootAndResultUUID(t *testing.T) {
	// The document root and the Result are per-run: they change with the
	// assessment date. Per-subject assemblies must not.
	log := makeEvaluationLog(gemara.Actor{
		Name: "test-tool",
		Type: gemara.Software,
	}, []*gemara.AssessmentLog{
		makeAssessmentLog("REQ-1", "check", gemara.Passed, "", nil),
	})
	log.Metadata.Id = "eval-date-test"
	log.Metadata.Version = "1.0.0"
	log.Target = gemara.Resource{Id: "sys-1", Name: "System", Type: gemara.Software}

	log.Metadata.Date = "2026-01-01"
	ar1, err := EvaluationLogToOSCALAssessmentResults(log, WithImportApHref("#"))
	require.NoError(t, err)

	log.Metadata.Date = "2026-01-02"
	ar2, err := EvaluationLogToOSCALAssessmentResults(log, WithImportApHref("#"))
	require.NoError(t, err)

	assert.NotEqual(t, ar1.UUID, ar2.UUID, "date change should change the document UUID")
	assert.NotEqual(t, ar1.Results[0].UUID, ar2.Results[0].UUID, "date change should change the result UUID")

	assert.Equal(t,
		(*ar1.Results[0].Findings)[0].UUID,
		(*ar2.Results[0].Findings)[0].UUID,
		"finding UUID should not change with the date",
	)
	assert.Equal(t,
		(*ar1.Results[0].Observations)[0].UUID,
		(*ar2.Results[0].Observations)[0].UUID,
		"observation UUID should not change with the date",
	)
	assert.Equal(t,
		ar1.Results[0].AssessmentLog.Entries[0].UUID,
		ar2.Results[0].AssessmentLog.Entries[0].UUID,
		"log entry UUID should not change with the date",
	)
	assert.Equal(t,
		(*ar1.Results[0].LocalDefinitions.InventoryItems)[0].UUID,
		(*ar2.Results[0].LocalDefinitions.InventoryItems)[0].UUID,
		"inventory item UUID should not change with the date",
	)
}

func TestDuplicateControlIds_ReturnError(t *testing.T) {
	log := gemara.EvaluationLog{
		Metadata: gemara.Metadata{
			Id:      "eval-duplicate-control",
			Version: "1.0.0",
			Author:  gemara.Actor{Name: "tool", Type: gemara.Software},
		},
		Evaluations: []*gemara.ControlEvaluation{
			{
				Name:    "First",
				Control: gemara.EntryMapping{EntryId: "AC-1"},
				Result:  gemara.Passed,
			},
			{
				Name:    "Second",
				Control: gemara.EntryMapping{EntryId: "AC-1"},
				Result:  gemara.Failed,
			},
		},
		Target: gemara.Resource{Id: "sys-1", Name: "System", Type: gemara.Software},
	}

	_, err := EvaluationLogToOSCALAssessmentResults(log, WithImportApHref("#"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "AC-1", "error should name the duplicated control ID")
}

func TestSameEntryId_DifferentReferences_NotDuplicates(t *testing.T) {
	// The schema defines a control evaluation's identity as its mapping's
	// reference id plus entry id: two catalogs can map the same entry id.
	// The duplicate check and the finding UUID sources must key on both, so
	// the same entry id from two references is two controls, not a duplicate.
	log := gemara.EvaluationLog{
		Metadata: gemara.Metadata{
			Id:      "eval-two-references",
			Version: "1.0.0",
			Author:  gemara.Actor{Name: "tool", Type: gemara.Software},
		},
		Evaluations: []*gemara.ControlEvaluation{
			{
				Name:    "OSPS access control",
				Control: gemara.EntryMapping{ReferenceId: "osps", EntryId: "OSPS-AC-01"},
				Result:  gemara.Passed,
				AssessmentLogs: []*gemara.AssessmentLog{
					makeAssessmentLog("REQ-1", "check", gemara.Passed, "", nil),
				},
			},
			{
				Name:    "NIST access control",
				Control: gemara.EntryMapping{ReferenceId: "nist", EntryId: "OSPS-AC-01"},
				Result:  gemara.Failed,
				AssessmentLogs: []*gemara.AssessmentLog{
					makeAssessmentLog("REQ-1", "check differently", gemara.Failed, "", nil),
				},
			},
		},
		Target: gemara.Resource{Id: "sys-1", Name: "System", Type: gemara.Software},
	}

	ar, err := EvaluationLogToOSCALAssessmentResults(log, WithImportApHref("#"))
	require.NoError(t, err,
		"the same entry id mapped through two references is two controls, not a duplicate")

	require.NotNil(t, ar.Results[0].Findings)
	findings := *ar.Results[0].Findings
	require.Len(t, findings, 2)
	assert.NotEqual(t, findings[0].UUID, findings[1].UUID,
		"findings for the same entry id under different references must not share a UUID")
}

func TestDuplicateControlEvaluations_SameReference_StillRejected(t *testing.T) {
	// Keying on reference id plus entry id must not relax the rejection of
	// true duplicates: the same mapping evaluated twice still errors.
	log := gemara.EvaluationLog{
		Metadata: gemara.Metadata{
			Id:      "eval-duplicate-with-reference",
			Version: "1.0.0",
			Author:  gemara.Actor{Name: "tool", Type: gemara.Software},
		},
		Evaluations: []*gemara.ControlEvaluation{
			{
				Name:    "First",
				Control: gemara.EntryMapping{ReferenceId: "osps", EntryId: "AC-1"},
				Result:  gemara.Passed,
			},
			{
				Name:    "Second",
				Control: gemara.EntryMapping{ReferenceId: "osps", EntryId: "AC-1"},
				Result:  gemara.Failed,
			},
		},
		Target: gemara.Resource{Id: "sys-1", Name: "System", Type: gemara.Software},
	}

	_, err := EvaluationLogToOSCALAssessmentResults(log, WithImportApHref("#"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "osps/AC-1",
		"error should name the duplicated control by reference and entry id")
}

func TestPlan_DistinguishesSameRequirementLogs(t *testing.T) {
	// A recorded plan is part of the assessment log's identity: two logs
	// for the same control and requirement under different plans get
	// distinct observation and log-entry UUIDs without spending occurrence
	// counter values on them.
	log := makeEvaluationLog(gemara.Actor{Name: "tool", Type: gemara.Software}, []*gemara.AssessmentLog{
		makeAssessmentLog("REQ-1", "check", gemara.Passed, "", nil),
		makeAssessmentLog("REQ-1", "check differently", gemara.Failed, "", nil),
	})
	log.Metadata.Id = "eval-two-plans"
	log.Metadata.Version = "1.0.0"
	log.Evaluations[0].AssessmentLogs[0].Plan = &gemara.EntryMapping{ReferenceId: "plans", EntryId: "plan-a"}
	log.Evaluations[0].AssessmentLogs[1].Plan = &gemara.EntryMapping{ReferenceId: "plans", EntryId: "plan-b"}

	ar, err := EvaluationLogToOSCALAssessmentResults(log, WithImportApHref("#"))
	require.NoError(t, err)

	require.NotNil(t, ar.Results[0].Observations)
	observations := *ar.Results[0].Observations
	require.Len(t, observations, 2)
	assert.NotEqual(t, observations[0].UUID, observations[1].UUID,
		"logs for the same requirement under different plans must not share an observation UUID")

	require.NotNil(t, ar.Results[0].AssessmentLog)
	entries := ar.Results[0].AssessmentLog.Entries
	require.Len(t, entries, 2)
	assert.NotEqual(t, entries[0].UUID, entries[1].UUID,
		"logs for the same requirement under different plans must not share a log-entry UUID")
}

func TestConflictingMappingReferences_ReturnError(t *testing.T) {
	log := makeEvaluationLog(gemara.Actor{
		Name: "test-tool",
		Type: gemara.Software,
	}, []*gemara.AssessmentLog{
		makeAssessmentLog("REQ-1", "check", gemara.Passed, "", nil),
	})
	log.Metadata.Id = "eval-conflicting-refs"
	log.Metadata.Version = "1.0.0"
	log.Metadata.MappingReferences = []gemara.MappingReference{
		{Id: "ref-1", Title: "First title", Url: "https://example.com/a"},
		{Id: "ref-1", Title: "Different title", Url: "https://example.com/a"},
	}
	log.Target = gemara.Resource{Id: "sys-1", Name: "System", Type: gemara.Software}

	_, err := EvaluationLogToOSCALAssessmentResults(log, WithImportApHref("#"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ref-1", "error should name the conflicting mapping reference")
}

func TestSameAuthor_SamePartyUUIDAcrossDocuments(t *testing.T) {
	author := gemara.Actor{Name: "Alice", Type: gemara.Human}

	catalog := gemara.ControlCatalog{
		Metadata: gemara.Metadata{
			Id:      "party-catalog",
			Version: "1.0.0",
			Author:  author,
		},
		Title:  "Party Catalog",
		Groups: []gemara.Group{{Id: "G1", Description: "G"}},
		Controls: []gemara.Control{{
			Id: "C-1", Group: "G1", Title: "C",
			AssessmentRequirements: []gemara.AssessmentRequirement{{Id: "C-1.1", Text: "R"}},
		}},
	}

	log := makeEvaluationLog(author, []*gemara.AssessmentLog{
		makeAssessmentLog("REQ-1", "check", gemara.Passed, "", nil),
	})
	log.Metadata.Id = "party-results"
	log.Metadata.Version = "1.0.0"
	log.Target = gemara.Resource{Id: "sys-1", Name: "System", Type: gemara.Software}

	cat, err := CatalogToOSCAL(catalog)
	require.NoError(t, err)
	ar, err := EvaluationLogToOSCALAssessmentResults(log, WithImportApHref("#"))
	require.NoError(t, err)

	require.NotNil(t, cat.Metadata.Parties, "catalog should have an author party")
	require.NotNil(t, ar.Metadata.Parties, "assessment results should have an author party")

	assert.Equal(t,
		(*cat.Metadata.Parties)[0].UUID,
		(*ar.Metadata.Parties)[0].UUID,
		"the same author must get the same Party UUID in a catalog and in assessment results",
	)
}

func TestActorId_RenameKeepsPartyUUID(t *testing.T) {
	base := gemara.Actor{Id: "author-1", Name: "Alice", Type: gemara.Human}
	renamed := gemara.Actor{Id: "author-1", Name: "Alice Smith", Type: gemara.Human}

	baseUUID := firstPartyUUID(t, base)
	renamedUUID := firstPartyUUID(t, renamed)

	assert.Equal(t, baseUUID, renamedUUID,
		"renaming an actor must keep its Party UUID when the id is unchanged",
	)
}

func TestActorId_SameNameDifferentId_DistinctUUID(t *testing.T) {
	a := gemara.Actor{Id: "author-1", Name: "Alice", Type: gemara.Human}
	b := gemara.Actor{Id: "author-2", Name: "Alice", Type: gemara.Human}

	assert.NotEqual(t, firstPartyUUID(t, a), firstPartyUUID(t, b),
		"two actors sharing a name must not share a Party UUID",
	)
}

func TestActorId_EmptyFallsBackToNameType(t *testing.T) {
	// No id: the source falls back to name and type, so the same actor
	// still gets the same UUID as before the Actor.Id change.
	author := gemara.Actor{Name: "Alice", Type: gemara.Human}

	catalog := gemara.ControlCatalog{
		Metadata: gemara.Metadata{Id: "fallback-catalog", Version: "1.0.0", Author: author},
		Title:    "Fallback",
		Groups:   []gemara.Group{{Id: "G1", Description: "G"}},
		Controls: []gemara.Control{{
			Id: "C-1", Group: "G1", Title: "C",
			AssessmentRequirements: []gemara.AssessmentRequirement{{Id: "C-1.1", Text: "R"}},
		}},
	}

	r, err := CatalogToOSCAL(catalog)
	require.NoError(t, err)
	require.NotNil(t, r.Metadata.Parties)

	want := uuid.NewUUIDWithSource(fmt.Sprintf("gemara:%s/%s/party",
		url.PathEscape("Alice"), url.PathEscape(gemara.Human.String())))
	assert.Equal(t, want, (*r.Metadata.Parties)[0].UUID,
		"an author without an id falls back to the name/type source",
	)
}

// firstPartyUUID converts a catalog with the given author and returns the
// UUID of its single author party.
func firstPartyUUID(t *testing.T, author gemara.Actor) string {
	t.Helper()

	catalog := gemara.ControlCatalog{
		Metadata: gemara.Metadata{Id: "party-catalog", Version: "1.0.0", Author: author},
		Title:    "Party Catalog",
		Groups:   []gemara.Group{{Id: "G1", Description: "G"}},
		Controls: []gemara.Control{{
			Id: "C-1", Group: "G1", Title: "C",
			AssessmentRequirements: []gemara.AssessmentRequirement{{Id: "C-1.1", Text: "R"}},
		}},
	}

	r, err := CatalogToOSCAL(catalog)
	require.NoError(t, err)
	require.NotNil(t, r.Metadata.Parties)
	require.Len(t, *r.Metadata.Parties, 1)
	return (*r.Metadata.Parties)[0].UUID
}

func TestDuplicateMappingReferences_Deduplicated(t *testing.T) {
	one := gemara.MappingReference{Id: "osps", Title: "OSPS", Url: "https://example.com/osps"}
	identical := []gemara.MappingReference{one, one}
	differentTitle := []gemara.MappingReference{one, {Id: "osps", Title: "Other", Url: one.Url}}
	differentURL := []gemara.MappingReference{one, {Id: "osps", Title: one.Title, Url: "https://example.com/other"}}

	bm, _, err := mappingToBackMatterWithUUIDs([]gemara.MappingReference{one})
	require.NoError(t, err)
	require.Len(t, *bm.Resources, 1)

	deduped, _, err := mappingToBackMatterWithUUIDs(identical)
	require.NoError(t, err)
	require.Len(t, *deduped.Resources, 1,
		"an identical duplicate mapping reference is deduplicated, not repeated")
	assert.Equal(t, (*bm.Resources)[0].UUID, (*deduped.Resources)[0].UUID,
		"the kept resource keeps the UUID a single reference would get")

	_, _, err = mappingToBackMatterWithUUIDs(differentTitle)
	require.Error(t, err, "same id with a different title must error")
	assert.Contains(t, err.Error(), "osps")

	_, _, err = mappingToBackMatterWithUUIDs(differentURL)
	require.Error(t, err, "same id with a different url must error")
	assert.Contains(t, err.Error(), "osps")
}

func TestGuidanceConflictingMappingReferences_ReturnError(t *testing.T) {
	g := guidanceWithExternalExtends()
	g.Metadata.MappingReferences = append(g.Metadata.MappingReferences, gemara.MappingReference{
		Id:    "NIST-800-53",
		Title: "A different title for the same id",
	})

	_, _, err := GuidanceToOSCAL(g, "catalog.json",
		WithOSCALImports(map[string]string{"NIST-800-53": "https://nist.gov/800-53/catalog.json"}))
	require.Error(t, err, "GuidanceToOSCAL must propagate the mapping-reference conflict")
	assert.Contains(t, err.Error(), "NIST-800-53")
}

func TestEmptyVersion_CatalogDefaultsBeforeUUIDGeneration(t *testing.T) {
	catalog := gemara.ControlCatalog{
		Metadata: gemara.Metadata{Id: "empty-version-catalog", Author: gemara.Actor{Id: "a1", Name: "A", Type: gemara.Human}},
		Title:    "Empty Version",
		Groups:   []gemara.Group{{Id: "G1", Description: "G"}},
		Controls: []gemara.Control{{
			Id: "C-1", Group: "G1", Title: "C",
			AssessmentRequirements: []gemara.AssessmentRequirement{{Id: "C-1.1", Text: "R"}},
		}},
	}

	defaulted, err := CatalogToOSCAL(catalog)
	require.NoError(t, err)
	pinned, err := CatalogToOSCAL(catalog, WithVersion("0.0.0"))
	require.NoError(t, err)

	assert.Equal(t, pinned.UUID, defaulted.UUID,
		"an empty version must default to 0.0.0 before the root UUID is generated")
	assert.Equal(t, "0.0.0", defaulted.Metadata.Version)
}

func TestEmptyVersion_GuidanceDefaultsBeforeUUIDGeneration(t *testing.T) {
	g := guidanceWithExternalExtends()
	g.Metadata.Version = ""

	catDefaulted, profDefaulted, err := GuidanceToOSCAL(g, "catalog.json",
		WithOSCALImports(map[string]string{"NIST-800-53": "https://nist.gov/800-53/catalog.json"}))
	require.NoError(t, err)
	catPinned, profPinned, err := GuidanceToOSCAL(g, "catalog.json",
		WithOSCALImports(map[string]string{"NIST-800-53": "https://nist.gov/800-53/catalog.json"}),
		WithVersion("0.0.0"))
	require.NoError(t, err)

	assert.Equal(t, catPinned.UUID, catDefaulted.UUID,
		"an empty version must default to 0.0.0 before the guidance catalog root UUID is generated")
	assert.Equal(t, profPinned.UUID, profDefaulted.UUID,
		"an empty version must default to 0.0.0 before the profile root UUID is generated")
	assert.Equal(t, "0.0.0", catDefaulted.Metadata.Version)
}

func TestSameResourceAndInventory_StableAcrossDocuments(t *testing.T) {
	author := gemara.Actor{Id: "author-1", Name: "Tool", Type: gemara.Software}
	ref := gemara.MappingReference{Id: "osps", Title: "OSPS", Url: "https://example.com/osps"}

	logA := makeEvaluationLog(author, []*gemara.AssessmentLog{
		makeAssessmentLog("REQ-1", "check", gemara.Passed, "", nil),
	})
	logA.Metadata.Id = "log-a"
	logA.Metadata.Version = "1.0.0"
	logA.Metadata.MappingReferences = []gemara.MappingReference{ref}
	logA.Target = gemara.Resource{Id: "sys-1", Name: "System", Type: gemara.Software}

	logB := makeEvaluationLog(author, []*gemara.AssessmentLog{
		makeAssessmentLog("REQ-2", "check differently", gemara.Failed, "", nil),
	})
	logB.Metadata.Id = "log-b"
	logB.Metadata.Version = "1.0.0"
	logB.Metadata.MappingReferences = []gemara.MappingReference{ref}
	logB.Target = gemara.Resource{Id: "sys-1", Name: "System", Type: gemara.Software}

	arA, err := EvaluationLogToOSCALAssessmentResults(logA, WithImportApHref("#"))
	require.NoError(t, err)
	arB, err := EvaluationLogToOSCALAssessmentResults(logB, WithImportApHref("#"))
	require.NoError(t, err)

	require.NotNil(t, arA.BackMatter.Resources)
	require.NotNil(t, arB.BackMatter.Resources)
	assert.Equal(t,
		(*arA.BackMatter.Resources)[0].UUID,
		(*arB.BackMatter.Resources)[0].UUID,
		"the same mapping reference id must get the same resource UUID across documents",
	)

	require.NotNil(t, arA.Results[0].LocalDefinitions.InventoryItems)
	require.NotNil(t, arB.Results[0].LocalDefinitions.InventoryItems)
	assert.Equal(t,
		(*arA.Results[0].LocalDefinitions.InventoryItems)[0].UUID,
		(*arB.Results[0].LocalDefinitions.InventoryItems)[0].UUID,
		"the same target id must get the same inventory-item UUID across documents",
	)
}

func TestEmptyMetadataId_ReturnsError(t *testing.T) {
	// The metadata id keys the finding, observation, and log-entry UUID
	// sources; a log without one would share those UUIDs with every other
	// unidentified log evaluating the same control.
	log := makeEvaluationLog(gemara.Actor{Name: "tool", Type: gemara.Software}, []*gemara.AssessmentLog{
		makeAssessmentLog("REQ-1", "check", gemara.Passed, "", nil),
	})
	log.Metadata.Id = ""
	log.Metadata.Version = "1.0.0"
	log.Target = gemara.Resource{Id: "sys-1", Name: "System", Type: gemara.Software}

	_, err := EvaluationLogToOSCALAssessmentResults(log, WithImportApHref("#"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "metadata id", "error should name the missing field")
}

func TestEmptyTargetId_ReturnsError(t *testing.T) {
	// The target id keys the inventory-item UUID source; a log without one
	// would share that UUID with every other unidentified target.
	log := makeEvaluationLog(gemara.Actor{Name: "tool", Type: gemara.Software}, []*gemara.AssessmentLog{
		makeAssessmentLog("REQ-1", "check", gemara.Passed, "", nil),
	})
	log.Metadata.Id = "eval-no-target"
	log.Metadata.Version = "1.0.0"
	log.Target = gemara.Resource{Name: "System", Type: gemara.Software}

	_, err := EvaluationLogToOSCALAssessmentResults(log, WithImportApHref("#"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "target id", "error should name the missing field")
}

func TestEmptyDate_FallsBackToConversionDate(t *testing.T) {
	// The root and Result UUIDs are per-run: a log without a date must not
	// freeze them onto a shared empty-string date segment, but fall back to
	// the conversion date. The clock is pinned so the explicit date below
	// and the fallback inside the conversion cannot disagree.
	fixed := time.Date(2025, 8, 22, 12, 0, 0, 0, time.UTC)
	pinConversionClock(t, fixed)

	log := makeEvaluationLog(gemara.Actor{Name: "tool", Type: gemara.Software}, []*gemara.AssessmentLog{
		makeAssessmentLog("REQ-1", "check", gemara.Passed, "", nil),
	})
	log.Metadata.Id = "eval-empty-date"
	log.Metadata.Version = "1.0.0"
	log.Target = gemara.Resource{Id: "sys-1", Name: "System", Type: gemara.Software}

	ar, err := EvaluationLogToOSCALAssessmentResults(log, WithImportApHref("#"))
	require.NoError(t, err)

	explicit := log
	explicit.Metadata.Date = gemara.Datetime(fixed.UTC().Format("2006-01-02"))
	arExplicit, err := EvaluationLogToOSCALAssessmentResults(explicit, WithImportApHref("#"))
	require.NoError(t, err)

	assert.Equal(t, arExplicit.UUID, ar.UUID,
		"an empty date should fall back to the conversion date for the root UUID")
	assert.Equal(t, arExplicit.Results[0].UUID, ar.Results[0].UUID,
		"an empty date should fall back to the conversion date for the Result UUID")
}

func TestFindingsObservationsLogEntries_DistinctAcrossDocuments(t *testing.T) {
	author := gemara.Actor{Id: "author-1", Name: "Tool", Type: gemara.Software}

	// Both logs evaluate the same control (makeEvaluationLog's fixed CTRL-1)
	// and the same requirement, so only the log id differs: distinct finding,
	// observation, and log-entry UUIDs are attributable to the log id alone.
	logA := makeEvaluationLog(author, []*gemara.AssessmentLog{
		makeAssessmentLog("REQ-1", "check", gemara.Passed, "", nil),
	})
	logA.Metadata.Id = "log-a"
	logA.Metadata.Version = "1.0.0"

	logB := makeEvaluationLog(author, []*gemara.AssessmentLog{
		makeAssessmentLog("REQ-1", "check differently", gemara.Failed, "", nil),
	})
	logB.Metadata.Id = "log-b"
	logB.Metadata.Version = "1.0.0"

	arA, err := EvaluationLogToOSCALAssessmentResults(logA, WithImportApHref("#"))
	require.NoError(t, err)
	arB, err := EvaluationLogToOSCALAssessmentResults(logB, WithImportApHref("#"))
	require.NoError(t, err)

	require.NotNil(t, arA.Results[0].Findings)
	require.NotNil(t, arB.Results[0].Findings)
	assert.NotEqual(t,
		(*arA.Results[0].Findings)[0].UUID,
		(*arB.Results[0].Findings)[0].UUID,
		"findings for the same control in two different logs must not share a UUID",
	)

	require.NotNil(t, arA.Results[0].Observations)
	require.NotNil(t, arB.Results[0].Observations)
	assert.NotEqual(t,
		(*arA.Results[0].Observations)[0].UUID,
		(*arB.Results[0].Observations)[0].UUID,
		"observations for the same control and requirement in two different logs must not share a UUID",
	)

	require.NotNil(t, arA.Results[0].AssessmentLog)
	require.NotNil(t, arB.Results[0].AssessmentLog)
	assert.NotEqual(t,
		arA.Results[0].AssessmentLog.Entries[0].UUID,
		arB.Results[0].AssessmentLog.Entries[0].UUID,
		"log entries for the same control and requirement in two different logs must not share a UUID",
	)
}

func TestAnonymousAuthor_OmitsPartyAndOrigins(t *testing.T) {
	// An author with neither id nor name has no identity to key a party UUID
	// on; the conversion must not derive one, or every anonymous actor of the
	// same entity type would collapse onto a single shared party.
	log := makeEvaluationLog(gemara.Actor{Type: gemara.Software}, []*gemara.AssessmentLog{
		makeAssessmentLog("REQ-1", "check", gemara.Passed, "", nil),
	})
	log.Metadata.Id = "log-anon"
	log.Metadata.Version = "1.0.0"

	ar, err := EvaluationLogToOSCALAssessmentResults(log, WithImportApHref("#"))
	require.NoError(t, err)

	require.Nil(t, ar.Metadata.Parties, "an anonymous author must not produce a party")

	require.NotNil(t, ar.Results[0].Findings)
	assert.Nil(t, (*ar.Results[0].Findings)[0].Origins,
		"findings from an anonymous author must carry no origin actors")

	require.NotNil(t, ar.Results[0].Observations)
	assert.Nil(t, (*ar.Results[0].Observations)[0].Origins,
		"observations from an anonymous author must carry no origin actors")

	require.NotNil(t, ar.Results[0].AssessmentLog)
	assert.Nil(t, ar.Results[0].AssessmentLog.Entries[0].LoggedBy,
		"log entries from an anonymous author must not be logged by a phantom party")
}

func TestAuthorWithIdButNoName_OmitsPartyAndOrigins(t *testing.T) {
	// The party-emission gate (author name) and the reference gate (origins,
	// logged-by) must agree: an author with an id but no name must not get
	// origin actors or logged-by entries pointing at a party the document
	// never defines.
	log := makeEvaluationLog(gemara.Actor{Id: "author-1", Type: gemara.Software}, []*gemara.AssessmentLog{
		makeAssessmentLog("REQ-1", "check", gemara.Passed, "", nil),
	})
	log.Metadata.Id = "log-id-only"
	log.Metadata.Version = "1.0.0"

	ar, err := EvaluationLogToOSCALAssessmentResults(log, WithImportApHref("#"))
	require.NoError(t, err)

	require.Nil(t, ar.Metadata.Parties,
		"an author with no name must not produce a party, even when an id is present")

	require.NotNil(t, ar.Results[0].Findings)
	assert.Nil(t, (*ar.Results[0].Findings)[0].Origins,
		"findings from an author with no name must carry no origin actors")

	require.NotNil(t, ar.Results[0].Observations)
	assert.Nil(t, (*ar.Results[0].Observations)[0].Origins,
		"observations from an author with no name must carry no origin actors")

	require.NotNil(t, ar.Results[0].AssessmentLog)
	assert.Nil(t, ar.Results[0].AssessmentLog.Entries[0].LoggedBy,
		"log entries from an author with no name must not reference an undefined party")
}

func TestAuthorPartyType_FollowsEntityType(t *testing.T) {
	// Pins the emitted Party.Type end to end: the unit test on mapPartyType
	// alone would keep passing if createMetadata or
	// createAssessmentResultsMetadata reverted to a hardcoded "person".
	softwareAuthor := gemara.Actor{Id: "scanbot-1", Name: "ScanBot", Type: gemara.Software}
	humanAuthor := gemara.Actor{Id: "alice-1", Name: "Alice", Type: gemara.Human}

	ar, err := EvaluationLogToOSCALAssessmentResults(
		makeEvaluationLog(softwareAuthor, []*gemara.AssessmentLog{
			makeAssessmentLog("REQ-1", "check", gemara.Passed, "", nil),
		}), WithImportApHref("#"))
	require.NoError(t, err)
	require.NotNil(t, ar.Metadata.Parties)
	assert.Equal(t, "organization", (*ar.Metadata.Parties)[0].Type,
		"assessment-results author parties must be typed by entity type")

	catalog := gemara.ControlCatalog{
		Metadata: gemara.Metadata{
			Id:      "party-type-catalog",
			Version: "1.0.0",
			Author:  softwareAuthor,
		},
		Title: "Party Type Catalog",
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
	oscalCatalog, err := CatalogToOSCAL(catalog)
	require.NoError(t, err)
	require.NotNil(t, oscalCatalog.Metadata.Parties)
	assert.Equal(t, "organization", (*oscalCatalog.Metadata.Parties)[0].Type,
		"catalog author parties must be typed by entity type")

	catalog.Metadata.Author = humanAuthor
	oscalCatalog, err = CatalogToOSCAL(catalog)
	require.NoError(t, err)
	require.NotNil(t, oscalCatalog.Metadata.Parties)
	assert.Equal(t, "person", (*oscalCatalog.Metadata.Parties)[0].Type,
		"human author parties must be typed as person")
}
