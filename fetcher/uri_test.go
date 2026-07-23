// SPDX-License-Identifier: Apache-2.0

package fetcher

import (
	"context"
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
	rc, err := f.Fetch(context.Background(), "file://"+p)
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
	rc, err := f.Fetch(context.Background(), srv.URL+"/remote.yaml")
	require.NoError(t, err)
	defer rc.Close() //nolint:errcheck

	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, "remote: true\n", string(data))
}

func TestURI_UnsupportedScheme(t *testing.T) {
	f := &URI{}
	_, err := f.Fetch(context.Background(), "ftp://example.com/file.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported URI scheme")
}

func TestURI_BarePath_Absolute(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "data.yaml")
	require.NoError(t, os.WriteFile(p, []byte("ok: true\n"), 0600))

	f := &URI{}
	rc, err := f.Fetch(context.Background(), p)
	require.NoError(t, err)
	defer rc.Close() //nolint:errcheck

	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, "ok: true\n", string(data))
}

func TestURI_BarePath_Relative(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "data.yaml"), []byte("ok: true\n"), 0600))
	t.Chdir(tmp)

	f := &URI{}
	rc, err := f.Fetch(context.Background(), "./data.yaml")
	require.NoError(t, err)
	defer rc.Close() //nolint:errcheck

	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, "ok: true\n", string(data))
}

func TestURI_TypoScheme(t *testing.T) {
	f := &URI{}
	_, err := f.Fetch(context.Background(), "htps://example.com/file.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported URI scheme")
}

func TestURI_BasePath_FileScheme(t *testing.T) {
	base := t.TempDir()
	sub := filepath.Join(base, "catalogs")
	require.NoError(t, os.MkdirAll(sub, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(sub, "c.yaml"), []byte("found: true\n"), 0600))

	tests := []struct {
		name     string
		basePath string
		source   string
		want     string
	}{
		{
			name:     "relative file:// resolved against BasePath",
			basePath: base,
			source:   "file://catalogs/c.yaml",
			want:     "found: true\n",
		},
		{
			name:     "dot-slash relative file:// resolved against BasePath",
			basePath: base,
			source:   "file://./catalogs/c.yaml",
			want:     "found: true\n",
		},
		{
			name:     "parent-relative file:// resolved against BasePath",
			basePath: sub,
			source:   "file://../catalogs/c.yaml",
			want:     "found: true\n",
		},
		{
			name:     "absolute file:// ignores BasePath",
			basePath: "/should/not/matter",
			source:   "file://" + filepath.Join(sub, "c.yaml"),
			want:     "found: true\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &URI{BasePath: tt.basePath}
			rc, err := f.Fetch(context.Background(), tt.source)
			require.NoError(t, err)
			defer rc.Close() //nolint:errcheck

			data, err := io.ReadAll(rc)
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(data))
		})
	}
}

func TestURI_BasePath_BarePath(t *testing.T) {
	base := t.TempDir()
	sub := filepath.Join(base, "catalogs")
	require.NoError(t, os.MkdirAll(sub, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(sub, "c.yaml"), []byte("bare: true\n"), 0600))

	tests := []struct {
		name     string
		basePath string
		source   string
		want     string
	}{
		{
			name:     "bare relative path resolved against BasePath",
			basePath: base,
			source:   "catalogs/c.yaml",
			want:     "bare: true\n",
		},
		{
			name:     "dot-slash bare path resolved against BasePath",
			basePath: base,
			source:   "./catalogs/c.yaml",
			want:     "bare: true\n",
		},
		{
			name:     "absolute bare path ignores BasePath",
			basePath: "/should/not/matter",
			source:   filepath.Join(sub, "c.yaml"),
			want:     "bare: true\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &URI{BasePath: tt.basePath}
			rc, err := f.Fetch(context.Background(), tt.source)
			require.NoError(t, err)
			defer rc.Close() //nolint:errcheck

			data, err := io.ReadAll(rc)
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(data))
		})
	}
}

func TestURI_BasePath_Empty_BackwardCompatible(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "data.yaml"), []byte("compat: true\n"), 0600))
	t.Chdir(tmp)

	f := &URI{}
	rc, err := f.Fetch(context.Background(), "file://./data.yaml")
	require.NoError(t, err)
	defer rc.Close() //nolint:errcheck

	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, "compat: true\n", string(data))
}
