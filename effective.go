// SPDX-License-Identifier: Apache-2.0

package gemara

import (
	"fmt"
	"strings"

	"github.com/gemaraproj/go-gemara/internal/codec"
)

// RequirementProvenance records how an effective assessment requirement was derived.
type RequirementProvenance struct {
	// RequirementID identifies the assessment requirement.
	RequirementID string `json:"requirement-id" yaml:"requirement-id"`

	// ControlID identifies the parent control containing the requirement.
	ControlID string `json:"control-id" yaml:"control-id"`

	// SourceCatalogID identifies the source catalog from which the requirement originated.
	SourceCatalogID string `json:"source-catalog-id" yaml:"source-catalog-id"`

	// ModifierID is the identifier of the modification directive from the policy, if modified.
	ModifierID string `json:"modifier-id,omitempty" yaml:"modifier-id,omitempty"`

	// ModificationType records the ModType applied (Add, Modify, Remove, Replace, Override).
	ModificationType ModType `json:"modification-type,omitempty" yaml:"modification-type,omitempty"`

	// Rationale captures the modification-rationale from the policy directive.
	Rationale string `json:"rationale,omitempty" yaml:"rationale,omitempty"`
}

// EffectiveCatalog is the resolved in-memory representation of a Gemara ControlCatalog,
// with all extends and policy imports/modifications fully processed.
type EffectiveCatalog struct {
	ControlCatalog `yaml:",inline"`

	// ResolvedFromPolicy is the ID of the policy that resolved this catalog, if applicable.
	ResolvedFromPolicy string `json:"resolved-from-policy,omitempty" yaml:"resolved-from-policy,omitempty"`

	// Provenance records the origin and modification history for each requirement.
	Provenance []RequirementProvenance `json:"provenance,omitempty" yaml:"provenance,omitempty"`
}

// UnmarshalYAML decodes an EffectiveCatalog, unmarshaling both the underlying
// ControlCatalog and the resolution-specific metadata (ResolvedFromPolicy, Provenance).
func (ec *EffectiveCatalog) UnmarshalYAML(data []byte) error {
	type alias struct {
		ResolvedFromPolicy string                  `json:"resolved-from-policy,omitempty" yaml:"resolved-from-policy,omitempty"`
		Provenance         []RequirementProvenance `json:"provenance,omitempty" yaml:"provenance,omitempty"`
	}
	if err := ec.ControlCatalog.UnmarshalYAML(data); err != nil {
		return err
	}
	var a alias
	if err := codec.UnmarshalYAML(data, &a); err != nil {
		return err
	}
	ec.ResolvedFromPolicy = a.ResolvedFromPolicy
	ec.Provenance = a.Provenance
	return nil
}

// EffectiveGuidanceCatalog is the resolved in-memory representation of a Gemara GuidanceCatalog,
// with all extends and policy imports/modifications fully processed.
type EffectiveGuidanceCatalog struct {
	GuidanceCatalog `yaml:",inline"`

	ResolvedFromPolicy string `json:"resolved-from-policy,omitempty" yaml:"resolved-from-policy,omitempty"`
}

// UnmarshalYAML decodes an EffectiveGuidanceCatalog, unmarshaling both the underlying
// GuidanceCatalog and the resolution-specific metadata.
func (eg *EffectiveGuidanceCatalog) UnmarshalYAML(data []byte) error {
	type alias struct {
		ResolvedFromPolicy string `json:"resolved-from-policy,omitempty" yaml:"resolved-from-policy,omitempty"`
	}
	if err := eg.GuidanceCatalog.UnmarshalYAML(data); err != nil {
		return err
	}
	var a alias
	if err := codec.UnmarshalYAML(data, &a); err != nil {
		return err
	}
	eg.ResolvedFromPolicy = a.ResolvedFromPolicy
	return nil
}

// EffectivePolicy is the resolved representation of a Gemara Policy with all its
// imported catalogs, policies, and guidelines fully resolved into effective sets.
type EffectivePolicy struct {
	Policy

	ResolvedCatalogs []*EffectiveCatalog         `json:"resolved-catalogs,omitempty" yaml:"resolved-catalogs,omitempty"`
	ResolvedGuidance []*EffectiveGuidanceCatalog `json:"resolved-guidance,omitempty" yaml:"resolved-guidance,omitempty"`
}

// Catalog returns the resolved EffectiveCatalog matching the given catalog reference ID,
// or nil if no such catalog was resolved.
func (ep *EffectivePolicy) Catalog(refID string) *EffectiveCatalog {
	for _, c := range ep.ResolvedCatalogs {
		if c.Metadata.Id == refID {
			return c
		}
	}
	return nil
}

