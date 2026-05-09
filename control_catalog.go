package gemara

import (
	"fmt"
	"sync"

	"github.com/gemaraproj/go-gemara/internal/codec"
)

// SControlCatalog wraps a ControlCatalog with pre-built indexes for
// efficient group, control, and requirement lookups.
type SControlCatalog struct {
	ControlCatalog

	groupsOnce  sync.Once
	groupsCache []string

	sugarControlsOnce  sync.Once
	sugarControlsCache []*SControl

	controlsByGroupOnce  sync.Once
	controlsByGroupCache map[string][]*SControl

	requirementsOnce  sync.Once
	requirementsCache map[string][]AssessmentRequirement
}

// Sugar wraps this ControlCatalog in a SControlCatalog for convenient
// cached helper access. Cached results are computed once on first access
// and never invalidated, so the wrapper should not be reused after the
// underlying data has changed. Call Sugar again or use FromBase to reset
// caches.
func (c *ControlCatalog) Sugar() *SControlCatalog {
	return &SControlCatalog{ControlCatalog: *c}
}

func (c *SControlCatalog) ToBase() ControlCatalog {
	return c.ControlCatalog
}

func (c *SControlCatalog) FromBase(s *ControlCatalog) {
	c.ControlCatalog = *s
	c.groupsOnce = sync.Once{}
	c.groupsCache = nil
	c.sugarControlsOnce = sync.Once{}
	c.sugarControlsCache = nil
	c.controlsByGroupOnce = sync.Once{}
	c.controlsByGroupCache = nil
	c.requirementsOnce = sync.Once{}
	c.requirementsCache = nil
}

func (c *SControlCatalog) MarshalYAML() ([]byte, error) {
	return codec.MarshalBaseYAML[ControlCatalog](c)
}

func (c *SControlCatalog) UnmarshalYAML(data []byte) error {
	return codec.UnmarshalBaseYAML[ControlCatalog](data, c)
}

// SControls returns all controls as cached SControl instances.
func (c *SControlCatalog) SControls() []*SControl {
	c.sugarControlsOnce.Do(func() {
		c.sugarControlsCache = make([]*SControl, len(c.Controls))
		for i := range c.Controls {
			c.sugarControlsCache[i] = c.Controls[i].Sugar()
		}
	})
	return c.sugarControlsCache
}

func (c *SControlCatalog) GetGroupNames() []string {
	c.groupsOnce.Do(func() {
		for _, group := range c.Groups {
			c.groupsCache = append(c.groupsCache, group.Title)
		}
	})
	return c.groupsCache
}

func (c *SControlCatalog) GetControlsForGroup(group string) []*SControl {
	c.controlsByGroupOnce.Do(func() {
		c.controlsByGroupCache = make(map[string][]*SControl)
		for _, sc := range c.SControls() {
			c.controlsByGroupCache[sc.Group] = append(
				c.controlsByGroupCache[sc.Group], sc,
			)
		}
	})
	return c.controlsByGroupCache[group]
}

func (c *SControlCatalog) GetRequirementForApplicability(applicability string) []AssessmentRequirement {
	c.requirementsOnce.Do(func() {
		c.requirementsCache = make(map[string][]AssessmentRequirement)
		for _, control := range c.Controls {
			for _, req := range control.AssessmentRequirements {
				for _, app := range req.Applicability {
					c.requirementsCache[app] = append(
						c.requirementsCache[app], req,
					)
				}
			}
		}
	})
	return c.requirementsCache[applicability]
}

var controlAccessor = catalogAccessor[ControlCatalog, Control]{
	typeName:   "catalog",
	metadataID: func(c ControlCatalog) string { return c.Metadata.Id },
	extends:    func(c ControlCatalog) []ArtifactMapping { return c.Extends },
	entries:    func(c ControlCatalog) []Control { return c.Controls },
	entryID:    func(e Control) string { return e.Id },
	deepCopy:   deepCopyControls,
}

// Resolve flattens the catalog's transitive extends chain against the
// pool, returning a new SControlCatalog with fresh caches. Delegates to
// ResolveControlCatalog.
func (c *SControlCatalog) Resolve(pool []ControlCatalog) (*SControlCatalog, error) {
	resolved, err := ResolveControlCatalog(c.ControlCatalog, pool)
	if err != nil {
		return nil, err
	}
	return resolved.Sugar(), nil
}

// ResolveControlCatalog flattens a catalog's transitive extends chain
// against the pool, returning a new ControlCatalog with all inherited
// controls merged. The result is a self-contained catalog with no
// unresolved extends.
func ResolveControlCatalog(primary ControlCatalog, pool []ControlCatalog) (*ControlCatalog, error) {
	if primary.Metadata.Id == "" {
		return nil, fmt.Errorf("primary catalog has no metadata.id")
	}

	poolByID, err := poolIndex(pool, controlAccessor.metadataID, controlAccessor.typeName)
	if err != nil {
		return nil, fmt.Errorf("building catalog pool: %w", err)
	}

	resolved := ControlCatalog{
		Title:    primary.Title,
		Metadata: deepCopyMetadata(primary.Metadata),
		Groups:   deepCopyGroups(primary.Groups),
		Controls: flattenEntries(primary, poolByID, controlAccessor),
	}
	return &resolved, nil
}

