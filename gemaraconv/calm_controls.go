// SPDX-License-Identifier: Apache-2.0

package gemaraconv

import (
	"fmt"
	"regexp"
	"strings"

	gemara "github.com/gemaraproj/go-gemara"
	"github.com/gemaraproj/go-gemara/gemaraconv/calm"
)

// CALM control-name keys must match ^[a-zA-Z0-9-]+$ (control.json#/defs/controls).
var nonCALMKeyChars = regexp.MustCompile(`[^A-Za-z0-9-]+`)

// normalizeCALMKey rewrites s into a valid CALM key: disallowed-char runs become a
// hyphen, leading/trailing hyphens are trimmed.
func normalizeCALMKey(s string) string {
	return strings.Trim(nonCALMKeyChars.ReplaceAllString(s, "-"), "-")
}

// ControlCatalogToCALM converts a Gemara ControlCatalog into a CALM `controls`
// block (release/1.2): one CALM control-name per Gemara control, with each of the
// control's assessment requirements as a control-detail.
func ControlCatalogToCALM(catalog gemara.ControlCatalog) (calm.Controls, error) {
	// Traceability anchor (grc.store: {author}/{id}/versions/{version}), carried
	// verbatim. Input is trusted as a vetted Gemara catalog.
	author := catalog.Metadata.Author.Id
	catalogID := catalog.Metadata.Id
	version := catalog.Metadata.Version

	controls := make(calm.Controls)
	keySource := make(map[string]string) // normalized key -> control id, to catch collisions

	for _, control := range catalog.Controls {
		key := normalizeCALMKey(control.Id)
		// Each control needs a unique key; reject collisions (distinct ids that
		// normalize alike, or exact duplicates) that would silently merge controls.
		if prev, ok := keySource[key]; ok {
			if prev == control.Id {
				return nil, fmt.Errorf("duplicate control identifier %q yields a repeated CALM control-name key %q; control ids must be unique", control.Id, key)
			}
			return nil, fmt.Errorf("Gemara identifiers %q and %q both normalize to CALM control-name key %q; rename one to avoid silently merging distinct controls", prev, control.Id, key)
		}
		keySource[key] = control.Id

		entry := calm.Control{Description: controlDescription(control)}
		for _, ar := range control.AssessmentRequirements {
			entry.Requirements = append(entry.Requirements, calm.ControlDetail{
				RequirementURL: calm.RequirementURL,
				Config: &calm.GemaraControlRequirement{
					ControlID:       ar.Id,
					GemaraControlID: control.Id,
					Name:            collapseWhitespace(control.Title),
					Description:     collapseWhitespace(ar.Text),
					Group:           control.Group,
					Applicability:   ar.Applicability,
					Recommendation:  collapseWhitespace(ar.Recommendation),
					State:           ar.State.String(),
					CatalogAuthor:   author,
					CatalogID:       catalogID,
					CatalogVersion:  version,
				},
			})
		}
		controls[key] = entry
	}

	return controls, nil
}

// controlDescription is the CALM control-name entry description: the control title,
// with its objective appended when present.
func controlDescription(control gemara.Control) string {
	desc := collapseWhitespace(control.Title)
	if obj := collapseWhitespace(control.Objective); obj != "" {
		if desc != "" {
			desc += " — " + obj
		} else {
			desc = obj
		}
	}
	return desc
}

// collapseWhitespace collapses whitespace runs (incl. folded-YAML newlines) to single spaces.
func collapseWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
