// SPDX-License-Identifier: Apache-2.0

package gemara

import "sync"

// SugarControl wraps the generated Control with cached
// cross-reference lookups.
type SugarControl struct {
	Control

	referencesOnce  sync.Once
	referencesCache []string
}

func (c *SugarControl) GetMappingReferences() []string {
	c.referencesOnce.Do(func() {
		for _, ref := range c.Guidelines {
			c.referencesCache = append(c.referencesCache, ref.ReferenceId)
		}
		for _, ref := range c.Threats {
			c.referencesCache = append(c.referencesCache, ref.ReferenceId)
		}
	})
	return c.referencesCache
}
