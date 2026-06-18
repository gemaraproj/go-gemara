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
	rc, err := f.Fetch(context.Background(), FileURI(p))
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

func TestFileURI(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Unix absolute path",
			input: "/home/user/data.yaml",
			want:  "file:///home/user/data.yaml",
		},
		{
			name:  "Windows absolute path",
			input: `C:\Users\foo\data.yaml`,
			want:  "file:///C:/Users/foo/data.yaml",
		},
		{
			name:  "Relative path unchanged",
			input: "data/file.yaml",
			want:  "data/file.yaml",
		},
		{
			name:  "Dot-relative path unchanged",
			input: "./data/file.yaml",
			want:  "data/file.yaml",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FileURI(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
