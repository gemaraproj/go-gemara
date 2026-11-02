package gemaraconv

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/gemaraproj/go-gemara"
	"github.com/gemaraproj/go-gemara/fetcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromEvaluationLog(t *testing.T) {
	testCatalog := makeCatalog("CTRL-1", "Test Control Title", "Test control objective", "REQ-1", "This is the requirement text that should appear in SARIF", "This is the catalog recommendation")

	tests := []struct {
		name          string
		opts          []EvalOption
		evaluationLog gemara.EvaluationLog
		wantRules     int
		wantResults   int
		wantLevels    map[string]string
		wantKinds     map[string]string
		wantToolName  string
		wantToolURI   string
		wantToolVer   string
		checkLocation func(*testing.T, *Location)
		checkRule     func(*testing.T, *ReportingDescriptor)
	}{
		{
			name: "basic conversion with multiple results",
			opts: nil,
			evaluationLog: makeEvaluationLog(gemara.Actor{
				Name:    "gemara",
				Uri:     "https://github.com/gemaraproj/go-gemara",
				Version: "1.0.0",
			}, []*gemara.AssessmentLog{
				makeAssessmentLog("REQ-1", "should do a thing", gemara.Failed, "thing was not done", nil),
				makeAssessmentLog("REQ-2", "should maybe do a thing", gemara.NeedsReview, "", nil),
				makeAssessmentLog("REQ-3", "should do another thing", gemara.Passed, "", nil),
			}),
			wantRules:   3,
			wantResults: 3,
			wantLevels: map[string]string{
				"REQ-1": "error",
				"REQ-2": "",
				"REQ-3": "",
			},
			wantKinds: map[string]string{
				"REQ-1": "fail",
				"REQ-2": "review",
				"REQ-3": "pass",
			},
			wantToolName: "gemara",
			wantToolURI:  "https://github.com/gemaraproj/go-gemara",
			wantToolVer:  "1.0.0",
			checkLocation: func(t *testing.T, loc *Location) {
				require.NotNil(t, loc.PhysicalLocation)
				require.Equal(t, emptyArtifactURIMessage, loc.PhysicalLocation.ArtifactLocation.URI)
				require.NotEmpty(t, loc.LogicalLocations)
			},
		},
		{
			name: "with artifactURI parameter",
			opts: []EvalOption{WithArtifactURI("README.md")},
			evaluationLog: makeEvaluationLog(gemara.Actor{
				Name:    "gemara",
				Uri:     "https://github.com/test/repo",
				Version: "1.0.0",
			}, []*gemara.AssessmentLog{
				makeAssessmentLog("REQ-1", "Test requirement", gemara.Failed, "Test message", nil),
			}),
			wantRules:   1,
			wantResults: 1,
			wantLevels: map[string]string{
				"REQ-1": "error",
			},
			wantToolName: "gemara",
			wantToolURI:  "https://github.com/test/repo",
			wantToolVer:  "1.0.0",
			checkLocation: func(t *testing.T, loc *Location) {
				require.NotNil(t, loc.PhysicalLocation)
				require.Equal(t, "README.md", loc.PhysicalLocation.ArtifactLocation.URI)
				require.NotEmpty(t, loc.LogicalLocations)
			},
		},
		{
			name: "empty author URI",
			opts: nil,
			evaluationLog: makeEvaluationLog(gemara.Actor{
				Name:    "gemara",
				Uri:     "",
				Version: "1.0.0",
			}, []*gemara.AssessmentLog{
				makeAssessmentLog("REQ-1", "should do a thing", gemara.Failed, "thing was not done", nil),
			}),
			wantRules:   1,
			wantResults: 1,
			wantLevels: map[string]string{
				"REQ-1": "error",
			},
			wantToolName: "gemara",
			wantToolURI:  "",
			wantToolVer:  "1.0.0",
			checkLocation: func(t *testing.T, loc *Location) {
				require.NotNil(t, loc.PhysicalLocation)
				require.Equal(t, emptyArtifactURIMessage, loc.PhysicalLocation.ArtifactLocation.URI)
				require.NotEmpty(t, loc.LogicalLocations)
			},
		},
		{
			name: "with catalog enrichment",
			opts: []EvalOption{WithArtifactURI("README.md"), WithCatalog(testCatalog)},
			evaluationLog: makeEvaluationLog(gemara.Actor{
				Name:    "test-tool",
				Uri:     "https://github.com/test/tool",
				Version: "1.0.0",
			}, []*gemara.AssessmentLog{
				{
					Requirement:    gemara.EntryMapping{EntryId: "REQ-1"},
					Description:    "Test description",
					Result:         gemara.Failed,
					Message:        "Test failed",
					Recommendation: "Fix this issue by doing X",
					Steps: []gemara.AssessmentStep{func(interface{}) (gemara.Result, string, gemara.ConfidenceLevel) {
						return gemara.Failed, "", gemara.Low
					}},
					StepsExecuted: 1,
				},
			}),
			wantRules:   1,
			wantResults: 1,
			wantLevels: map[string]string{
				"REQ-1": "error",
			},
			wantToolName: "test-tool",
			wantToolURI:  "https://github.com/test/tool",
			wantToolVer:  "1.0.0",
			checkRule: func(t *testing.T, rule *ReportingDescriptor) {
				require.Equal(t, "REQ-1", rule.ID)
				require.NotNil(t, rule.ShortDescription)
				require.Equal(t, "This is the requirement text that should appear in SARIF", rule.ShortDescription.Text)
				require.NotNil(t, rule.FullDescription)
				require.Contains(t, rule.FullDescription.Text, "Test control objective")
				require.Contains(t, rule.FullDescription.Text, "This is the requirement text")
				require.NotNil(t, rule.Help)
				require.Equal(t, "Fix this issue by doing X", rule.Help.Text, "should prefer AssessmentLog recommendation over catalog")
				require.Empty(t, rule.HelpUri)
			},
		},
		{
			name: "without catalog",
			opts: []EvalOption{WithArtifactURI("README.md")},
			evaluationLog: makeEvaluationLog(gemara.Actor{
				Name:    "test-tool",
				Uri:     "https://github.com/test/tool",
				Version: "1.0.0",
			}, []*gemara.AssessmentLog{
				{
					Requirement:    gemara.EntryMapping{EntryId: "REQ-1"},
					Description:    "Test description",
					Result:         gemara.Failed,
					Message:        "Test failed",
					Recommendation: "Fix this issue by doing X",
					Steps: []gemara.AssessmentStep{func(interface{}) (gemara.Result, string, gemara.ConfidenceLevel) {
						return gemara.Failed, "", gemara.Low
					}},
					StepsExecuted: 1,
				},
			}),
			wantRules:   1,
			wantResults: 1,
			wantLevels: map[string]string{
				"REQ-1": "error",
			},
			wantToolName: "test-tool",
			wantToolURI:  "https://github.com/test/tool",
			wantToolVer:  "1.0.0",
			checkRule: func(t *testing.T, rule *ReportingDescriptor) {
				require.Equal(t, "REQ-1", rule.ID)
				require.Nil(t, rule.ShortDescription)
				require.Nil(t, rule.FullDescription)
				require.Nil(t, rule.Help)
				require.Empty(t, rule.HelpUri)
			},
		},
		{
			name: "catalog recommendation when assessment log has none",
			opts: []EvalOption{WithArtifactURI("README.md"), WithCatalog(testCatalog)},
			evaluationLog: makeEvaluationLog(gemara.Actor{
				Name:    "test-tool",
				Uri:     "https://github.com/test/tool",
				Version: "1.0.0",
			}, []*gemara.AssessmentLog{
				{
					Requirement: gemara.EntryMapping{EntryId: "REQ-1"},
					Description: "Test description",
					Result:      gemara.Failed,
					Message:     "Test failed",
					Steps: []gemara.AssessmentStep{func(interface{}) (gemara.Result, string, gemara.ConfidenceLevel) {
						return gemara.Failed, "", gemara.Low
					}},
					StepsExecuted: 1,
				},
			}),
			wantRules:   1,
			wantResults: 1,
			wantLevels: map[string]string{
				"REQ-1": "error",
			},
			wantToolName: "test-tool",
			wantToolURI:  "https://github.com/test/tool",
			wantToolVer:  "1.0.0",
			checkRule: func(t *testing.T, rule *ReportingDescriptor) {
				require.NotNil(t, rule.Help)
				require.Equal(t, "This is the catalog recommendation", rule.Help.Text, "should use catalog recommendation when assessment log has none")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sarifBytes, err := ToSARIF(tt.evaluationLog, tt.opts...)
			require.NoError(t, err)

			sarif := toSARIFReport(t, sarifBytes)
			require.Len(t, sarif.Runs, 1)

			run := sarif.Runs[0]

			require.Len(t, run.Tool.Driver.Rules, tt.wantRules)
			require.Len(t, run.Results, tt.wantResults)

			require.Equal(t, tt.wantToolName, run.Tool.Driver.Name)
			require.Equal(t, tt.wantToolURI, run.Tool.Driver.InformationURI)
			require.Equal(t, tt.wantToolVer, run.Tool.Driver.Version)

			levels := make(map[string]string)
			kinds := make(map[string]string)
			for _, r := range run.Results {
				levels[r.RuleID] = r.Level
				kinds[r.RuleID] = r.Kind
				if tt.checkLocation != nil {
					require.NotEmpty(t, r.Locations)
					tt.checkLocation(t, &r.Locations[0])
				}
			}

			for ruleID, wantLevel := range tt.wantLevels {
				require.Equal(t, wantLevel, levels[ruleID], "rule %s should have level %s", ruleID, wantLevel)
			}
			for ruleID, wantKind := range tt.wantKinds {
				require.Equal(t, wantKind, kinds[ruleID], "rule %s should have kind %s", ruleID, wantKind)
			}

			if tt.checkRule != nil && len(run.Tool.Driver.Rules) > 0 {
				tt.checkRule(t, &run.Tool.Driver.Rules[0])
			}

			_, err = json.Marshal(sarif)
			require.NoError(t, err)
		})
	}
}

