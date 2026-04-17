// SPDX-License-Identifier: Apache-2.0

package bundle

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"oras.land/oras-go/v2/content/memory"
)

func TestPackUnpack(t *testing.T) {
	tests := []struct {
		name    string
		bundle  *Bundle
		tag     string
		opts    []PackOption
		wantErr string
		check   func(t *testing.T, original *Bundle, got *Bundle)
	}{
		{
			name: "round trip preserves files and imports",
			bundle: &Bundle{
				Manifest: Manifest{BundleVersion: "1", GemaraVersion: "v1.0.0"},
				Files: []File{
					{Name: "controls.yaml", Data: []byte("id: ctrl-catalog\ncontrols: []")},
				},
				Imports: []File{
					{Name: "imported-guidance.yaml", Data: []byte("id: guidance-import\nguidelines: []")},
				},
			},
			tag: "v1.0.0",
			check: func(t *testing.T, original *Bundle, got *Bundle) {
				assert.Equal(t, original.Manifest, got.Manifest)
				assert.NotEmpty(t, got.Etag)
				require.Len(t, got.Files, 1)
				assert.Equal(t, "controls.yaml", got.Files[0].Name)
				assert.Equal(t, original.Files[0].Data, got.Files[0].Data)
				require.Len(t, got.Imports, 1)
				assert.Equal(t, "imported-guidance.yaml", got.Imports[0].Name)
				assert.Equal(t, original.Imports[0].Data, got.Imports[0].Data)
			},
		},
		{
			name: "multiple files without imports",
			bundle: &Bundle{
				Manifest: Manifest{BundleVersion: "1", GemaraVersion: "v1.0.0"},
				Files: []File{
					{Name: "controls.yaml", Data: []byte("controls: [one]")},
					{Name: "threats.yaml", Data: []byte("threats: [two]")},
				},
			},
			tag: "latest",
			check: func(t *testing.T, original *Bundle, got *Bundle) {
				assert.Equal(t, original.Manifest, got.Manifest)
				require.Len(t, got.Files, 2)
				assert.Nil(t, got.Imports)
			},
		},
		{
			name: "custom annotations propagated",
			bundle: &Bundle{
				Manifest: Manifest{BundleVersion: "1", GemaraVersion: "v1.0.0"},
				Files:    []File{{Name: "c.yaml", Data: []byte("data")}},
			},
			tag:  "annotated",
			opts: []PackOption{WithAnnotations(map[string]string{"org.example.source": "ci"})},
			check: func(t *testing.T, _ *Bundle, got *Bundle) {
				require.Len(t, got.Files, 1)
			},
		},
		{
			name: "duplicate content across files and imports",
			bundle: &Bundle{
				Manifest: Manifest{BundleVersion: "1", GemaraVersion: "v1.0.0"},
				Files:    []File{{Name: "a.yaml", Data: []byte("identical content")}},
				Imports:  []File{{Name: "b.yaml", Data: []byte("identical content")}},
			},
			tag: "dup",
			check: func(t *testing.T, _ *Bundle, got *Bundle) {
				require.Len(t, got.Files, 1)
				require.Len(t, got.Imports, 1)
				assert.Equal(t, "a.yaml", got.Files[0].Name)
				assert.Equal(t, "b.yaml", got.Imports[0].Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			store := memory.New()

			desc, err := Pack(ctx, store, tt.bundle, tt.opts...)
			require.NoError(t, err)
			require.NotEmpty(t, desc.Digest)
			require.NoError(t, store.Tag(ctx, desc, tt.tag))

			got, err := Unpack(ctx, store, tt.tag)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			tt.check(t, tt.bundle, got)
		})
	}
}

func TestPackUnpack_Annotations(t *testing.T) {
	ctx := context.Background()
	store := memory.New()

	b := &Bundle{
		Manifest: Manifest{BundleVersion: "1", GemaraVersion: "v1.0.0"},
		Files:    []File{{Name: "c.yaml", Data: []byte("data")}},
	}
	desc, err := Pack(ctx, store, b, WithAnnotations(map[string]string{
		"org.example.source": "ci",
	}))
	require.NoError(t, err)
	assert.Equal(t, "ci", desc.Annotations["org.example.source"])
}

func TestPack_Errors(t *testing.T) {
	tests := []struct {
		name    string
		bundle  *Bundle
		wantErr string
	}{
		{
			name:    "nil bundle",
			bundle:  nil,
			wantErr: "bundle must not be nil",
		},
		{
			name:    "empty files",
			bundle:  &Bundle{},
			wantErr: "at least one artifact file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Pack(context.Background(), memory.New(), tt.bundle)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestUnpack_BadRef(t *testing.T) {
	_, err := Unpack(context.Background(), memory.New(), "does-not-exist")
	assert.Error(t, err)
}
