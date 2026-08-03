package gemara

import (
	"testing"
)

var controlEvaluationTestData = []struct {
	testName          string
	control           *ControlEvaluation
	failBeforePass    bool
	expectedResult    Result
	expectedCorrupted bool
}{
	{
		testName:          "ControlEvaluation with no AssessmentLogs",
		expectedResult:    NeedsReview,
		expectedCorrupted: false,
		control: &ControlEvaluation{
			AssessmentLogs: []*AssessmentLog{},
		},
	},
	{
		testName:          "ControlEvaluation with one passing AssessmentLog",
		expectedResult:    Passed,
		expectedCorrupted: false,
		control: &ControlEvaluation{
			AssessmentLogs: []*AssessmentLog{passingAssessmentPtr()},
		},
	},
	{
		testName:          "ControlEvaluation with one failing AssessmentLog",
		expectedResult:    Failed,
		expectedCorrupted: false,
		control: &ControlEvaluation{
			AssessmentLogs: []*AssessmentLog{failingAssessmentPtr()},
		},
	},
	{
		testName:          "ControlEvaluation with one NeedsReview AssessmentLog",
		expectedResult:    NeedsReview,
		expectedCorrupted: false,
		control: &ControlEvaluation{
			AssessmentLogs: []*AssessmentLog{needsReviewAssessmentPtr()},
		},
	},
	{
		testName:          "ControlEvaluation with one Unknown AssessmentLog",
		expectedResult:    Unknown,
		expectedCorrupted: false,
		control: &ControlEvaluation{
			AssessmentLogs: []*AssessmentLog{unknownAssessmentPtr()},
		},
	},
	{
		testName:          "ControlEvaluation with first NeedsReview and then Unknown AssessmentLog",
		expectedResult:    Unknown,
		expectedCorrupted: false,
		control: &ControlEvaluation{
			AssessmentLogs: []*AssessmentLog{
				needsReviewAssessmentPtr(),
				unknownAssessmentPtr(),
			},
		},
	},
	{
		testName:          "ControlEvaluation with first Unknown and then NeedsReview AssessmentLog",
		expectedResult:    Unknown,
		expectedCorrupted: false,
		control: &ControlEvaluation{
			AssessmentLogs: []*AssessmentLog{
				unknownAssessmentPtr(),
				needsReviewAssessmentPtr(),
			},
		},
	},
	{
		testName:          "ControlEvaluation with first Failed and then NeedsReview AssessmentLog",
		expectedResult:    Failed,
		expectedCorrupted: false,
		control: &ControlEvaluation{
			AssessmentLogs: []*AssessmentLog{
				failingAssessmentPtr(),
				needsReviewAssessmentPtr(),
			},
		},
	},
	{
		testName:          "ControlEvaluation with first Failing and then Passing AssessmentLog",
		expectedResult:    Failed,
		failBeforePass:    true,
		expectedCorrupted: false,
		control: &ControlEvaluation{
			AssessmentLogs: []*AssessmentLog{
				failingAssessmentPtr(),
				passingAssessmentPtr(),
			},
		},
	},
}

// TestEvaluate runs a series of tests on the ControlEvaluation.Evaluate method
func TestEvaluate(t *testing.T) {
	for _, test := range controlEvaluationTestData {
		t.Run(test.testName, func(t *testing.T) {
			c := test.control // copy the control to avoid duplication in the next test
			c.Evaluate(nil, testingApplicability)

			if c.Result != test.expectedResult {
				t.Errorf("Expected Result to be %v, but it was %v", test.expectedResult, c.Result)
			}
		})
	}
}

