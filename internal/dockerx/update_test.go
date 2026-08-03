package dockerx

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

const testUpdateDigest = "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestRemoteImageDigestUsesOfficialRegistryWithoutFallback(t *testing.T) {
	var requests []string
	countryCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests = append(requests, request.RequestURI)
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"Descriptor":{"digest":"` + testUpdateDigest + `"}}`))
	}))
	defer server.Close()
	client := testHTTPClient(server)
	client.ConfigureImageUpdateFallback(func(context.Context) (string, error) {
		countryCalls++
		return "CN", nil
	})

	digest, err := client.remoteImageDigestForUpdate(context.Background(), "redis:alpine")
	if err != nil || digest != testUpdateDigest {
		t.Fatalf("digest = %q, err = %v", digest, err)
	}
	if countryCalls != 1 || len(requests) != 1 || !strings.Contains(requests[0], "redis:alpine") {
		t.Fatalf("requests = %#v, countryCalls = %d", requests, countryCalls)
	}
}

func TestRemoteImageDigestFallsBackForMainlandDockerHubImage(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests = append(requests, request.RequestURI)
		switch {
		case strings.Contains(request.RequestURI, "docker.1ms.run"):
			http.Error(response, "temporary mirror failure", http.StatusBadGateway)
		case strings.Contains(request.RequestURI, "gh.kejilion.pro"):
			response.Header().Set("Content-Type", "application/json")
			_, _ = response.Write([]byte(`{"Descriptor":{"digest":"` + testUpdateDigest + `"}}`))
		default:
			http.Error(response, "official registry unavailable", http.StatusServiceUnavailable)
		}
	}))
	defer server.Close()
	client := testHTTPClient(server)
	client.ConfigureImageUpdateFallback(func(context.Context) (string, error) {
		return "cn", nil
	})

	digest, err := client.remoteImageDigestForUpdate(context.Background(), "redis:alpine")
	if err != nil || digest != testUpdateDigest {
		t.Fatalf("digest = %q, err = %v", digest, err)
	}
	if len(requests) != 3 || !strings.Contains(requests[1], "docker.1ms.run") ||
		!strings.Contains(requests[1], "library") ||
		!strings.Contains(requests[2], "gh.kejilion.pro") {
		t.Fatalf("fallback requests = %#v", requests)
	}
}

func TestRemoteImageDigestFallsBackWhenCountryLookupIsUnavailable(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests = append(requests, request.RequestURI)
		if strings.Contains(request.RequestURI, "docker.1ms.run") {
			response.Header().Set("Content-Type", "application/json")
			_, _ = response.Write([]byte(`{"Descriptor":{"digest":"` + testUpdateDigest + `"}}`))
			return
		}
		http.Error(response, "official registry unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	client := testHTTPClient(server)
	client.ConfigureImageUpdateFallback(func(context.Context) (string, error) {
		return "", errors.New("country service blocked")
	})

	if _, err := client.remoteImageDigestForUpdate(context.Background(), "cloudreve/cloudreve:latest"); err != nil {
		t.Fatal(err)
	}
	if len(requests) != 2 || !strings.Contains(requests[1], "docker.1ms.run") {
		t.Fatalf("fallback requests = %#v", requests)
	}
}

func TestRemoteImageDigestDoesNotProxyKnownOverseasOrPrivateRegistry(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests = append(requests, request.RequestURI)
		http.Error(response, "registry unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	client := testHTTPClient(server)
	client.ConfigureImageUpdateFallback(func(context.Context) (string, error) {
		return "US", nil
	})

	if _, err := client.remoteImageDigestForUpdate(context.Background(), "redis:alpine"); err == nil {
		t.Fatal("known overseas lookup unexpectedly used an accelerator")
	}
	if _, err := client.remoteImageDigestForUpdate(context.Background(), "ghcr.io/example/app:latest"); err == nil {
		t.Fatal("private registry lookup unexpectedly succeeded")
	}
	if len(requests) != 2 {
		t.Fatalf("requests = %#v, want only two official requests", requests)
	}
}

func TestNormalizedDockerHubReference(t *testing.T) {
	tests := []struct {
		image      string
		repository string
		tag        string
		ok         bool
	}{
		{image: "redis", repository: "library/redis", tag: "latest", ok: true},
		{image: "redis:alpine", repository: "library/redis", tag: "alpine", ok: true},
		{image: "cloudreve/cloudreve:latest", repository: "cloudreve/cloudreve", tag: "latest", ok: true},
		{image: "docker.io/kjlion/kejilion-panel:latest", repository: "kjlion/kejilion-panel", tag: "latest", ok: true},
		{image: "index.docker.io/library/nginx:alpine", repository: "library/nginx", tag: "alpine", ok: true},
		{image: "ghcr.io/example/app:latest"},
		{image: "registry.example:5000/example/app:latest"},
		{image: "example/app@sha256:" + strings.Repeat("a", 64)},
		{image: "../example:latest"},
		{image: "UPPER/example:latest"},
	}
	for _, test := range tests {
		t.Run(test.image, func(t *testing.T) {
			repository, tag, ok := normalizedDockerHubReference(test.image)
			if !reflect.DeepEqual([]any{repository, tag, ok}, []any{test.repository, test.tag, test.ok}) {
				t.Fatalf("got (%q, %q, %t), want (%q, %q, %t)", repository, tag, ok, test.repository, test.tag, test.ok)
			}
		})
	}
}
