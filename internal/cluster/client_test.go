package cluster

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"strings"
	"sync"
	"testing"
)

type staticResolver map[string][]net.IP

func (r staticResolver) LookupNetIP(_ context.Context, _, host string) ([]net.IP, error) {
	addresses, ok := r[host]
	if !ok {
		return nil, errors.New("host not found")
	}
	return append([]net.IP(nil), addresses...), nil
}

type sequenceResolver struct {
	mu        sync.Mutex
	responses [][]net.IP
}

func (r *sequenceResolver) LookupNetIP(_ context.Context, _, _ string) ([]net.IP, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.responses) == 0 {
		return nil, errors.New("no resolver response")
	}
	response := append([]net.IP(nil), r.responses[0]...)
	r.responses = r.responses[1:]
	return response, nil
}

func TestNormalizeOrigin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "fqdn", raw: "https://panel.example.com", want: "https://panel.example.com"},
		{name: "trim and lowercase", raw: "  https://PANEL.EXAMPLE.COM  ", want: "https://panel.example.com"},
		{name: "default port removed", raw: "https://panel.example.com:443", want: "https://panel.example.com"},
		{name: "custom port retained", raw: "https://panel.example.com:8443", want: "https://panel.example.com:8443"},
		{
			name: "ipv6 canonicalized",
			raw:  "https://[2606:4700:4700:0:0:0:0:1111]:8443",
			want: "https://[2606:4700:4700::1111]:8443",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := NormalizeOrigin(test.raw)
			if err != nil {
				t.Fatalf("NormalizeOrigin(%q) returned error: %v", test.raw, err)
			}
			if got != test.want {
				t.Fatalf("NormalizeOrigin(%q) = %q, want %q", test.raw, got, test.want)
			}
		})
	}
}

func TestNormalizeOriginRejectsUnsafeAndAmbiguousInputs(t *testing.T) {
	t.Parallel()

	inputs := []string{
		"",
		"http://panel.example.com",
		"https://panel.example.com/",
		"https://panel.example.com/path",
		"https://panel.example.com?next=https://127.0.0.1",
		"https://panel.example.com#fragment",
		"https://user:password@panel.example.com",
		"https://panel",
		"https://123456",
		"https://127.0.0.1.",
		"https://panel..example.com",
		"https://-panel.example.com",
		"https://panel-.example.com",
		"https://面板.example.com",
		"https://panel.example.com:0",
		"https://panel.example.com:65536",
		"https://[fe80::1%25eth0]",
		"https://panel.example.com\n.evil.example",
		strings.Repeat("a", 513),
	}
	for _, input := range inputs {
		input := input
		t.Run(input, func(t *testing.T) {
			t.Parallel()
			if got, err := NormalizeOrigin(input); !errors.Is(err, ErrInvalidOrigin) {
				t.Fatalf("NormalizeOrigin(%q) = %q, %v; want ErrInvalidOrigin", input, got, err)
			}
		})
	}
}

