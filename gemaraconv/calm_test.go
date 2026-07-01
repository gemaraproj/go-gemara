// SPDX-License-Identifier: Apache-2.0

package gemaraconv

import (
	"encoding/json"
	"flag"
	"net/url"
	"os"
	"strings"
	"testing"

	gemara "github.com/gemaraproj/go-gemara"
	"github.com/gemaraproj/go-gemara/gemaraconv/calm"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sampleCatalog returns a small ControlCatalog modelled on real FINOS CCC core
// controls: two controls (one with two assessment requirements, one with one).
func sampleCatalog() gemara.ControlCatalog {
	return gemara.ControlCatalog{
		Title: "Common Cloud Controls Core",
		Metadata: gemara.Metadata{
			Id:      "CCC.Core",
			Version: "2025.1",
			Author:  gemara.Actor{Id: "finos-ccc", Name: "FINOS"},
		},
		Groups: []gemara.Group{
			{Id: "Encryption", Title: "Encryption", Description: "Encryption controls."},
			{Id: "Logging", Title: "Logging", Description: "Logging controls."},
		},
		Controls: []gemara.Control{
			{
				Id:        "CCC.Core.CN01",
				Group:     "Encryption",
				Title:     "Encrypt Data for Transmission",
				Objective: "Ensure that all communications are encrypted in transit.",
				AssessmentRequirements: []gemara.AssessmentRequirement{
					{
						Id:             "CCC.Core.CN01.AR01",
						Text:           "When a port is exposed for non-SSH network traffic, all traffic MUST use TLS 1.3 or higher.",
						Applicability:  []string{"tlp-green", "tlp-amber", "tlp-red"},
						Recommendation: "Most cloud services enable TLS 1.3 by default.",
						State:          gemara.LifecycleActive,
					},
					{
						Id:            "CCC.Core.CN01.AR02",
						Text:          "When a port is exposed for SSH network traffic, all traffic MUST use SSHv2 or higher.",
						Applicability: []string{"tlp-clear", "tlp-green", "tlp-amber", "tlp-red"},
						State:         gemara.LifecycleActive,
					},
				},
			},
			{
				Id:        "CCC.Core.CN02",
				Group:     "Logging",
				Title:     "Protect Access Logs",
				Objective: "Ensure access logs cannot be tampered with or deleted.",
				AssessmentRequirements: []gemara.AssessmentRequirement{
					{
						Id:            "CCC.Core.CN02.AR01",
						Text:          "When access logs are stored, they MUST be protected from unauthorized modification.",
						Applicability: []string{"tlp-amber", "tlp-red"},
						State:         gemara.LifecycleActive,
					},
				},
			},
		},
	}
}

// updateGolden is shared with the CCC fixture round-trip test
// (calm_fixture_test.go), which is the canonical golden for exact-output drift.
var updateGolden = flag.Bool("update", false, "update golden files")

func TestNormalizeCALMKey(t *testing.T) {
	cases := map[string]string{
		"data-protection": "data-protection",
		"CCC.Core.CN01":   "CCC-Core-CN01",
		"Data Protection": "Data-Protection",
		".leading.dot.":   "leading-dot",
	}
	for in, want := range cases {
		assert.Equal(t, want, normalizeCALMKey(in), "normalizeCALMKey(%q)", in)
	}
}

func TestControlCatalogToCALM(t *testing.T) {
	controls, err := ControlCatalogToCALM(sampleCatalog())
	require.NoError(t, err)

	// One CALM control-name per Gemara control, its requirements grouped under it.
	require.Len(t, controls, 2)
	require.Contains(t, controls, "CCC-Core-CN01")
	require.Contains(t, controls, "CCC-Core-CN02")

	cn01 := controls["CCC-Core-CN01"]
	assert.Equal(t, "Encrypt Data for Transmission — Ensure that all communications are encrypted in transit.", cn01.Description)
	require.Len(t, cn01.Requirements, 2)

	first := cn01.Requirements[0]
	assert.Equal(t, calm.RequirementURL, first.RequirementURL)
	require.NotNil(t, first.Config)
	// Source identifiers preserved verbatim, dots and all.
	assert.Equal(t, "CCC.Core.CN01.AR01", first.Config.ControlID)
	assert.Equal(t, "CCC.Core.CN01", first.Config.GemaraControlID)
	assert.Equal(t, "Encrypt Data for Transmission", first.Config.Name)
	assert.Equal(t, "Encryption", first.Config.Group)
	assert.Equal(t, "CCC.Core", first.Config.CatalogID)
	assert.Equal(t, []string{"tlp-green", "tlp-amber", "tlp-red"}, first.Config.Applicability)
	// Traceability: author + id + version pin the requirement to a published release.
	assert.Equal(t, "finos-ccc", first.Config.CatalogAuthor)
	assert.Equal(t, "2025.1", first.Config.CatalogVersion)
	assert.Equal(t, "Active", first.Config.State)

	// The second control is its own key with its own requirement.
	cn02 := controls["CCC-Core-CN02"]
	require.Len(t, cn02.Requirements, 1)
	assert.Equal(t, "CCC.Core.CN02.AR01", cn02.Requirements[0].Config.ControlID)
	assert.Equal(t, "Logging", cn02.Requirements[0].Config.Group)
}

// TestControlCatalogToCALM_DescriptionFallbacks covers the entry-description
// fallbacks: a control with no objective (title alone) and no title (objective alone).
func TestControlCatalogToCALM_DescriptionFallbacks(t *testing.T) {
	// Objective empty -> description is the title alone.
	cat := sampleCatalog()
	cat.Controls[0].Objective = ""
	controls, err := ControlCatalogToCALM(cat)
	require.NoError(t, err)
	assert.Equal(t, "Encrypt Data for Transmission", controls["CCC-Core-CN01"].Description)

	// Title empty -> description is the objective alone.
	cat = sampleCatalog()
	cat.Controls[0].Title = ""
	controls, err = ControlCatalogToCALM(cat)
	require.NoError(t, err)
	assert.Equal(t, "Ensure that all communications are encrypted in transit.", controls["CCC-Core-CN01"].Description)
}

// TestControlCatalogConverter_ToCALM exercises the public wrapper entry point
// (the call documented in the README), not just the underlying free function.
func TestControlCatalogConverter_ToCALM(t *testing.T) {
	controls, err := ControlCatalog(sampleCatalog()).ToCALM()
	require.NoError(t, err)
	require.Contains(t, controls, "CCC-Core-CN01")
}

// TestControlCatalogToCALM_KeyCollision asserts that two distinct control ids that
// normalize to the same CALM key are rejected, not silently merged.
func TestControlCatalogToCALM_KeyCollision(t *testing.T) {
	cat := sampleCatalog()
	// "CCC-Core.CN01" normalizes to the same key ("CCC-Core-CN01") as the existing
	// "CCC.Core.CN01" — two distinct controls must not silently merge.
	cat.Controls = append(cat.Controls, gemara.Control{
		Id:    "CCC-Core.CN01",
		Group: "Encryption",
		Title: "Another",
		AssessmentRequirements: []gemara.AssessmentRequirement{
			{Id: "CCC-Core.CN01.AR01", Text: "x", State: gemara.LifecycleActive},
		},
	})
	_, err := ControlCatalogToCALM(cat)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "normalize")
}

