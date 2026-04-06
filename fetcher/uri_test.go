// SPDX-License-Identifier: Apache-2.0

package fetcher

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestURI_FileScheme(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "data.yaml")
	require.NoError(t, os.WriteFile(p, []byte("ok: true\n"), 0600))

	f := &URI{}
	rc, err := f.Fetch("file://" + p)
	require.NoError(t, err)
	defer rc.Close() //nolint:errcheck

	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, "ok: true\n", string(data))
}

func TestURI_HTTPScheme(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("remote: true\n"))
	}))
	defer srv.Close()

	f := &URI{Client: srv.Client()}
	rc, err := f.Fetch(srv.URL + "/remote.yaml")
	require.NoError(t, err)
	defer rc.Close() //nolint:errcheck

	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, "remote: true\n", string(data))
}

func TestURI_UnsupportedScheme(t *testing.T) {
	f := &URI{}
	_, err := f.Fetch("ftp://example.com/file.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported URI scheme")
}