func TestRemoteClientResolveEnforcesSSRFPolicy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		host         string
		resolved     []net.IP
		privateCIDRs []string
		want         []netip.Addr
		wantErr      error
	}{
		{
			name:     "public address",
			host:     "panel.example.com",
			resolved: []net.IP{net.ParseIP("8.8.8.8")},
			want:     []netip.Addr{netip.MustParseAddr("8.8.8.8")},
		},
		{
			name:     "duplicates removed",
			host:     "panel.example.com",
			resolved: []net.IP{net.ParseIP("8.8.8.8"), net.ParseIP("8.8.8.8")},
			want:     []netip.Addr{netip.MustParseAddr("8.8.8.8")},
		},
		{
			name:     "private blocked by default",
			host:     "panel.example.com",
			resolved: []net.IP{net.ParseIP("10.1.2.3")},
			wantErr:  ErrPrivateOrigin,
		},
		{
			name:         "explicit private allowlist",
			host:         "panel.example.com",
			resolved:     []net.IP{net.ParseIP("10.1.2.3")},
			privateCIDRs: []string{"10.1.0.0/16"},
			want:         []netip.Addr{netip.MustParseAddr("10.1.2.3")},
		},
		{
			name:         "allowlist cannot enable loopback",
			host:         "panel.example.com",
			resolved:     []net.IP{net.ParseIP("127.0.0.1")},
			privateCIDRs: []string{"127.0.0.0/8"},
			wantErr:      ErrPrivateOrigin,
		},
		{
			name:         "allowlist cannot enable metadata link local",
			host:         "panel.example.com",
			resolved:     []net.IP{net.ParseIP("169.254.169.254")},
			privateCIDRs: []string{"169.254.0.0/16"},
			wantErr:      ErrPrivateOrigin,
		},
		{
			name:     "mixed public and private fails closed",
			host:     "panel.example.com",
			resolved: []net.IP{net.ParseIP("8.8.8.8"), net.ParseIP("10.0.0.5")},
			wantErr:  ErrPrivateOrigin,
		},
		{
			name:     "mapped loopback is unmapped and blocked",
			host:     "panel.example.com",
			resolved: []net.IP{net.ParseIP("::ffff:127.0.0.1")},
			wantErr:  ErrPrivateOrigin,
		},
		{
			name:     "documentation range blocked",
			host:     "panel.example.com",
			resolved: []net.IP{net.ParseIP("203.0.113.10")},
			wantErr:  ErrPrivateOrigin,
		},
		{
			name:     "benchmark range blocked",
			host:     "panel.example.com",
			resolved: []net.IP{net.ParseIP("198.18.0.1")},
			wantErr:  ErrPrivateOrigin,
		},
		{
			name:     "well known NAT64 metadata mapping blocked",
			host:     "panel.example.com",
			resolved: []net.IP{net.ParseIP("64:ff9b::a9fe:a9fe")},
			wantErr:  ErrPrivateOrigin,
		},
		{
			name:     "deprecated IPv4 compatible metadata mapping blocked",
			host:     "panel.example.com",
			resolved: []net.IP{net.ParseIP("::a9fe:a9fe")},
			wantErr:  ErrPrivateOrigin,
		},
		{
			name:     "local use NAT64 mapping blocked",
			host:     "panel.example.com",
			resolved: []net.IP{net.ParseIP("64:ff9b:1::a9fe:a9fe")},
			wantErr:  ErrPrivateOrigin,
		},
		{
			name:     "6to4 metadata mapping blocked",
			host:     "panel.example.com",
			resolved: []net.IP{net.ParseIP("2002:a9fe:a9fe::1")},
			wantErr:  ErrPrivateOrigin,
		},
		{
			name:     "teredo transition address blocked",
			host:     "panel.example.com",
			resolved: []net.IP{net.ParseIP("2001:0000:4136:e378:8000:63bf:3fff:fdd2")},
			wantErr:  ErrPrivateOrigin,
		},
		{
			name:     "deprecated site local address blocked",
			host:     "panel.example.com",
			resolved: []net.IP{net.ParseIP("fec0::1")},
			wantErr:  ErrPrivateOrigin,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			client, err := NewRemoteClient(RemoteClientConfig{
				PrivateCIDRs: test.privateCIDRs,
				Resolver: staticResolver{
					test.host: test.resolved,
				},
			})
			if err != nil {
				t.Fatalf("NewRemoteClient() error: %v", err)
			}
			got, err := client.resolve(context.Background(), test.host)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("resolve() error = %v, want %v", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolve() error: %v", err)
			}
			if len(got) != len(test.want) {
				t.Fatalf("resolve() = %v, want %v", got, test.want)
			}
			for index := range got {
				if got[index] != test.want[index] {
					t.Fatalf("resolve() = %v, want %v", got, test.want)
				}
			}
		})
	}
}

func TestRemoteClientDialPinsValidatedAddressAndRejectsRebinding(t *testing.T) {
	t.Parallel()

	resolver := &sequenceResolver{responses: [][]net.IP{
		{net.ParseIP("8.8.8.8")},
		{net.ParseIP("127.0.0.1")},
	}}
	var (
		mu      sync.Mutex
		dialled []string
	)
	client, err := NewRemoteClient(RemoteClientConfig{
		Resolver: resolver,
		Dialer: func(_ context.Context, _, address string) (net.Conn, error) {
			mu.Lock()
			dialled = append(dialled, address)
			mu.Unlock()
			return nil, errors.New("test dial stopped")
		},
	})
	if err != nil {
		t.Fatalf("NewRemoteClient() error: %v", err)
	}

	if _, err := client.dialContext(context.Background(), "tcp", "panel.example.com:443"); err == nil {
		t.Fatal("first dialContext() unexpectedly succeeded")
	}
	mu.Lock()
	if len(dialled) != 1 || dialled[0] != "8.8.8.8:443" {
		t.Fatalf("first dial attempted %v, want exact validated address", dialled)
	}
	mu.Unlock()

	if _, err := client.dialContext(context.Background(), "tcp", "panel.example.com:443"); !errors.Is(err, ErrPrivateOrigin) {
		t.Fatalf("second dialContext() error = %v, want ErrPrivateOrigin", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(dialled) != 1 {
		t.Fatalf("rebound private address reached dialer: %v", dialled)
	}
}
