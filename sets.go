// SPDX-License-Identifier: Apache-2.0

package gemara

import (
	"fmt"

	"github.com/gemaraproj/go-gemara/internal/codec"
)

// ArtifactSet holds artifacts grouped by type after classification.
type ArtifactSet struct {
	Policies         []Policy
	ControlCatalogs  []ControlCatalog
	GuidanceCatalogs []GuidanceCatalog
}

// Classify detects and unmarshals raw artifact data into typed slices.
// Unrecognised artifact types are silently skipped; malformed YAML
// returns an error immediately.
func Classify(data ...[]byte) (*ArtifactSet, error) {
	as := &ArtifactSet{}
	for i, d := range data {
		artType, err := DetectType(d)
		if err != nil {
			return nil, fmt.Errorf("artifact %d: %w", i, err)
		}
		switch artType {
		case PolicyArtifact:
			var p Policy
			if err := codec.UnmarshalYAML(d, &p); err != nil {
				return nil, fmt.Errorf("artifact %d (Policy): %w", i, err)
			}
			as.Policies = append(as.Policies, p)
		case ControlCatalogArtifact:
			var cc ControlCatalog
			if err := codec.UnmarshalYAML(d, &cc); err != nil {
				return nil, fmt.Errorf("artifact %d (ControlCatalog): %w", i, err)
			}
			as.ControlCatalogs = append(as.ControlCatalogs, cc)
		case GuidanceCatalogArtifact:
			var gc GuidanceCatalog
			if err := codec.UnmarshalYAML(d, &gc); err != nil {
				return nil, fmt.Errorf("artifact %d (GuidanceCatalog): %w", i, err)
			}
			as.GuidanceCatalogs = append(as.GuidanceCatalogs, gc)
		}
	}
	return as, nil
}
