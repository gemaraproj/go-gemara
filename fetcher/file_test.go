// SPDX-License-Identifier: Apache-2.0

package fetcher

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFile_Success(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "test.yaml")
	require.NoError(t, os.WriteFile(p, []byte("field: value\n"), 0600))

	f := &File{}
	rc, err := f.Fetch(context.Background(), p)
	require.NoError(t, err)
	defer rc.Close() //nolint:errcheck

	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, "field: value\n", string(data))
}

func TestFile_NotFound(t *testing.T) {
	f := &File{}
	_, err := f.Fetch(context.Background(), "/nonexistent/path/to/file.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error opening file")
}
