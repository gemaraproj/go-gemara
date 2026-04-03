package gemara

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestResultString(t *testing.T) {
	tests := []struct {
		name     string
		result   Result
		expected string
	}{
		{
			result:   Passed,
			expected: "Passed",
		},
		{
			result:   Failed,
			expected: "Failed",
		},
		{
			result:   NeedsReview,
			expected: "Needs Review",
		},
		{
			result:   NotRun,
			expected: "Not Run",
		},
		{
			result:   NotApplicable,
			expected: "Not Applicable",
		},
		{
			result:   Unknown,
			expected: "Unknown",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := test.result.String()
			if actual != test.expected {
				t.Errorf("expected %q, got %q", test.expected, actual)
			}
		})
	}
}

func TestUpdateAggregateResult(t *testing.T) {
	tests := []struct {
		name     string
		prev     Result
		new      Result
		expected Result
	}{
		{
			name:     "NotRun should not overwrite anything",
			prev:     Passed,
			new:      NotRun,
			expected: Passed,
		},
		{
			name:     "Failed should not be overwritten by anything",
			prev:     Failed,
			new:      Passed,
			expected: Failed,
		},
		{
			name:     "Failed should overwrite anything",
			prev:     Passed,
			new:      Failed,
			expected: Failed,
		},
		{
			name:     "Unknown should not be overwritten by NeedsReview",
			prev:     Unknown,
			new:      NeedsReview,
			expected: Unknown,
		},
		{
			name:     "Unknown should not be overwritten by Passed",
			prev:     Unknown,
			new:      Passed,
			expected: Unknown,
		},
		{
			name:     "NeedsReview should not be overwritten by Passed",
			prev:     NeedsReview,
			new:      Passed,
			expected: NeedsReview,
		},
		{
			name:     "NeedsReview should overwrite Passed",
			prev:     Passed,
			new:      NeedsReview,
			expected: NeedsReview,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := UpdateAggregateResult(test.prev, test.new)
			if actual != test.expected {
				t.Errorf("expected %s, got %s", test.expected, actual)
			}
		})
	}
}

