package remotedownload

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"testing"
	"time"
)

type resolverFunc func(context.Context, string, string) ([]netip.Addr, error)

func (function resolverFunc) LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error) {
	return function(ctx, network, host)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestValidateURLAcceptsOnlyUnambiguousPublicHTTPURLSyntax(t *testing.T) {
	valid := []string{
		"https://downloads.example.com/releases/app.tar.zst?signature=secret",
		"http://8.8.8.8/file.iso",
		"https://[2606:4700:4700::1111]:8443/file",
	}
	for _, raw := range valid {
		parsed, err := ValidateURL(raw)
		if err != nil || parsed.String() == "" {
			t.Fatalf("ValidateURL(%q) = %#v, %v", raw, parsed, err)
		}
	}
	invalid := []string{
		"", "ftp://downloads.example.com/file", "https://user:secret@downloads.example.com/file",
		"https://downloads.example.com/file#fragment", "https://downloads.example.com\\file",
		"https://localhost/file", "https://intranet/file", "https://0177.0.0.1/file",
		"https://downloads.example.com:0/file", "https://downloads.example.com:65536/file",
		"https://[fe80::1%25eth0]/file", "https://下载.example.com/file",
		"https://downloads.example.com./file",
		"https://downloads.example.com/file\r\nX-Test: yes",
	}
	for _, raw := range invalid {
		if _, err := ValidateURL(raw); !errors.Is(err, ErrInvalidURL) {
			t.Fatalf("ValidateURL(%q) error = %v, want ErrInvalidURL", raw, err)
		}
	}
}

func TestResolveRejectsEveryRestrictedOrMixedDNSAnswer(t *testing.T) {
	tests := []struct {
		name      string
		addresses []netip.Addr
	}{
		{name: "loopback", addresses: []netip.Addr{netip.MustParseAddr("127.0.0.1")}},
		{name: "private", addresses: []netip.Addr{netip.MustParseAddr("10.0.0.8")}},
		{name: "link local metadata", addresses: []netip.Addr{netip.MustParseAddr("169.254.169.254")}},
		{name: "CGNAT", addresses: []netip.Addr{netip.MustParseAddr("100.64.0.1")}},
		{name: "documentation", addresses: []netip.Addr{netip.MustParseAddr("192.0.2.1")}},
		{name: "IPv4 mapped", addresses: []netip.Addr{netip.MustParseAddr("::ffff:127.0.0.1")}},
		{name: "NAT64", addresses: []netip.Addr{netip.MustParseAddr("64:ff9b::808:808")}},
		{name: "mixed", addresses: []netip.Addr{netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("10.0.0.8")}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := NewClient(Config{Resolver: resolverFunc(func(context.Context, string, string) ([]netip.Addr, error) {
				return test.addresses, nil
			})})
			if _, err := client.resolve(context.Background(), "downloads.example.com"); !errors.Is(err, ErrAddressBlocked) {
				t.Fatalf("resolve error = %v, want ErrAddressBlocked", err)
			}
		})
	}
}

func TestDialContextPinsValidatedIPAddress(t *testing.T) {
	var dialed string
	client := NewClient(Config{
		Resolver: resolverFunc(func(context.Context, string, string) ([]netip.Addr, error) {
			return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
		}),
		Dialer: func(_ context.Context, network, address string) (net.Conn, error) {
			if network != "tcp" {
				t.Fatalf("network = %q", network)
			}
			dialed = address
			return nil, errors.New("test dial stopped")
		},
	})
	_, _ = client.dialContext(context.Background(), "tcp", "downloads.example.com:443")
	if dialed != "8.8.8.8:443" {
		t.Fatalf("dialed = %q, want validated IP", dialed)
	}
}

func TestDialContextUsesOneDeadlineAcrossResolutionAndAddresses(t *testing.T) {
	dialCalls := 0
	client := NewClient(Config{
		ConnectTimeout: 20 * time.Millisecond,
		Resolver: resolverFunc(func(context.Context, string, string) ([]netip.Addr, error) {
			return []netip.Addr{netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("1.1.1.1")}, nil
		}),
		Dialer: func(ctx context.Context, _, _ string) (net.Conn, error) {
			dialCalls++
			<-ctx.Done()
			return nil, ctx.Err()
		},
	})
	started := time.Now()
	_, err := client.dialContext(context.Background(), "tcp", "downloads.example.com:443")
	if err == nil || dialCalls != 1 || time.Since(started) > time.Second {
		t.Fatalf("err=%v dialCalls=%d elapsed=%s", err, dialCalls, time.Since(started))
	}
}

