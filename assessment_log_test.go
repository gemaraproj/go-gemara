package gemara

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gemaraproj/go-gemara/internal/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml3 "gopkg.in/yaml.v3"
)

func getAssessmentsTestData() []struct {
	testName           string
	assessment         AssessmentLog
	numberOfSteps      int
	numberOfStepsToRun int
	expectedResult     Result
} {
	return []struct {
		testName           string
		assessment         AssessmentLog
		numberOfSteps      int
		numberOfStepsToRun int
		expectedResult     Result
	}{
		{
			// An empty assessment fails precheck, which reports Unknown without
			// running any steps.
			testName:       "AssessmentLog with no steps",
			assessment:     AssessmentLog{},
			expectedResult: Unknown,
		},
		{
			testName:           "AssessmentLog with one step",
			assessment:         passingAssessment(),
			numberOfSteps:      1,
			numberOfStepsToRun: 1,
			expectedResult:     Passed,
		},
		{
			testName:           "AssessmentLog with two steps",
			assessment:         failingAssessment(),
			numberOfSteps:      2,
			numberOfStepsToRun: 1,
			expectedResult:     Failed,
		},
		{
			testName:           "AssessmentLog with three steps",
			assessment:         needsReviewAssessment(),
			numberOfSteps:      3,
			numberOfStepsToRun: 3,
			expectedResult:     NeedsReview,
		},
		{
			testName:           "AssessmentLog with four steps",
			assessment:         badRevertPassingAssessment(),
			numberOfSteps:      4,
			numberOfStepsToRun: 4,
			expectedResult:     Passed,
		},
		{
			testName:           "AssessmentLog halting on a NotApplicable guard",
			assessment:         notApplicableAssessment(),
			numberOfSteps:      2,
			numberOfStepsToRun: 1,
			expectedResult:     NotApplicable,
		},
	}
}

// TestNewStep ensures that NewStep queues a new step in the AssessmentLog
func TestAddStep(t *testing.T) {
	for _, test := range getAssessmentsTestData() {
		t.Run(test.testName, func(t *testing.T) {
			if len(test.assessment.Steps) != test.numberOfSteps {
				t.Errorf("Bad test data: expected to start with %d, got %d", test.numberOfSteps, len(test.assessment.Steps))
			}
			test.assessment.AddStep(passingAssessmentStep)
			if len(test.assessment.Steps) != test.numberOfSteps+1 {
				t.Errorf("expected %d, got %d", test.numberOfSteps, len(test.assessment.Steps))
			}
		})
	}
}

// TestRunStep ensures that runStep runs the step and updates the AssessmentLog
func TestRunStep(t *testing.T) {
	stepsTestData := []struct {
		testName        string
		step            AssessmentStep
		result          Result
		confidenceLevel ConfidenceLevel
	}{
		{
			testName:        "Failing step",
			step:            failingAssessmentStep,
			result:          Failed,
			confidenceLevel: Low,
		},
		{
			testName:        "Passing step",
			step:            passingAssessmentStep,
			result:          Passed,
			confidenceLevel: High,
		},
		{
			testName:        "Needs review step",
			step:            needsReviewAssessmentStep,
			result:          NeedsReview,
			confidenceLevel: Medium,
		},
		{
			testName:        "Unknown step",
			step:            unknownAssessmentStep,
			result:          Unknown,
			confidenceLevel: Undetermined,
		},
		{
			testName:        "Not applicable step",
			step:            notApplicableAssessmentStep,
			result:          NotApplicable,
			confidenceLevel: High,
		},
	}
	for _, test := range stepsTestData {
		t.Run(test.testName, func(t *testing.T) {
			anyOldAssessment := AssessmentLog{}
			result := anyOldAssessment.runStep(nil, test.step)
			if result != test.result {
				t.Errorf("expected %s, got %s", test.result, result)
			}
			if anyOldAssessment.Result != test.result {
				t.Errorf("expected %s, got %s", test.result, anyOldAssessment.Result)
			}
			if anyOldAssessment.ConfidenceLevel != test.confidenceLevel {
				t.Errorf("expected confidence %s, got %s", test.confidenceLevel, anyOldAssessment.ConfidenceLevel)
			}
		})
	}
}

