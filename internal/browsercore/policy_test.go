package browsercore

import (
	"context"
	"errors"
	"net/netip"
	"testing"
)

type staticResolver map[string][]netip.Addr

func (r staticResolver) LookupNetIP(_ context.Context, _, host string) ([]netip.Addr, error) {
	addresses, ok := r[host]
	if !ok {
		return nil, errors.New("not found")
	}
	return addresses, nil
}

func TestTargetPolicyAcceptsOnlyPublicHTTPSTargets(t *testing.T) {
	policy := NewTargetPolicy(staticResolver{
		"example.com": {netip.MustParseAddr("93.184.216.34")},
	})
	target, err := policy.Resolve(context.Background(), "https://Example.COM:8443/path?q=1#fragment")
	if err != nil {
		t.Fatal(err)
	}
	if got := target.URL.String(); got != "https://example.com:8443/path?q=1" {
		t.Fatalf("normalized URL = %q", got)
	}
	if len(target.Addresses) != 1 || target.Addresses[0].String() != "93.184.216.34" {
		t.Fatalf("addresses = %#v", target.Addresses)
	}
}

func TestTargetPolicyRejectsPrivateOrMixedDNSAnswers(t *testing.T) {
	policy := NewTargetPolicy(staticResolver{
		"private.example": {netip.MustParseAddr("10.0.0.2")},
		"rebind.example": {
			netip.MustParseAddr("93.184.216.34"),
			netip.MustParseAddr("127.0.0.1"),
		},
	})
	for _, raw := range []string{
		"http://127.0.0.1/",
		"http://[::1]/",
		"https://private.example/",
		"https://rebind.example/",
		"https://service.local/",
		"https://user:secret@example.com/",
		"file:///etc/passwd",
		"javascript:alert(1)",
	} {
		if _, err := policy.Resolve(context.Background(), raw); err == nil {
			t.Errorf("Resolve(%q) succeeded", raw)
		}
	}
}

func TestTargetPolicyRejectsSpecialUsePublicLookingAddresses(t *testing.T) {
	policy := NewTargetPolicy(staticResolver{})
	for _, raw := range []string{
		"https://192.0.2.1/",
		"https://192.88.99.2/",
		"https://198.51.100.1/",
		"https://203.0.113.1/",
		"https://[64:ff9b::7f00:1]/",
		"https://[100::1]/",
		"https://[2001:2::1]/",
		"https://[2001:db8::1]/",
		"https://[2002:7f00:1::]/",
		"https://[3fff::1]/",
	} {
		if _, err := policy.Resolve(context.Background(), raw); !errors.Is(err, ErrBlockedTarget) {
			t.Errorf("Resolve(%q) error = %v", raw, err)
		}
	}
}

func TestTargetPolicyAcceptsBrowserNormalizedInternationalHostname(t *testing.T) {
	policy := NewTargetPolicy(staticResolver{
		"xn--fsqu00a.xn--0zwm56d": {netip.MustParseAddr("1.1.1.1")},
	})
	target, err := policy.Resolve(context.Background(), "https://xn--fsqu00a.xn--0zwm56d/path")
	if err != nil {
		t.Fatal(err)
	}
	if got := target.URL.Hostname(); got != "xn--fsqu00a.xn--0zwm56d" {
		t.Fatalf("hostname = %q", got)
	}
}
