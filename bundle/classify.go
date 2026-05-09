// SPDX-License-Identifier: Apache-2.0

package bundle

import (
	"fmt"

	"github.com/gemaraproj/go-gemara"
)

// ClassifiedBundle holds the result of classifying a bundle's raw files
// into typed Gemara artifacts
type ClassifiedBundle struct {
	Policy          *gemara.Policy
	ControlCatalog  *gemara.ControlCatalog
	GuidanceCatalog *gemara.GuidanceCatalog
	Imports         *gemara.ArtifactSet
}

// Classify unmarshals the bundle's raw files into typed artifacts.
// All Files (role=artifact) are classified as leaf artifacts; all
// Imports (role=import) are classified into the Imports ArtifactSet.
// The leaf fields are set from the classified Files results.
func (b *Bundle) Classify() (*ClassifiedBundle, error) {
	if len(b.Files) == 0 {
		return nil, fmt.Errorf("bundle has no primary files")
	}

	var leafData [][]byte
	for _, f := range b.Files {
		leafData = append(leafData, f.Data)
	}

	leafSet, err := gemara.Classify(leafData...)
	if err != nil {
		return nil, fmt.Errorf("classifying leaf artifacts: %w", err)
	}

	cb := &ClassifiedBundle{}
	if len(leafSet.Policies) > 0 {
		cb.Policy = &leafSet.Policies[0]
	}
	if len(leafSet.ControlCatalogs) > 0 {
		cb.ControlCatalog = &leafSet.ControlCatalogs[0]
	}
	if len(leafSet.GuidanceCatalogs) > 0 {
		cb.GuidanceCatalog = &leafSet.GuidanceCatalogs[0]
	}

	var depData [][]byte
	for _, f := range b.Imports {
		depData = append(depData, f.Data)
	}

	if len(depData) > 0 {
		imports, err := gemara.Classify(depData...)
		if err != nil {
			return nil, fmt.Errorf("classifying imports: %w", err)
		}
		cb.Imports = imports
	} else {
		cb.Imports = &gemara.ArtifactSet{}
	}

	return cb, nil
}
