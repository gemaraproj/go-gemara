// SPDX-License-Identifier: Apache-2.0

package gemara

import (
	"fmt"
	"strings"
)

// ResolvableCatalog is the type constraint for artifact types that support
// extends-based resolution.
type ResolvableCatalog interface {
	ControlCatalog | GuidanceCatalog
}

// ResolveCatalogOpts configures extends flattening when resolving a catalog
// against a pool.
type ResolveCatalogOpts struct {
	// StrictExtends causes resolution to fail when any extends reference-id is
	// missing from the pool. When false, missing targets are skipped (legacy).
	StrictExtends bool
}

// catalogAccessor provides field-level access to a catalog's entries so the
// generic resolution pipeline can operate without knowing concrete types.
type catalogAccessor[C ResolvableCatalog, E any] struct {
	typeName   string
	metadataID func(C) string
	extends    func(C) []ArtifactMapping
	entries    func(C) []E
	entryID    func(E) string
	deepCopy   func([]E) []E
}

// poolIndex builds an ID-keyed index of catalogs, returning an error if
// duplicate metadata.id values are found. Entries with empty IDs are skipped.
// The typeName is used in error messages to identify the catalog kind.
func poolIndex[C ResolvableCatalog](pool []C, id func(C) string, typeName string) (map[string]C, error) {
	idx := make(map[string]C, len(pool))
	for _, c := range pool {
		cid := id(c)
		if cid == "" {
			continue
		}
		if _, exists := idx[cid]; exists {
			return nil, fmt.Errorf("duplicate %s metadata.id: %s", typeName, cid)
		}
		idx[cid] = c
	}
	return idx, nil
}

// flattenEntries merges entries from the primary catalog with entries from
// all catalogs in its transitive extends chain.
func flattenEntries[C ResolvableCatalog, E any](primary C, pool map[string]C, acc catalogAccessor[C, E]) []E {
	entries, _ := flattenEntriesWithUnresolved(primary, pool, acc)
	return entries
}

// flattenEntriesWithUnresolved merges entries like flattenEntries and returns
// reference-ids from extends that were not found in the pool (non-empty ids only).
func flattenEntriesWithUnresolved[C ResolvableCatalog, E any](primary C, pool map[string]C, acc catalogAccessor[C, E]) ([]E, []string) {
	entries := acc.deepCopy(acc.entries(primary))

	exts := acc.extends(primary)
	if len(exts) == 0 {
		return entries, nil
	}

	seen := map[string]bool{acc.metadataID(primary): true}
	more, unresolved := acc.walkExtendsCollect(exts, pool, seen)
	entries = append(entries, more...)
	return entries, unresolved
}

// walkExtendsCollect recursively collects entries from extended catalogs in
// declaration order, skipping already-visited IDs to break cycles. Missing
// pool entries append ext.ReferenceId to the unresolved slice.
func (acc catalogAccessor[C, E]) walkExtendsCollect(extends []ArtifactMapping, pool map[string]C, seen map[string]bool) ([]E, []string) {
	var result []E
	var unresolved []string
	for _, ext := range extends {
		if ext.ReferenceId == "" || seen[ext.ReferenceId] {
			continue
		}
		seen[ext.ReferenceId] = true

		cat, ok := pool[ext.ReferenceId]
		if !ok {
			unresolved = append(unresolved, ext.ReferenceId)
			continue
		}
		result = append(result, acc.deepCopy(acc.entries(cat))...)

		if exts := acc.extends(cat); len(exts) > 0 {
			sub, miss := acc.walkExtendsCollect(exts, pool, seen)
			result = append(result, sub...)
			unresolved = append(unresolved, miss...)
		}
	}
	return result, unresolved
}

func formatUnresolvedExtends(kind string, catalogID string, missing []string) error {
	return fmt.Errorf("%s %q: unresolved extends reference-ids: %s", kind, catalogID, strings.Join(missing, ", "))
}

// applyEntryExclusions removes entries whose IDs appear in the exclusion list.
func applyEntryExclusions[E any](entries []E, exclusions []string, entryID func(E) string) []E {
	if len(exclusions) == 0 {
		return entries
	}
	excluded := toSet(exclusions)
	filtered := make([]E, 0, len(entries))
	for _, e := range entries {
		if !excluded[entryID(e)] {
			filtered = append(filtered, e)
		}
	}
	return filtered
}
