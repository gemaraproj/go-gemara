// SPDX-License-Identifier: Apache-2.0

package bundle

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	godigest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
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
			name: "round trip preserves source and imports",
			bundle: &Bundle{
				Manifest: Manifest{BundleVersion: "1", GemaraVersion: "v1.0.0"},
				Source:   File{Name: "controls.yaml", Type: "ControlCatalog", Data: []byte("id: ctrl-catalog\ncontrols: []")},
				Imports: []File{
					{Name: "imported-guidance.yaml", Type: "GuidanceCatalog", Data: []byte("id: guidance-import\nguidelines: []")},
				},
			},
			tag: "v1.0.0",
			check: func(t *testing.T, original *Bundle, got *Bundle) {
				assert.Equal(t, original.Manifest, got.Manifest)
				assert.NotEmpty(t, got.Etag)
				assert.Equal(t, "controls.yaml", got.Source.Name)
				assert.Equal(t, "ControlCatalog", got.Source.Type)
				assert.Equal(t, original.Source.Data, got.Source.Data)
				require.Len(t, got.Imports, 1)
				assert.Equal(t, "imported-guidance.yaml", got.Imports[0].Name)
				assert.Equal(t, "GuidanceCatalog", got.Imports[0].Type)
				assert.Equal(t, original.Imports[0].Data, got.Imports[0].Data)
			},
		},
		{
			name: "source without imports",
			bundle: &Bundle{
				Manifest: Manifest{BundleVersion: "1", GemaraVersion: "v1.0.0"},
				Source:   File{Name: "controls.yaml", Type: "ControlCatalog", Data: []byte("controls: [one]")},
			},
			tag: "latest",
			check: func(t *testing.T, original *Bundle, got *Bundle) {
				assert.Equal(t, original.Manifest, got.Manifest)
				assert.Equal(t, "ControlCatalog", got.Source.Type)
				assert.Nil(t, got.Imports)
			},
		},
		{
			name: "custom annotations propagated",
			bundle: &Bundle{
				Manifest: Manifest{BundleVersion: "1", GemaraVersion: "v1.0.0"},
				Source:   File{Name: "c.yaml", Data: []byte("data")},
			},
			tag:  "annotated",
			opts: []PackOption{WithAnnotations(map[string]string{"org.example.source": "ci"})},
			check: func(t *testing.T, _ *Bundle, got *Bundle) {
				assert.Equal(t, "c.yaml", got.Source.Name)
			},
		},
		{
			name: "duplicate content across source and imports",
			bundle: &Bundle{
				Manifest: Manifest{BundleVersion: "1", GemaraVersion: "v1.0.0"},
				Source:   File{Name: "a.yaml", Data: []byte("identical content")},
				Imports:  []File{{Name: "b.yaml", Data: []byte("identical content")}},
			},
			tag: "dup",
			check: func(t *testing.T, _ *Bundle, got *Bundle) {
				assert.Equal(t, "a.yaml", got.Source.Name)
				require.Len(t, got.Imports, 1)
				assert.Equal(t, "b.yaml", got.Imports[0].Name)
			},
		},
		{
			name: "omitted type round-trips as empty string",
			bundle: &Bundle{
				Manifest: Manifest{BundleVersion: "1", GemaraVersion: "v1.0.0"},
				Source:   File{Name: "plain.yaml", Data: []byte("data")},
			},
			tag: "no-type",
			check: func(t *testing.T, _ *Bundle, got *Bundle) {
				assert.Empty(t, got.Source.Type)
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
		Source:   File{Name: "c.yaml", Data: []byte("data")},
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
			name:    "empty source",
			bundle:  &Bundle{},
			wantErr: "source artifact",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Pack(context.Background(), memory.New(), tt.bundle)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestPack_WithVersion(t *testing.T) {
	tests := []struct {
		name     string
		initial  string
		override string
		wantVer  string
	}{
		{
			name:     "overrides existing BundleVersion",
			initial:  "1.0.0",
			override: "2.0.0",
			wantVer:  "2.0.0",
		},
		{
			name:     "sets BundleVersion when empty",
			initial:  "",
			override: "3.0.0",
			wantVer:  "3.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			store := memory.New()

			b := &Bundle{
				Manifest: Manifest{BundleVersion: tt.initial, GemaraVersion: "v1.0.0"},
				Source:   File{Name: "c.yaml", Data: []byte("data")},
			}
			desc, err := Pack(ctx, store, b, WithVersion(tt.override))
			require.NoError(t, err)
			require.NotEmpty(t, desc.Digest)
			assert.Equal(t, tt.initial, b.Manifest.BundleVersion, "Pack must not mutate the caller's Bundle")
			require.NoError(t, store.Tag(ctx, desc, "v"))

			got, err := Unpack(ctx, store, "v")
			require.NoError(t, err)
			assert.Equal(t, tt.wantVer, got.Manifest.BundleVersion)
			assert.Equal(t, tt.wantVer, got.Version())
		})
	}
}

func TestBundle_Version(t *testing.T) {
	b := &Bundle{Manifest: Manifest{BundleVersion: "4.5.6"}}
	assert.Equal(t, "4.5.6", b.Version())

	b = &Bundle{}
	assert.Equal(t, "", b.Version())
}

func TestUnpack_MultipleSourcesError(t *testing.T) {
	ctx := context.Background()
	store := memory.New()

	b := &Bundle{
		Manifest: Manifest{BundleVersion: "1", GemaraVersion: "v1.0.0"},
		Source:   File{Name: "a.yaml", Data: []byte("first")},
		Imports:  []File{{Name: "b.yaml", Data: []byte("import")}},
	}
	desc, err := Pack(ctx, store, b)
	require.NoError(t, err)

	// Push a second artifact-role layer into the same manifest to
	// simulate a bundle produced by an older multi-source format.
	extraDesc, err := pushLayer(ctx, store, File{Name: "c.yaml", Data: []byte("second")}, roleArtifact)
	require.NoError(t, err)

	manifestData, err := fetchAll(ctx, store, desc)
	require.NoError(t, err)
	var ociManifest ocispec.Manifest
	require.NoError(t, json.Unmarshal(manifestData, &ociManifest))
	ociManifest.Layers = append(ociManifest.Layers, extraDesc)

	modifiedManifest, err := json.Marshal(ociManifest)
	require.NoError(t, err)
	modifiedDesc := ocispec.Descriptor{
		MediaType: ociManifest.MediaType,
		Digest:    godigest.FromBytes(modifiedManifest),
		Size:      int64(len(modifiedManifest)),
	}
	require.NoError(t, store.Push(ctx, modifiedDesc, bytes.NewReader(modifiedManifest)))
	require.NoError(t, store.Tag(ctx, modifiedDesc, "multi-source"))

	_, err = Unpack(ctx, store, "multi-source")
	assert.ErrorContains(t, err, "multiple source artifacts")
}

func TestUnpack_BadRef(t *testing.T) {
	_, err := Unpack(context.Background(), memory.New(), "does-not-exist")
	assert.Error(t, err)
}
