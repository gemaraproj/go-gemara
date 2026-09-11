// SPDX-License-Identifier: Apache-2.0

package gemaraconv

import (
	"encoding/json"
	"testing"

	oscal "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/gemaraproj/go-gemara"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testImportApHref = "#6ba7b810-9dad-11d1-80b4-00c04fd430c8"

func TestEvaluationLogToOSCALAssessmentResults(t *testing.T) {
	log := makeEvaluationLog(gemara.Actor{
		Name:    "test-tool",
		Uri:     "https://github.com/test/tool",
		Version: "1.0.0",
		Type:    gemara.Software,
	}, []*gemara.AssessmentLog{
		makeAssessmentLog("REQ-1", "check first requirement", gemara.Failed, "requirement not met", nil),
		makeAssessmentLog("REQ-2", "check second requirement", gemara.Passed, "", nil),
	})
	log.Metadata.Id = "eval-001"
	log.Target = gemara.Resource{Id: "sys-1", Name: "Test System", Type: gemara.Software}

	ar, err := EvaluationLogToOSCALAssessmentResults(log, WithImportApHref(testImportApHref))
	require.NoError(t, err)

	assert.NotEmpty(t, ar.UUID)
	assert.Equal(t, testImportApHref, ar.ImportAp.Href)
	assert.Contains(t, ar.Metadata.Title, "eval-001")
	require.Len(t, ar.Results, 1)

	result := ar.Results[0]
	assert.Contains(t, result.Title, "eval-001")
	require.NotNil(t, result.Findings)
	require.NotNil(t, result.Observations)
	require.Len(t, *result.Findings, 1)
	require.Len(t, *result.Observations, 2)

	finding := (*result.Findings)[0]
	assert.Equal(t, "CTRL-1", finding.Target.TargetId)
	assert.Equal(t, "objective-id", finding.Target.Type)

	require.NotNil(t, result.ReviewedControls.ControlSelections)
	require.Len(t, result.ReviewedControls.ControlSelections, 1)
	sel := result.ReviewedControls.ControlSelections[0]
	require.NotNil(t, sel.IncludeControls)
	require.Len(t, *sel.IncludeControls, 1)
	assert.Equal(t, "CTRL-1", (*sel.IncludeControls)[0].ControlId)

	assertValidJSON(t, ar)
}

func TestEvaluationLogToOSCALAssessmentResults_WithCatalogEnrichment(t *testing.T) {
	catalog := makeCatalog("CTRL-1", "Access Control", "Enforce access controls", "REQ-1", "Verify access is restricted", "Use RBAC")

	log := makeEvaluationLog(gemara.Actor{Name: "tool", Type: gemara.Software}, []*gemara.AssessmentLog{
		makeAssessmentLog("REQ-1", "check access", gemara.Failed, "access unrestricted", nil),
	})
	log.Metadata.Id = "eval-enriched"

	ar, err := EvaluationLogToOSCALAssessmentResults(log, WithImportApHref(testImportApHref), WithCatalog(catalog))
	require.NoError(t, err)
	require.Len(t, ar.Results, 1)

	finding := (*ar.Results[0].Findings)[0]
	assert.Equal(t, "Access Control", finding.Title)
	assertValidJSON(t, ar)
}

func TestEvaluationLogToOSCALAssessmentResults_ResultMapping(t *testing.T) {
	tests := []struct {
		result      gemara.Result
		wantState   string
		wantReason  string
		description string
	}{
		{gemara.Passed, "satisfied", "", "passed maps to satisfied"},
		{gemara.Failed, "not-satisfied", "", "failed maps to not-satisfied"},
		{gemara.NeedsReview, "not-satisfied", "Needs Review", "needs-review maps to not-satisfied with reason"},
		{gemara.Unknown, "not-satisfied", "Unknown", "unknown maps to not-satisfied with reason"},
		{gemara.NotApplicable, "not-satisfied", "Not Applicable", "not-applicable maps to not-satisfied with reason"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			log := makeEvaluationLog(gemara.Actor{Name: "tool", Type: gemara.Software}, []*gemara.AssessmentLog{
				makeAssessmentLog("REQ-1", "check", tt.result, "", nil),
			})
			log.Evaluations[0].Result = tt.result

			ar, err := EvaluationLogToOSCALAssessmentResults(log)
			require.NoError(t, err)

			finding := (*ar.Results[0].Findings)[0]
			assert.Equal(t, tt.wantState, finding.Target.Status.State)
			assert.Equal(t, tt.wantReason, finding.Target.Status.Reason)
		})
	}
}