// Guidance returns the resolved EffectiveGuidanceCatalog matching the given reference ID,
// or nil if no such guidance was resolved.
func (ep *EffectivePolicy) Guidance(refID string) *EffectiveGuidanceCatalog {
	for _, g := range ep.ResolvedGuidance {
		if g.Metadata.Id == refID {
			return g
		}
	}
	return nil
}

// FindControl locates a control in the effective catalog by ID.
func (ec *EffectiveCatalog) FindControl(id string) *Control {
	for i := range ec.Controls {
		if ec.Controls[i].Id == id {
			return &ec.Controls[i]
		}
	}
	return nil
}

// FindRequirement locates an assessment requirement and its parent control by requirement ID.
func (ec *EffectiveCatalog) FindRequirement(reqID string) (*Control, *AssessmentRequirement) {
	for i := range ec.Controls {
		for j := range ec.Controls[i].AssessmentRequirements {
			if ec.Controls[i].AssessmentRequirements[j].Id == reqID {
				return &ec.Controls[i], &ec.Controls[i].AssessmentRequirements[j]
			}
		}
	}
	return nil, nil
}

// GetProvenance returns the provenance record for a given requirement ID, if one exists.
func (ec *EffectiveCatalog) GetProvenance(reqID string) *RequirementProvenance {
	for i := range ec.Provenance {
		if ec.Provenance[i].RequirementID == reqID {
			return &ec.Provenance[i]
		}
	}
	return nil
}