// TestControlCatalogToCALM_CollapsesWhitespace asserts folded-YAML newlines in
// prose fields (title, requirement text, recommendation) are collapsed, not
// leaked verbatim into the config.
func TestControlCatalogToCALM_CollapsesWhitespace(t *testing.T) {
	cat := sampleCatalog()
	cat.Controls[0].Title = "Encrypt\nData"
	cat.Controls[0].AssessmentRequirements[0].Text = "Line one.\n  Line two."
	cat.Controls[0].AssessmentRequirements[0].Recommendation = "Do\n\tthis."

	controls, err := ControlCatalogToCALM(cat)
	require.NoError(t, err)
	cfg := controls["CCC-Core-CN01"].Requirements[0].Config
	require.NotNil(t, cfg)
	assert.Equal(t, "Encrypt Data", cfg.Name)
	assert.Equal(t, "Line one. Line two.", cfg.Description)
	assert.Equal(t, "Do this.", cfg.Recommendation)

	// No embedded newline escapes survive into the emitted document.
	raw, err := controls.MarshalDocument()
	require.NoError(t, err)
	assert.NotContains(t, string(raw), `\n`)
}

// TestControlCatalogToCALM_DuplicateControlID asserts exact-duplicate control ids
// error rather than silently merging.
func TestControlCatalogToCALM_DuplicateControlID(t *testing.T) {
	cat := sampleCatalog()
	cat.Controls = append(cat.Controls, cat.Controls[0]) // same Id repeated
	_, err := ControlCatalogToCALM(cat)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate control identifier")
}

