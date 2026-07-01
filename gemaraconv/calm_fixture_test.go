// SPDX-License-Identifier: Apache-2.0

package gemaraconv

import (
	"context"
	"os"
	"testing"

	gemara "github.com/gemaraproj/go-gemara"
	"github.com/gemaraproj/go-gemara/fetcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestControlCatalogToCALM_CCCFixture loads a real CCC catalog from disk, converts
// it, and pins the exact CALM document (same input -> same artifact).
// Regenerate: go test ./gemaraconv/ -run CCCFixture -update
func TestControlCatalogToCALM_CCCFixture(t *testing.T) {
	catalog, err := gemara.Load[gemara.ControlCatalog](
		context.Background(), &fetcher.File{}, "testdata/ccc.marefarc.cn.yaml")
	require.NoError(t, err)

	// author + id + version carry the release coordinate so a consumer can resolve
	// back to finos-ccc/ccc.marefarc.cn @ v2026.06-rc1.
	require.Equal(t, "finos-ccc", catalog.Metadata.Author.Id)
	require.Equal(t, "ccc.marefarc.cn", catalog.Metadata.Id)
	require.Equal(t, "v2026.06-rc1", catalog.Metadata.Version)

	controls, err := ControlCatalogToCALM(*catalog)
	require.NoError(t, err)

	// The emitted document conforms to the real upstream CALM controls schema.
	assertConformsToControlsSchema(t, controls)

	got, err := controls.MarshalDocument()
	require.NoError(t, err)
	got = append(got, '\n')

	const goldenPath = "testdata/ccc.marefarc.cn.controls.json"
	if *updateGolden {
		require.NoError(t, os.WriteFile(goldenPath, got, 0o600))
	}

	want, err := os.ReadFile(goldenPath)
	require.NoError(t, err)
	assert.Equal(t, string(want), string(got),
		"CCC fixture drifted from golden; regenerate with: go test ./gemaraconv/ -run CCCFixture -update")
}