// ResolveRequirements applies a Policy's catalog import directives (exclusions, constraints,
// and assessment-requirement modifications) to a candidate ControlCatalog per the
// Requirements Resolution specification (gemaraproj/website#16).
func ResolveRequirements(policy *Policy, baseCatalog *ControlCatalog, importRefID string) (*EffectiveCatalog, error) {
	if policy == nil {
		return nil, fmt.Errorf("policy is required")
	}
	if baseCatalog == nil {
		return nil, fmt.Errorf("base catalog is required")
	}

	var targetImport *CatalogImport
	for i := range policy.Imports.Catalogs {
		if policy.Imports.Catalogs[i].ReferenceId == importRefID {
			targetImport = &policy.Imports.Catalogs[i]
			break
		}
	}

	if targetImport == nil {
		return nil, fmt.Errorf("catalog import %q not found in policy %q", importRefID, policy.Metadata.Id)
	}

	eff := &EffectiveCatalog{
		ControlCatalog:     deepCopyControlCatalog(*baseCatalog),
		ResolvedFromPolicy: policy.Metadata.Id,
	}
	for _, ctrl := range eff.Controls {
		for _, ar := range ctrl.AssessmentRequirements {
			eff.Provenance = append(eff.Provenance, RequirementProvenance{
				RequirementID:   ar.Id,
				ControlID:       ctrl.Id,
				SourceCatalogID: baseCatalog.Metadata.Id,
			})
		}
	}

	if len(targetImport.Exclusions) > 0 {
		excludeSet := make(map[string]bool, len(targetImport.Exclusions))
		for _, ex := range targetImport.Exclusions {
			excludeSet[ex] = true
		}

		var filteredControls []Control
		for _, ctrl := range eff.Controls {
			if excludeSet[ctrl.Id] {
				continue
			}

			var filteredReqs []AssessmentRequirement
			for _, ar := range ctrl.AssessmentRequirements {
				if excludeSet[ar.Id] {
					continue
				}
				filteredReqs = append(filteredReqs, ar)
			}
			ctrl.AssessmentRequirements = filteredReqs
			filteredControls = append(filteredControls, ctrl)
		}
		eff.Controls = filteredControls
	}

	for _, constraint := range targetImport.Constraints {
		if constraint.TargetId != "" {
			ctrl, req := eff.FindRequirement(constraint.TargetId)
			if ctrl == nil && eff.FindControl(constraint.TargetId) == nil && req == nil {
				return nil, fmt.Errorf("constraint %q: target %q not found in catalog %q",
					constraint.Id, constraint.TargetId, baseCatalog.Metadata.Id)
			}
		}
	}

	for _, mod := range targetImport.AssessmentRequirementModifications {
		switch mod.ModificationType {
		case ModAdd:
			if strings.TrimSpace(mod.Text) == "" {
				return nil, fmt.Errorf("add modification %q: text is required", mod.Id)
			}
			if _, existing := eff.FindRequirement(mod.TargetId); existing != nil {
				return nil, fmt.Errorf("add modification %q: target requirement %q already exists", mod.Id, mod.TargetId)
			}

			parentCtrl := findParentControl(eff.Controls, mod.TargetId)
			if parentCtrl == nil {
				return nil, fmt.Errorf("add modification %q: no matching control found for target requirement %q", mod.Id, mod.TargetId)
			}

			newReq := AssessmentRequirement{
				Id:             mod.TargetId,
				Text:           mod.Text,
				Applicability:  append([]string(nil), mod.Applicability...),
				Recommendation: mod.Recommendation,
				State:          LifecycleActive,
			}
			parentCtrl.AssessmentRequirements = append(parentCtrl.AssessmentRequirements, newReq)

			eff.Provenance = append(eff.Provenance, RequirementProvenance{
				RequirementID:    mod.TargetId,
				ControlID:        parentCtrl.Id,
				SourceCatalogID:  baseCatalog.Metadata.Id,
				ModifierID:       mod.Id,
				ModificationType: ModAdd,
				Rationale:        mod.ModificationRationale,
			})

		case ModModify:
			ctrl, req := eff.FindRequirement(mod.TargetId)
			if req == nil {
				return nil, fmt.Errorf("modify modification %q: target requirement %q not found", mod.Id, mod.TargetId)
			}

			if mod.Text != "" {
				req.Text = mod.Text
			}
			if len(mod.Applicability) > 0 {
				req.Applicability = append([]string(nil), mod.Applicability...)
			}
			if mod.Recommendation != "" {
				req.Recommendation = mod.Recommendation
			}

			updateProvenance(&eff.Provenance, RequirementProvenance{
				RequirementID:    mod.TargetId,
				ControlID:        ctrl.Id,
				SourceCatalogID:  baseCatalog.Metadata.Id,
				ModifierID:       mod.Id,
				ModificationType: ModModify,
				Rationale:        mod.ModificationRationale,
			})

		case ModRemove:
			ctrl, req := eff.FindRequirement(mod.TargetId)
			if req == nil {
				return nil, fmt.Errorf("remove modification %q: target requirement %q not found", mod.Id, mod.TargetId)
			}

			var remaining []AssessmentRequirement
			for _, r := range ctrl.AssessmentRequirements {
				if r.Id != mod.TargetId {
					remaining = append(remaining, r)
				}
			}
			ctrl.AssessmentRequirements = remaining

			updateProvenance(&eff.Provenance, RequirementProvenance{
				RequirementID:    mod.TargetId,
				ControlID:        ctrl.Id,
				SourceCatalogID:  baseCatalog.Metadata.Id,
				ModifierID:       mod.Id,
				ModificationType: ModRemove,
				Rationale:        mod.ModificationRationale,
			})

		case ModReplace:
			ctrl, req := eff.FindRequirement(mod.TargetId)
			if req == nil {
				return nil, fmt.Errorf("replace modification %q: target requirement %q not found", mod.Id, mod.TargetId)
			}
			if strings.TrimSpace(mod.Text) == "" {
				return nil, fmt.Errorf("replace modification %q: text is required", mod.Id)
			}

			*req = AssessmentRequirement{
				Id:             req.Id,
				Text:           mod.Text,
				Applicability:  append([]string(nil), mod.Applicability...),
				Recommendation: mod.Recommendation,
				State:          LifecycleActive,
			}

			updateProvenance(&eff.Provenance, RequirementProvenance{
				RequirementID:    mod.TargetId,
				ControlID:        ctrl.Id,
				SourceCatalogID:  baseCatalog.Metadata.Id,
				ModifierID:       mod.Id,
				ModificationType: ModReplace,
				Rationale:        mod.ModificationRationale,
			})

		case ModOverride:
			ctrl, req := eff.FindRequirement(mod.TargetId)
			if req == nil {
				return nil, fmt.Errorf("override modification %q: target requirement %q not found", mod.Id, mod.TargetId)
			}
			if strings.TrimSpace(mod.Text) == "" {
				return nil, fmt.Errorf("override modification %q: text is required", mod.Id)
			}
			if strings.TrimSpace(mod.ModificationRationale) == "" {
				return nil, fmt.Errorf("override modification %q: modification-rationale is required", mod.Id)
			}

			req.Text = mod.Text
			if len(mod.Applicability) > 0 {
				req.Applicability = append([]string(nil), mod.Applicability...)
			}
			if mod.Recommendation != "" {
				req.Recommendation = mod.Recommendation
			}

			updateProvenance(&eff.Provenance, RequirementProvenance{
				RequirementID:    mod.TargetId,
				ControlID:        ctrl.Id,
				SourceCatalogID:  baseCatalog.Metadata.Id,
				ModifierID:       mod.Id,
				ModificationType: ModOverride,
				Rationale:        mod.ModificationRationale,
			})

		default:
			return nil, fmt.Errorf("unsupported modification type %v for %q", mod.ModificationType, mod.Id)
		}
	}

	return eff, nil
}

