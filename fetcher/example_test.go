// SPDX-License-Identifier: Apache-2.0

package fetcher_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
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