// TestRun ensures that Run executes all steps, halting if any step does not return Passed
func TestRun(t *testing.T) {
	for _, data := range getAssessmentsTestData() {
		t.Run(data.testName, func(t *testing.T) {
			a := data.assessment // copy the assessment to prevent duplicate executions in the next test
			result := a.Run(nil)
			if result != a.Result {
				t.Errorf("expected match between Run return value (%s) and assessment Result value (%s)", result, data.expectedResult)
			}
			if result != data.expectedResult {
				t.Errorf("expected result %s, got %s", data.expectedResult, result)
			}
			if a.StepsExecuted != int64(data.numberOfStepsToRun) {
				t.Errorf("expected to run %d tests, got %d", data.numberOfStepsToRun, a.StepsExecuted)
			}
		})
	}
}

func TestNewAssessment(t *testing.T) {
	newAssessmentsTestData := []struct {
		testName      string
		requirementId string
		description   string
		applicability []string
		steps         []AssessmentStep
		expectedError bool
	}{
		{
			testName:      "Empty requirementId",
			requirementId: "",
			description:   "test",
			applicability: []string{"test"},
			steps:         []AssessmentStep{passingAssessmentStep},
			expectedError: true,
		},
		{
			testName:      "Empty description",
			requirementId: "test",
			description:   "",
			applicability: []string{"test"},
			steps:         []AssessmentStep{passingAssessmentStep},
			expectedError: true,
		},
		{
			testName:      "Empty applicability",
			requirementId: "test",
			description:   "test",
			applicability: []string{},
			steps:         []AssessmentStep{passingAssessmentStep},
			expectedError: true,
		},
		{
			testName:      "Empty steps",
			requirementId: "test",
			description:   "test",
			applicability: []string{"test"},
			steps:         []AssessmentStep{},
			expectedError: true,
		},
		{
			testName:      "Good data",
			requirementId: "test",
			description:   "test",
			applicability: []string{"test"},
			steps:         []AssessmentStep{passingAssessmentStep},
			expectedError: false,
		},
	}
	for _, data := range newAssessmentsTestData {
		t.Run(data.testName, func(t *testing.T) {
			assessment, err := NewAssessment(data.requirementId, data.description, data.applicability, data.steps)
			if data.expectedError && err == nil {
				t.Error("expected error, got nil")
			}
			if !data.expectedError && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if assessment == nil && !data.expectedError {
				t.Error("expected assessment object, got nil")
			}
		})
	}
}

