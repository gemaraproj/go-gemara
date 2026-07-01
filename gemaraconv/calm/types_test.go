// SPDX-License-Identifier: Apache-2.0

package calm

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestStandardSchemaReturnsCopy asserts the accessor hands back an independent
// copy, so a caller mutating the result cannot corrupt the embedded schema.
func TestStandardSchemaReturnsCopy(t *testing.T) {
	a := StandardSchema()
	if len(a) == 0 {
		t.Fatal("StandardSchema() returned no bytes")
	}
	a[0] = 'X'

	b := StandardSchema()
	if b[0] == 'X' {
		t.Fatal("StandardSchema() shares mutable state across calls")
	}
	if !json.Valid(b) {
		t.Fatal("embedded Standard is not valid JSON")
	}
}

// TestMarshalDocumentShape asserts the {"controls": {...}} wrapper and that a
// requirements-less control still serializes requirements as [] (never null),
// which CALM control.json requires.
func TestMarshalDocumentShape(t *testing.T) {
	controls := Controls{
		"Encryption": Control{Description: "d"}, // nil Requirements
	}
	raw, err := controls.MarshalDocument()
	if err != nil {
		t.Fatal(err)
	}
	got := string(raw)
	if !strings.Contains(got, `"controls"`) {
		t.Errorf("expected a controls wrapper, got:\n%s", got)
	}
	if strings.Contains(got, `"requirements": null`) {
		t.Errorf("requirements must never marshal to null, got:\n%s", got)
	}
	if !strings.Contains(got, `"requirements": []`) {
		t.Errorf("expected requirements to be [], got:\n%s", got)
	}
}