func TestArtifactTypeString(t *testing.T) {
	tests := []struct {
		v        ArtifactType
		expected string
	}{
		{InvalidArtifact, "Invalid"},
		{AuditLogArtifact, "AuditLog"},
		{CapabilityCatalogArtifact, "CapabilityCatalog"},
		{ControlCatalogArtifact, "ControlCatalog"},
		{EnforcementLogArtifact, "EnforcementLog"},
		{EvaluationLogArtifact, "EvaluationLog"},
		{GuidanceCatalogArtifact, "GuidanceCatalog"},
		{LexiconArtifact, "Lexicon"},
		{MappingDocumentArtifact, "MappingDocument"},
		{PolicyArtifact, "Policy"},
		{PrincipleCatalogArtifact, "PrincipleCatalog"},
		{RiskCatalogArtifact, "RiskCatalog"},
		{ThreatCatalogArtifact, "ThreatCatalog"},
		{VectorCatalogArtifact, "VectorCatalog"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.v.String(); got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestLifecycleString(t *testing.T) {
	tests := []struct {
		v        Lifecycle
		expected string
	}{
		{LifecycleActive, "Active"},
		{LifecycleDraft, "Draft"},
		{LifecycleDeprecated, "Deprecated"},
		{LifecycleRetired, "Retired"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.v.String(); got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestEntryTypeString(t *testing.T) {
	tests := []struct {
		v        EntryType
		expected string
	}{
		{EntryTypeGuideline, "Guideline"},
		{EntryTypeStatement, "Statement"},
		{EntryTypeControl, "Control"},
		{EntryTypeAssessmentRequirement, "AssessmentRequirement"},
		{EntryTypeCapability, "Capability"},
		{EntryTypeThreat, "Threat"},
		{EntryTypeRisk, "Risk"},
		{EntryTypeVector, "Vector"},
		{EntryTypePrinciple, "Principle"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.v.String(); got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestConfidenceLevelString(t *testing.T) {
	tests := []struct {
		v        ConfidenceLevel
		expected string
	}{
		{Undetermined, "Undetermined"},
		{Low, "Low"},
		{Medium, "Medium"},
		{High, "High"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.v.String(); got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestRelationshipTypeString(t *testing.T) {
	tests := []struct {
		v        RelationshipType
		expected string
	}{
		{RelImplements, "implements"},
		{RelImplementedBy, "implemented-by"},
		{RelSupports, "supports"},
		{RelSupportedBy, "supported-by"},
		{RelEquivalent, "equivalent"},
		{RelSubsumes, "subsumes"},
		{RelNoMatch, "no-match"},
		{RelRelatesTo, "relates-to"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.v.String(); got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestMethodTypeString(t *testing.T) {
	tests := []struct {
		v        MethodType
		expected string
	}{
		{MethodBehavioral, "Behavioral"},
		{MethodIntent, "Intent"},
		{MethodRemediation, "Remediation"},
		{MethodGate, "Gate"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.v.String(); got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestModeTypeString(t *testing.T) {
	tests := []struct {
		v        ModeType
		expected string
	}{
		{ModeManual, "Manual"},
		{ModeAutomated, "Automated"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.v.String(); got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDispositionString(t *testing.T) {
	tests := []struct {
		v        Disposition
		expected string
	}{
		{DispositionEnforced, "Enforced"},
		{DispositionTolerated, "Tolerated"},
		{DispositionClear, "Clear"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.v.String(); got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSeverityString(t *testing.T) {
	tests := []struct {
		v        Severity
		expected string
	}{
		{SeverityLow, "Low"},
		{SeverityMedium, "Medium"},
		{SeverityHigh, "High"},
		{SeverityCritical, "Critical"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.v.String(); got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestGuidanceTypeString(t *testing.T) {
	tests := []struct {
		v        GuidanceType
		expected string
	}{
		{GuidanceStandard, "Standard"},
		{GuidanceRegulation, "Regulation"},
		{GuidanceBestPractice, "Best Practice"},
		{GuidanceFramework, "Framework"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.v.String(); got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestRiskAppetiteString(t *testing.T) {
	tests := []struct {
		v        RiskAppetite
		expected string
	}{
		{RiskAppetiteMinimal, "Minimal"},
		{RiskAppetiteLow, "Low"},
		{RiskAppetiteModerate, "Moderate"},
		{RiskAppetiteHigh, "High"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.v.String(); got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestModTypeString(t *testing.T) {
	tests := []struct {
		v        ModType
		expected string
	}{
		{ModAdd, "Add"},
		{ModModify, "Modify"},
		{ModRemove, "Remove"},
		{ModReplace, "Replace"},
		{ModOverride, "Override"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.v.String(); got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestEntityTypeString(t *testing.T) {
	tests := []struct {
		v        EntityType
		expected string
	}{
		{Human, "Human"},
		{Software, "Software"},
		{SoftwareAssisted, "Software Assisted"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.v.String(); got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestResultStringUnknownValue(t *testing.T) {
	// Out-of-range or unknown int should not return empty string
	const unknown Result = 99
	got := unknown.String()
	if got == "" {
		t.Error("String() for unknown Result should not return empty string")
	}
	if !strings.Contains(got, "99") {
		t.Errorf("String() for unknown Result should include numeric value, got %q", got)
	}
}

func TestResultTypeStringUnknownValue(t *testing.T) {
	const unknown ResultType = 99
	got := unknown.String()
	if got == "" {
		t.Error("String() for unknown ResultType should not return empty string")
	}
	if !strings.Contains(got, "99") {
		t.Errorf("String() for unknown ResultType should include numeric value, got %q", got)
	}
}

func TestResultTypeString(t *testing.T) {
	tests := []struct {
		v        ResultType
		expected string
	}{
		{ResultObservation, "Observation"},
		{ResultStrength, "Strength"},
		{ResultFinding, "Finding"},
		{ResultGap, "Gap"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.v.String(); got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestLifecycleMarshalUnmarshalJSON(t *testing.T) {
	var l Lifecycle
	if err := l.UnmarshalJSON([]byte(`"Draft"`)); err != nil {
		t.Fatal(err)
	}
	if l != LifecycleDraft {
		t.Errorf("UnmarshalJSON: got %v, want LifecycleDraft", l)
	}
	out, err := l.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != `"Draft"` {
		t.Errorf("MarshalJSON: got %s", out)
	}
}

func TestConfidenceLevelMarshalUnmarshalJSON(t *testing.T) {
	var c ConfidenceLevel
	if err := c.UnmarshalJSON([]byte(`"High"`)); err != nil {
		t.Fatal(err)
	}
	if c != High {
		t.Errorf("UnmarshalJSON: got %v, want High", c)
	}
	out, err := c.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != `"High"` {
		t.Errorf("MarshalJSON: got %s", out)
	}
}

func TestRelationshipTypeUnmarshalJSONInvalid(t *testing.T) {
	var r RelationshipType
	err := r.UnmarshalJSON([]byte(`"invalid"`))
	if err == nil {
		t.Error("expected error for invalid RelationshipType")
	}
	// Error should include the invalid value and list valid values
	if err != nil && err.Error() != "" {
		if !strings.Contains(err.Error(), "invalid") {
			t.Errorf("error should mention invalid: %s", err.Error())
		}
		if !strings.Contains(err.Error(), "valid:") {
			t.Errorf("error should list valid values: %s", err.Error())
		}
	}
}

func TestResultTypeMarshalUnmarshalJSON(t *testing.T) {
	tests := []struct {
		value    ResultType
		jsonRepr string
	}{
		{ResultObservation, `"Observation"`},
		{ResultStrength, `"Strength"`},
		{ResultFinding, `"Finding"`},
		{ResultGap, `"Gap"`},
	}
	for _, tt := range tests {
		t.Run(tt.jsonRepr, func(t *testing.T) {
			out, err := tt.value.MarshalJSON()
			if err != nil {
				t.Fatalf("MarshalJSON: %v", err)
			}
			if string(out) != tt.jsonRepr {
				t.Errorf("MarshalJSON = %s, want %s", out, tt.jsonRepr)
			}

			var got ResultType
			if err := got.UnmarshalJSON(out); err != nil {
				t.Fatalf("UnmarshalJSON: %v", err)
			}
			if got != tt.value {
				t.Errorf("UnmarshalJSON round-trip: got %v, want %v", got, tt.value)
			}
		})
	}
}

func TestEvidenceTypeToArtifactType(t *testing.T) {
	tests := []struct {
		name     string
		ev       EvidenceType
		expected ArtifactType
		wantErr  bool
	}{
		{"AuditLog", EvidenceType("AuditLog"), AuditLogArtifact, false},
		{"CapabilityCatalog", EvidenceType("CapabilityCatalog"), CapabilityCatalogArtifact, false},
		{"ControlCatalog", EvidenceType("ControlCatalog"), ControlCatalogArtifact, false},
		{"EnforcementLog", EvidenceType("EnforcementLog"), EnforcementLogArtifact, false},
		{"EvaluationLog", EvidenceType("EvaluationLog"), EvaluationLogArtifact, false},
		{"GuidanceCatalog", EvidenceType("GuidanceCatalog"), GuidanceCatalogArtifact, false},
		{"Lexicon", EvidenceType("Lexicon"), LexiconArtifact, false},
		{"MappingDocument", EvidenceType("MappingDocument"), MappingDocumentArtifact, false},
		{"Policy", EvidenceType("Policy"), PolicyArtifact, false},
		{"PrincipleCatalog", EvidenceType("PrincipleCatalog"), PrincipleCatalogArtifact, false},
		{"RiskCatalog", EvidenceType("RiskCatalog"), RiskCatalogArtifact, false},
		{"ThreatCatalog", EvidenceType("ThreatCatalog"), ThreatCatalogArtifact, false},
		{"VectorCatalog", EvidenceType("VectorCatalog"), VectorCatalogArtifact, false},
		{"invalid value", EvidenceType("not-an-artifact"), 0, true},
		{"empty string", EvidenceType(""), 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.ev.ToArtifactType()
			if (err != nil) != tt.wantErr {
				t.Errorf("ToArtifactType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.expected {
				t.Errorf("ToArtifactType() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestEvidenceTypeJSONRoundTrip(t *testing.T) {
	type wrapper struct {
		Type EvidenceType `json:"type"`
	}

	tests := []string{"document", "interview", "automated-scan", "custom-value"}
	for _, val := range tests {
		t.Run(val, func(t *testing.T) {
			input := wrapper{Type: EvidenceType(val)}
			data, err := json.Marshal(input)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}

			var got wrapper
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			if got.Type != input.Type {
				t.Errorf("round-trip: got %q, want %q", got.Type, input.Type)
			}
		})
	}
}
