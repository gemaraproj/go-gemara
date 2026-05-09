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
//
// UnresolvedCatalogs and UnresolvedGuidance list reference-ids that could
// not be matched during resolution (missing mapping-reference or absent
// from the artifact pool). Callers should inspect these to detect partial
// resolution.
type EffectivePolicy struct {
	Policy           Policy
	ControlCatalogs  []ControlCatalog
	GuidanceCatalogs []GuidanceCatalog

	UnresolvedCatalogs []string
	UnresolvedGuidance []string

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

// ResolveWithOpts resolves imports like Resolve but honors ResolveEffectivePolicyOpts
// (strict imports and/or strict extends).
func (p *SPolicy) ResolveWithOpts(catalogs []ControlCatalog, guidance []GuidanceCatalog, opts ResolveEffectivePolicyOpts) (*EffectivePolicy, error) {
	return ResolveEffectivePolicyWithOpts(p.Policy, catalogs, guidance, opts)
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

// ResolveEffectivePolicyOpts configures ResolveEffectivePolicyWithOpts.
type ResolveEffectivePolicyOpts struct {
	// StrictImports causes resolution to fail if any declared catalog or guidance
	// import cannot be matched (mapping-reference or pool). When false,
	// unresolved ids are recorded on EffectivePolicy only.
	StrictImports bool
	// StrictExtends causes resolution to fail if any catalog or guidance
	// extends chain references a missing pool member. When false, missing
	// extends targets are skipped.
	StrictExtends bool
}

// ResolveEffectivePolicy resolves a policy's imports against the provided
// catalogs and guidance catalogs. Each import's reference-id must appear
// in the policy's metadata.mapping-references; the matching entry's id is
// used to locate the artifact in the pool by metadata.id. Currently the
// mapping is an identity (reference-id == metadata.id).
//
// Missing individual imports are skipped and listed on EffectivePolicy
// (UnresolvedCatalogs / UnresolvedGuidance) to support partial bundles.
// Use ResolveEffectivePolicyWithOpts with StrictImports: true for fail-closed resolution.
func ResolveEffectivePolicy(policy Policy, catalogs []ControlCatalog, guidance []GuidanceCatalog) (*EffectivePolicy, error) {
	return ResolveEffectivePolicyWithOpts(policy, catalogs, guidance, ResolveEffectivePolicyOpts{})
}

// ResolveEffectivePolicyWithOpts resolves like ResolveEffectivePolicy with optional
// strict behavior for imports and extends.
func ResolveEffectivePolicyWithOpts(policy Policy, catalogs []ControlCatalog, guidance []GuidanceCatalog, opts ResolveEffectivePolicyOpts) (*EffectivePolicy, error) {
	if err := checkDuplicateImportRefs(policy.Imports); err != nil {
		return nil, err
	}

	refIndex, err := buildRefIndex(policy.Metadata.MappingReferences)
	if err != nil {
		return nil, err
	}

	catalogsByID, err := poolIndex(catalogs, controlAccessor.metadataID, controlAccessor.typeName)
	if err != nil {
		return nil, fmt.Errorf("building catalog pool: %w", err)
	}
	guidanceByID, err := poolIndex(guidance, guidanceAccessor.metadataID, guidanceAccessor.typeName)
	if err != nil {
		return nil, fmt.Errorf("building guidance pool: %w", err)
	}

	ep := &EffectivePolicy{Policy: policy}
	catOpts := ResolveCatalogOpts{StrictExtends: opts.StrictExtends}

	for _, imp := range policy.Imports.Catalogs {
		metaID, ok := refIndex[imp.ReferenceId]
		if !ok {
			ep.UnresolvedCatalogs = append(ep.UnresolvedCatalogs, imp.ReferenceId)
			continue
		}
		cat, ok := catalogsByID[metaID]
		if !ok {
			ep.UnresolvedCatalogs = append(ep.UnresolvedCatalogs, imp.ReferenceId)
			continue
		}

		resolved, err := ResolveControlCatalogWithOpts(cat, catalogs, catOpts)
		if err != nil {
			return nil, fmt.Errorf("resolving catalog %q: %w", imp.ReferenceId, err)
		}

		ec := ApplyCatalogOverlays(*resolved, imp)
		ep.ControlCatalogs = append(ep.ControlCatalogs, *ec)
	}

	for _, imp := range policy.Imports.Guidance {
		metaID, ok := refIndex[imp.ReferenceId]
		if !ok {
			ep.UnresolvedGuidance = append(ep.UnresolvedGuidance, imp.ReferenceId)
			continue
		}
		gc, ok := guidanceByID[metaID]
		if !ok {
			ep.UnresolvedGuidance = append(ep.UnresolvedGuidance, imp.ReferenceId)
			continue
		}

		resolved, err := ResolveGuidanceCatalogWithOpts(gc, guidance, catOpts)
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

	if opts.StrictImports {
		if len(ep.UnresolvedCatalogs) > 0 {
			return nil, fmt.Errorf("strict imports: unresolved catalog reference-ids: %v", ep.UnresolvedCatalogs)
		}
		if len(ep.UnresolvedGuidance) > 0 {
			return nil, fmt.Errorf("strict imports: unresolved guidance reference-ids: %v", ep.UnresolvedGuidance)
		}
	}

	return ep, nil
}

// checkDuplicateImportRefs returns an error if any reference-id appears
// more than once within the catalog or guidance import lists.
func checkDuplicateImportRefs(imports Imports) error {
	seen := make(map[string]bool, len(imports.Catalogs)+len(imports.Guidance))
	for _, imp := range imports.Catalogs {
		if seen[imp.ReferenceId] {
			return fmt.Errorf("duplicate catalog import reference-id: %s", imp.ReferenceId)
		}
		seen[imp.ReferenceId] = true
	}
	for _, imp := range imports.Guidance {
		if seen[imp.ReferenceId] {
			return fmt.Errorf("duplicate guidance import reference-id: %s", imp.ReferenceId)
		}
		seen[imp.ReferenceId] = true
	}
	return nil
}

// buildRefIndex builds a lookup from mapping-reference ID to the artifact
// metadata ID it resolves to. In the current Gemara specification these
// are always equal (identity mapping). If the spec introduces indirection
// (e.g., reference-id != metadata.id) this function is the single place
// to change.
func buildRefIndex(refs []MappingReference) (map[string]string, error) {
	idx := make(map[string]string, len(refs))
	for _, ref := range refs {
		if ref.Id == "" {
			continue
		}
		if _, exists := idx[ref.Id]; exists {
			return nil, fmt.Errorf("duplicate mapping-reference id: %s", ref.Id)
		}
		idx[ref.Id] = ref.Id
	}
	return idx, nil
}