func TestToSARIF_ResultMapping(t *testing.T) {
	tests := []struct {
		result    gemara.Result
		wantKind  string
		wantLevel string
		wantCount int
	}{
		{gemara.Failed, "fail", "error", 1},
		{gemara.NeedsReview, "review", "", 1},
		{gemara.Unknown, "open", "", 1},
		{gemara.Passed, "pass", "", 1},
		{gemara.NotApplicable, "", "", 0},
		{gemara.NotRun, "", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.result.String(), func(t *testing.T) {
			evaluationLog := makeEvaluationLog(gemara.Actor{
				Name:    "test",
				Uri:     "https://test",
				Version: "1.0.0",
			}, []*gemara.AssessmentLog{
				makeAssessmentLog("REQ-1", "test", tt.result, "", nil),
			})

			sarifBytes, err := ToSARIF(evaluationLog)
			require.NoError(t, err)

			sarif := toSARIFReport(t, sarifBytes)
			require.Len(t, sarif.Runs[0].Results, tt.wantCount)

			if tt.wantCount > 0 {
				result := sarif.Runs[0].Results[0]
				require.Equal(t, tt.wantKind, result.Kind)
				require.Equal(t, tt.wantLevel, result.Level)
				require.Equal(t, tt.result.String(), result.Properties["gemara/result"])
				require.Equal(t, "REQ-1", result.PartialFingerprints["gemara/requirementId/v1"])
			}
		})
	}
}

