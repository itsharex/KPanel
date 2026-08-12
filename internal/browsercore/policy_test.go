package browsercore

import (
	"context"
	"errors"
	"net/netip"
	"strings"
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
		"https://168.63.129.16/",
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

func TestTargetPolicyEnforcesUTF8URLByteLimit(t *testing.T) {
	policy := NewTargetPolicy(staticResolver{
		"example.com": {netip.MustParseAddr("93.184.216.34")},
	})

	for _, prefix := range []string{
		"https://example.com/?q=",
		"https://example.com/%E8%B7%AF%E5%BE%84?q=",
	} {
		exact := prefix + strings.Repeat("a", DefaultMaxURLBytes-len(prefix))
		if got := len(exact); got != DefaultMaxURLBytes {
			t.Fatalf("fixture byte length = %d, want %d", got, DefaultMaxURLBytes)
		}
		if _, err := policy.Resolve(context.Background(), exact); err != nil {
			t.Fatalf("Resolve(exact UTF-8 limit) error = %v", err)
		}
		if _, err := policy.Resolve(context.Background(), exact+"a"); !errors.Is(err, ErrInvalidTarget) {
			t.Fatalf("Resolve(over UTF-8 limit) error = %v, want %v", err, ErrInvalidTarget)
		}
	}

	unicodePrefix := "https://example.com/路径?q="
	rawExact := unicodePrefix + strings.Repeat("a", DefaultMaxURLBytes-len(unicodePrefix))
	if got := len(rawExact); got != DefaultMaxURLBytes {
		t.Fatalf("raw Unicode fixture byte length = %d, want %d", got, DefaultMaxURLBytes)
	}
	if _, err := policy.Resolve(context.Background(), rawExact); !errors.Is(err, ErrInvalidTarget) {
		t.Fatalf("Resolve(URL expanded beyond normalized limit) error = %v, want %v", err, ErrInvalidTarget)
	}
	if _, err := policy.Resolve(context.Background(), "https://example.com/路径?q=ok"); err != nil {
		t.Fatalf("Resolve(short normalized Unicode URL) error = %v", err)
	}
}
