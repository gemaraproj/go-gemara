// SPDX-License-Identifier: Apache-2.0

package fetcher

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTP_Success(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("title: Test\n"))
	}))
	defer srv.Close()

	f := &HTTP{Client: srv.Client()}
	rc, err := f.Fetch(srv.URL + "/test.yaml")
	require.NoError(t, err)
	defer rc.Close() //nolint:errcheck

	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, "title: Test\n", string(data))
}

func TestHTTP_NotFound(t *testing.T) {
	srv := httptest.NewTLSServer(http.NotFoundHandler())
	defer srv.Close()

	f := &HTTP{Client: srv.Client()}
	_, err := f.Fetch(srv.URL + "/missing.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404 Not Found")
}

func TestHTTP_DefaultClient(t *testing.T) {
	f := &HTTP{}
	assert.Equal(t, http.DefaultClient, f.httpClient())
}
