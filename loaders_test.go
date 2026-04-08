// SPDX-License-Identifier: Apache-2.0

package gemara

// This file contains table-driven tests for loader functions:
// - PolicyDocument.LoadFile
// - GuidanceCatalog.LoadFile and LoadFiles
// - Catalog.LoadFile, LoadFiles, and LoadNestedCatalog
//
// Test data is pulled from ./test-data/

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gemaraproj/go-gemara/fetcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var fileFetcher = &fetcher.File{}

// ============================================================================
// Policy Tests
// ============================================================================

func TestLoad_Policy(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		{
			name:    "File not found",
			source:  "bad-path.yaml",
			wantErr: true,
		},
		{
			name:    "Bad YAML",
			source:  "test-data/bad.yaml",
			wantErr: true,
		},
		{
			name:    "Unsupported file extension",
			source:  "test-data/unsupported.txt",
			wantErr: true,
		},
		{
			name:    "Good YAML — Policy Document",
			source:  "test-data/good-policy.yaml",
			wantErr: false,
		},
		{
			name:    "Good YAML — Security Policy",
			source:  "test-data/good-security-policy.yml",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := Load[Policy](context.Background(), fileFetcher, tt.source)
			if tt.wantErr {
				assert.Error(t, err, "expected error but got none")
			} else {
				require.NoError(t, err, "unexpected error loading file")
				assert.NotEmpty(t, p.Metadata.Id, "Policy document ID should not be empty")
				assert.NotEmpty(t, p.Metadata.Version, "Policy document version should not be empty")
			}
		})
	}
}

func TestLoad_Policy_HTTP(t *testing.T) {
	srv := httptest.NewTLSServer(http.NotFoundHandler())
	defer srv.Close()
	f := &fetcher.HTTP{Client: srv.Client()}

	_, err := Load[Policy](context.Background(), f, srv.URL+"/nonexistent.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch URL; response status: 404 Not Found")
}

func TestLoad_URLWithQueryParams(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "test-data/good-policy.yaml")
	}))
	defer srv.Close()
	f := &fetcher.HTTP{Client: srv.Client()}

	p, err := Load[Policy](context.Background(), f, srv.URL+"/policy.yaml?token=abc&ref=main")
	require.NoError(t, err)
	assert.NotEmpty(t, p.Metadata.Id)
}

func TestLoad_URLWithFragment(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "test-data/good-policy.yaml")
	}))
	defer srv.Close()
	f := &fetcher.HTTP{Client: srv.Client()}

	p, err := Load[Policy](context.Background(), f, srv.URL+"/policy.yaml#section")
	require.NoError(t, err)
	assert.NotEmpty(t, p.Metadata.Id)
}

// ============================================================================
// GuidanceCatalog Tests
// ============================================================================

func TestLoad_GuidanceCatalog(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		{
			name:    "Bad YAML",
			source:  "test-data/bad.yaml",
			wantErr: true,
		},
		{
			name:    "Unsupported file extension",
			source:  "test-data/unsupported.txt",
			wantErr: true,
		},
		{
			name:    "Good YAML — AIGF",
			source:  "test-data/good-aigf.yaml",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, err := Load[GuidanceCatalog](context.Background(), fileFetcher, tt.source)
			if tt.wantErr {
				assert.Error(t, err, "expected error but got none")
			} else {
				require.NoError(t, err, "unexpected error loading file")
				assert.NotEmpty(t, g.Metadata.Id, "Guidance document ID should not be empty")
				assert.NotEmpty(t, g.Title, "Guidance document title should not be empty")
				assert.NotEmpty(t, g.Groups, "Guidance document should have at least one family")
				assert.NotEmpty(t, g.Guidelines, "Guidance document should have at least one guideline")
			}
		})
	}
}

func TestGuidanceCatalog_LoadFiles_AppendsData(t *testing.T) {
	singleDoc, err := Load[GuidanceCatalog](context.Background(), fileFetcher, "test-data/good-aigf.yaml")
	require.NoError(t, err)
	require.Greater(t, len(singleDoc.Groups), 0, "expected at least one family")
	require.Greater(t, len(singleDoc.Guidelines), 0, "expected at least one guideline")

	multiDoc := &GuidanceCatalog{}
	err = multiDoc.LoadFiles(context.Background(), fileFetcher, []string{
		"test-data/good-aigf.yaml",
		"test-data/good-aigf.yaml",
	})
	require.NoError(t, err)

	assert.Equal(t, singleDoc.Metadata, multiDoc.Metadata,
		"first document's metadata should be preserved")
	assert.Equal(t, len(singleDoc.Groups)*2, len(multiDoc.Groups),
		"families should be appended across multiple files")
	assert.Equal(t, len(singleDoc.Guidelines)*2, len(multiDoc.Guidelines),
		"guidelines should be appended across multiple files")
}

func TestLoad_GuidanceCatalog_HTTP(t *testing.T) {
	srv := httptest.NewTLSServer(http.NotFoundHandler())
	defer srv.Close()
	f := &fetcher.HTTP{Client: srv.Client()}

	_, err := Load[GuidanceCatalog](context.Background(), f, srv.URL+"/nonexistent.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch URL; response status: 404 Not Found")
}

