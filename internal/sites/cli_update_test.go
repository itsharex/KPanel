package sites

import (
	"strings"
	"testing"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

func TestPatchCLIProxyPreservesScriptSpecificConfiguration(t *testing.T) {
	old := []byte(`server {
    listen 443 ssl;
    server_name old.example.com www.old.example.com;
    ssl_certificate /etc/nginx/certs/old.example.com_cert.pem;
    location /custom-security/ { deny all; }
    location / {
        proxy_pass http://127.0.0.1:3000;
    }
}`)
	current := contract.SiteSummary{
		Kind: contract.SiteReverseProxy, Target: "http://127.0.0.1:3000",
	}
	got, err := patchCLIConfig(old, current, managedSpec{
		Primary: "old.example.com", Aliases: []string{"new.example.com"},
		Kind: contract.SiteReverseProxy, Upstream: "http://127.0.0.1:8080",
	})
	if err != nil {
		t.Fatal(err)
	}
	value := string(got)
	for _, expected := range []string{
		"server_name old.example.com new.example.com;",
		"proxy_pass http://127.0.0.1:8080;",
		"ssl_certificate /etc/nginx/certs/old.example.com_cert.pem;",
		"location /custom-security/ { deny all; }",
	} {
		if !strings.Contains(value, expected) {
			t.Fatalf("patched configuration is missing %q:\n%s", expected, value)
		}
	}
}

func TestPatchCLILoadBalanceAndRedirect(t *testing.T) {
	proxy := []byte(`upstream backend_name {
    least_conn;
    server 127.0.0.1:3000;
    server 127.0.0.1:3001;
}
server {
    server_name balance.example.com;
    location / { proxy_pass http://backend_name; }
}`)
	got, err := patchCLIConfig(proxy, contract.SiteSummary{
		Kind: contract.SiteLoadBalance,
	}, managedSpec{
		Primary:   "balance.example.com",
		Kind:      contract.SiteLoadBalance,
		Upstreams: []string{"http://10.0.0.2:80", "http://10.0.0.3:80"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "127.0.0.1:3000") ||
		!strings.Contains(string(got), "server 10.0.0.2:80;") ||
		!strings.Contains(string(got), "least_conn;") {
		t.Fatalf("load balancer patch changed unknown directives:\n%s", got)
	}

	redirect := []byte(`server {
    server_name go.example.com;
    return 301 https://old.example.com$request_uri;
}`)
	got, err = patchCLIConfig(redirect, contract.SiteSummary{
		Kind: contract.SiteRedirect,
	}, managedSpec{
		Primary: "go.example.com", Kind: contract.SiteRedirect,
		RedirectCode: 308, RedirectTarget: "https://new.example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "return 308 https://new.example.com$request_uri;") {
		t.Fatalf("redirect request URI was not preserved:\n%s", got)
	}
}
