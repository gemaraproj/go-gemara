// SPDX-License-Identifier: Apache-2.0

package gemara

import (
	"testing"

	"github.com/gemaraproj/go-gemara/internal/codec"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSControlCatalog_RoundTrip(t *testing.T) {
	original, err := Load[ControlCatalog](fileFetcher, "test-data/good-ccc.yaml")
	require.NoError(t, err)

	sc := original.Sugar()

	yamlBytes, err := codec.MarshalYAML(sc)
	require.NoError(t, err)

	var roundTripped SControlCatalog
	require.NoError(t, codec.UnmarshalYAML(yamlBytes, &roundTripped))

	assert.Equal(t, original.Title, roundTripped.Title)
	assert.Equal(t, original.Metadata.Id, roundTripped.Metadata.Id)
	assert.Equal(t, len(original.Groups), len(roundTripped.Groups))
	assert.Equal(t, len(original.Controls), len(roundTripped.Controls))

	if diff := cmp.Diff(original.Controls, roundTripped.Controls); diff != "" {
		t.Errorf("controls mismatch (-original +roundtripped):\n%s", diff)
	}
}

func TestSControlCatalog_CacheResetOnUnmarshal(t *testing.T) {
	original, err := Load[ControlCatalog](fileFetcher, "test-data/good-ccc.yaml")
	require.NoError(t, err)
	sc := original.Sugar()

	_ = sc.GetGroupNames()
	require.NotEmpty(t, sc.GetGroupNames(), "cache should be populated")

	yamlBytes, err := codec.MarshalYAML(sc)
	require.NoError(t, err)
	require.NoError(t, codec.UnmarshalYAML(yamlBytes, sc))

	groups := sc.GetGroupNames()
	require.NotEmpty(t, groups, "cache should repopulate after unmarshal")
}