func TestEvaluationLogToOSCALAssessmentResults_DefaultImportApHref(t *testing.T) {
	log := makeEvaluationLog(gemara.Actor{Name: "tool", Type: gemara.Software}, []*gemara.AssessmentLog{
		makeAssessmentLog("REQ-1", "check", gemara.Passed, "", nil),
	})

	ar, err := EvaluationLogToOSCALAssessmentResults(log)
	require.NoError(t, err)
	assert.Equal(t, "#", ar.ImportAp.Href)
}

func TestEvaluationLogToOSCALAssessmentResults_ObservationMethod(t *testing.T) {
	log := makeEvaluationLog(gemara.Actor{Name: "tool", Type: gemara.Software}, []*gemara.AssessmentLog{
		makeAssessmentLog("REQ-1", "automated check", gemara.Passed, "", nil),
	})

	ar, err := EvaluationLogToOSCALAssessmentResults(log)
	require.NoError(t, err)

	obs := (*ar.Results[0].Observations)[0]
	assert.Contains(t, obs.Methods, "TEST")
}

func TestEvaluationLogToOSCALAssessmentResults_AssessmentLogEntries(t *testing.T) {
	log := makeEvaluationLog(gemara.Actor{Name: "tool", Type: gemara.Software}, []*gemara.AssessmentLog{
		makeAssessmentLog("REQ-1", "first check", gemara.Passed, "", nil),
		makeAssessmentLog("REQ-2", "second check", gemara.Failed, "broke", nil),
	})

	ar, err := EvaluationLogToOSCALAssessmentResults(log)
	require.NoError(t, err)

	result := ar.Results[0]
	require.NotNil(t, result.AssessmentLog)
	require.Len(t, result.AssessmentLog.Entries, 2)
	assert.Contains(t, result.AssessmentLog.Entries[0].Title, "REQ-1")
	assert.Contains(t, result.AssessmentLog.Entries[1].Title, "REQ-2")
}

func TestEvaluationLogToOSCALAssessmentResults_TargetInventoryItem(t *testing.T) {
	log := makeEvaluationLog(gemara.Actor{Name: "tool", Type: gemara.Software}, []*gemara.AssessmentLog{
		makeAssessmentLog("REQ-1", "check", gemara.Passed, "", nil),
	})
	log.Target = gemara.Resource{Id: "my-sys", Name: "Production System", Description: "The prod system"}

	ar, err := EvaluationLogToOSCALAssessmentResults(log)
	require.NoError(t, err)

	result := ar.Results[0]
	require.NotNil(t, result.LocalDefinitions)
	require.NotNil(t, result.LocalDefinitions.InventoryItems)
	require.Len(t, *result.LocalDefinitions.InventoryItems, 1)

	item := (*result.LocalDefinitions.InventoryItems)[0]
	assert.Equal(t, "The prod system", item.Description)
	require.NotNil(t, item.Props)
	props := *item.Props
	assert.Equal(t, "my-sys", props[0].Value)
	assert.Equal(t, "Production System", props[1].Value)
}

func TestEvaluationLogConverter_ToOSCALAssessmentResults(t *testing.T) {
	log := makeEvaluationLog(gemara.Actor{Name: "tool", Type: gemara.Software},
		[]*gemara.AssessmentLog{makeAssessmentLog("REQ-1", "check", gemara.Passed, "", nil)})
	log.Metadata.Id = "eval-converter"

	converter := EvaluationLog(log)
	ar, err := converter.ToOSCALAssessmentResults(WithImportApHref(testImportApHref))
	require.NoError(t, err)
	require.Len(t, ar.Results, 1)
	assert.Contains(t, ar.Results[0].Title, "eval-converter")
}

func TestEvaluationLogToOSCALAssessmentResults_BackMatter(t *testing.T) {
	log := makeEvaluationLog(gemara.Actor{Name: "tool", Type: gemara.Software}, []*gemara.AssessmentLog{
		makeAssessmentLog("REQ-1", "check", gemara.Passed, "", nil),
	})
	log.Metadata.MappingReferences = []gemara.MappingReference{
		{
			Id:          "CNSC",
			Title:       "Cloud Native Security Controls",
			Version:     "1.0.0",
			Description: "CNCF security controls catalog",
			Url:         "https://example.com/cnsc",
		},
	}

	ar, err := EvaluationLogToOSCALAssessmentResults(log)
	require.NoError(t, err)

	require.NotNil(t, ar.BackMatter)
	require.NotNil(t, ar.BackMatter.Resources)
	require.Len(t, *ar.BackMatter.Resources, 1)

	resource := (*ar.BackMatter.Resources)[0]
	assert.Equal(t, "Cloud Native Security Controls", resource.Title)
	assert.NotEmpty(t, resource.UUID)
	assertValidJSON(t, ar)
}

