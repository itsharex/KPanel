package browsercore

import (
	"net/netip"
	"net/url"
	"strings"
)

// SupportsServiceWorkerOrigin implements the secure-context subset required by
// the embedded browser runtime: HTTPS everywhere, plus HTTP loopback origins
// used for local development. Public and LAN HTTP origins fail closed.
func SupportsServiceWorkerOrigin(raw string) bool {
	origin, err := NormalizeOrigin(raw)
	if err != nil {
		return false
	}
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	if parsed.Scheme == "https" {
		return true
	}
	if parsed.Scheme != "http" {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	address, err := netip.ParseAddr(host)
	return err == nil && address.Unmap().IsLoopback()
}