func TestConfidenceLevelFromSteps(t *testing.T) {
	tests := []struct {
		name               string
		steps              []AssessmentStep
		expectedResult     Result
		expectedConfidence ConfidenceLevel
	}{
		{
			name: "Passed then NeedsReview",
			steps: []AssessmentStep{
				func(interface{}) (Result, string, ConfidenceLevel) {
					return Passed, "Step 1 passed", High
				},
				func(interface{}) (Result, string, ConfidenceLevel) {
					return NeedsReview, "Step 2 needs review", Medium
				},
			},
			expectedResult:     NeedsReview,
			expectedConfidence: Medium,
		},
		{
			name: "Passed then Passed with different confidence",
			steps: []AssessmentStep{
				func(interface{}) (Result, string, ConfidenceLevel) {
					return Passed, "Step 1 passed", High
				},
				func(interface{}) (Result, string, ConfidenceLevel) {
					return Passed, "Step 2 passed", Low
				},
			},
			expectedResult:     Passed,
			expectedConfidence: Low, // Use last step's confidence
		},
		{
			name: "NeedsReview then Passed",
			steps: []AssessmentStep{
				func(interface{}) (Result, string, ConfidenceLevel) {
					return NeedsReview, "Step 1 needs review", Medium
				},
				func(interface{}) (Result, string, ConfidenceLevel) {
					return Passed, "Step 2 passed", High
				},
			},
			expectedResult:     NeedsReview,
			expectedConfidence: High, // Use last step's confidence
		},
		{
			name: "Multiple NeedsReview steps",
			steps: []AssessmentStep{
				func(interface{}) (Result, string, ConfidenceLevel) {
					return NeedsReview, "Step 1 needs review", High
				},
				func(interface{}) (Result, string, ConfidenceLevel) {
					return NeedsReview, "Step 2 needs review", Low
				},
			},
			expectedResult:     NeedsReview,
			expectedConfidence: Low,
		},
		{
			name: "Unknown then Passed",
			steps: []AssessmentStep{
				func(interface{}) (Result, string, ConfidenceLevel) {
					return Unknown, "Step 1 unknown", Undetermined
				},
				func(interface{}) (Result, string, ConfidenceLevel) {
					return Passed, "Step 2 passed", High
				},
			},
			expectedResult:     Unknown,
			expectedConfidence: High, // Use last step's confidence
		},
		{
			name: "Passed then Unknown",
			steps: []AssessmentStep{
				func(interface{}) (Result, string, ConfidenceLevel) {
					return Passed, "Step 1 passed", High
				},
				func(interface{}) (Result, string, ConfidenceLevel) {
					return Unknown, "Step 2 unknown", Undetermined
				},
			},
			expectedResult:     Unknown,
			expectedConfidence: Undetermined,
		},
		{
			name: "Failed stops execution",
			steps: []AssessmentStep{
				func(interface{}) (Result, string, ConfidenceLevel) {
					return Passed, "Step 1 passed", High
				},
				func(interface{}) (Result, string, ConfidenceLevel) {
					return Failed, "Step 2 failed", Low
				},
				func(interface{}) (Result, string, ConfidenceLevel) {
					return Passed, "Step 3 passed", High
				},
			},
			expectedResult:     Failed,
			expectedConfidence: Low,
		},
		{
			name: "Prereqs, then NeedsReview, then Passed",
			steps: []AssessmentStep{
				func(interface{}) (Result, string, ConfidenceLevel) {
					return Passed, "Prereq 1", High
				},
				func(interface{}) (Result, string, ConfidenceLevel) {
					return Passed, "Prereq 2", High
				},
				func(interface{}) (Result, string, ConfidenceLevel) {
					return NeedsReview, "Check 1", Medium
				},
				func(interface{}) (Result, string, ConfidenceLevel) {
					return Passed, "Check 2", High
				},
			},
			expectedResult:     NeedsReview,
			expectedConfidence: High, // Use last step's confidence
		},
		{
			name: "Prereqs, then Passed Low, then Passed High",
			steps: []AssessmentStep{
				func(interface{}) (Result, string, ConfidenceLevel) {
					return Passed, "Prereq 1", High
				},
				func(interface{}) (Result, string, ConfidenceLevel) {
					return Passed, "Prereq 2", High
				},
				func(interface{}) (Result, string, ConfidenceLevel) {
					return Passed, "Check 1", Low
				},
				func(interface{}) (Result, string, ConfidenceLevel) {
					return Passed, "Check 2", High
				},
			},
			expectedResult:     Passed,
			expectedConfidence: High, // Use last step's confidence
		},
		{
			name: "Passed High, then NeedsReview Low, then Passed High",
			steps: []AssessmentStep{
				func(interface{}) (Result, string, ConfidenceLevel) {
					return Passed, "Step 1", High
				},
				func(interface{}) (Result, string, ConfidenceLevel) {
					return NeedsReview, "Step 2", Low
				},
				func(interface{}) (Result, string, ConfidenceLevel) {
					return Passed, "Step 3", High
				},
			},
			expectedResult:     NeedsReview,
			expectedConfidence: High, // Use last step's confidence
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assessment, err := NewAssessment("test-id", "test description", []string{"test"}, tt.steps)
			require.NoError(t, err)
			result := assessment.Run(nil)

			assert.Equal(t, tt.expectedResult, result)
			assert.Equal(t, tt.expectedConfidence, assessment.ConfidenceLevel)
		})
	}
}

