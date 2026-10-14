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
	{
		// Every applicable assessment was inapplicable, so the control is too.
		testName:          "ControlEvaluation with only a NotApplicable AssessmentLog",
		expectedResult:    NotApplicable,
		expectedCorrupted: false,
		control: &ControlEvaluation{
			AssessmentLogs: []*AssessmentLog{notApplicableAssessmentPtr()},
		},
	},
	{
		// Roll-up absorption: one inapplicable assessment does not make the whole
		// control inapplicable when a sibling produced a real result.
		testName:          "ControlEvaluation with first NotApplicable and then Passing AssessmentLog",
		expectedResult:    Passed,
		expectedCorrupted: false,
		control: &ControlEvaluation{
			AssessmentLogs: []*AssessmentLog{
				notApplicableAssessmentPtr(),
				passingAssessmentPtr(),
			},
		},
	},
	{
		testName:          "ControlEvaluation with first Passing and then NotApplicable AssessmentLog",
		expectedResult:    Passed,
		expectedCorrupted: false,
		control: &ControlEvaluation{
			AssessmentLogs: []*AssessmentLog{
				passingAssessmentPtr(),
				notApplicableAssessmentPtr(),
			},
		},
	},
	{
		// A NotApplicable assessment must not halt the roll-up, so the failing
		// sibling after it still runs and still decides the control.
		testName:          "ControlEvaluation with first NotApplicable and then Failing AssessmentLog",
		expectedResult:    Failed,
		expectedCorrupted: false,
		control: &ControlEvaluation{
			AssessmentLogs: []*AssessmentLog{
				notApplicableAssessmentPtr(),
				failingAssessmentPtr(),
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

// outOfScopeAssessmentPtr returns an assessment tagged for an applicability that
// testingApplicability does not request. Its step flips *ran so tests can prove
// Evaluate never executed it.
func outOfScopeAssessmentPtr(ran *bool) *AssessmentLog {
	return &AssessmentLog{
		Requirement:   EntryMapping{EntryId: "outOfScopeAssessment()"},
		Description:   "out of scope assessment",
		Applicability: []string{"maturity-1"},
		Steps: []AssessmentStep{func(interface{}) (Result, string, ConfidenceLevel) {
			*ran = true
			return Failed, "out of scope step ran", Low
		}},
	}
}

// TestEvaluate_MarksOutOfScopeAssessmentNotApplicable is a regression test for
// assessments whose applicability matched none of the requested values being
// left at their zero value, NotRun, with an empty message.
func TestEvaluate_MarksOutOfScopeAssessmentNotApplicable(t *testing.T) {
	var ran bool
	assessment := outOfScopeAssessmentPtr(&ran)
	c := &ControlEvaluation{AssessmentLogs: []*AssessmentLog{assessment}}

	c.Evaluate(nil, []string{"maturity-2"})

	if assessment.Result != NotApplicable {
		t.Errorf("assessment Result: got = %v, want = %v", assessment.Result, NotApplicable)
	}
	if want := "Not applicable: requirement applies to [maturity-1], evaluation requested [maturity-2]"; assessment.Message != want {
		t.Errorf("assessment Message: got = %q, want = %q", assessment.Message, want)
	}
	if ran {
		t.Error("expected the out of scope assessment's steps not to run")
	}
	if assessment.StepsExecuted != 0 {
		t.Errorf("assessment StepsExecuted: got = %d, want = 0", assessment.StepsExecuted)
	}
	if assessment.Start != "" || assessment.End != "" {
		t.Errorf("assessment Start/End: got = %q/%q, want both empty", assessment.Start, assessment.End)
	}
	if c.Result != NotApplicable {
		t.Errorf("control Result: got = %v, want = %v", c.Result, NotApplicable)
	}
	if c.Message != assessment.Message {
		t.Errorf("control Message: got = %q, want = %q", c.Message, assessment.Message)
	}
}

// TestEvaluate_AggregatesOutOfScopeAssessments verifies that an out of scope
// assessment rolls into the control result as NotApplicable, the weakest
// reportable result, so any applicable sibling's result wins.
func TestEvaluate_AggregatesOutOfScopeAssessments(t *testing.T) {
	tests := []struct {
		name          string
		applicable    func() *AssessmentLog
		outOfScope    int
		wantResult    Result
		inScopeBefore bool
	}{
		{name: "passed then out of scope", applicable: passingAssessmentPtr, outOfScope: 1, wantResult: Passed, inScopeBefore: true},
		{name: "out of scope then passed", applicable: passingAssessmentPtr, outOfScope: 1, wantResult: Passed},
		{name: "failed then out of scope", applicable: failingAssessmentPtr, outOfScope: 1, wantResult: Failed, inScopeBefore: true},
		{name: "out of scope then failed", applicable: failingAssessmentPtr, outOfScope: 1, wantResult: Failed},
		{name: "nothing applies", outOfScope: 2, wantResult: NotApplicable},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var ran bool
			logs := make([]*AssessmentLog, 0, test.outOfScope+1)
			for range test.outOfScope {
				logs = append(logs, outOfScopeAssessmentPtr(&ran))
			}
			var inScope *AssessmentLog
			if test.applicable != nil {
				inScope = test.applicable()
				if test.inScopeBefore {
					logs = append([]*AssessmentLog{inScope}, logs...)
				} else {
					logs = append(logs, inScope)
				}
			}
			c := &ControlEvaluation{AssessmentLogs: logs}

			c.Evaluate(nil, testingApplicability)

			if c.Result != test.wantResult {
				t.Errorf("control Result: got = %v, want = %v", c.Result, test.wantResult)
			}
			if ran {
				t.Error("expected out of scope steps not to run")
			}
			for _, a := range logs {
				if a == inScope {
					if a.StepsExecuted == 0 {
						t.Error("expected the applicable assessment to run")
					}
					continue
				}
				if a.Result != NotApplicable {
					t.Errorf("out of scope assessment Result: got = %v, want = %v", a.Result, NotApplicable)
				}
				if a.Message == "" {
					t.Error("expected out of scope assessment to explain why it was skipped")
				}
			}
		})
	}
}
