// SPDX-License-Identifier: Apache-2.0

package fetcher

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

// HTTP reads from HTTP/HTTPS URLs.
// If Client is nil, http.DefaultClient is used.
type HTTP struct {
	Client *http.Client
}

func (h *HTTP) httpClient() *http.Client {
	if h.Client != nil {
		return h.Client
	}
	return http.DefaultClient
}

func (h *HTTP) Fetch(source string) (io.ReadCloser, error) {
	resp, err := h.httpClient().Get(source) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Printf("failed to close response body: %v", err)
			}
		}()
		return nil, fmt.Errorf("failed to fetch URL; response status: %v", resp.Status)
	}
	return resp.Body, nil
}