func TestToSARIF_ExcludedStatuses(t *testing.T) {
	logs := []*gemara.AssessmentLog{
		makeAssessmentLog("REQ-1", "should do a thing", gemara.Failed, "thing was not done", nil),
		makeAssessmentLog("REQ-2", "should maybe do a thing", gemara.NeedsReview, "", nil),
		makeAssessmentLog("REQ-3", "should do another thing", gemara.Passed, "", nil),
	}

	tests := []struct {
		name        string
		opts        []EvalOption
		wantRuleIDs []string
	}{
		{
			name:        "no exclusions keeps all results",
			opts:        nil,
			wantRuleIDs: []string{"REQ-1", "REQ-2", "REQ-3"},
		},
		{
			name:        "exclude needs review",
			opts:        []EvalOption{WithExcludedStatuses(gemara.NeedsReview)},
			wantRuleIDs: []string{"REQ-1", "REQ-3"},
		},
		{
			name:        "exclude all but failed",
			opts:        []EvalOption{WithExcludedStatuses(gemara.NeedsReview, gemara.Passed)},
			wantRuleIDs: []string{"REQ-1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluationLog := makeEvaluationLog(gemara.Actor{
				Name:    "test",
				Uri:     "https://test",
				Version: "1.0.0",
			}, logs)

			sarifBytes, err := ToSARIF(evaluationLog, tt.opts...)
			require.NoError(t, err)

			sarif := toSARIFReport(t, sarifBytes)
			gotRuleIDs := make([]string, 0, len(sarif.Runs[0].Results))
			for _, r := range sarif.Runs[0].Results {
				gotRuleIDs = append(gotRuleIDs, r.RuleID)
			}
			require.Equal(t, tt.wantRuleIDs, gotRuleIDs)

			gotRules := make([]string, 0, len(sarif.Runs[0].Tool.Driver.Rules))
			for _, rule := range sarif.Runs[0].Tool.Driver.Rules {
				gotRules = append(gotRules, rule.ID)
			}
			require.Equal(t, tt.wantRuleIDs, gotRules, "rules should only be emitted for included results")
		})
	}
}

