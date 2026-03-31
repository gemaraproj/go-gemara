package gemara

import (
	"slices"

	"github.com/goccy/go-yaml"
)

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

func (c *ControlCatalog) GetGroupNames() (groups []string) {
	for _, group := range c.Groups {
		groups = append(groups, group.Title)
	}
	return groups
}

func (c *ControlCatalog) GetControlsForGroup(group string) (controls []Control) {
	for _, control := range c.Controls {
		if control.Group == group {
			controls = append(controls, control)
		}
	}
	return controls
}

func (c *ControlCatalog) GetRequirementForApplicability(applicability string) (reqs []AssessmentRequirement) {
	for _, control := range c.Controls {
		for _, assessment := range control.AssessmentRequirements {
			if slices.Contains(assessment.Applicability, applicability) {
				reqs = append(reqs, assessment)
			}
		}
	}
	return reqs
}