// TestControlCatalogToCALM_ZeroRequirements asserts a control with no assessment
// requirements still produces a CALM-valid entry (requirements is [], never null).
func TestControlCatalogToCALM_ZeroRequirements(t *testing.T) {
	cat := sampleCatalog()
	cat.Controls = append(cat.Controls, gemara.Control{
		Id: "CCC.Core.CN99", Title: "Stub",
	})

	controls, err := ControlCatalogToCALM(cat)
	require.NoError(t, err)

	raw, err := controls.MarshalDocument()
	require.NoError(t, err)
	assert.NotContains(t, string(raw), `"requirements": null`)

	// And the whole controls block conforms to the real upstream CALM control.json.
	assertConformsToControlsSchema(t, controls)
}

// resolvedStandard compiles the shipped Gemara Standard, resolving its external
// $ref to the CALM base control-requirement against the vendored testdata copy so
// the check runs offline.
func resolvedStandard(t *testing.T) *jsonschema.Resolved {
	t.Helper()

	var schema jsonschema.Schema
	require.NoError(t, json.Unmarshal(calm.StandardSchema(), &schema))

	baseBytes, err := os.ReadFile("testdata/control-requirement.json")
	require.NoError(t, err)

	loader := func(_ *url.URL) (*jsonschema.Schema, error) {
		var base jsonschema.Schema
		if err := json.Unmarshal(baseBytes, &base); err != nil {
			return nil, err
		}
		return &base, nil
	}

	resolved, err := schema.Resolve(&jsonschema.ResolveOptions{Loader: loader})
	require.NoError(t, err)
	return resolved
}

// assertConformsToControlsSchema validates a Controls block against the real
// upstream CALM control.json (#/defs/controls), vendored in testdata so the check
// runs offline. This guards the OUTER shape (control-name keys, control-detail),
// which the Gemara Standard (an inner-config schema) does not cover.
func assertConformsToControlsSchema(t *testing.T, controls calm.Controls) {
	t.Helper()

	controlBytes, err := os.ReadFile("testdata/control.json")
	require.NoError(t, err)

	// CALM control.json uses the non-standard "defs" keyword; rewrite it to the
	// JSON Schema 2020-12 "$defs" (and its internal refs) so a strict validator can
	// resolve them, then validate the controls block against #/$defs/controls. The
	// vendored file stays a pristine upstream copy for drift comparison.
	var doc map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(controlBytes, &doc))
	defs := strings.ReplaceAll(string(doc["defs"]), "#/defs/", "#/$defs/")
	wrapper := `{"$schema":"https://json-schema.org/draft/2020-12/schema","$ref":"#/$defs/controls","$defs":` + defs + `}`

	var schema jsonschema.Schema
	require.NoError(t, json.Unmarshal([]byte(wrapper), &schema))
	resolved, err := schema.Resolve(nil)
	require.NoError(t, err)

	raw, err := json.Marshal(controls)
	require.NoError(t, err)
	var v any
	require.NoError(t, json.Unmarshal(raw, &v))
	require.NoError(t, resolved.Validate(v), "controls block should conform to CALM control.json")
}

// TestControlCatalogToCALM_SchemaConformance validates the emitted configs against
// the Gemara Standard, and the emitted controls block against the real CALM
// control.json. `calm validate` does NOT check control configs against their
// requirement-url, so this conformance guard lives in the converter's test suite.
func TestControlCatalogToCALM_SchemaConformance(t *testing.T) {
	controls, err := ControlCatalogToCALM(sampleCatalog())
	require.NoError(t, err)

	// Outer shape: conforms to the upstream CALM controls schema.
	assertConformsToControlsSchema(t, controls)

	resolved := resolvedStandard(t)

	count := 0
	for _, ctrl := range controls {
		for _, req := range ctrl.Requirements {
			require.NotNil(t, req.Config)
			raw, err := json.Marshal(req.Config)
			require.NoError(t, err)
			var v any
			require.NoError(t, json.Unmarshal(raw, &v))
			assert.NoErrorf(t, resolved.Validate(v), "config %s should conform to the Gemara Standard", req.Config.ControlID)
			count++
		}
	}
	require.Positive(t, count)

	// Negative controls: each isolates exactly ONE missing required field (all
	// others present), so a dropped requirement on any single field is caught
	// without being masked by a co-missing field.
	for _, drop := range []string{"gemara-control-id", "catalog-author", "catalog-id", "catalog-version"} {
		v := map[string]any{
			"control-id":        "X.1",
			"gemara-control-id": "X",
			"name":              "X",
			"description":       "d",
			"catalog-author":    "A",
			"catalog-id":        "C",
			"catalog-version":   "1",
		}
		delete(v, drop)
		assert.Errorf(t, resolved.Validate(v), "a config missing %s must be rejected", drop)
	}
}
