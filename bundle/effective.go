// SPDX-License-Identifier: Apache-2.0

package bundle

import (
	"github.com/gemaraproj/go-gemara"
)

// EffectiveCatalog is the resolved in-memory representation of a Gemara ControlCatalog,
// with all extends and policy imports/modifications fully processed.
type EffectiveCatalog = gemara.EffectiveCatalog

// EffectivePolicy is the resolved representation of a Gemara Policy with all its
// imported catalogs, policies, and guidelines fully resolved into effective sets.
type EffectivePolicy = gemara.EffectivePolicy

// EffectiveGuidanceCatalog is the resolved in-memory representation of a Gemara GuidanceCatalog,
// with all extends and policy imports/modifications fully processed.
type EffectiveGuidanceCatalog = gemara.EffectiveGuidanceCatalog

// RequirementProvenance records how an effective assessment requirement was derived.
type RequirementProvenance = gemara.RequirementProvenance