// Helper functions

func makeEvaluationLog(author gemara.Actor, logs []*gemara.AssessmentLog) gemara.EvaluationLog {
	return gemara.EvaluationLog{
		Evaluations: []*gemara.ControlEvaluation{
			{
				Name:           "Example Control",
				Control:        gemara.EntryMapping{EntryId: "CTRL-1"},
				Result:         gemara.Passed,
				AssessmentLogs: logs,
			},
		},
		Metadata: gemara.Metadata{Author: author},
		Target:   gemara.Resource{Id: "example-target", Name: "Example Target", Type: gemara.Software},
	}
}

func makeAssessmentLog(entryID, description string, result gemara.Result, message string, steps []gemara.AssessmentStep) *gemara.AssessmentLog {
	if steps == nil {
		steps = []gemara.AssessmentStep{func(interface{}) (gemara.Result, string, gemara.ConfidenceLevel) { return result, "", gemara.Medium }}
	}
	return &gemara.AssessmentLog{
		Requirement:   gemara.EntryMapping{EntryId: entryID},
		Description:   description,
		Result:        result,
		Message:       message,
		Steps:         steps,
		StepsExecuted: int64(len(steps)),
	}
}

func makeCatalog(controlID, controlTitle, controlObjective, reqID, reqText, reqRecommendation string) *gemara.ControlCatalog {
	return &gemara.ControlCatalog{
		Groups: []gemara.Group{
			{
				Id:    "test-family",
				Title: "Test Group",
			},
		},
		Controls: []gemara.Control{
			{
				Id:        controlID,
				Group:     "test-family",
				Title:     controlTitle,
				Objective: controlObjective,
				AssessmentRequirements: []gemara.AssessmentRequirement{
					{
						Id:             reqID,
						Text:           reqText,
						Recommendation: reqRecommendation,
					},
				},
			},
		},
	}
}

func toSARIFReport(t *testing.T, data []byte) *SarifReport {
	t.Helper()
	var sarif SarifReport
	err := json.Unmarshal(data, &sarif)
	require.NoError(t, err)
	require.NotNil(t, &sarif)
	return &sarif
}

