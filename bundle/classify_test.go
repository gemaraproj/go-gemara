// SPDX-License-Identifier: Apache-2.0

package bundle

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var policyYAML = []byte(`metadata:
  id: pol-1
  type: Policy
  gemara-version: "1.0.0"
  description: test
  author: {id: a, type: Human}
title: P
contacts: {responsible: [{id: a, type: Human}]}
scope: {in: {}}
imports: {}
adherence: {}
`)

var catalogYAML = []byte(`metadata:
  id: cat-1
  type: ControlCatalog
  gemara-version: "1.0.0"
  description: test
  author: {id: a, type: Human}
title: C
controls: []
`)

var guidanceYAML = []byte(`metadata:
  id: gc-1
  type: GuidanceCatalog
  gemara-version: "1.0.0"
  description: test
  author: {id: a, type: Human}
title: G
guidance-type: best-practice
guidelines: []
`)

func TestClassify_PolicyLeaf(t *testing.T) {
	b := &Bundle{
		Files:   []File{{Name: "policy.yaml", Type: "Policy", Data: policyYAML}},
		Imports: []File{{Name: "catalog.yaml", Type: "ControlCatalog", Data: catalogYAML}},
	}

	cb, err := b.Classify()
	require.NoError(t, err)
	require.NotNil(t, cb.Policy)
	assert.Equal(t, "pol-1", cb.Policy.Metadata.Id)
	assert.Nil(t, cb.ControlCatalog)
	assert.Nil(t, cb.GuidanceCatalog)
	assert.Len(t, cb.Imports.ControlCatalogs, 1)
}

func TestClassify_CatalogLeaf(t *testing.T) {
	b := &Bundle{
		Files: []File{{Name: "catalog.yaml", Type: "ControlCatalog", Data: catalogYAML}},
	}

	cb, err := b.Classify()
	require.NoError(t, err)
	assert.Nil(t, cb.Policy)
	require.NotNil(t, cb.ControlCatalog)
	assert.Equal(t, "cat-1", cb.ControlCatalog.Metadata.Id)
	assert.Nil(t, cb.GuidanceCatalog)
}

func TestClassify_GuidanceLeaf(t *testing.T) {
	b := &Bundle{
		Files: []File{{Name: "guidance.yaml", Type: "GuidanceCatalog", Data: guidanceYAML}},
	}

	cb, err := b.Classify()
	require.NoError(t, err)
	assert.Nil(t, cb.Policy)
	assert.Nil(t, cb.ControlCatalog)
	require.NotNil(t, cb.GuidanceCatalog)
	assert.Equal(t, "gc-1", cb.GuidanceCatalog.Metadata.Id)
}

func TestClassify_WithMultipleImports(t *testing.T) {
	b := &Bundle{
		Files: []File{{Name: "policy.yaml", Type: "Policy", Data: policyYAML}},
		Imports: []File{
			{Name: "catalog.yaml", Type: "ControlCatalog", Data: catalogYAML},
			{Name: "guidance.yaml", Type: "GuidanceCatalog", Data: guidanceYAML},
		},
	}

	cb, err := b.Classify()
	require.NoError(t, err)
	require.NotNil(t, cb.Policy)
	assert.Len(t, cb.Imports.ControlCatalogs, 1)
	assert.Len(t, cb.Imports.GuidanceCatalogs, 1)
}

func TestClassify_MultipleLeafFiles(t *testing.T) {
	b := &Bundle{
		Files: []File{
			{Name: "policy.yaml", Type: "Policy", Data: policyYAML},
			{Name: "catalog.yaml", Type: "ControlCatalog", Data: catalogYAML},
		},
	}

	cb, err := b.Classify()
	require.NoError(t, err)
	require.NotNil(t, cb.Policy, "policy should be classified as leaf")
	require.NotNil(t, cb.ControlCatalog, "catalog should be classified as leaf")
}

func TestClassify_EmptyBundle(t *testing.T) {
	b := &Bundle{}

	_, err := b.Classify()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no primary files")
}

func TestClassify_MalformedLeaf(t *testing.T) {
	b := &Bundle{
		Files: []File{{Name: "bad.yaml", Data: []byte("{{{bad yaml")}},
	}

	_, err := b.Classify()
	require.Error(t, err)
}
