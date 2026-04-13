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

	ar, err := EvaluationLogToOSCALAssessmentResults(log, WithImportApHref("#ap-1"))
	require.NoError(t, err)

	assert.NotEmpty(t, ar.UUID)
	assert.Equal(t, "#ap-1", ar.ImportAp.Href)
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

	ar, err := EvaluationLogToOSCALAssessmentResults(log, WithImportApHref("#ap-1"), WithCatalog(catalog))
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

func TestEvaluationLogToOSCALAssessmentResults_TargetComponent(t *testing.T) {
	log := makeEvaluationLog(gemara.Actor{Name: "tool", Type: gemara.Software}, []*gemara.AssessmentLog{
		makeAssessmentLog("REQ-1", "check", gemara.Passed, "", nil),
	})
	log.Target = gemara.Resource{Id: "my-sys", Name: "Production System", Description: "The prod system"}

	ar, err := EvaluationLogToOSCALAssessmentResults(log)
	require.NoError(t, err)

	result := ar.Results[0]
	require.NotNil(t, result.LocalDefinitions)
	require.NotNil(t, result.LocalDefinitions.Components)
	require.Len(t, *result.LocalDefinitions.Components, 1)

	comp := (*result.LocalDefinitions.Components)[0]
	assert.Equal(t, "Production System", comp.Title)
	assert.Equal(t, "The prod system", comp.Description)
}

func TestEvaluationLogConverter_ToOSCALAssessmentResults(t *testing.T) {
	log := makeEvaluationLog(gemara.Actor{Name: "tool", Type: gemara.Software},
		[]*gemara.AssessmentLog{makeAssessmentLog("REQ-1", "check", gemara.Passed, "", nil)})
	log.Metadata.Id = "eval-converter"

	converter := EvaluationLog(log)
	ar, err := converter.ToOSCALAssessmentResults(WithImportApHref("#ap"))
	require.NoError(t, err)
	require.Len(t, ar.Results, 1)
	assert.Contains(t, ar.Results[0].Title, "eval-converter")
}

func TestMapEntityType(t *testing.T) {
	assert.Equal(t, "person", mapEntityType(gemara.Human))
	assert.Equal(t, "tool", mapEntityType(gemara.Software))
	assert.Equal(t, "tool", mapEntityType(gemara.SoftwareAssisted))
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