// ApplyCatalogOverlays applies a policy's import-level exclusions and
// assessment requirement modifications to a resolved catalog, returning
// a new ControlCatalog with overlays applied.
func ApplyCatalogOverlays(catalog ControlCatalog, imp CatalogImport) *ControlCatalog {
	controls := deepCopyControls(catalog.Controls)
	controls = applyEntryExclusions(controls, imp.Exclusions, controlAccessor.entryID)
	controls = applyARModifications(controls, imp.AssessmentRequirementModifications)

	result := catalog
	result.Controls = controls
	result.Metadata = deepCopyMetadata(catalog.Metadata)
	result.Groups = deepCopyGroups(catalog.Groups)
	return &result
}


// applyARModifications applies assessment requirement modifications to
// controls. Supports Add, Remove, Replace, Override, and Modify operations.
func applyARModifications(controls []Control, mods []AssessmentRequirementModifier) []Control {
	if len(mods) == 0 {
		return controls
	}

	modsByTarget := make(map[string][]AssessmentRequirementModifier, len(mods))
	for _, m := range mods {
		modsByTarget[m.TargetId] = append(modsByTarget[m.TargetId], m)
	}

	for i, ctrl := range controls {
		var modified []AssessmentRequirement
		for _, ar := range ctrl.AssessmentRequirements {
			targetMods, hasMods := modsByTarget[ar.Id]
			if !hasMods {
				modified = append(modified, ar)
				continue
			}

			removed := false
			for _, m := range targetMods {
				switch m.ModificationType {
				case ModRemove:
					removed = true
				case ModReplace, ModOverride, ModModify:
					ar = mergeARFields(ar, m)
				case ModAdd:
					modified = append(modified, ar)
					ar = newARFromModifier(m)
				default:
					// Unknown modification types are silently skipped to allow
					// forward compatibility with future ModType values.
				}
			}
			if !removed {
				modified = append(modified, ar)
			}
		}
		controls[i].AssessmentRequirements = modified
	}
	return controls
}

// mergeARFields applies non-zero modifier fields onto a copy of the AR.
// Slice fields are deep-copied to prevent aliasing with the modifier.
func mergeARFields(ar AssessmentRequirement, m AssessmentRequirementModifier) AssessmentRequirement {
	if m.Text != "" {
		ar.Text = m.Text
	}
	if len(m.Applicability) > 0 {
		ar.Applicability = copyStrings(m.Applicability)
	}
	if m.Recommendation != "" {
		ar.Recommendation = m.Recommendation
	}
	return ar
}

// newARFromModifier creates a new AssessmentRequirement from a modifier.
// Slice fields are deep-copied to prevent aliasing.
func newARFromModifier(m AssessmentRequirementModifier) AssessmentRequirement {
	return AssessmentRequirement{
		Id:             m.Id,
		Text:           m.Text,
		Applicability:  copyStrings(m.Applicability),
		Recommendation: m.Recommendation,
		State:          LifecycleActive,
	}
}

// deepCopyControls returns a new slice of controls with all nested slices
// and pointer fields deep-copied to prevent aliasing with the source.
func deepCopyControls(src []Control) []Control {
	if src == nil {
		return nil
	}
	dst := make([]Control, len(src))
	copy(dst, src)
	for i, c := range dst {
		if c.AssessmentRequirements != nil {
			ars := make([]AssessmentRequirement, len(c.AssessmentRequirements))
			for j, ar := range c.AssessmentRequirements {
				ar.Applicability = copyStrings(ar.Applicability)
				ar.ReplacedBy = cloneEntryMapping(ar.ReplacedBy)
				ars[j] = ar
			}
			dst[i].AssessmentRequirements = ars
		}
		dst[i].Guidelines = copyMultiEntryMappings(c.Guidelines)
		dst[i].Threats = copyMultiEntryMappings(c.Threats)
		dst[i].ReplacedBy = cloneEntryMapping(c.ReplacedBy)
	}
	return dst
}

func copyStrings(src []string) []string {
	if src == nil {
		return nil
	}
	dst := make([]string, len(src))
	copy(dst, src)
	return dst
}

func copyMultiEntryMappings(src []MultiEntryMapping) []MultiEntryMapping {
	if src == nil {
		return nil
	}
	dst := make([]MultiEntryMapping, len(src))
	for i, m := range src {
		dst[i] = m
		if m.Entries != nil {
			entries := make([]ArtifactMapping, len(m.Entries))
			copy(entries, m.Entries)
			dst[i].Entries = entries
		}
	}
	return dst
}

func toSet(items []string) map[string]bool {
	s := make(map[string]bool, len(items))
	for _, item := range items {
		s[item] = true
	}
	return s
}

func cloneEntryMapping(src *EntryMapping) *EntryMapping {
	if src == nil {
		return nil
	}
	cp := *src
	return &cp
}

func deepCopyMetadata(src Metadata) Metadata {
	src.MappingReferences = copyMappingReferences(src.MappingReferences)
	src.ApplicabilityGroups = deepCopyGroups(src.ApplicabilityGroups)
	if src.Lexicon != nil {
		cp := *src.Lexicon
		src.Lexicon = &cp
	}
	return src
}

func copyMappingReferences(src []MappingReference) []MappingReference {
	if src == nil {
		return nil
	}
	dst := make([]MappingReference, len(src))
	copy(dst, src)
	return dst
}

func deepCopyGroups(src []Group) []Group {
	if src == nil {
		return nil
	}
	dst := make([]Group, len(src))
	copy(dst, src)
	return dst
}