// ============================================================================
// ControlCatalog Tests
// ============================================================================

func TestLoad_ControlCatalog(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		{
			name:    "File not found",
			source:  "bad-path.yaml",
			wantErr: true,
		},
		{
			name:    "Bad YAML",
			source:  "test-data/bad.yaml",
			wantErr: true,
		},
		{
			name:    "Unsupported file extension",
			source:  "test-data/unsupported.txt",
			wantErr: true,
		},
		{
			name:    "Good YAML — CCC",
			source:  "test-data/good-ccc.yaml",
			wantErr: false,
		},
		{
			name:    "Good YAML — OSPS",
			source:  "test-data/good-osps.yml",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := Load[ControlCatalog](context.Background(), fileFetcher, tt.source)
			if tt.wantErr {
				assert.Error(t, err, "expected error but got none")
			} else {
				require.NoError(t, err, "unexpected error loading file")
				assert.NotEmpty(t, c.Groups, "catalog should have at least one family")
				assert.NotEmpty(t, c.Controls, "catalog should have at least one control")
				if len(c.Groups) > 0 {
					assert.NotEmpty(t, c.Groups[0].Title, "family title should not be empty")
					assert.NotEmpty(t, c.Groups[0].Description, "family description should not be empty")
				}
			}
		})
	}
}

func TestControlCatalog_LoadFiles_NilImports(t *testing.T) {
	c := &ControlCatalog{}
	err := c.LoadFiles(context.Background(), fileFetcher, []string{
		"test-data/good-ccc.yaml",
		"test-data/good-osps.yml",
	})
	require.NoError(t, err)
	assert.Nil(t, c.Imports, "imports should remain nil when no source files contain imports")
	assert.NotEmpty(t, c.Controls, "controls should be appended from both files")
}

func TestControlCatalog_LoadFiles(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		{
			name:    "File not found",
			source:  "bad-path.yaml",
			wantErr: true,
		},
		{
			name:    "Bad YAML",
			source:  "test-data/bad.yaml",
			wantErr: true,
		},
		{
			name:    "Good YAML — CCC",
			source:  "test-data/good-ccc.yaml",
			wantErr: false,
		},
		{
			name:    "Good YAML — OSPS",
			source:  "test-data/good-osps.yml",
			wantErr: false,
		},
		{
			name:    "Unsupported file extension",
			source:  "test-data/unsupported.txt",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ControlCatalog{}
			err := c.LoadFiles(context.Background(), fileFetcher, []string{tt.source})
			if tt.wantErr {
				assert.Error(t, err, "expected error but got none")
			} else {
				require.NoError(t, err, "unexpected error loading files")
				assert.NotEmpty(t, c.Groups, "catalog should have at least one family")
				assert.NotEmpty(t, c.Controls, "catalog should have at least one control")
			}
		})
	}
}

func TestControlCatalog_LoadNestedCatalog(t *testing.T) {
	t.Run("Empty field name", func(t *testing.T) {
		c := &ControlCatalog{}
		err := c.LoadNestedCatalog(context.Background(), fileFetcher, "test-data/nested-good-ccc.yaml", "")
		assert.Error(t, err, "empty fieldName should return error")
	})

	tests := []struct {
		name      string
		source    string
		fieldName string
		wantErr   bool
	}{
		{
			name:      "File not found",
			source:    "wonky-file-name.yaml",
			fieldName: "catalog",
			wantErr:   true,
		},
		{
			name:      "Bad YAML",
			source:    "test-data/bad.yaml",
			fieldName: "catalog",
			wantErr:   true,
		},
		{
			name:      "Field not in non-nested file",
			source:    "test-data/good-policy.yaml",
			fieldName: "catalog",
			wantErr:   true,
		},
		{
			name:      "Empty nested catalog",
			source:    "test-data/nested-empty.yaml",
			fieldName: "catalog",
			wantErr:   true,
		},
		{
			name:      "Nested field name present",
			source:    "test-data/nested-good-ccc.yaml",
			fieldName: "catalog",
			wantErr:   false,
		},
		{
			name:      "Nested field name not in file",
			source:    "test-data/nested-good-ccc.yaml",
			fieldName: "doesnt-exist",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ControlCatalog{}
			err := c.LoadNestedCatalog(context.Background(), fileFetcher, tt.source, tt.fieldName)
			if tt.wantErr {
				assert.Error(t, err, "expected error but got none")
			} else {
				require.NoError(t, err, "unexpected error loading nested catalog")
				assert.Equal(t, "FINOS Cloud Control Catalog", c.Title,
					"catalog title should match expected value")
				assert.NotEmpty(t, c.Groups,
					"catalog should have at least one family")
				assert.NotEmpty(t, c.Controls,
					"catalog should have at least one control")
				if len(c.Groups) > 0 {
					assert.NotEmpty(t, c.Groups[0].Title,
						"family title should not be empty")
					assert.NotEmpty(t, c.Groups[0].Description,
						"family description should not be empty")
				}
			}
		})
	}
}