// findParentControl finds the parent control for a newly added assessment requirement.
// It searches for the longest matching control ID that prefixes targetID.
func findParentControl(controls []Control, targetID string) *Control {
	var bestMatch *Control
	maxLen := 0
	for i := range controls {
		cID := controls[i].Id
		if strings.HasPrefix(targetID, cID) && len(cID) > maxLen {
			bestMatch = &controls[i]
			maxLen = len(cID)
		}
	}
	if bestMatch != nil {
		return bestMatch
	}
	if len(controls) == 1 {
		return &controls[0]
	}
	return nil
}

func updateProvenance(prov *[]RequirementProvenance, p RequirementProvenance) {
	for i := range *prov {
		if (*prov)[i].RequirementID == p.RequirementID {
			(*prov)[i] = p
			return
		}
	}
	*prov = append(*prov, p)
}

func deepCopyControlCatalog(src ControlCatalog) ControlCatalog {
	dst := src

	if len(src.Groups) > 0 {
		dst.Groups = make([]Group, len(src.Groups))
		copy(dst.Groups, src.Groups)
	}

	if len(src.Controls) > 0 {
		dst.Controls = make([]Control, len(src.Controls))
		for i, c := range src.Controls {
			dst.Controls[i] = c
			if len(c.AssessmentRequirements) > 0 {
				dst.Controls[i].AssessmentRequirements = make([]AssessmentRequirement, len(c.AssessmentRequirements))
				for j, ar := range c.AssessmentRequirements {
					dst.Controls[i].AssessmentRequirements[j] = ar
					if len(ar.Applicability) > 0 {
						dst.Controls[i].AssessmentRequirements[j].Applicability = append([]string(nil), ar.Applicability...)
					}
				}
			}
		}
	}

	return dst
}

// ResolveGuidanceRequirements applies a Policy's guidance import directives (exclusions, constraints)
// to a candidate GuidanceCatalog.
func ResolveGuidanceRequirements(policy *Policy, baseCatalog *GuidanceCatalog, importRefID string) (*EffectiveGuidanceCatalog, error) {
	if policy == nil {
		return nil, fmt.Errorf("policy is required")
	}
	if baseCatalog == nil {
		return nil, fmt.Errorf("base catalog is required")
	}

	var targetImport *GuidanceImport
	for i := range policy.Imports.Guidance {
		if policy.Imports.Guidance[i].ReferenceId == importRefID {
			targetImport = &policy.Imports.Guidance[i]
			break
		}
	}

	if targetImport == nil {
		return nil, fmt.Errorf("guidance import %q not found in policy %q", importRefID, policy.Metadata.Id)
	}

	eff := &EffectiveGuidanceCatalog{
		GuidanceCatalog:    deepCopyGuidanceCatalog(*baseCatalog),
		ResolvedFromPolicy: policy.Metadata.Id,
	}

	if len(targetImport.Exclusions) > 0 {
		excludeSet := make(map[string]bool, len(targetImport.Exclusions))
		for _, ex := range targetImport.Exclusions {
			excludeSet[ex] = true
		}

		var filtered []Guideline
		for _, gl := range eff.Guidelines {
			if !excludeSet[gl.Id] {
				filtered = append(filtered, gl)
			}
		}
		eff.Guidelines = filtered
	}

	for _, constraint := range targetImport.Constraints {
		if constraint.TargetId != "" {
			found := false
			for _, gl := range eff.Guidelines {
				if gl.Id == constraint.TargetId {
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("constraint %q: target %q not found in guidance %q",
					constraint.Id, constraint.TargetId, baseCatalog.Metadata.Id)
			}
		}
	}

	return eff, nil
}

func deepCopyGuidanceCatalog(src GuidanceCatalog) GuidanceCatalog {
	dst := src
	if len(src.Groups) > 0 {
		dst.Groups = make([]Group, len(src.Groups))
		copy(dst.Groups, src.Groups)
	}
	if len(src.Guidelines) > 0 {
		dst.Guidelines = make([]Guideline, len(src.Guidelines))
		copy(dst.Guidelines, src.Guidelines)
	}
	return dst
}
