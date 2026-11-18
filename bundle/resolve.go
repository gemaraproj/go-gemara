// SPDX-License-Identifier: Apache-2.0

package bundle

import (
	"context"
	"fmt"
	"strings"

	"github.com/gemaraproj/go-gemara"
	"github.com/gemaraproj/go-gemara/internal/codec"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry/remote"
)

// ResolveOption configures artifact resolution behaviour.
type ResolveOption func(*resolveOptions)

type resolveOptions struct {
	catalogID string
	target    oras.ReadOnlyTarget
	fetcher   gemara.Fetcher
}

// WithCatalogID specifies which imported catalog to resolve when resolving
// an EffectiveCatalog from a Policy containing multiple catalog imports.
func WithCatalogID(id string) ResolveOption {
	return func(o *resolveOptions) {
		o.catalogID = id
	}
}

// WithResolveTarget configures the OCI target used for resolving referenced artifacts.
func WithResolveTarget(target oras.ReadOnlyTarget) ResolveOption {
	return func(o *resolveOptions) {
		o.target = target
	}
}

// WithResolveFetcher configures a Fetcher for resolving external artifact URLs.
func WithResolveFetcher(f gemara.Fetcher) ResolveOption {
	return func(o *resolveOptions) {
		o.fetcher = f
	}
}

// ResolverOption configures a Resolver instance.
type ResolverOption func(*Resolver)

// WithTarget sets the default OCI target for the Resolver.
func WithTarget(target oras.ReadOnlyTarget) ResolverOption {
	return func(r *Resolver) {
		r.target = target
	}
}

// WithFetcher sets the fallback Fetcher for the Resolver.
func WithFetcher(f gemara.Fetcher) ResolverOption {
	return func(r *Resolver) {
		r.fetcher = f
	}
}

// Resolver resolves Gemara bundles and artifacts into effective types,
// processing extends hierarchies and policy imports per the Gemara specification.
type Resolver struct {
	target  oras.ReadOnlyTarget
	fetcher gemara.Fetcher
}