// TestRunNotApplicableGuard covers the scope-guard contract end to end: a step
// that reports NotApplicable must decide the assessment outright and stop the
// chain. Aggregating it through UpdateAggregateResult is not sufficient, because
// NotApplicable is intentionally the weakest non-NotRun value there and would be
// absorbed by any real result on either side of it.
func TestRunNotApplicableGuard(t *testing.T) {
	tests := []struct {
		name          string
		steps         []AssessmentStep
		expected      Result
		expectedSteps int64
	}{
		{
			// The regression that motivated this behavior: an inapplicable target was
			// reported as having passed the requirement.
			name:          "NotApplicable guard is not overwritten by a later Passed",
			steps:         []AssessmentStep{notApplicableAssessmentStep, passingAssessmentStep},
			expected:      NotApplicable,
			expectedSteps: 1,
		},
		{
			name:          "NotApplicable guard halts before a later Failed",
			steps:         []AssessmentStep{notApplicableAssessmentStep, failingAssessmentStep},
			expected:      NotApplicable,
			expectedSteps: 1,
		},
		{
			name:          "NotApplicable guard halts before a later NeedsReview",
			steps:         []AssessmentStep{notApplicableAssessmentStep, needsReviewAssessmentStep},
			expected:      NotApplicable,
			expectedSteps: 1,
		},
		{
			name:          "NotApplicable guard halts before a later Unknown",
			steps:         []AssessmentStep{notApplicableAssessmentStep, unknownAssessmentStep},
			expected:      NotApplicable,
			expectedSteps: 1,
		},
		{
			// Scope determination outranks findings gathered before the requirement
			// was known not to apply, so a passing prerequisite does not absorb it.
			name:          "NotApplicable after a passing prerequisite still wins",
			steps:         []AssessmentStep{passingAssessmentStep, notApplicableAssessmentStep, passingAssessmentStep},
			expected:      NotApplicable,
			expectedSteps: 2,
		},
		{
			name:          "NotApplicable after NeedsReview still wins",
			steps:         []AssessmentStep{needsReviewAssessmentStep, notApplicableAssessmentStep, passingAssessmentStep},
			expected:      NotApplicable,
			expectedSteps: 2,
		},
		{
			// Failed halts first, so a guard placed after it is never reached.
			name:          "Failed halts before reaching a later NotApplicable",
			steps:         []AssessmentStep{failingAssessmentStep, notApplicableAssessmentStep},
			expected:      Failed,
			expectedSteps: 1,
		},
		{
			name:          "consecutive NotApplicable steps halt on the first",
			steps:         []AssessmentStep{notApplicableAssessmentStep, notApplicableAssessmentStep},
			expected:      NotApplicable,
			expectedSteps: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, err := NewAssessment("test-id", "test description", testingApplicability, tt.steps)
			require.NoError(t, err)

			result := a.Run(nil)

			assert.Equal(t, tt.expected, result, "Run return value")
			assert.Equal(t, tt.expected, a.Result, "AssessmentLog.Result")
			assert.Equal(t, tt.expectedSteps, a.StepsExecuted, "steps executed")
		})
	}
}

// TestRunNotApplicableGuardRetainsMessage ensures the guard's own message reaches
// the log rather than being overwritten by a step that runs after it, since
// runStep assigns a.Message unconditionally.
func TestRunNotApplicableGuardRetainsMessage(t *testing.T) {
	later := func(interface{}) (Result, string, ConfidenceLevel) {
		return Passed, "this step should never have run", High
	}
	a, err := NewAssessment("test-id", "test description", testingApplicability,
		[]AssessmentStep{notApplicableAssessmentStep, later})
	require.NoError(t, err)

	require.Equal(t, NotApplicable, a.Run(nil))
	assert.Equal(t, "out of scope for this target", a.Message)
}

