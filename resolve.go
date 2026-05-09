// SPDX-License-Identifier: Apache-2.0

package gemara

import "fmt"

// ResolvableCatalog is the type constraint for artifact types that support
// extends-based resolution.
type ResolvableCatalog interface {
	ControlCatalog | GuidanceCatalog
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
	entries := acc.deepCopy(acc.entries(primary))

	exts := acc.extends(primary)
	if len(exts) == 0 {
		return entries
	}

	seen := map[string]bool{acc.metadataID(primary): true}
	entries = append(entries, acc.walkExtends(exts, pool, seen)...)
	return entries
}

// walkExtends recursively collects entries from extended catalogs in
// declaration order, skipping already-visited IDs to break cycles.
func (acc catalogAccessor[C, E]) walkExtends(extends []ArtifactMapping, pool map[string]C, seen map[string]bool) []E {
	var result []E
	for _, ext := range extends {
		if ext.ReferenceId == "" || seen[ext.ReferenceId] {
			continue
		}
		seen[ext.ReferenceId] = true

		cat, ok := pool[ext.ReferenceId]
		if !ok {
			continue
		}
		result = append(result, acc.deepCopy(acc.entries(cat))...)

		if exts := acc.extends(cat); len(exts) > 0 {
			result = append(result, acc.walkExtends(exts, pool, seen)...)
		}
	}
	return result
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