// TestEvaluate_EvaluatesAllSubRequirementsAfterFailure is a regression test for a
// bug where a failing sub-requirement caused Evaluate to break out of the loop,
// leaving later sub-requirements of the same control unevaluated (reported as
// NotRun with StepsExecuted=0). Each sub-requirement is independent and must be
// evaluated regardless of a sibling's result, while the control still aggregates
// to Failed.
func TestEvaluate_EvaluatesAllSubRequirementsAfterFailure(t *testing.T) {
	tests := []struct {
		name        string
		laterResult Result
	}{
		{name: "Passed", laterResult: Passed},
		{name: "NeedsReview", laterResult: NeedsReview},
		{name: "Unknown", laterResult: Unknown},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			first := failingAssessmentPtr()
			first.Steps = []AssessmentStep{func(interface{}) (Result, string, ConfidenceLevel) {
				return Failed, "failure context", Low
			}}
			second := passingAssessmentPtr()
			second.Steps = []AssessmentStep{func(interface{}) (Result, string, ConfidenceLevel) {
				return test.laterResult, "later context", High
			}}
			c := &ControlEvaluation{AssessmentLogs: []*AssessmentLog{first, second}}

			c.Evaluate(nil, testingApplicability)

			if second.StepsExecuted == 0 {
				t.Errorf("expected the sub-requirement after a failing sibling to be evaluated, but it was skipped (StepsExecuted=0)")
			}
			if second.Result != test.laterResult {
				t.Errorf("expected the later sub-requirement to record its own result %v, got %v", test.laterResult, second.Result)
			}
			if c.Result != Failed {
				t.Errorf("expected the control to still aggregate to Failed, got %v", c.Result)
			}
			if c.Message != "failure context" {
				t.Errorf("expected the control to retain the failing sub-requirement's message, got %q", c.Message)
			}
		})
	}
}

// TestEvaluate_RetainsFirstMessageForTiedAggregateResult documents that when two
// sibling assessments share the winning precedence, the control-level message is
// a stable summary that keeps the first tied assessment's message. Each assessment
// still retains its own message in AssessmentLogs so consumers can report every
// condition. The rule holds for every precedence level that can tie, not just Failed.
func TestEvaluate_RetainsFirstMessageForTiedAggregateResult(t *testing.T) {
	tests := []struct {
		name            string
		tiedResult      Result
		expectAggregate Result
	}{
		{name: "Failed", tiedResult: Failed, expectAggregate: Failed},
		{name: "NeedsReview", tiedResult: NeedsReview, expectAggregate: NeedsReview},
		{name: "Unknown", tiedResult: Unknown, expectAggregate: Unknown},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			first := failingAssessmentPtr()
			first.Steps = []AssessmentStep{func(interface{}) (Result, string, ConfidenceLevel) {
				return test.tiedResult, "first context", Low
			}}
			second := failingAssessmentPtr()
			second.Steps = []AssessmentStep{func(interface{}) (Result, string, ConfidenceLevel) {
				return test.tiedResult, "second context", High
			}}
			c := &ControlEvaluation{AssessmentLogs: []*AssessmentLog{first, second}}

			c.Evaluate(nil, testingApplicability)

			if c.Result != test.expectAggregate {
				t.Errorf("expected the control to aggregate to %v, got %v", test.expectAggregate, c.Result)
			}
			if c.Message != "first context" {
				t.Errorf("expected the control to retain the first tied message, got %q", c.Message)
			}
			if first.Message != "first context" {
				t.Errorf("expected the first assessment to retain its message, got %q", first.Message)
			}
			if second.Message != "second context" {
				t.Errorf("expected the second assessment to retain its message, got %q", second.Message)
			}
		})
	}
}

func TestAddAssesment(t *testing.T) {

	controlEvaluationTestData[0].control.AddAssessment("test", "test", []string{}, []AssessmentStep{})

	if controlEvaluationTestData[0].control.Result != Failed {
		t.Errorf("Expected Result to be Failed, but it was %v", controlEvaluationTestData[0].control.Result)
	}

	if controlEvaluationTestData[0].control.Message != "expected all AssessmentLog fields to have a value, but got: requirementId=len(4), description=len=(4), applicability=len(0), steps=len(0)" {
		t.Errorf("Expected error message to be 'expected all AssessmentLog fields to have a value, but got: requirementId=len(4), description=len=(4), applicability=len(0), steps=len(0)', but instead it was '%v'", controlEvaluationTestData[0].control.Message)
	}

}