// TestRunStampsEndOnHalt ensures an assessment that executed at least one step
// records an end time on every exit path, including the early returns.
func TestRunStampsEndOnHalt(t *testing.T) {
	cases := []struct {
		name  string
		steps []AssessmentStep
	}{
		{"halt on Failed", []AssessmentStep{failingAssessmentStep}},
		{"halt on NotApplicable", []AssessmentStep{notApplicableAssessmentStep}},
		{"ran to completion", []AssessmentStep{passingAssessmentStep}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, err := NewAssessment("test-id", "test description", testingApplicability, tc.steps)
			require.NoError(t, err)

			a.Run(nil)

			assert.NotEmpty(t, a.Start, "Start should be stamped")
			assert.NotEmpty(t, a.End, "End should be stamped")
		})
	}
}

// evidenceTarget is a sample targetData payload that opts into evidence
// collection by embedding EvidenceCollector (no methods written by hand).
type evidenceTarget struct {
	EvidenceCollector
}

// evidenceStep returns a step that records one piece of evidence with the given id.
func evidenceStep(id string) AssessmentStep {
	return func(payload interface{}) (Result, string, ConfidenceLevel) {
		if t, ok := payload.(*evidenceTarget); ok {
			t.AddEvidence(Evidence{Id: id, Description: "recorded by step"})
		}
		return Passed, "recorded evidence", High
	}
}

// TestRunCollectsEvidence ensures evidence a step records into the payload is
// harvested into the AssessmentLog and the payload is cleared afterward, so it
// does not linger as a field on the target data.
func TestRunCollectsEvidence(t *testing.T) {
	a, err := NewAssessment("req", "desc", testingApplicability, []AssessmentStep{evidenceStep("ev-1")})
	require.NoError(t, err)

	target := &evidenceTarget{}
	result := a.Run(target)

	require.Equal(t, Passed, result)
	require.Len(t, a.Evidence, 1)
	assert.Equal(t, "ev-1", a.Evidence[0].Id)
	// The assessment clears the payload after copying the evidence out.
	assert.Empty(t, target.GetEvidence())
}

// TestRunCollectsMultipleEvidencePerStep ensures a single step may record evidence
// more than once, and every piece is harvested into the AssessmentLog.
func TestRunCollectsMultipleEvidencePerStep(t *testing.T) {
	multiStep := func(payload interface{}) (Result, string, ConfidenceLevel) {
		if t, ok := payload.(*evidenceTarget); ok {
			t.AddEvidence(Evidence{Id: "ev-1"})
			t.AddEvidence(Evidence{Id: "ev-2"})
		}
		return Passed, "recorded twice", High
	}
	a, err := NewAssessment("req", "desc", testingApplicability, []AssessmentStep{multiStep})
	require.NoError(t, err)

	target := &evidenceTarget{}
	result := a.Run(target)

	require.Equal(t, Passed, result)
	require.Len(t, a.Evidence, 2)
	assert.Equal(t, "ev-1", a.Evidence[0].Id)
	assert.Equal(t, "ev-2", a.Evidence[1].Id)
	assert.Empty(t, target.GetEvidence())
}

// TestRunCollectsEvidencePerStep ensures each recording step contributes exactly
// one piece of evidence, and a step that records nothing does not cause the prior
// step's evidence to be re-copied.
func TestRunCollectsEvidencePerStep(t *testing.T) {
	steps := []AssessmentStep{
		evidenceStep("ev-1"),
		passingAssessmentStep, // records nothing; must not duplicate ev-1
		evidenceStep("ev-2"),
	}
	a, err := NewAssessment("req", "desc", testingApplicability, steps)
	require.NoError(t, err)

	target := &evidenceTarget{}
	result := a.Run(target)

	require.Equal(t, Passed, result)
	require.Len(t, a.Evidence, 2)
	assert.Equal(t, "ev-1", a.Evidence[0].Id)
	assert.Equal(t, "ev-2", a.Evidence[1].Id)
	assert.Empty(t, target.GetEvidence())
}

