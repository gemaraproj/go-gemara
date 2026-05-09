// SPDX-License-Identifier: Apache-2.0

package gemara

import (
	"fmt"
	"sync"

	"github.com/gemaraproj/go-gemara/internal/codec"
)

// EffectivePolicy is a fully resolved Policy with all catalog and guidance
// imports resolved against the provided artifacts. Lookup methods use
// lazy-initialized caches for efficient repeated access.
type EffectivePolicy struct {
	Policy   Policy
	ControlCatalogs  []ControlCatalog
	GuidanceCatalogs []GuidanceCatalog

	// TODO: Imports.Policies (recursive policy imports) are not yet resolved.

	controlCatalogsByIDOnce  sync.Once
	controlCatalogsByIDCache map[string]*ControlCatalog

	guidanceCatalogsByIDOnce  sync.Once
	guidanceCatalogsByIDCache map[string]*GuidanceCatalog

	controlCatalogIDsOnce  sync.Once
	controlCatalogIDsCache []string

	guidanceCatalogIDsOnce  sync.Once
	guidanceCatalogIDsCache []string
}

// SPolicy wraps a Policy with a convenience Resolve method that produces
// an EffectivePolicy in a single call.
type SPolicy struct {
	Policy
}

// Sugar wraps this Policy in an SPolicy for convenient method access.
func (p *Policy) Sugar() *SPolicy {
	return &SPolicy{Policy: *p}
}

func (p *SPolicy) ToBase() Policy {
	return p.Policy
}

func (p *SPolicy) FromBase(s *Policy) {
	p.Policy = *s
}

func (p *SPolicy) MarshalYAML() ([]byte, error) {
	return codec.MarshalBaseYAML[Policy](p)
}

func (p *SPolicy) UnmarshalYAML(data []byte) error {
	return codec.UnmarshalBaseYAML[Policy](data, p)
}

// Resolve resolves the policy's imports against the provided catalogs and
// guidance, returning an EffectivePolicy with cached lookups ready.
func (p *SPolicy) Resolve(catalogs []ControlCatalog, guidance []GuidanceCatalog) (*EffectivePolicy, error) {
	return ResolveEffectivePolicy(p.Policy, catalogs, guidance)
}

func (ep *EffectivePolicy) buildCatalogIndex() {
	ep.controlCatalogsByIDOnce.Do(func() {
		ep.controlCatalogsByIDCache = make(map[string]*ControlCatalog, len(ep.ControlCatalogs))
		for i := range ep.ControlCatalogs {
			ep.controlCatalogsByIDCache[ep.ControlCatalogs[i].Metadata.Id] = &ep.ControlCatalogs[i]
		}
	})
}

func (ep *EffectivePolicy) buildGuidanceIndex() {
	ep.guidanceCatalogsByIDOnce.Do(func() {
		ep.guidanceCatalogsByIDCache = make(map[string]*GuidanceCatalog, len(ep.GuidanceCatalogs))
		for i := range ep.GuidanceCatalogs {
			ep.guidanceCatalogsByIDCache[ep.GuidanceCatalogs[i].Metadata.Id] = &ep.GuidanceCatalogs[i]
		}
	})
}

// GetControlCatalog returns the resolved ControlCatalog with the given ID,
// or nil if no catalog with that ID was resolved.
func (ep *EffectivePolicy) GetControlCatalog(id string) *ControlCatalog {
	ep.buildCatalogIndex()
	return ep.controlCatalogsByIDCache[id]
}

// GetGuidanceCatalog returns the resolved GuidanceCatalog with the given ID,
// or nil if no guidance catalog with that ID was resolved.
func (ep *EffectivePolicy) GetGuidanceCatalog(id string) *GuidanceCatalog {
	ep.buildGuidanceIndex()
	return ep.guidanceCatalogsByIDCache[id]
}