func TestEvaluationLogToOSCALAssessmentResults_RelevantEvidence(t *testing.T) {
	log := makeEvaluationLog(gemara.Actor{Name: "tool", Type: gemara.Software}, []*gemara.AssessmentLog{
		makeAssessmentLog("REQ-1", "check", gemara.Passed, "", nil),
	})
	log.Evaluations[0].AssessmentLogs[0].Evidence = []gemara.Evidence{
		{
			Id:          "ev-1",
			Type:        gemara.EvidenceType("configuration"),
			CollectedAt: "2026-01-02T03:04:05Z",
			Payload:     map[string]any{"enabled": true},
			Source: gemara.EvidenceMapping{
				ReferenceId: "source-1",
				Coordinate:  "/etc/example.conf",
				Digest:      "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				Remarks:     "collected remotely",
			},
			Description: "The observed configuration.",
		},
		{
			Id:          "ev-2",
			Type:        gemara.EvidenceType("command-output"),
			CollectedAt: "2026-01-02T03:04:06Z",
			Source: gemara.EvidenceMapping{
				ReferenceId: "source-1",
				EntryId:     "entry-1",
			},
			Description: "The observed command output.",
		},
		{
			Id:          "ev-3",
			Type:        gemara.EvidenceType("command-output"),
			CollectedAt: "2026-01-02T03:04:07Z",
			Description: "The observed command output without a source.",
		},
	}
	log.Metadata.MappingReferences = []gemara.MappingReference{{Id: "source-1", Title: "Source"}}

	ar, err := EvaluationLogToOSCALAssessmentResults(log)
	require.NoError(t, err)

	obs := (*ar.Results[0].Observations)[0]
	require.NotNil(t, obs.RelevantEvidence)
	require.Len(t, *obs.RelevantEvidence, 3)
	evidence := (*obs.RelevantEvidence)[0]
	assert.Equal(t, "The observed configuration.", evidence.Description)
	assert.Equal(t, "collected remotely", evidence.Remarks)
	require.NotNil(t, evidence.Links)
	require.Len(t, *evidence.Links, 1)
	assert.Equal(t, "reference", (*evidence.Links)[0].Rel)
	require.NotNil(t, ar.BackMatter)
	assert.Equal(t, "#"+(*ar.BackMatter.Resources)[0].UUID, (*evidence.Links)[0].Href)
	require.NotNil(t, evidence.Props)
	props := make(map[string]string, len(*evidence.Props))
	for _, prop := range *evidence.Props {
		assert.Equal(t, "https://github.com/gemaraproj/go-gemara/ns/oscal", prop.Ns)
		props[prop.Name] = prop.Value
	}
	assert.Equal(t, "ev-1", props["id"])
	assert.Equal(t, "configuration", props["type"])
	assert.Equal(t, "2026-01-02T03:04:05Z", props["collected-at"])
	assert.Equal(t, "source-1", props["source-reference-id"])
	assert.Equal(t, "/etc/example.conf", props["source-coordinate"])
	assert.NotContains(t, props, "payload")
	assert.NotContains(t, props, "source-digest")
	require.NotNil(t, (*ar.BackMatter.Resources)[0].Rlinks)
	hashes := (*(*ar.BackMatter.Resources)[0].Rlinks)[0].Hashes
	require.NotNil(t, hashes)
	require.Len(t, *hashes, 1)
	assert.Equal(t, "sha256", (*hashes)[0].Algorithm)
	assert.Equal(t, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", (*hashes)[0].Value)
	secondEvidence := (*obs.RelevantEvidence)[1]
	assert.Equal(t, "The observed command output.", secondEvidence.Description)
	require.NotNil(t, secondEvidence.Links)
	assert.Equal(t, "#"+(*ar.BackMatter.Resources)[0].UUID, (*secondEvidence.Links)[0].Href)
	require.NotNil(t, secondEvidence.Props)
	assert.Len(t, *secondEvidence.Props, 5)
	assert.Equal(t, "entry-1", (*secondEvidence.Props)[4].Value)
	thirdEvidence := (*obs.RelevantEvidence)[2]
	assert.Equal(t, "The observed command output without a source.", thirdEvidence.Description)
	assert.Nil(t, thirdEvidence.Links)
	require.NotNil(t, thirdEvidence.Props)
	assert.Len(t, *thirdEvidence.Props, 3)
	assertValidJSON(t, ar)
}

func TestEvaluationLogToOSCALAssessmentResults_OmitsRelevantEvidenceWhenEmpty(t *testing.T) {
	log := makeEvaluationLog(gemara.Actor{Name: "tool", Type: gemara.Software}, []*gemara.AssessmentLog{
		makeAssessmentLog("REQ-1", "check", gemara.Passed, "", nil),
	})

	ar, err := EvaluationLogToOSCALAssessmentResults(log)
	require.NoError(t, err)

	assert.Nil(t, (*ar.Results[0].Observations)[0].RelevantEvidence)
}

func TestEvaluationLogToOSCALAssessmentResults_OmitsEvidencePayload(t *testing.T) {
	log := makeEvaluationLog(gemara.Actor{Name: "tool", Type: gemara.Software}, []*gemara.AssessmentLog{
		makeAssessmentLog("REQ-1", "check", gemara.Passed, "", nil),
	})
	log.Evaluations[0].AssessmentLogs[0].Evidence = []gemara.Evidence{{
		Id:      "ev-1",
		Payload: func() {},
	}}

	ar, err := EvaluationLogToOSCALAssessmentResults(log)
	require.NoError(t, err)

	props := *(*(*ar.Results[0].Observations)[0].RelevantEvidence)[0].Props
	for _, prop := range props {
		assert.NotEqual(t, "payload", prop.Name)
	}
}

func TestEvaluationLogToOSCALAssessmentResults_InvalidEvidenceDigest(t *testing.T) {
	log := makeEvaluationLog(gemara.Actor{Name: "tool", Type: gemara.Software}, []*gemara.AssessmentLog{
		makeAssessmentLog("REQ-1", "check", gemara.Passed, "", nil),
	})
	log.Evaluations[0].AssessmentLogs[0].Evidence = []gemara.Evidence{{
		Source: gemara.EvidenceMapping{Digest: "not-a-digest"},
	}}

	_, err := EvaluationLogToOSCALAssessmentResults(log)
	require.Error(t, err)
	assert.ErrorContains(t, err, "parsing evidence digest")
}

func TestEvaluationLogToOSCALAssessmentResults_InvalidEvidenceDigestEncoding(t *testing.T) {
	log := makeEvaluationLog(gemara.Actor{Name: "tool", Type: gemara.Software}, []*gemara.AssessmentLog{
		makeAssessmentLog("REQ-1", "check", gemara.Passed, "", nil),
	})
	log.Evaluations[0].AssessmentLogs[0].Evidence = []gemara.Evidence{{
		Source: gemara.EvidenceMapping{Digest: "sha256:abc123"},
	}}

	_, err := EvaluationLogToOSCALAssessmentResults(log)
	require.Error(t, err)
}

func TestEvaluationLogToOSCALAssessmentResults_NoBackMatterWhenEmpty(t *testing.T) {
	log := makeEvaluationLog(gemara.Actor{Name: "tool", Type: gemara.Software}, []*gemara.AssessmentLog{
		makeAssessmentLog("REQ-1", "check", gemara.Passed, "", nil),
	})

	ar, err := EvaluationLogToOSCALAssessmentResults(log)
	require.NoError(t, err)
	assert.Nil(t, ar.BackMatter)
}

func TestEvaluationLogToOSCALAssessmentResults_PartyUUIDConsistency(t *testing.T) {
	log := makeEvaluationLog(gemara.Actor{Name: "tool", Type: gemara.Software}, []*gemara.AssessmentLog{
		makeAssessmentLog("REQ-1", "check", gemara.Passed, "", nil),
	})

	ar, err := EvaluationLogToOSCALAssessmentResults(log)
	require.NoError(t, err)

	require.NotNil(t, ar.Metadata.Parties)
	metadataPartyUUID := (*ar.Metadata.Parties)[0].UUID

	result := ar.Results[0]
	require.NotNil(t, result.Findings)
	finding := (*result.Findings)[0]
	require.NotNil(t, finding.Origins)
	originActorUUID := (*finding.Origins)[0].Actors[0].ActorUuid

	assert.Equal(t, metadataPartyUUID, originActorUUID, "metadata party UUID must match origin actor UUID")

	require.NotNil(t, result.AssessmentLog)
	loggedByUUID := (*result.AssessmentLog.Entries[0].LoggedBy)[0].PartyUuid
	assert.Equal(t, metadataPartyUUID, loggedByUUID, "metadata party UUID must match log entry party UUID")
}

func TestMapActorType(t *testing.T) {
	assert.Equal(t, "person", mapActorType(gemara.Human))
	assert.Equal(t, "tool", mapActorType(gemara.Software))
	assert.Equal(t, "tool", mapActorType(gemara.SoftwareAssisted))
}

func TestMapPartyType(t *testing.T) {
	assert.Equal(t, "person", mapPartyType(gemara.Human))
	assert.Equal(t, "organization", mapPartyType(gemara.Software))
	assert.Equal(t, "organization", mapPartyType(gemara.SoftwareAssisted))
}

// Helpers

func assertValidJSON(t *testing.T, ar oscal.AssessmentResults) {
	t.Helper()
	model := oscal.OscalModels{AssessmentResults: &ar}
	data, err := json.MarshalIndent(model, "", "  ")
	require.NoError(t, err)
	require.NotEmpty(t, data)

	var roundtrip oscal.OscalModels
	require.NoError(t, json.Unmarshal(data, &roundtrip))
	require.NotNil(t, roundtrip.AssessmentResults)
}