// TestRunCollectsEvidenceOnHalt ensures evidence recorded before a failing step
// is harvested exactly once, since Run halts early on the first non-passing result.
func TestRunCollectsEvidenceOnHalt(t *testing.T) {
	steps := []AssessmentStep{evidenceStep("ev-1"), failingAssessmentStep}
	a, err := NewAssessment("req", "desc", testingApplicability, steps)
	require.NoError(t, err)

	target := &evidenceTarget{}
	result := a.Run(target)

	require.Equal(t, Failed, result)
	require.Len(t, a.Evidence, 1)
	assert.Equal(t, "ev-1", a.Evidence[0].Id)
}

// TestRunWithoutEvidence ensures backward compatibility: payloads that predate
// the evidence channel (nil, or any type that does not implement HasEvidence)
// run unchanged and leave AssessmentLog.Evidence empty.
func TestRunWithoutEvidence(t *testing.T) {
	cases := []struct {
		name    string
		payload interface{}
	}{
		{"nil payload", nil},
		{"payload without evidence", struct{ Config string }{Config: "x"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := passingAssessment()
			result := a.Run(tc.payload)

			assert.Equal(t, Passed, result)
			assert.Empty(t, a.Evidence)
		})
	}
}

// TestAssessmentStepRoundTrip covers the defect that made a published
// EvaluationLog unreadable: steps marshalled to their function names but had no
// decoder. The name assertions are also what catch a decodedStepPC that has
// stopped identifying decoded steps.
func TestAssessmentStepRoundTrip(t *testing.T) {
	want := AssessmentStep(passingAssessmentStep).String()
	require.NotEmpty(t, want)
	require.NotContains(t, want, "decodedStep", "sanity: want the real step name, not the decoder closure")

	in := AssessmentLog{
		Requirement:   EntryMapping{EntryId: "test"},
		Description:   "round trip",
		Applicability: testingApplicability,
		Result:        Passed,
		Steps:         []AssessmentStep{passingAssessmentStep},
	}

	// goccy/go-yaml reaches AssessmentStep via the BytesUnmarshaler signature;
	// encoding/json and yaml.v3 both reach it via encoding.TextUnmarshaler.
	codecs := []struct {
		name      string
		marshal   func(interface{}) ([]byte, error)
		unmarshal func([]byte, interface{}) error
	}{
		{"json", json.Marshal, json.Unmarshal},
		{"goccy-yaml", codec.MarshalYAML, codec.UnmarshalYAML},
		{"yaml.v3", yaml3.Marshal, yaml3.Unmarshal},
	}

	for _, c := range codecs {
		t.Run(c.name, func(t *testing.T) {
			data, err := c.marshal(in)
			require.NoError(t, err)

			var out AssessmentLog
			require.NoError(t, c.unmarshal(data, &out))
			require.Len(t, out.Steps, 1)
			assert.Equal(t, want, out.Steps[0].String(), "step name must survive decoding")

			// Re-encoding a decoded log must reproduce the same bytes, or a log that
			// passes through this package is corrupted by the trip.
			again, err := c.marshal(out)
			require.NoError(t, err)
			assert.Equal(t, string(data), string(again), "round trip must be lossless")
		})
	}

	// A decoded log has names but no functions, so running it reports Unknown
	// and says why rather than a result it never assessed.
	t.Run("running a decoded log reports Unknown", func(t *testing.T) {
		data, err := json.Marshal(in)
		require.NoError(t, err)
		var out AssessmentLog
		require.NoError(t, json.Unmarshal(data, &out))

		assert.Equal(t, Unknown, out.Run(nil))
		assert.Contains(t, out.Message, "cannot be re-run")
	})

	// The probe is an internal detail; a consumer step must never receive it.
	t.Run("real steps are never probed", func(t *testing.T) {
		var sawProbe bool
		step := AssessmentStep(func(payload interface{}) (Result, string, ConfidenceLevel) {
			if _, ok := payload.(stepNameProbe); ok {
				sawProbe = true
			}
			return Passed, "ok", High
		})
		_ = step.String()
		assert.False(t, sawProbe, "String must not invoke a consumer's step")
	})
}

