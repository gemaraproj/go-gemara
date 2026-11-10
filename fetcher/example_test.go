// SPDX-License-Identifier: Apache-2.0

package fetcher_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gemaraproj/go-gemara/fetcher"
)

// This example shows how to configure a [fetcher.URI] that blocks requests
// to private, loopback, and link-local IP addresses at connect time.
// Inspecting the resolved IP inside a custom [net.Dialer] is the
// standard Go approach for SSRF mitigation and is immune to DNS
// rebinding because the check runs on the address that will actually
// be dialed.
func ExampleURI_ssrfSafe() {
	safeDialer := func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}
		for _, ip := range ips {
			if ip.IP.IsLoopback() || ip.IP.IsPrivate() || ip.IP.IsLinkLocalUnicast() {
				return nil, fmt.Errorf("request to private address blocked: %s", ip.IP)
			}
		}
		return (&net.Dialer{}).DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
	}

	f := &fetcher.URI{
		Client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DialContext: safeDialer,
			},
		},
	}

	_, err := f.Fetch(context.Background(), "http://127.0.0.1/internal.yaml")
	fmt.Println(err)
	// Output:
	// failed to fetch URL: Get "http://127.0.0.1/internal.yaml": request to private address blocked: 127.0.0.1
}

// This example shows how [fetcher.URI.BasePath] anchors relative file paths
// to a chosen directory instead of the process working directory, which is
// useful when resolving references from an artifact loaded elsewhere on
// disk. BasePath is a convenience anchor, not a security boundary: relative
// paths containing ".." and absolute paths both resolve outside it, so
// applications resolving references from untrusted artifacts must validate
// paths before fetching.
func ExampleURI_basePath() {
	base, err := os.MkdirTemp("", "gemara-basepath")
	if err != nil {
		panic(err)
	}
	defer func() { _ = os.RemoveAll(base) }()

	inner := filepath.Join(base, "inner")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(inner, "notes.txt"), []byte("inside BasePath"), 0o600); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(base, "artifact.yaml"), []byte("outside BasePath"), 0o600); err != nil {
		panic(err)
	}

	f := &fetcher.URI{BasePath: inner}

	rc, err := f.Fetch(context.Background(), "notes.txt")
	if err != nil {
		panic(err)
	}
	data, err := io.ReadAll(rc)
	if err != nil {
		panic(err)
	}
	if err := rc.Close(); err != nil {
		panic(err)
	}
	fmt.Println(string(data))

	// ".." is not blocked by BasePath: this path escapes the anchor.
	rc, err = f.Fetch(context.Background(), "../artifact.yaml")
	if err != nil {
		panic(err)
	}
	data, err = io.ReadAll(rc)
	if err != nil {
		panic(err)
	}
	if err := rc.Close(); err != nil {
		panic(err)
	}
	fmt.Println(string(data))
	// Output:
	// inside BasePath
	// outside BasePath
}