func TestDialContextValidatesAllAnswersBeforeLimitingDialCandidates(t *testing.T) {
	publicAddresses := []netip.Addr{
		netip.MustParseAddr("8.8.8.1"),
		netip.MustParseAddr("8.8.8.2"),
		netip.MustParseAddr("8.8.8.3"),
		netip.MustParseAddr("8.8.8.4"),
		netip.MustParseAddr("8.8.8.5"),
		netip.MustParseAddr("8.8.8.6"),
		netip.MustParseAddr("8.8.8.7"),
		netip.MustParseAddr("8.8.8.8"),
		netip.MustParseAddr("1.1.1.1"),
	}
	t.Run("ninth restricted answer rejects before dialing", func(t *testing.T) {
		dialCalls := 0
		answers := append([]netip.Addr(nil), publicAddresses[:8]...)
		answers = append(answers, netip.MustParseAddr("10.0.0.8"))
		client := NewClient(Config{
			Resolver: resolverFunc(func(context.Context, string, string) ([]netip.Addr, error) {
				return answers, nil
			}),
			Dialer: func(context.Context, string, string) (net.Conn, error) {
				dialCalls++
				return nil, errors.New("test dial stopped")
			},
		})
		if _, err := client.dialContext(context.Background(), "tcp", "downloads.example.com:443"); !errors.Is(err, ErrAddressBlocked) || dialCalls != 0 {
			t.Fatalf("error=%v dialCalls=%d, want blocked before dialing", err, dialCalls)
		}
	})
	t.Run("only eight validated answers are dialed", func(t *testing.T) {
		dialCalls := 0
		client := NewClient(Config{
			Resolver: resolverFunc(func(context.Context, string, string) ([]netip.Addr, error) {
				return publicAddresses, nil
			}),
			Dialer: func(context.Context, string, string) (net.Conn, error) {
				dialCalls++
				return nil, errors.New("test dial stopped")
			},
		})
		if _, err := client.dialContext(context.Background(), "tcp", "downloads.example.com:443"); err == nil || dialCalls != maxDialAddresses {
			t.Fatalf("error=%v dialCalls=%d, want %d", err, dialCalls, maxDialAddresses)
		}
	})
}

func TestOpenValidatesRedirectAndStripsReferer(t *testing.T) {
	client := NewClient(Config{IdleTimeout: time.Second})
	requests := 0
	client.httpClient.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests++
		if requests == 1 {
			return &http.Response{
				StatusCode: http.StatusFound,
				Header:     http.Header{"Location": []string{"https://cdn.example.net/file.tar.zst"}},
				Body:       io.NopCloser(strings.NewReader("")), Request: request,
			}, nil
		}
		if request.Header.Get("Referer") != "" {
			t.Fatalf("redirect leaked Referer %q", request.Header.Get("Referer"))
		}
		return &http.Response{
			StatusCode: http.StatusOK, Header: make(http.Header), ContentLength: 2,
			Body: io.NopCloser(strings.NewReader("ok")), Request: request,
		}, nil
	})
	response, err := client.Open(context.Background(), "https://downloads.example.com/file?token=secret")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	content, err := io.ReadAll(response.Body)
	if err != nil || string(content) != "ok" || requests != 2 {
		t.Fatalf("content=%q requests=%d err=%v", content, requests, err)
	}
}

func TestOpenRejectsHTTPSDowngradeWithoutFollowingIt(t *testing.T) {
	client := NewClient(Config{})
	requests := 0
	client.httpClient.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests++
		return &http.Response{
			StatusCode: http.StatusFound,
			Header:     http.Header{"Location": []string{"http://cdn.example.net/file"}},
			Body:       io.NopCloser(strings.NewReader("")), Request: request,
		}, nil
	})
	if _, err := client.Open(context.Background(), "https://downloads.example.com/file"); !errors.Is(err, ErrRedirectRejected) {
		t.Fatalf("error = %v, want ErrRedirectRejected", err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, downgrade target was contacted", requests)
	}
}

func TestClientDisablesProxyAndLimitsRedirects(t *testing.T) {
	client := NewClient(Config{})
	transport, ok := client.httpClient.Transport.(*http.Transport)
	if !ok || transport.Proxy != nil {
		t.Fatalf("transport proxy = %#v", transport)
	}
	requests := 0
	client.httpClient.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests++
		return &http.Response{
			StatusCode: http.StatusFound,
			Header:     http.Header{"Location": []string{"https://cdn" + string(rune('a'+requests)) + ".example.net/file"}},
			Body:       io.NopCloser(strings.NewReader("")), Request: request,
		}, nil
	})
	if _, err := client.Open(context.Background(), "https://downloads.example.com/file"); !errors.Is(err, ErrRedirectRejected) {
		t.Fatalf("error = %v, want ErrRedirectRejected", err)
	}
	if requests != MaxRedirects+1 {
		t.Fatalf("requests = %d, want initial request plus %d redirects", requests, MaxRedirects)
	}
}