// TestAssessmentStepNullDecodesToNil checks that a null step stays nil instead
// of becoming a non-nil step with an empty name, which would slip past a
// caller's nil check -- gemaraconv/sarif.go guards on nil before calling
// String() to build a SARIF logical location.
func TestAssessmentStepNullDecodesToNil(t *testing.T) {
	t.Run("json", func(t *testing.T) {
		var step AssessmentStep
		require.NoError(t, json.Unmarshal([]byte("null"), &step))
		assert.Nil(t, step)
	})

	t.Run("goccy-yaml", func(t *testing.T) {
		var doc struct {
			Steps []AssessmentStep `yaml:"steps"`
		}
		require.NoError(t, codec.UnmarshalYAML([]byte("steps:\n  - null\n"), &doc))
		require.Len(t, doc.Steps, 1)
		assert.Nil(t, doc.Steps[0])
	})

	t.Run("yaml.v3", func(t *testing.T) {
		var doc struct {
			Steps []AssessmentStep `yaml:"steps"`
		}
		require.NoError(t, yaml3.Unmarshal([]byte("steps:\n  - null\n"), &doc))
		// yaml.v3 drops the entry rather than keeping a nil one, so a null shifts
		// later steps down and desyncs len(Steps) from steps-executed; goccy and
		// JSON keep position. The SDK decodes through goccy, so this only pins
		// the shape for consumers who decode with yaml.v3.
		assert.Empty(t, doc.Steps)
	})

	// An explicitly empty name is a name, not a null, and must still decode.
	t.Run("empty name is not null", func(t *testing.T) {
		var step AssessmentStep
		require.NoError(t, json.Unmarshal([]byte(`""`), &step))
		require.NotNil(t, step)
		assert.Empty(t, step.String())
	})
}

// TestMalformedStepDiagnostics locks in the reason AssessmentStep has an
// UnmarshalYAML but deliberately no UnmarshalJSON: each decoder's own error is
// better than one routed through an inner unmarshal. Re-adding UnmarshalJSON, or
// dropping UnmarshalYAML, degrades one of these.
//
// A wrong-typed YAML scalar is not malformed here: a plain scalar carries no
// type, so `steps: [123]` decodes to a step named "123" under both goccy and
// yaml.v3. Only JSON, whose numbers are typed, rejects it.
func TestMalformedStepDiagnostics(t *testing.T) {
	t.Run("json names the field and type", func(t *testing.T) {
		var doc struct {
			Steps []AssessmentStep `json:"steps"`
		}
		err := json.Unmarshal([]byte(`{"steps":[123]}`), &doc)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ".steps.0", "error must locate the bad entry")
		assert.Contains(t, err.Error(), "AssessmentStep", "error must name the real type")
	})

	t.Run("goccy reports the source position", func(t *testing.T) {
		var doc struct {
			Steps []AssessmentStep `yaml:"steps"`
		}
		err := codec.UnmarshalYAML([]byte("steps:\n  - [nested, seq]\n"), &doc)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Steps", "error must name the field")
		assert.NotContains(t, err.Error(), "does not implemented Unmarshaler",
			"goccy's fallback error carries no position; UnmarshalYAML exists to avoid it")
	})
}

// TestEvaluationLogFixtureRoundTrip is the fixture-level counterpart of
// TestAssessmentStepRoundTrip: a log as a plugin emits it, step names included,
// must load through the public loader and survive a re-encode unchanged.
func TestEvaluationLogFixtureRoundTrip(t *testing.T) {
	log, err := Load[EvaluationLog](context.Background(), fileFetcher, "test-data/good-evaluation-log.yaml")
	require.NoError(t, err)

	require.Len(t, log.Evaluations, 2)
	steps := log.Evaluations[1].AssessmentLogs[0].Steps
	require.Len(t, steps, 3)
	assert.Equal(t, "github.com/revanite-io/pvtr-github-repo/evaluation_plans/osps/docs.hasUserGuides", steps[2].String())

	first, err := codec.MarshalYAML(log)
	require.NoError(t, err)
	var again EvaluationLog
	require.NoError(t, codec.UnmarshalYAML(first, &again))
	second, err := codec.MarshalYAML(again)
	require.NoError(t, err)
	assert.Equal(t, string(first), string(second), "re-encoding a loaded log must be lossless")
}

