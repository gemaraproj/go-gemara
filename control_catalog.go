// SPDX-License-Identifier: Apache-2.0

package gemara

import "sync"

// SugarControlCatalog wraps the generated ControlCatalog with
// pre-built indexes for efficient group, control, and requirement lookups.
type SugarControlCatalog struct {
	ControlCatalog

	groupsOnce        sync.Once
	groupsCache       []string

	controlsOnce      sync.Once
	controlsCache     map[string][]Control

	requirementsOnce  sync.Once
	requirementsCache map[string][]AssessmentRequirement
}

func (c *SugarControlCatalog) GetGroupNames() []string {
	c.groupsOnce.Do(func() {
		for _, group := range c.Groups {
			c.groupsCache = append(c.groupsCache, group.Title)
		}
	})
	return c.groupsCache
}

func (c *SugarControlCatalog) GetControlsForGroup(group string) []Control {
	c.controlsOnce.Do(func() {
		c.controlsCache = make(map[string][]Control)
		for _, control := range c.Controls {
			c.controlsCache[control.Group] = append(
				c.controlsCache[control.Group], control,
			)
		}
	})
	return c.controlsCache[group]
}

func (c *SugarControlCatalog) GetRequirementForApplicability(applicability string) []AssessmentRequirement {
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
