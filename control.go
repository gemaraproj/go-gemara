package gemara

import "github.com/goccy/go-yaml"

// Control describes a safeguard or countermeasure with a clear objective and assessment requirements
type Control struct {
	// id allows this entry to be referenced by other elements
	Id string `json:"id" yaml:"id"`

	// title describes the purpose of this control at a glance
	Title string `json:"title" yaml:"title"`

	// objective is a unified statement of intent, which may encompass multiple situationally applicable requirements
	Objective string `json:"objective" yaml:"objective"`

	// group references by id a catalog group that this control belongs to
	Group string `json:"group" yaml:"group"`

	// assessment-requirements is a list of requirements that must be verified to confirm the control objective has been met
	AssessmentRequirements []AssessmentRequirement `json:"assessment-requirements" yaml:"assessment-requirements"`

	// guidelines documents relationships between this control and Layer 1 guideline artifacts
	Guidelines []MultiEntryMapping `json:"guidelines,omitempty" yaml:"guidelines,omitempty"`

	// threats documents relationships between this control and Layer 2 threat artifacts
	Threats []MultiEntryMapping `json:"threats,omitempty" yaml:"threats,omitempty"`

	// state is the lifecycle state of this control
	State Lifecycle `json:"state" yaml:"state"`

	// replaced-by references the control that supersedes this one when deprecated or retired
	ReplacedBy *EntryMapping `json:"replaced-by,omitempty" yaml:"replaced-by,omitempty"`

	references_cache []string
}

// UnmarshalYAML allows decoding controls from older/alternate YAML schemas.
// In particular, it supports using `family` instead of the struct's `group` key.
func (c *Control) UnmarshalYAML(data []byte) error {
	type controlYAML struct {
		Id        string `yaml:"id"`
		Title     string `yaml:"title"`
		Objective string `yaml:"objective"`
		Group     string `yaml:"group,omitempty"`
		Family    string `yaml:"family,omitempty"`

		AssessmentRequirements []AssessmentRequirement `yaml:"assessment-requirements,omitempty"`

		Guidelines []MultiEntryMapping `yaml:"guidelines,omitempty"`
		Threats    []MultiEntryMapping `yaml:"threats,omitempty"`

		State      Lifecycle     `yaml:"state"`
		ReplacedBy *EntryMapping `yaml:"replaced-by,omitempty"`
	}

	var tmp controlYAML
	if err := yaml.Unmarshal(data, &tmp); err != nil {
		return err
	}

	c.Id = tmp.Id
	c.Title = tmp.Title
	c.Objective = tmp.Objective
	if tmp.Group != "" {
		c.Group = tmp.Group
	} else {
		c.Group = tmp.Family
	}

	c.AssessmentRequirements = tmp.AssessmentRequirements
	c.Guidelines = tmp.Guidelines
	c.Threats = tmp.Threats
	c.State = tmp.State
	c.ReplacedBy = tmp.ReplacedBy

	return nil
}

func (c *Control) GetMappingReferences() (refs []string) {
	if len(c.references_cache) > 0 {
		return c.references_cache
	}
	for _, ref := range c.Guidelines {
		refs = append(refs, ref.ReferenceId)
	}
	for _, ref := range c.Threats {
		refs = append(refs, ref.ReferenceId)
	}
	return refs
}
