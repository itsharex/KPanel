package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func lightTokenForTest(t *testing.T, origin string, expiresAt time.Time) string {
	t.Helper()
	wire := tokenWire{
		Version: 1, Origin: origin, ID: strings.Repeat("a", 32),
		Secret: base64.RawURLEncoding.EncodeToString(make([]byte, 32)), ExpiresAt: expiresAt.Unix(),
	}
	content, err := json.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	return lightTokenPrefix + base64.RawURLEncoding.EncodeToString(content)
}

func TestOriginFromTokenRequiresStrictHTTPSOriginAndFutureExpiry(t *testing.T) {
	valid := lightTokenForTest(t, "https://panel.example:8443", time.Now().UTC().Add(time.Minute))
	if origin, err := originFromToken(valid); err != nil || origin != "https://panel.example:8443" {
		t.Fatalf("originFromToken(valid) = %q, %v", origin, err)
	}
	for _, token := range []string{
		lightTokenForTest(t, "http://panel.example", time.Now().UTC().Add(time.Minute)),
		lightTokenForTest(t, "https://user:pass@panel.example", time.Now().UTC().Add(time.Minute)),
		lightTokenForTest(t, "https://panel.example/path", time.Now().UTC().Add(time.Minute)),
		lightTokenForTest(t, "https://panel.example", time.Now().UTC().Add(-time.Minute)),
		lightTokenForTest(t, "https://panel.example", time.Now().UTC().Add(11*time.Minute)),
		"kpl1.invalid",
	} {
		if _, err := originFromToken(token); err == nil {
			t.Fatalf("originFromToken(%q) accepted an unsafe token", token)
		}
	}
}

func TestNodeConfigRoundTripIsStrictAndRejectsNonRegularTargets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "node.json")
	config := nodeConfig{
		SchemaVersion: 1, Origin: "https://panel.example", NodeID: strings.Repeat("b", 32),
		ReportingKey: base64.RawURLEncoding.EncodeToString(make([]byte, 32)), ReportInterval: 30,
	}
	if err := writeConfigAtomic(path, config); err != nil {
		t.Fatalf("writeConfigAtomic() error = %v", err)
	}
	loaded, secret, err := readConfig(path)
	if err != nil || loaded != config || len(secret) != 32 {
		t.Fatalf("readConfig() = %#v, %d, %v", loaded, len(secret), err)
	}
	if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
		t.Fatalf("configuration is not a regular file: %#v, %v", info, err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content = append(content[:len(content)-2], []byte(`,"unexpected":true}`+"\n")...)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readConfig(path); err == nil {
		t.Fatal("readConfig() accepted an unknown field")
	}
	directoryTarget := filepath.Join(t.TempDir(), "directory")
	if err := os.Mkdir(directoryTarget, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeConfigAtomic(directoryTarget, config); err == nil {
		t.Fatal("writeConfigAtomic() accepted a directory target")
	}
}

func TestNodeHTTPClientDoesNotFollowRedirects(t *testing.T) {
	targetCalls := 0
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		targetCalls++
	}))
	defer target.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", target.URL)
		w.WriteHeader(http.StatusTemporaryRedirect)
	}))
	defer redirect.Close()
	var output map[string]any
	err := postRawJSON(t.Context(), redirect.URL, []byte(`{}`), nil, &output)
	if err == nil || !strings.Contains(err.Error(), "HTTP 307") {
		t.Fatalf("postRawJSON() error = %v, want redirect rejection", err)
	}
	if targetCalls != 0 {
		t.Fatalf("redirect target received %d requests", targetCalls)
	}
}

func TestRunRejectsUnsupportedCommands(t *testing.T) {
	if err := run(nil); err == nil {
		t.Fatal("run(nil) succeeded")
	}
	if err := run([]string{"shell"}); err == nil {
		t.Fatal("run(shell) exposed an unsupported command")
	}
	if err := run([]string{"run", "unexpected"}); err == nil {
		t.Fatal("run accepted an unexpected argument")
	}
	if _, err := validateHTTPSOrigin("https://panel.example/?token=secret"); err == nil {
		t.Fatal("validateHTTPSOrigin accepted a query string")
	}
	if !errors.Is(newHTTPClient().CheckRedirect(nil, nil), http.ErrUseLastResponse) {
		t.Fatal("HTTP client redirect policy changed")
	}
}