func TestSourceDisplayOmitsPathQueryAndFragment(t *testing.T) {
	parsed, err := ValidateURL("https://downloads.example.com:8443/private/file?token=secret")
	if err != nil {
		t.Fatal(err)
	}
	if display := SourceDisplay(parsed); display != "https://downloads.example.com:8443" {
		t.Fatalf("display = %q", display)
	}
}

func TestOpenReturnsSanitizedStatusAndRejectsEncodedBody(t *testing.T) {
	client := NewClient(Config{})
	client.httpClient.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusForbidden, Header: make(http.Header),
			Body: io.NopCloser(strings.NewReader("secret upstream body")), Request: request,
		}, nil
	})
	_, err := client.Open(context.Background(), "https://downloads.example.com/file?token=secret")
	var statusError *StatusError
	if !errors.As(err, &statusError) || statusError.StatusCode != http.StatusForbidden ||
		strings.Contains(err.Error(), "token") || strings.Contains(err.Error(), "upstream body") {
		t.Fatalf("status error was not sanitized: %v", err)
	}

	client.httpClient.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK, Header: http.Header{"Content-Encoding": []string{"gzip"}},
			Body: io.NopCloser(strings.NewReader("encoded")), Request: request,
		}, nil
	})
	if _, err := client.Open(context.Background(), "https://downloads.example.com/file"); !errors.Is(err, ErrEncoding) {
		t.Fatalf("encoding error = %v", err)
	}
}

func TestOpenRejectsPartialAndNonFinalSuccessResponses(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		header http.Header
		want   error
	}{
		{name: "partial status", status: http.StatusPartialContent, header: make(http.Header), want: ErrPartialContent},
		{name: "accepted without file", status: http.StatusAccepted, header: make(http.Header)},
		{name: "no content", status: http.StatusNoContent, header: make(http.Header)},
		{name: "content range on 200", status: http.StatusOK, header: http.Header{"Content-Range": []string{"bytes 0-3/8"}}, want: ErrPartialContent},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := NewClient(Config{})
			client.httpClient.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: test.status, Header: test.header,
					Body: io.NopCloser(strings.NewReader("part")), Request: request,
				}, nil
			})
			_, err := client.Open(context.Background(), "https://downloads.example.com/file")
			if test.want != nil {
				if !errors.Is(err, test.want) {
					t.Fatalf("error=%v, want %v", err, test.want)
				}
				return
			}
			var statusError *StatusError
			if !errors.As(err, &statusError) || statusError.StatusCode != test.status {
				t.Fatalf("error=%v, want status %d", err, test.status)
			}
		})
	}
}

type blockingBody struct {
	closed chan struct{}
	once   sync.Once
}

func (body *blockingBody) Read([]byte) (int, error) {
	<-body.closed
	return 0, io.ErrClosedPipe
}

func (body *blockingBody) Close() error {
	body.once.Do(func() { close(body.closed) })
	return nil
}

func TestIdleReadCloserReturnsExplicitTimeout(t *testing.T) {
	source := &blockingBody{closed: make(chan struct{})}
	reader := newIdleReadCloser(source, 10*time.Millisecond)
	defer reader.Close()
	started := time.Now()
	_, err := reader.Read(make([]byte, 1))
	if !errors.Is(err, ErrIdleTimeout) {
		t.Fatalf("error = %v, want ErrIdleTimeout", err)
	}
	if time.Since(started) > time.Second {
		t.Fatal("idle timeout did not interrupt the blocked read")
	}
}

func TestIdleReadCloserDoesNotCountTimeOutsideReadCalls(t *testing.T) {
	reader := newIdleReadCloser(io.NopCloser(strings.NewReader("ab")), 10*time.Millisecond)
	defer reader.Close()
	time.Sleep(25 * time.Millisecond)
	buffer := make([]byte, 1)
	if count, err := reader.Read(buffer); err != nil || count != 1 || string(buffer) != "a" {
		t.Fatalf("first read count=%d content=%q err=%v", count, buffer, err)
	}
	time.Sleep(25 * time.Millisecond)
	if count, err := reader.Read(buffer); err != nil || count != 1 || string(buffer) != "b" {
		t.Fatalf("second read count=%d content=%q err=%v", count, buffer, err)
	}
}
