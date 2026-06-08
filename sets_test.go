// SPDX-License-Identifier: Apache-2.0

package gemara

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassify_MixedArtifacts(t *testing.T) {
	polYAML := []byte(`metadata:
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
	catYAML := []byte(`metadata:
  id: cat-1
  type: ControlCatalog
  gemara-version: "1.0.0"
  description: test
  author: {id: a, type: Human}
title: C
controls: []
`)
	gcYAML := []byte(`metadata:
  id: gc-1
  type: GuidanceCatalog
  gemara-version: "1.0.0"
  description: test
  author: {id: a, type: Human}
title: G
guidance-type: best-practice
guidelines: []
`)

	as, err := Classify(polYAML, catYAML, gcYAML)
	require.NoError(t, err)
	require.Len(t, as.Policies, 1)
	assert.Equal(t, "pol-1", as.Policies[0].Metadata.Id)
	require.Len(t, as.ControlCatalogs, 1)
	assert.Equal(t, "cat-1", as.ControlCatalogs[0].Metadata.Id)
	require.Len(t, as.GuidanceCatalogs, 1)
	assert.Equal(t, "gc-1", as.GuidanceCatalogs[0].Metadata.Id)
}

func TestClassify_SkipsUnknownTypes(t *testing.T) {
	evalYAML := []byte(`metadata:
  id: eval-1
  type: EvaluationLog
  gemara-version: "1.0.0"
  description: test
  author: {id: a, type: Human}
`)
	catYAML := []byte(`metadata:
  id: cat-1
  type: ControlCatalog
  gemara-version: "1.0.0"
  description: test
  author: {id: a, type: Human}
title: C
controls: []
`)

	as, err := Classify(evalYAML, catYAML)
	require.NoError(t, err)
	assert.Empty(t, as.Policies)
	assert.Len(t, as.ControlCatalogs, 1)
	assert.Empty(t, as.GuidanceCatalogs)
}

func TestClassify_Empty(t *testing.T) {
	as, err := Classify()
	require.NoError(t, err)
	assert.Empty(t, as.Policies)
	assert.Empty(t, as.ControlCatalogs)
	assert.Empty(t, as.GuidanceCatalogs)
}

func TestClassify_MalformedYAML(t *testing.T) {
	_, err := Classify([]byte(`{{{not valid yaml`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "artifact 0")
}

func TestClassify_MultipleSameType(t *testing.T) {
	cat1 := []byte(`metadata:
  id: cat-1
  type: ControlCatalog
  gemara-version: "1.0.0"
  description: test
  author: {id: a, type: Human}
title: C1
controls: []
`)
	cat2 := []byte(`metadata:
  id: cat-2
  type: ControlCatalog
  gemara-version: "1.0.0"
  description: test
  author: {id: a, type: Human}
title: C2
controls: []
`)

	as, err := Classify(cat1, cat2)
	require.NoError(t, err)
	require.Len(t, as.ControlCatalogs, 2)
	assert.Equal(t, "cat-1", as.ControlCatalogs[0].Metadata.Id)
	assert.Equal(t, "cat-2", as.ControlCatalogs[1].Metadata.Id)
}
