// SPDX-License-Identifier: Apache-2.0

// Package calm provides Go types for the subset of the FINOS Common Architecture
// Language Model (CALM, release/1.2) that go-gemara emits: the `controls` block
// (control.json#/defs/controls) and the Gemara control-requirement Standard that
// each requirement references as its schema.
package calm

import (
	_ "embed"
	"encoding/json"
)

// RequirementURL is the canonical $id of the Gemara CALM control-requirement
// Standard, and the default value emitted as each control-detail's requirement-url.
const RequirementURL = "https://gemara.openssf.org/schema/calm/v1/gemara-control-requirement.json"

//go:embed standards/gemara-control-requirement.json
var standardSchema []byte

// StandardSchema returns the embedded Gemara control-requirement Standard
// (a JSON Schema 2020-12 document) as a fresh copy of its raw bytes.
func StandardSchema() []byte {
	out := make([]byte, len(standardSchema))
	copy(out, standardSchema)
	return out
}

// Controls is a CALM `controls` block: a map keyed by control name (a grouping
// label that must match ^[a-zA-Z0-9-]+$), per control.json#/defs/controls.
type Controls map[string]Control

// Control is a single named entry in a CALM controls block.
type Control struct {
	Description  string          `json:"description"`
	Requirements []ControlDetail `json:"requirements"`
}

// MarshalJSON forces requirements to a JSON array, never null — CALM control.json
// requires an array, and a nil slice would marshal as null.
func (c Control) MarshalJSON() ([]byte, error) {
	type alias Control
	a := alias(c)
	if a.Requirements == nil {
		a.Requirements = []ControlDetail{}
	}
	return json.Marshal(a)
}

// ControlDetail is one requirement within a control (control.json#/defs/control-detail).
// RequirementURL is required; exactly one of Config or ConfigURL must be set.
type ControlDetail struct {
	RequirementURL string                    `json:"requirement-url"`
	Config         *GemaraControlRequirement `json:"config,omitempty"`
	ConfigURL      string                    `json:"config-url,omitempty"`
}

// GemaraControlRequirement is an inline control `config` conforming to the Gemara
// control-requirement Standard (which extends the CALM base control-requirement).
// Field names mirror their CALM JSON keys. catalog-author/id/version are the
// required traceability anchor back to the source release.
type GemaraControlRequirement struct {
	// ControlID is CALM's control-id; it holds the assessment-requirement id
	// (e.g. "CCC.C01.TR01"), not the control id.
	ControlID string `json:"control-id"`
	// GemaraControlID is the parent control id (e.g. "CCC.C01").
	GemaraControlID string   `json:"gemara-control-id"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	Group           string   `json:"group,omitempty"`
	Applicability   []string `json:"applicability,omitempty"`
	Recommendation  string   `json:"recommendation,omitempty"`
	State           string   `json:"state,omitempty"`
	CatalogAuthor   string   `json:"catalog-author"`
	CatalogID       string   `json:"catalog-id"`
	CatalogVersion  string   `json:"catalog-version"`
}

// Document wraps a Controls block as a standalone CALM fragment: {"controls": {...}}.
type Document struct {
	Controls Controls `json:"controls"`
}

// MarshalDocument renders the controls block as an indented standalone
// {"controls": {...}} JSON document.
func (c Controls) MarshalDocument() ([]byte, error) {
	return json.MarshalIndent(Document{Controls: c}, "", "  ")
}