func TestToSARIF_EvidenceRealArtifactURIs(t *testing.T) {
	evalLog := makeEvaluationLog(gemara.Actor{
		Name:    "evaluator",
		Uri:     "https://github.com/test/evaluator",
		Version: "1.0.0",
	}, []*gemara.AssessmentLog{
		{
			Requirement:   gemara.EntryMapping{EntryId: "REQ-1"},
			Description:   "Check branch protection",
			Result:        gemara.Failed,
			Message:       "Branch protection is missing status checks",
			StepsExecuted: 1,
			Evidence: []gemara.Evidence{
				{
					Id:          "ev-api",
					Type:        "configuration",
					Description: "Branch protection configuration",
					Source: gemara.EvidenceMapping{
						ReferenceId: "github-api",
						Coordinate:  "/repos/org/repo/branches/main/protection",
						Digest:      "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
						Remarks:     "Fetched via GitHub REST API",
					},
				},
				{
					Id:          "ev-osps",
					Type:        "command-output",
					Description: "Baseline definition",
					Source: gemara.EvidenceMapping{
						ReferenceId: "osps",
						EntryId:     "OSPS-AC-01.01",
						Digest:      "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
					},
				},
			},
		},
		{
			Requirement:   gemara.EntryMapping{EntryId: "REQ-2"},
			Description:   "Check file directly",
			Result:        gemara.Passed,
			Message:       "Security insights file found",
			StepsExecuted: 1,
			Evidence: []gemara.Evidence{
				{
					Id:          "ev-file",
					Type:        "file",
					Description: "Security Insights file",
					Source: gemara.EvidenceMapping{
						Coordinate: "SECURITY-INSIGHTS.yml",
						Digest:     "sha-512:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
					},
				},
			},
		},
	})
	evalLog.Metadata.MappingReferences = []gemara.MappingReference{
		{
			Id:    "github-api",
			Title: "GitHub API",
			Url:   "https://api.github.com",
		},
		{
			Id:    "osps",
			Title: "OpenSSF Baseline",
			Url:   "https://baseline.openssf.org",
		},
	}

	data, err := ToSARIF(evalLog)
	require.NoError(t, err)

	sarif := toSARIFReport(t, data)
	require.Len(t, sarif.Runs, 1)
	run := sarif.Runs[0]

	// Verify run.artifacts
	require.Len(t, run.Artifacts, 3)
	assert.Equal(t, "https://api.github.com/repos/org/repo/branches/main/protection", run.Artifacts[0].Location.URI)
	assert.Equal(t, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", run.Artifacts[0].Hashes["sha-256"])
	assert.Equal(t, "Branch protection configuration", run.Artifacts[0].Description.Text)

	assert.Equal(t, "https://baseline.openssf.org", run.Artifacts[1].Location.URI)
	assert.Equal(t, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", run.Artifacts[1].Hashes["sha-256"])

	assert.Equal(t, "SECURITY-INSIGHTS.yml", run.Artifacts[2].Location.URI)
	assert.Equal(t, "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", run.Artifacts[2].Hashes["sha-512"])

	// Verify Results
	require.Len(t, run.Results, 2)

	// REQ-1 has 2 evidence locations
	res1 := run.Results[0]
	assert.Equal(t, "REQ-1", res1.RuleID)
	require.Len(t, res1.Locations, 2)

	loc1_0 := res1.Locations[0]
	assert.Equal(t, "https://api.github.com/repos/org/repo/branches/main/protection", loc1_0.PhysicalLocation.ArtifactLocation.URI)
	require.NotNil(t, loc1_0.PhysicalLocation.ArtifactLocation.Index)
	assert.Equal(t, 0, *loc1_0.PhysicalLocation.ArtifactLocation.Index)
	assert.Equal(t, "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", loc1_0.PhysicalLocation.ArtifactLocation.Properties["gemara/digest"])
	assert.Equal(t, "github-api", loc1_0.PhysicalLocation.ArtifactLocation.Properties["gemara/source-reference-id"])
	assert.Equal(t, "/repos/org/repo/branches/main/protection", loc1_0.PhysicalLocation.ArtifactLocation.Properties["gemara/source-coordinate"])

	loc1_1 := res1.Locations[1]
	assert.Equal(t, "https://baseline.openssf.org", loc1_1.PhysicalLocation.ArtifactLocation.URI)
	require.NotNil(t, loc1_1.PhysicalLocation.ArtifactLocation.Index)
	assert.Equal(t, 1, *loc1_1.PhysicalLocation.ArtifactLocation.Index)
	assert.Equal(t, "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", loc1_1.PhysicalLocation.ArtifactLocation.Properties["gemara/digest"])
	assert.Equal(t, "osps", loc1_1.PhysicalLocation.ArtifactLocation.Properties["gemara/source-reference-id"])
	assert.Equal(t, "OSPS-AC-01.01", loc1_1.PhysicalLocation.ArtifactLocation.Properties["gemara/source-entry-id"])

	// REQ-2 has 1 evidence location
	res2 := run.Results[1]
	assert.Equal(t, "REQ-2", res2.RuleID)
	require.Len(t, res2.Locations, 1)
	loc2_0 := res2.Locations[0]
	assert.Equal(t, "SECURITY-INSIGHTS.yml", loc2_0.PhysicalLocation.ArtifactLocation.URI)
	require.NotNil(t, loc2_0.PhysicalLocation.ArtifactLocation.Index)
	assert.Equal(t, 2, *loc2_0.PhysicalLocation.ArtifactLocation.Index)
	assert.Equal(t, "sha-512:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", loc2_0.PhysicalLocation.ArtifactLocation.Properties["gemara/digest"])
}

func TestToSARIF_EvidenceFallbackWhenAbsent(t *testing.T) {
	evalLog := makeEvaluationLog(gemara.Actor{
		Name:    "evaluator",
		Uri:     "https://github.com/test/evaluator",
		Version: "1.0.0",
	}, []*gemara.AssessmentLog{
		{
			Requirement:   gemara.EntryMapping{EntryId: "REQ-NO-EV"},
			Description:   "Check without evidence",
			Result:        gemara.Failed,
			StepsExecuted: 1,
			Evidence:      nil,
		},
		{
			Requirement:   gemara.EntryMapping{EntryId: "REQ-PAYLOAD-ONLY"},
			Description:   "Check with inline payload only",
			Result:        gemara.Failed,
			StepsExecuted: 1,
			Evidence: []gemara.Evidence{
				{
					Id:          "ev-payload",
					Type:        "command-output",
					Payload:     map[string]any{"ok": false},
					Description: "Inline check without source",
				},
			},
		},
	})

	t.Run("default placeholder fallback", func(t *testing.T) {
		data, err := ToSARIF(evalLog)
		require.NoError(t, err)

		sarif := toSARIFReport(t, data)
		run := sarif.Runs[0]
		assert.Empty(t, run.Artifacts)
		require.Len(t, run.Results, 2)

		for _, res := range run.Results {
			require.Len(t, res.Locations, 1)
			assert.Equal(t, emptyArtifactURIMessage, res.Locations[0].PhysicalLocation.ArtifactLocation.URI)
			assert.Nil(t, res.Locations[0].PhysicalLocation.ArtifactLocation.Index)
			assert.Nil(t, res.Locations[0].PhysicalLocation.ArtifactLocation.Properties)
		}
	})

	t.Run("configured WithArtifactURI fallback", func(t *testing.T) {
		data, err := ToSARIF(evalLog, WithArtifactURI("fallback.txt"))
		require.NoError(t, err)

		sarif := toSARIFReport(t, data)
		run := sarif.Runs[0]
		assert.Empty(t, run.Artifacts)
		require.Len(t, run.Results, 2)

		for _, res := range run.Results {
			require.Len(t, res.Locations, 1)
			assert.Equal(t, "fallback.txt", res.Locations[0].PhysicalLocation.ArtifactLocation.URI)
		}
	})
}

func TestToSARIF_EvidencePreservesDigestAndAlgorithm(t *testing.T) {
	evalLog := makeEvaluationLog(gemara.Actor{Name: "test"}, []*gemara.AssessmentLog{
		{
			Requirement:   gemara.EntryMapping{EntryId: "REQ-DIGESTS"},
			Result:        gemara.Failed,
			StepsExecuted: 1,
			Evidence: []gemara.Evidence{
				{
					Source: gemara.EvidenceMapping{
						Coordinate: "file1",
						Digest:     "sha3-512:arbitrary-encoding==",
					},
				},
				{
					Source: gemara.EvidenceMapping{
						Coordinate: "file2",
						Digest:     "not-a-digest",
					},
				},
			},
		},
	})

	data, err := ToSARIF(evalLog)
	require.NoError(t, err)

	sarif := toSARIFReport(t, data)
	run := sarif.Runs[0]

	require.Len(t, run.Results[0].Locations, 2)

	// file1 has valid algorithm:hash
	loc1 := run.Results[0].Locations[0].PhysicalLocation.ArtifactLocation
	assert.Equal(t, "file1", loc1.URI)
	assert.Equal(t, "sha3-512:arbitrary-encoding==", loc1.Properties["gemara/digest"])
	require.NotNil(t, loc1.Index)
	assert.Equal(t, "arbitrary-encoding==", run.Artifacts[*loc1.Index].Hashes["sha3-512"])

	// file2 has unvalidated digest - preserved in properties, no crash in hashes
	loc2 := run.Results[0].Locations[1].PhysicalLocation.ArtifactLocation
	assert.Equal(t, "file2", loc2.URI)
	assert.Equal(t, "not-a-digest", loc2.Properties["gemara/digest"])
	require.NotNil(t, loc2.Index)
	assert.Nil(t, run.Artifacts[*loc2.Index].Hashes)
}

func TestToSARIF_GoodEvaluationLogFixture(t *testing.T) {
	log, err := gemara.Load[gemara.EvaluationLog](context.Background(), &fetcher.File{}, filepath.Join("..", "test-data", "good-evaluation-log.yaml"))
	require.NoError(t, err)

	data, err := ToSARIF(*log)
	require.NoError(t, err)

	sarif := toSARIFReport(t, data)
	require.Len(t, sarif.Runs, 1)
	run := sarif.Runs[0]

	// Verify driver metadata
	assert.Equal(t, "pvtr-github-repo", run.Tool.Driver.Name)
	assert.Equal(t, "0.1.0", run.Tool.Driver.Version)

	// In good-evaluation-log.yaml:
	// Access Control - MFA (OSPS-AC-01.01) is Passed.
	// Documentation - User Guides (OSPS-DO-01.01) is Failed.
	require.Len(t, run.Results, 2)

	// Check REQ 1: OSPS-AC-01.01
	res1 := run.Results[0]
	assert.Equal(t, "OSPS-AC-01.01", res1.RuleID)
	assert.Equal(t, "pass", res1.Kind)
	require.Len(t, res1.Locations, 2)
	assert.Equal(t, "https://api.github.com/repos/revanite-io/pvtr-github-repo/branches/main/protection", res1.Locations[0].PhysicalLocation.ArtifactLocation.URI)
	assert.Equal(t, "https://baseline.openssf.org", res1.Locations[1].PhysicalLocation.ArtifactLocation.URI)

	// Check REQ 2: OSPS-DO-01.01 has no evidence, falls back to placeholder
	res2 := run.Results[1]
	assert.Equal(t, "OSPS-DO-01.01", res2.RuleID)
	assert.Equal(t, "fail", res2.Kind)
	assert.Equal(t, "error", res2.Level)
	require.Len(t, res2.Locations, 1)
	assert.Equal(t, emptyArtifactURIMessage, res2.Locations[0].PhysicalLocation.ArtifactLocation.URI)

	// Verify run.artifacts collected the 2 URIs from evidence
	require.Len(t, run.Artifacts, 2)
	assert.Equal(t, "https://api.github.com/repos/revanite-io/pvtr-github-repo/branches/main/protection", run.Artifacts[0].Location.URI)
	assert.Equal(t, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", run.Artifacts[0].Hashes["sha-256"])
	assert.Equal(t, "https://baseline.openssf.org", run.Artifacts[1].Location.URI)
	assert.Equal(t, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", run.Artifacts[1].Hashes["sha-256"])
}

func TestEvaluationLogConverter_ToSARIF(t *testing.T) {
	evalLog := makeEvaluationLog(gemara.Actor{Name: "test"}, []*gemara.AssessmentLog{
		makeAssessmentLog("REQ-1", "test requirement", gemara.Failed, "", nil),
	})
	converter := EvaluationLog(evalLog)
	data, err := converter.ToSARIF()
	require.NoError(t, err)
	sarif := toSARIFReport(t, data)
	require.Len(t, sarif.Runs, 1)
	require.Len(t, sarif.Runs[0].Results, 1)
}

func TestResolveArtifactURI_EdgeCases(t *testing.T) {
	refMap := map[string]gemara.MappingReference{
		"base-with-slash": {
			Id:  "base-with-slash",
			Url: "https://example.com/base/",
		},
		"base-no-slash": {
			Id:  "base-no-slash",
			Url: "https://example.com/base",
		},
		"no-url": {
			Id: "no-url",
		},
		"invalid-url": {
			Id:  "invalid-url",
			Url: "http://[::1]:namedport",
		},
	}

	tests := []struct {
		name     string
		source   gemara.EvidenceMapping
		expected string
	}{
		{
			name: "coordinate is already absolute URL",
			source: gemara.EvidenceMapping{
				ReferenceId: "base-no-slash",
				Coordinate:  "https://other.com/file.txt",
			},
			expected: "https://other.com/file.txt",
		},
		{
			name: "relative coordinate to base with slash",
			source: gemara.EvidenceMapping{
				ReferenceId: "base-with-slash",
				Coordinate:  "dir/file.txt",
			},
			expected: "https://example.com/base/dir/file.txt",
		},
		{
			name: "relative coordinate to base without slash",
			source: gemara.EvidenceMapping{
				ReferenceId: "base-no-slash",
				Coordinate:  "dir/file.txt",
			},
			expected: "https://example.com/base/dir/file.txt",
		},
		{
			name: "reference has no url but has coordinate",
			source: gemara.EvidenceMapping{
				ReferenceId: "no-url",
				Coordinate:  "local/path.go",
			},
			expected: "local/path.go",
		},
		{
			name: "reference has no url and no coordinate",
			source: gemara.EvidenceMapping{
				ReferenceId: "no-url",
			},
			expected: "no-url",
		},
		{
			name: "reference not in map, has coordinate",
			source: gemara.EvidenceMapping{
				ReferenceId: "unknown-ref",
				Coordinate:  "custom/coord",
			},
			expected: "custom/coord",
		},
		{
			name: "reference not in map, no coordinate",
			source: gemara.EvidenceMapping{
				ReferenceId: "unknown-ref",
			},
			expected: "unknown-ref",
		},
		{
			name:     "empty source",
			source:   gemara.EvidenceMapping{},
			expected: "",
		},
		{
			name: "fallback join on invalid url",
			source: gemara.EvidenceMapping{
				ReferenceId: "invalid-url",
				Coordinate:  "path",
			},
			expected: "http://[::1]:namedport/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveArtifactURI(refMap, tt.source)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestToSARIF_EvidenceDuplicateAndSharedArtifacts(t *testing.T) {
	evalLog := makeEvaluationLog(gemara.Actor{Name: "test"}, []*gemara.AssessmentLog{
		{
			Requirement:   gemara.EntryMapping{EntryId: "REQ-DUP"},
			Result:        gemara.Failed,
			StepsExecuted: 1,
			Evidence: []gemara.Evidence{
				{
					Source: gemara.EvidenceMapping{
						Coordinate: "shared/file.go",
					},
				},
				{
					// Duplicate URI on same result - should be skipped
					Source: gemara.EvidenceMapping{
						Coordinate: "shared/file.go",
					},
				},
			},
		},
		{
			Requirement:   gemara.EntryMapping{EntryId: "REQ-SECOND"},
			Result:        gemara.Failed,
			StepsExecuted: 1,
			Evidence: []gemara.Evidence{
				{
					// Reuses same URI, provides digest to populate hashes
					Source: gemara.EvidenceMapping{
						Coordinate: "shared/file.go",
						Digest:     "sha256:1111111111111111111111111111111111111111111111111111111111111111",
					},
				},
			},
		},
	})

	data, err := ToSARIF(evalLog)
	require.NoError(t, err)

	sarif := toSARIFReport(t, data)
	run := sarif.Runs[0]

	// Only 1 unique artifact in run.artifacts
	require.Len(t, run.Artifacts, 1)
	assert.Equal(t, "shared/file.go", run.Artifacts[0].Location.URI)
	assert.Equal(t, "1111111111111111111111111111111111111111111111111111111111111111", run.Artifacts[0].Hashes["sha-256"])

	// REQ-DUP has only 1 location due to deduplication
	require.Len(t, run.Results[0].Locations, 1)
	assert.Equal(t, "shared/file.go", run.Results[0].Locations[0].PhysicalLocation.ArtifactLocation.URI)

	// REQ-SECOND also points to index 0
	require.Len(t, run.Results[1].Locations, 1)
	require.NotNil(t, run.Results[1].Locations[0].PhysicalLocation.ArtifactLocation.Index)
	assert.Equal(t, 0, *run.Results[1].Locations[0].PhysicalLocation.ArtifactLocation.Index)
}

func TestNormalizeHashAlgorithm(t *testing.T) {
	assert.Equal(t, "sha-256", normalizeHashAlgorithm("SHA256"))
	assert.Equal(t, "sha-512", normalizeHashAlgorithm("sha512"))
	assert.Equal(t, "sha-384", normalizeHashAlgorithm("SHA384"))
	assert.Equal(t, "sha-1", normalizeHashAlgorithm("sha1"))
	assert.Equal(t, "md5", normalizeHashAlgorithm("MD5"))
	assert.Equal(t, "unknown", normalizeHashAlgorithm("unknown"))
}
