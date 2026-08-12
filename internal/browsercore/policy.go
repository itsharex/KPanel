package browsercore

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
)

// DefaultMaxURLBytes matches the browser transport's UTF-8 target URL budget.
// Keep this bounded below MaxHeaderBytes so the URL and relay metadata can share
// one request header block without weakening the target policy.
const DefaultMaxURLBytes = 16 << 10

var (
	ErrInvalidTarget  = errors.New("invalid browser target")
	ErrBlockedTarget  = errors.New("browser target is not public")
	ErrTargetNotFound = errors.New("browser target did not resolve")
)

type Resolver interface {
	LookupNetIP(context.Context, string, string) ([]netip.Addr, error)
}

type Target struct {
	URL       *url.URL
	Addresses []netip.Addr
}

type TargetPolicy struct {
	Resolver    Resolver
	MaxURLBytes int
}

func NewTargetPolicy(resolver Resolver) *TargetPolicy {
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	return &TargetPolicy{Resolver: resolver, MaxURLBytes: DefaultMaxURLBytes}
}

func (p *TargetPolicy) Resolve(ctx context.Context, raw string) (Target, error) {
	maxURLBytes := p.MaxURLBytes
	if maxURLBytes <= 0 {
		maxURLBytes = DefaultMaxURLBytes
	}
	if raw == "" || len(raw) > maxURLBytes || strings.TrimSpace(raw) != raw {
		return Target{}, ErrInvalidTarget
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Opaque != "" || parsed.Host == "" || parsed.User != nil {
		return Target{}, ErrInvalidTarget
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return Target{}, ErrInvalidTarget
	}
	if parsed.Hostname() == "" || strings.ContainsAny(parsed.Host, "\r\n\t ") {
		return Target{}, ErrInvalidTarget
	}

	port := parsed.Port()
	if port != "" {
		number, parseErr := strconv.Atoi(port)
		if parseErr != nil || number < 1 || number > 65535 {
			return Target{}, ErrInvalidTarget
		}
	}

	host, err := normalizeHostname(parsed.Hostname())
	if err != nil || reservedHostname(host) {
		return Target{}, ErrBlockedTarget
	}

	normalized := *parsed
	if port == "" {
		if literal, parseErr := netip.ParseAddr(host); parseErr == nil && literal.Is6() {
			normalized.Host = "[" + host + "]"
		} else {
			normalized.Host = host
		}
	} else {
		normalized.Host = net.JoinHostPort(host, port)
	}
	normalized.Fragment = ""
	normalized.RawFragment = ""
	if len(normalized.String()) > maxURLBytes {
		return Target{}, ErrInvalidTarget
	}

	addresses, err := p.lookup(ctx, host)
	if err != nil {
		return Target{}, err
	}
	for _, address := range addresses {
		if !publicAddress(address) {
			return Target{}, fmt.Errorf("%w: %s", ErrBlockedTarget, address)
		}
	}
	return Target{URL: &normalized, Addresses: addresses}, nil
}

func (p *TargetPolicy) lookup(ctx context.Context, host string) ([]netip.Addr, error) {
	if literal, err := netip.ParseAddr(host); err == nil {
		return []netip.Addr{literal.Unmap()}, nil
	}
	addresses, err := p.Resolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTargetNotFound, err)
	}
	unique := make([]netip.Addr, 0, len(addresses))
	seen := make(map[netip.Addr]struct{}, len(addresses))
	for _, address := range addresses {
		address = address.Unmap()
		if !address.IsValid() {
			continue
		}
		if _, exists := seen[address]; exists {
			continue
		}
		seen[address] = struct{}{}
		unique = append(unique, address)
	}
	if len(unique) == 0 {
		return nil, ErrTargetNotFound
	}
	return unique, nil
}

func normalizeHostname(host string) (string, error) {
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	if literal, err := netip.ParseAddr(host); err == nil {
		return literal.Unmap().String(), nil
	}
	if host == "" || len(host) > 253 {
		return "", ErrInvalidTarget
	}
	for _, character := range host {
		if character > 127 || !((character >= 'a' && character <= 'z') ||
			(character >= '0' && character <= '9') || character == '-' || character == '.') {
			return "", ErrInvalidTarget
		}
	}
	for _, label := range strings.Split(host, ".") {
		if label == "" || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return "", ErrInvalidTarget
		}
	}
	return host, nil
}

func reservedHostname(host string) bool {
	return host == "localhost" || host == "localhost.localdomain" ||
		strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") ||
		strings.HasSuffix(host, ".internal") || strings.HasSuffix(host, ".home.arpa")
}

var nonPublicPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.31.196.0/24"),
	netip.MustParsePrefix("192.52.193.0/24"),
	netip.MustParsePrefix("192.88.99.0/24"),
	netip.MustParsePrefix("192.175.48.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("64:ff9b::/96"),
	netip.MustParsePrefix("64:ff9b:1::/48"),
	netip.MustParsePrefix("100::/64"),
	netip.MustParsePrefix("100:0:0:1::/64"),
	netip.MustParsePrefix("2001::/23"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("2002::/16"),
	netip.MustParsePrefix("2620:4f:8000::/48"),
	netip.MustParsePrefix("3fff::/20"),
	netip.MustParsePrefix("5f00::/16"),
}

func publicAddress(address netip.Addr) bool {
	address = address.Unmap()
	if !address.IsValid() || !address.IsGlobalUnicast() || address.IsPrivate() ||
		address.IsLoopback() || address.IsLinkLocalUnicast() || address.IsMulticast() ||
		address.IsUnspecified() {
		return false
	}
	for _, prefix := range nonPublicPrefixes {
		if prefix.Contains(address) {
			return false
		}
	}
	return true
}