// TestRunResetsCounts guards against Run compounding what an earlier run, or a
// decoded log, already recorded. gemaraconv branches on StepsExecuted > 0, so an
// inflated count feeds downstream.
func TestRunResetsCounts(t *testing.T) {
	t.Run("decoded log", func(t *testing.T) {
		var log AssessmentLog
		require.NoError(t, json.Unmarshal([]byte(`{"requirement":{"entry-id":"r"},"description":"d","applicability":["x"],"steps":["a","b"],"steps-executed":3,"evidence":[{"id":"old"}]}`), &log))
		log.Run(nil)
		assert.EqualValues(t, 2, log.StepsExecuted)
		assert.Empty(t, log.Evidence)
	})

	t.Run("run twice", func(t *testing.T) {
		log, err := NewAssessment("r", "d", testingApplicability, []AssessmentStep{passingAssessmentStep, passingAssessmentStep})
		require.NoError(t, err)
		log.Run(nil)
		log.Run(nil)
		assert.EqualValues(t, 2, log.StepsExecuted)
	})
}

// TestNilStep covers a null step that decoded to nil: running it must refuse
// rather than panic, and marshaling it must write null rather than a name, so
// the trip is lossless.
func TestNilStep(t *testing.T) {
	steps := []AssessmentStep{passingAssessmentStep, nil, passingAssessmentStep}

	t.Run("Run refuses", func(t *testing.T) {
		log := AssessmentLog{Requirement: EntryMapping{EntryId: "r"}, Description: "d", Applicability: testingApplicability, Steps: steps}
		assert.Equal(t, Unknown, log.Run(nil))
		assert.Contains(t, log.Message, "step 1 is nil")
		assert.Zero(t, log.StepsExecuted)
	})

	t.Run("NewAssessment refuses", func(t *testing.T) {
		_, err := NewAssessment("r", "d", testingApplicability, steps)
		require.ErrorContains(t, err, "step 1 is nil")
	})

	t.Run("json round trip", func(t *testing.T) {
		data, err := json.Marshal(steps)
		require.NoError(t, err)
		assert.NotContains(t, string(data), "unknown function")
		var out []AssessmentStep
		require.NoError(t, json.Unmarshal(data, &out))
		require.Len(t, out, 3)
		assert.Nil(t, out[1])
	})

	t.Run("goccy round trip", func(t *testing.T) {
		data, err := codec.MarshalYAML(steps)
		require.NoError(t, err)
		assert.NotContains(t, string(data), "unknown function")
		var out []AssessmentStep
		require.NoError(t, codec.UnmarshalYAML(data, &out))
		require.Len(t, out, 3)
		assert.Nil(t, out[1])
	})
}

func TestNamedStep(t *testing.T) {
	ran := false
	step := NamedStep("pkg.checkThing", func(payload interface{}) (Result, string, ConfidenceLevel) {
		ran = true
		return Passed, "ok", High
	})

	assert.Equal(t, "pkg.checkThing", step.String())
	assert.False(t, step.isDecoded())

	result, msg, conf := step(nil)
	assert.True(t, ran)
	assert.Equal(t, Passed, result)
	assert.Equal(t, "ok", msg)
	assert.Equal(t, High, conf)

	out, err := codec.MarshalYAML([]AssessmentStep{step})
	require.NoError(t, err)
	assert.Equal(t, "- pkg.checkThing\n", string(out))

	// A name without a function is a decoded step.
	nameOnly := NamedStep("pkg.gone", nil)
	assert.Equal(t, "pkg.gone", nameOnly.String())
	assert.True(t, nameOnly.isDecoded())
}