// NewResolver creates a new Resolver with the supplied options.
func NewResolver(opts ...ResolverOption) *Resolver {
	r := &Resolver{}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// ResolveCatalog resolves an OCI reference into an EffectiveCatalog.
func (r *Resolver) ResolveCatalog(ctx context.Context, ref string, opts ...ResolveOption) (*EffectiveCatalog, error) {
	b, err := r.ResolveBundle(ctx, ref)
	if err != nil {
		return nil, err
	}
	return b.ResolveCatalog(opts...)
}

// ResolvePolicy resolves an OCI reference into an EffectivePolicy.
func (r *Resolver) ResolvePolicy(ctx context.Context, ref string, opts ...ResolveOption) (*EffectivePolicy, error) {
	b, err := r.ResolveBundle(ctx, ref)
	if err != nil {
		return nil, err
	}
	return b.ResolvePolicy(opts...)
}

// ResolveGuidance resolves an OCI reference into an EffectiveGuidanceCatalog.
func (r *Resolver) ResolveGuidance(ctx context.Context, ref string, opts ...ResolveOption) (*EffectiveGuidanceCatalog, error) {
	b, err := r.ResolveBundle(ctx, ref)
	if err != nil {
		return nil, err
	}
	return b.ResolveGuidance(opts...)
}

// ResolveBundle unpacks an OCI reference into a Bundle using the Resolver's configured target.
func (r *Resolver) ResolveBundle(ctx context.Context, ref string) (*Bundle, error) {
	target := r.target
	if target == nil {
		repo, err := remote.NewRepository(ref)
		if err != nil {
			return nil, fmt.Errorf("initializing remote repository for %q: %w", ref, err)
		}
		target = repo
	}
	return Unpack(ctx, target, ref)
}

// ResolveCatalog resolves an OCI reference from target into an EffectiveCatalog.
func ResolveCatalog(ctx context.Context, target oras.ReadOnlyTarget, ref string, opts ...ResolveOption) (*EffectiveCatalog, error) {
	var ropts resolveOptions
	for _, o := range opts {
		o(&ropts)
	}
	if target == nil && ropts.target != nil {
		target = ropts.target
	}
	if target == nil {
		repo, err := remote.NewRepository(ref)
		if err != nil {
			return nil, fmt.Errorf("initializing remote repository for %q: %w", ref, err)
		}
		target = repo
	}

	b, err := Unpack(ctx, target, ref)
	if err != nil {
		return nil, err
	}
	return b.ResolveCatalog(opts...)
}

// ResolvePolicy resolves an OCI reference from target into an EffectivePolicy.
func ResolvePolicy(ctx context.Context, target oras.ReadOnlyTarget, ref string, opts ...ResolveOption) (*EffectivePolicy, error) {
	var ropts resolveOptions
	for _, o := range opts {
		o(&ropts)
	}
	if target == nil && ropts.target != nil {
		target = ropts.target
	}
	if target == nil {
		repo, err := remote.NewRepository(ref)
		if err != nil {
			return nil, fmt.Errorf("initializing remote repository for %q: %w", ref, err)
		}
		target = repo
	}

	b, err := Unpack(ctx, target, ref)
	if err != nil {
		return nil, err
	}
	return b.ResolvePolicy(opts...)
}

// ResolveGuidance resolves an OCI reference from target into an EffectiveGuidanceCatalog.
func ResolveGuidance(ctx context.Context, target oras.ReadOnlyTarget, ref string, opts ...ResolveOption) (*EffectiveGuidanceCatalog, error) {
	var ropts resolveOptions
	for _, o := range opts {
		o(&ropts)
	}
	if target == nil && ropts.target != nil {
		target = ropts.target
	}
	if target == nil {
		repo, err := remote.NewRepository(ref)
		if err != nil {
			return nil, fmt.Errorf("initializing remote repository for %q: %w", ref, err)
		}
		target = repo
	}

	b, err := Unpack(ctx, target, ref)
	if err != nil {
		return nil, err
	}
	return b.ResolveGuidance(opts...)
}

// ResolveCatalogFromBundle resolves an in-memory Bundle into an EffectiveCatalog.
func ResolveCatalogFromBundle(b *Bundle, opts ...ResolveOption) (*EffectiveCatalog, error) {
	return b.ResolveCatalog(opts...)
}

// ResolvePolicyFromBundle resolves an in-memory Bundle into an EffectivePolicy.
func ResolvePolicyFromBundle(b *Bundle, opts ...ResolveOption) (*EffectivePolicy, error) {
	return b.ResolvePolicy(opts...)
}

// ResolveGuidanceFromBundle resolves an in-memory Bundle into an EffectiveGuidanceCatalog.
func ResolveGuidanceFromBundle(b *Bundle, opts ...ResolveOption) (*EffectiveGuidanceCatalog, error) {
	return b.ResolveGuidance(opts...)
}

// ResolveCatalog resolves the bundle into an EffectiveCatalog.
func (b *Bundle) ResolveCatalog(opts ...ResolveOption) (*EffectiveCatalog, error) {
	if b == nil {
		return nil, fmt.Errorf("bundle is nil")
	}

	var ropts resolveOptions
	for _, o := range opts {
		o(&ropts)
	}

	idx, err := newBundleIndex(b, ropts.fetcher)
	if err != nil {
		return nil, err
	}

	sourcePF, err := parseFile(b.Source)
	if err != nil {
		return nil, fmt.Errorf("parsing bundle source: %w", err)
	}

	switch sourcePF.artType {
	case gemara.PolicyArtifact:
		effPol, err := resolvePolicyFile(sourcePF, idx, nil, ropts)
		if err != nil {
			return nil, err
		}

		if len(effPol.ResolvedCatalogs) == 0 {
			return nil, fmt.Errorf("policy %q does not import any control catalogs", sourcePF.id)
		}

		if ropts.catalogID != "" {
			for _, cat := range effPol.ResolvedCatalogs {
				if cat.Metadata.Id == ropts.catalogID {
					return cat, nil
				}
			}
			return nil, fmt.Errorf("catalog %q not found in policy imports", ropts.catalogID)
		}

		if len(effPol.ResolvedCatalogs) == 1 {
			return effPol.ResolvedCatalogs[0], nil
		}

		return nil, fmt.Errorf("policy imports %d catalogs; use WithCatalogID to select which one to resolve",
			len(effPol.ResolvedCatalogs))

	default:
		var cat gemara.ControlCatalog
		if err := codec.UnmarshalYAML(b.Source.Data, &cat); err != nil {
			return nil, fmt.Errorf("decoding control catalog: %w", err)
		}
		return resolveCatalogExtends(&cat, idx, []string{cat.Metadata.Id})
	}
}

// ResolvePolicy resolves the bundle into an EffectivePolicy with all imported catalogs resolved.
func (b *Bundle) ResolvePolicy(opts ...ResolveOption) (*EffectivePolicy, error) {
	if b == nil {
		return nil, fmt.Errorf("bundle is nil")
	}

	var ropts resolveOptions
	for _, o := range opts {
		o(&ropts)
	}

	idx, err := newBundleIndex(b, ropts.fetcher)
	if err != nil {
		return nil, err
	}

	sourcePF, err := parseFile(b.Source)
	if err != nil {
		return nil, fmt.Errorf("parsing bundle source: %w", err)
	}

	if sourcePF.artType != gemara.PolicyArtifact {
		return nil, fmt.Errorf("bundle source %q is not a Policy (type: %s)", b.Source.Name, sourcePF.artType)
	}

	return resolvePolicyFile(sourcePF, idx, []string{sourcePF.id}, ropts)
}

// ResolveGuidance resolves the bundle into an EffectiveGuidanceCatalog.
func (b *Bundle) ResolveGuidance(opts ...ResolveOption) (*EffectiveGuidanceCatalog, error) {
	if b == nil {
		return nil, fmt.Errorf("bundle is nil")
	}

	var ropts resolveOptions
	for _, o := range opts {
		o(&ropts)
	}

	idx, err := newBundleIndex(b, ropts.fetcher)
	if err != nil {
		return nil, err
	}

	sourcePF, err := parseFile(b.Source)
	if err != nil {
		return nil, fmt.Errorf("parsing bundle source: %w", err)
	}

	switch sourcePF.artType {
	case gemara.PolicyArtifact:
		effPol, err := resolvePolicyFile(sourcePF, idx, nil, ropts)
		if err != nil {
			return nil, err
		}
		if len(effPol.ResolvedGuidance) == 0 {
			return nil, fmt.Errorf("policy %q does not import any guidance catalogs", sourcePF.id)
		}
		if ropts.catalogID != "" {
			for _, g := range effPol.ResolvedGuidance {
				if g.Metadata.Id == ropts.catalogID {
					return g, nil
				}
			}
			return nil, fmt.Errorf("guidance %q not found in policy imports", ropts.catalogID)
		}
		if len(effPol.ResolvedGuidance) == 1 {
			return effPol.ResolvedGuidance[0], nil
		}
		return nil, fmt.Errorf("policy imports %d guidance catalogs; use WithCatalogID to select which one to resolve",
			len(effPol.ResolvedGuidance))
	default:
		var gCat gemara.GuidanceCatalog
		if err := codec.UnmarshalYAML(b.Source.Data, &gCat); err != nil {
			return nil, fmt.Errorf("decoding guidance catalog: %w", err)
		}
		return resolveGuidanceExtends(&gCat, idx, []string{gCat.Metadata.Id})
	}
}

// bundleIndex indexes bundle files by metadata.id and filename, with fallback fetching.
type bundleIndex struct {
	byID   map[string]*parsedFile
	byName map[string]*parsedFile
	fetch  gemara.Fetcher
}

func newBundleIndex(b *Bundle, fetcher gemara.Fetcher) (*bundleIndex, error) {
	idx := &bundleIndex{
		byID:   make(map[string]*parsedFile),
		byName: make(map[string]*parsedFile),
		fetch:  fetcher,
	}

	allFiles := append([]File{b.Source}, b.Imports...)
	for _, f := range allFiles {
		if len(f.Data) == 0 {
			continue
		}
		pf, err := parseFile(f)
		if err != nil {
			return nil, fmt.Errorf("indexing %s: %w", f.Name, err)
		}
		if pf.id != "" {
			idx.byID[pf.id] = pf
		}
		if pf.Name != "" {
			idx.byName[pf.Name] = pf
		}
	}

	return idx, nil
}

func (idx *bundleIndex) find(refID string, mappingRefs []gemara.MappingReference) (*parsedFile, error) {
	if pf, ok := idx.byID[refID]; ok {
		return pf, nil
	}

	if pf, ok := idx.byName[refID]; ok {
		return pf, nil
	}

	for _, mr := range mappingRefs {
		if mr.Id == refID {
			if mr.Url != "" {
				name := importFileName(mr.Id, mr.Url)
				if pf, ok := idx.byName[name]; ok {
					return pf, nil
				}
				if idx.fetch != nil {
					rc, err := idx.fetch.Fetch(context.Background(), mr.Url)
					if err == nil {
						defer rc.Close() //nolint:errcheck
						data, err := ioReadAll(rc)
						if err == nil {
							pf, err := parseFile(File{Name: name, Data: data})
							if err == nil {
								idx.byID[pf.id] = pf
								idx.byName[pf.Name] = pf
								return pf, nil
							}
						}
					}
				}
			}
			break
		}
	}

	return nil, fmt.Errorf("artifact %q not found in bundle", refID)
}

func ioReadAll(r ioReader) ([]byte, error) {
	var buf []byte
	b := make([]byte, 1024)
	for {
		n, err := r.Read(b)
		if n > 0 {
			buf = append(buf, b[:n]...)
		}
		if err != nil {
			if err.Error() == "EOF" {
				return buf, nil
			}
			return buf, err
		}
	}
}

type ioReader interface {
	Read(p []byte) (n int, err error)
}

// resolveCatalogExtends recursively resolves extends inheritance for a ControlCatalog.
func resolveCatalogExtends(cat *gemara.ControlCatalog, idx *bundleIndex, visited []string) (*EffectiveCatalog, error) {
	if len(cat.Extends) == 0 {
		return &EffectiveCatalog{
			ControlCatalog: *cat,
		}, nil
	}

	current := &EffectiveCatalog{
		ControlCatalog: *cat,
	}

	for _, ext := range cat.Extends {
		if ext.ReferenceId == "" {
			continue
		}

		for _, v := range visited {
			if v == ext.ReferenceId {
				return nil, fmt.Errorf("cyclic dependency detected in catalog extends: %s -> %s",
					strings.Join(visited, " -> "), ext.ReferenceId)
			}
		}

		parentPF, err := idx.find(ext.ReferenceId, cat.Metadata.MappingReferences)
		if err != nil {
			return nil, fmt.Errorf("resolving extends %q: %w", ext.ReferenceId, err)
		}

		var parentCat gemara.ControlCatalog
		if err := codec.UnmarshalYAML(parentPF.Data, &parentCat); err != nil {
			return nil, fmt.Errorf("decoding parent catalog %q: %w", parentPF.id, err)
		}

		nextVisited := append([]string(nil), visited...)
		nextVisited = append(nextVisited, parentCat.Metadata.Id)

		resolvedParent, err := resolveCatalogExtends(&parentCat, idx, nextVisited)
		if err != nil {
			return nil, err
		}

		current = mergeCatalogs(resolvedParent, current)
	}

	current.Extends = nil
	return current, nil
}

func mergeCatalogs(base *EffectiveCatalog, child *EffectiveCatalog) *EffectiveCatalog {
	merged := &EffectiveCatalog{
		ControlCatalog: child.ControlCatalog,
	}

	if merged.Title == "" {
		merged.Title = base.Title
	}

	groupMap := make(map[string]gemara.Group)
	var groupOrder []string
	for _, g := range base.Groups {
		groupMap[g.Id] = g
		groupOrder = append(groupOrder, g.Id)
	}
	for _, g := range child.Groups {
		if _, exists := groupMap[g.Id]; !exists {
			groupOrder = append(groupOrder, g.Id)
		}
		groupMap[g.Id] = g
	}

	merged.Groups = make([]gemara.Group, 0, len(groupOrder))
	for _, gid := range groupOrder {
		merged.Groups = append(merged.Groups, groupMap[gid])
	}

	ctrlMap := make(map[string]gemara.Control)
	var ctrlOrder []string
	for _, c := range base.Controls {
		ctrlMap[c.Id] = c
		ctrlOrder = append(ctrlOrder, c.Id)
	}
	for _, c := range child.Controls {
		if _, exists := ctrlMap[c.Id]; !exists {
			ctrlOrder = append(ctrlOrder, c.Id)
		}
		ctrlMap[c.Id] = c
	}

	merged.Controls = make([]gemara.Control, 0, len(ctrlOrder))
	for _, cid := range ctrlOrder {
		merged.Controls = append(merged.Controls, ctrlMap[cid])
	}

	agMap := make(map[string]gemara.Group)
	var agOrder []string
	for _, ag := range base.Metadata.ApplicabilityGroups {
		agMap[ag.Id] = ag
		agOrder = append(agOrder, ag.Id)
	}
	for _, ag := range child.Metadata.ApplicabilityGroups {
		if _, exists := agMap[ag.Id]; !exists {
			agOrder = append(agOrder, ag.Id)
		}
		agMap[ag.Id] = ag
	}
	merged.Metadata.ApplicabilityGroups = make([]gemara.Group, 0, len(agOrder))
	for _, id := range agOrder {
		merged.Metadata.ApplicabilityGroups = append(merged.Metadata.ApplicabilityGroups, agMap[id])
	}

	return merged
}

func resolveGuidanceExtends(cat *gemara.GuidanceCatalog, idx *bundleIndex, visited []string) (*EffectiveGuidanceCatalog, error) {
	if len(cat.Extends) == 0 {
		return &EffectiveGuidanceCatalog{
			GuidanceCatalog: *cat,
		}, nil
	}

	current := &EffectiveGuidanceCatalog{
		GuidanceCatalog: *cat,
	}

	for _, ext := range cat.Extends {
		if ext.ReferenceId == "" {
			continue
		}

		for _, v := range visited {
			if v == ext.ReferenceId {
				return nil, fmt.Errorf("cyclic dependency detected in guidance extends: %s -> %s",
					strings.Join(visited, " -> "), ext.ReferenceId)
			}
		}

		parentPF, err := idx.find(ext.ReferenceId, cat.Metadata.MappingReferences)
		if err != nil {
			return nil, fmt.Errorf("resolving extends %q: %w", ext.ReferenceId, err)
		}

		var parentCat gemara.GuidanceCatalog
		if err := codec.UnmarshalYAML(parentPF.Data, &parentCat); err != nil {
			return nil, fmt.Errorf("decoding parent guidance %q: %w", parentPF.id, err)
		}

		nextVisited := append([]string(nil), visited...)
		nextVisited = append(nextVisited, parentCat.Metadata.Id)

		resolvedParent, err := resolveGuidanceExtends(&parentCat, idx, nextVisited)
		if err != nil {
			return nil, err
		}

		current = mergeGuidanceCatalogs(resolvedParent, current)
	}

	current.Extends = nil
	return current, nil
}

func mergeGuidanceCatalogs(base *EffectiveGuidanceCatalog, child *EffectiveGuidanceCatalog) *EffectiveGuidanceCatalog {
	merged := &EffectiveGuidanceCatalog{
		GuidanceCatalog: child.GuidanceCatalog,
	}

	if merged.Title == "" {
		merged.Title = base.Title
	}

	groupMap := make(map[string]gemara.Group)
	var groupOrder []string
	for _, g := range base.Groups {
		groupMap[g.Id] = g
		groupOrder = append(groupOrder, g.Id)
	}
	for _, g := range child.Groups {
		if _, exists := groupMap[g.Id]; !exists {
			groupOrder = append(groupOrder, g.Id)
		}
		groupMap[g.Id] = g
	}
	merged.Groups = make([]gemara.Group, 0, len(groupOrder))
	for _, gid := range groupOrder {
		merged.Groups = append(merged.Groups, groupMap[gid])
	}

	glMap := make(map[string]gemara.Guideline)
	var glOrder []string
	for _, gl := range base.Guidelines {
		glMap[gl.Id] = gl
		glOrder = append(glOrder, gl.Id)
	}
	for _, gl := range child.Guidelines {
		if _, exists := glMap[gl.Id]; !exists {
			glOrder = append(glOrder, gl.Id)
		}
		glMap[gl.Id] = gl
	}
	merged.Guidelines = make([]gemara.Guideline, 0, len(glOrder))
	for _, glid := range glOrder {
		merged.Guidelines = append(merged.Guidelines, glMap[glid])
	}

	return merged
}

func resolvePolicyFile(pf *parsedFile, idx *bundleIndex, visited []string, ropts resolveOptions) (*EffectivePolicy, error) {
	var pol gemara.Policy
	if err := codec.UnmarshalYAML(pf.Data, &pol); err != nil {
		return nil, fmt.Errorf("decoding policy %q: %w", pf.id, err)
	}

	effPolicy := &EffectivePolicy{
		Policy: pol,
	}

	for _, impPolRef := range pol.Imports.Policies {
		if impPolRef.ReferenceId == "" {
			continue
		}

		for _, v := range visited {
			if v == impPolRef.ReferenceId {
				return nil, fmt.Errorf("cyclic dependency detected in policy imports: %s -> %s",
					strings.Join(visited, " -> "), impPolRef.ReferenceId)
			}
		}

		impPF, err := idx.find(impPolRef.ReferenceId, pol.Metadata.MappingReferences)
		if err != nil {
			return nil, fmt.Errorf("resolving imported policy %q: %w", impPolRef.ReferenceId, err)
		}

		nextVisited := append([]string(nil), visited...)
		nextVisited = append(nextVisited, impPF.id)

		nestedEff, err := resolvePolicyFile(impPF, idx, nextVisited, ropts)
		if err != nil {
			return nil, err
		}

		effPolicy.ResolvedCatalogs = append(effPolicy.ResolvedCatalogs, nestedEff.ResolvedCatalogs...)
		effPolicy.ResolvedGuidance = append(effPolicy.ResolvedGuidance, nestedEff.ResolvedGuidance...)
	}

	for _, catImport := range pol.Imports.Catalogs {
		if catImport.ReferenceId == "" {
			continue
		}

		catPF, err := idx.find(catImport.ReferenceId, pol.Metadata.MappingReferences)
		if err != nil {
			return nil, fmt.Errorf("locating imported catalog %q: %w", catImport.ReferenceId, err)
		}

		var rawCat gemara.ControlCatalog
		if err := codec.UnmarshalYAML(catPF.Data, &rawCat); err != nil {
			return nil, fmt.Errorf("decoding imported catalog %q: %w", catPF.id, err)
		}

		baseEffCat, err := resolveCatalogExtends(&rawCat, idx, []string{rawCat.Metadata.Id})
		if err != nil {
			return nil, fmt.Errorf("resolving extends for imported catalog %q: %w", rawCat.Metadata.Id, err)
		}

		effCat, err := gemara.ResolveRequirements(&pol, &baseEffCat.ControlCatalog, catImport.ReferenceId)
		if err != nil {
			return nil, fmt.Errorf("resolving requirements for catalog %q: %w", catImport.ReferenceId, err)
		}

		effPolicy.ResolvedCatalogs = append(effPolicy.ResolvedCatalogs, effCat)
	}

	for _, gImport := range pol.Imports.Guidance {
		if gImport.ReferenceId == "" {
			continue
		}

		gPF, err := idx.find(gImport.ReferenceId, pol.Metadata.MappingReferences)
		if err != nil {
			return nil, fmt.Errorf("locating imported guidance %q: %w", gImport.ReferenceId, err)
		}

		var rawG gemara.GuidanceCatalog
		if err := codec.UnmarshalYAML(gPF.Data, &rawG); err != nil {
			return nil, fmt.Errorf("decoding imported guidance %q: %w", gPF.id, err)
		}

		baseEffG, err := resolveGuidanceExtends(&rawG, idx, []string{rawG.Metadata.Id})
		if err != nil {
			return nil, fmt.Errorf("resolving extends for imported guidance %q: %w", rawG.Metadata.Id, err)
		}

		effG, err := gemara.ResolveGuidanceRequirements(&pol, &baseEffG.GuidanceCatalog, gImport.ReferenceId)
		if err != nil {
			return nil, fmt.Errorf("resolving requirements for guidance %q: %w", gImport.ReferenceId, err)
		}

		effPolicy.ResolvedGuidance = append(effPolicy.ResolvedGuidance, effG)
	}

	return effPolicy, nil
}
