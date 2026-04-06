// SPDX-License-Identifier: Apache-2.0

package fetcher

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// URI routes to File or HTTP based on the URI scheme.
// Supported schemes: file://, http://, https://.
type URI struct {
	Client *http.Client
}

func (u *URI) Fetch(source string) (io.ReadCloser, error) {
	parsed, err := url.Parse(source)
	if err != nil {
		return nil, fmt.Errorf("invalid URI %q: %w", source, err)
	}
	switch parsed.Scheme {
	case "file":
		return (&File{}).Fetch(parsed.Path)
	case "http", "https":
		return (&HTTP{Client: u.Client}).Fetch(source)
	default:
		return nil, fmt.Errorf("unsupported URI scheme in %q", source)
	}
}
