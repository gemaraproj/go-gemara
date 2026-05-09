package gemara

import (
	"fmt"
	"sync"

	"github.com/gemaraproj/go-gemara/internal/codec"
)

// SGuidanceCatalog wraps a GuidanceCatalog with pre-built indexes for
// efficient group and guideline lookups.
type SGuidanceCatalog struct {
	GuidanceCatalog

	groupsOnce  sync.Once
	groupsCache []string

	guidelinesByGroupOnce  sync.Once
	guidelinesByGroupCache map[string][]Guideline
}

// Sugar wraps this GuidanceCatalog in a SGuidanceCatalog for convenient
// cached helper access. Cached results are computed once on first access
// and never invalidated, so the wrapper should not be reused after the
// underlying data has changed. Call Sugar again or use FromBase to reset
// caches.
func (g *GuidanceCatalog) Sugar() *SGuidanceCatalog {
	return &SGuidanceCatalog{GuidanceCatalog: *g}
}

func (g *SGuidanceCatalog) ToBase() GuidanceCatalog {
	return g.GuidanceCatalog
}

func (g *SGuidanceCatalog) FromBase(s *GuidanceCatalog) {
	g.GuidanceCatalog = *s
	g.groupsOnce = sync.Once{}
	g.groupsCache = nil
	g.guidelinesByGroupOnce = sync.Once{}
	g.guidelinesByGroupCache = nil
}

func (g *SGuidanceCatalog) MarshalYAML() ([]byte, error) {
	return codec.MarshalBaseYAML[GuidanceCatalog](g)
}

func (g *SGuidanceCatalog) UnmarshalYAML(data []byte) error {
	return codec.UnmarshalBaseYAML[GuidanceCatalog](data, g)
}

// GetGroupNames returns all group titles from the catalog.
func (g *SGuidanceCatalog) GetGroupNames() []string {
	g.groupsOnce.Do(func() {
		for _, group := range g.Groups {
			g.groupsCache = append(g.groupsCache, group.Title)
		}
	})
	return g.groupsCache
}

// GetGuidelinesForGroup returns all guidelines belonging to the given group ID.
func (g *SGuidanceCatalog) GetGuidelinesForGroup(group string) []Guideline {
	g.guidelinesByGroupOnce.Do(func() {
		g.guidelinesByGroupCache = make(map[string][]Guideline)
		for _, gl := range g.Guidelines {
			g.guidelinesByGroupCache[gl.Group] = append(
				g.guidelinesByGroupCache[gl.Group], gl,
			)
		}
	})
	return g.guidelinesByGroupCache[group]
}

var guidanceAccessor = catalogAccessor[GuidanceCatalog, Guideline]{
	typeName:   "guidance catalog",
	metadataID: func(g GuidanceCatalog) string { return g.Metadata.Id },
	extends:    func(g GuidanceCatalog) []ArtifactMapping { return g.Extends },
	entries:    func(g GuidanceCatalog) []Guideline { return g.Guidelines },
	entryID:    func(e Guideline) string { return e.Id },
	deepCopy:   deepCopyGuidelines,
}

// Resolve flattens the guidance catalog's transitive extends chain against
// the pool, returning a new SGuidanceCatalog with fresh caches. Delegates
// to ResolveGuidanceCatalog.
func (g *SGuidanceCatalog) Resolve(pool []GuidanceCatalog) (*SGuidanceCatalog, error) {
	resolved, err := ResolveGuidanceCatalog(g.GuidanceCatalog, pool)
	if err != nil {
		return nil, err
	}
	return resolved.Sugar(), nil
}

// ResolveGuidanceCatalog flattens a guidance catalog's transitive extends
// chain against the pool, returning a new GuidanceCatalog with all inherited
// guidelines merged.
func ResolveGuidanceCatalog(primary GuidanceCatalog, pool []GuidanceCatalog) (*GuidanceCatalog, error) {
	if primary.Metadata.Id == "" {
		return nil, fmt.Errorf("primary guidance catalog has no metadata.id")
	}

	poolByID, err := poolIndex(pool, guidanceAccessor.metadataID, guidanceAccessor.typeName)
	if err != nil {
		return nil, fmt.Errorf("building guidance pool: %w", err)
	}

	resolved := GuidanceCatalog{
		Title:        primary.Title,
		Metadata:     deepCopyMetadata(primary.Metadata),
		Groups:       deepCopyGroups(primary.Groups),
		GuidanceType: primary.GuidanceType,
		FrontMatter:  primary.FrontMatter,
		Exemptions:   deepCopyExemptions(primary.Exemptions),
		Guidelines:   flattenEntries(primary, poolByID, guidanceAccessor),
	}
	return &resolved, nil
}

// ApplyGuidanceOverlays applies a policy's import-level exclusions to a
// resolved guidance catalog, returning a new GuidanceCatalog with overlays
// applied.
func ApplyGuidanceOverlays(catalog GuidanceCatalog, imp GuidanceImport) *GuidanceCatalog {
	guidelines := deepCopyGuidelines(catalog.Guidelines)
	guidelines = applyEntryExclusions(guidelines, imp.Exclusions, guidanceAccessor.entryID)

	result := catalog
	result.Guidelines = guidelines
	result.Metadata = deepCopyMetadata(catalog.Metadata)
	result.Groups = deepCopyGroups(catalog.Groups)
	result.Exemptions = deepCopyExemptions(catalog.Exemptions)
	return &result
}

// deepCopyGuidelines returns a new slice of guidelines with all nested
// slices and pointer fields deep-copied to prevent aliasing with the source.
func deepCopyGuidelines(src []Guideline) []Guideline {
	if src == nil {
		return nil
	}
	dst := make([]Guideline, len(src))
	copy(dst, src)
	for i, g := range dst {
		dst[i].Recommendations = copyStrings(g.Recommendations)
		dst[i].Applicability = copyStrings(g.Applicability)
		dst[i].Statements = copyStatements(g.Statements)
		dst[i].Principles = copyMultiEntryMappings(g.Principles)
		dst[i].Vectors = copyMultiEntryMappings(g.Vectors)
		dst[i].SeeAlso = copyStrings(g.SeeAlso)
		dst[i].Extends = cloneEntryMapping(g.Extends)
		dst[i].ReplacedBy = cloneEntryMapping(g.ReplacedBy)
		dst[i].Rationale = cloneRationale(g.Rationale)
	}
	return dst
}

func copyStatements(src []Statement) []Statement {
	if src == nil {
		return nil
	}
	dst := make([]Statement, len(src))
	for i, s := range src {
		dst[i] = s
		dst[i].Recommendations = copyStrings(s.Recommendations)
	}
	return dst
}

func cloneRationale(src *Rationale) *Rationale {
	if src == nil {
		return nil
	}
	cp := *src
	cp.Goals = copyStrings(src.Goals)
	return &cp
}

func deepCopyExemptions(src []Exemption) []Exemption {
	if src == nil {
		return nil
	}
	dst := make([]Exemption, len(src))
	copy(dst, src)
	for i, e := range dst {
		if e.Redirect != nil {
			cp := *e.Redirect
			cp.Entries = make([]ArtifactMapping, len(e.Redirect.Entries))
			copy(cp.Entries, e.Redirect.Entries)
			dst[i].Redirect = &cp
		}
	}
	return dst
}
