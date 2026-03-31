package gemara

import (
	"slices"

	"github.com/goccy/go-yaml"
)

// ControlCatalog describes a set of related controls and relevant metadata
type ControlCatalog struct {
	// title describes the purpose of this catalog at a glance
	Title string `json:"title" yaml:"title"`

	// metadata provides detailed data about this catalog
	Metadata Metadata `json:"metadata" yaml:"metadata"`

	// controls is a list of unique controls defined by this catalog
	Controls []Control `json:"controls,omitempty" yaml:"controls,omitempty"`

	// groups contains a list of groups that can be referenced by entries in this catalog
	Groups []Group `json:"groups,omitempty" yaml:"groups,omitempty"`

	// extends references catalogs that this catalog builds upon
	Extends []ArtifactMapping `json:"extends,omitempty" yaml:"extends,omitempty"`

	Imports []MultiEntryMapping `json:"imports,omitempty" yaml:"imports,omitempty"`

	groups_cache       []string
	controls_cache     map[string][]Control
	requirements_cache map[string][]AssessmentRequirement
}

// UnmarshalYAML allows decoding control catalogs from older/alternate YAML schemas.
// It supports mapping `families` -> `groups`.
func (c *ControlCatalog) UnmarshalYAML(data []byte) error {
	type controlCatalogYAML struct {
		Groups   []Group `yaml:"groups,omitempty"`
		Families []Group `yaml:"families,omitempty"`

		Title    string   `yaml:"title"`
		Metadata Metadata `yaml:"metadata"`

		Extends []ArtifactMapping   `yaml:"extends,omitempty"`
		Imports []MultiEntryMapping `yaml:"imports,omitempty"`

		Controls []Control `yaml:"controls,omitempty"`
	}

	var tmp controlCatalogYAML
	if err := yaml.Unmarshal(data, &tmp); err != nil {
		return err
	}

	c.Groups = tmp.Groups
	if len(c.Groups) == 0 {
		c.Groups = tmp.Families
	}
	c.Controls = tmp.Controls

	c.Title = tmp.Title
	c.Metadata = tmp.Metadata
	c.Extends = tmp.Extends

	// Keep imports exactly as decoded (nil vs empty can matter to tests).
	c.Imports = tmp.Imports

	return nil
}

func (c *ControlCatalog) GetGroupIds() (groups []string) {
	if len(c.groups_cache) > 0 {
		return c.groups_cache
	}
	for _, group := range c.Groups {
		groups = append(groups, group.Id)
	}
	return groups
}

func (c *ControlCatalog) GetControlsForGroup(group string) (controls []Control) {
	if c.controls_cache != nil && len(c.controls_cache[group]) > 0 {
		return c.controls_cache[group]
	}
	for _, control := range c.Controls {
		if control.Group == group {
			controls = append(controls, control)
		}
	}
	return controls
}

func (c *ControlCatalog) GetRequirementForApplicability(applicability string) (reqs []AssessmentRequirement) {
	if c.requirements_cache != nil && len(c.requirements_cache[applicability]) > 0 {
		return c.requirements_cache[applicability]
	}
	for _, control := range c.Controls {
		for _, assessment := range control.AssessmentRequirements {
			if slices.Contains(assessment.Applicability, applicability) {
				reqs = append(reqs, assessment)
			}
		}
	}
	return reqs
}