// ControlCatalogIDs returns the IDs of all resolved control catalogs.
func (ep *EffectivePolicy) ControlCatalogIDs() []string {
	ep.controlCatalogIDsOnce.Do(func() {
		ep.controlCatalogIDsCache = make([]string, len(ep.ControlCatalogs))
		for i, c := range ep.ControlCatalogs {
			ep.controlCatalogIDsCache[i] = c.Metadata.Id
		}
	})
	return ep.controlCatalogIDsCache
}

// GuidanceCatalogIDs returns the IDs of all resolved guidance catalogs.
func (ep *EffectivePolicy) GuidanceCatalogIDs() []string {
	ep.guidanceCatalogIDsOnce.Do(func() {
		ep.guidanceCatalogIDsCache = make([]string, len(ep.GuidanceCatalogs))
		for i, g := range ep.GuidanceCatalogs {
			ep.guidanceCatalogIDsCache[i] = g.Metadata.Id
		}
	})
	return ep.guidanceCatalogIDsCache
}

// ResolveEffectivePolicy resolves a policy's imports against the provided
// catalogs and guidance catalogs. Each import's reference-id must appear
// in the policy's metadata.mapping-references; the matching entry's id is
// used to locate the artifact in the pool by metadata.id. Currently the
// mapping is an identity (reference-id == metadata.id).
//
// Missing individual imports are silently skipped to support partial bundles.
func ResolveEffectivePolicy(policy Policy, catalogs []ControlCatalog, guidance []GuidanceCatalog) (*EffectivePolicy, error) {
	refIndex := buildRefIndex(policy.Metadata.MappingReferences)

	catalogsByID, err := poolIndex(catalogs, controlAccessor.metadataID, controlAccessor.typeName)
	if err != nil {
		return nil, fmt.Errorf("building catalog pool: %w", err)
	}
	guidanceByID, err := poolIndex(guidance, guidanceAccessor.metadataID, guidanceAccessor.typeName)
	if err != nil {
		return nil, fmt.Errorf("building guidance pool: %w", err)
	}

	ep := &EffectivePolicy{Policy: policy}

	for _, imp := range policy.Imports.Catalogs {
		metaID, ok := refIndex[imp.ReferenceId]
		if !ok {
			continue
		}
		cat, ok := catalogsByID[metaID]
		if !ok {
			continue
		}

		resolved, err := ResolveControlCatalog(cat, catalogs)
		if err != nil {
			return nil, fmt.Errorf("resolving catalog %q: %w", imp.ReferenceId, err)
		}

		ec := ApplyCatalogOverlays(*resolved, imp)
		ep.ControlCatalogs = append(ep.ControlCatalogs, *ec)
	}

	for _, imp := range policy.Imports.Guidance {
		metaID, ok := refIndex[imp.ReferenceId]
		if !ok {
			continue
		}
		gc, ok := guidanceByID[metaID]
		if !ok {
			continue
		}

		resolved, err := ResolveGuidanceCatalog(gc, guidance)
		if err != nil {
			return nil, fmt.Errorf("resolving guidance %q: %w", imp.ReferenceId, err)
		}

		eg := ApplyGuidanceOverlays(*resolved, imp)
		ep.GuidanceCatalogs = append(ep.GuidanceCatalogs, *eg)
	}

	hasCatalogImports := len(policy.Imports.Catalogs) > 0
	hasGuidanceImports := len(policy.Imports.Guidance) > 0
	resolvedNothing := len(ep.ControlCatalogs) == 0 && len(ep.GuidanceCatalogs) == 0

	if (hasCatalogImports || hasGuidanceImports) && resolvedNothing {
		return nil, fmt.Errorf("no imports could be resolved for policy %s", policy.Metadata.Id)
	}

	return ep, nil
}

// buildRefIndex builds a lookup from mapping-reference ID to the artifact
// metadata ID it resolves to. In the current Gemara specification these
// are always equal (identity mapping). If the spec introduces indirection
// (e.g., reference-id != metadata.id) this function is the single place
// to change.
func buildRefIndex(refs []MappingReference) map[string]string {
	idx := make(map[string]string, len(refs))
	for _, ref := range refs {
		idx[ref.Id] = ref.Id
	}
	return idx
}
