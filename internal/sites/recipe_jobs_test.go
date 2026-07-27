package sites

import (
	"reflect"
	"strings"
	"testing"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

func TestNormalizeRecipeInputUsesFixedKejilionSelectors(t *testing.T) {
	domain, selector, err := normalizeRecipeInput(SiteInput{
		PrimaryDomain: "forum.example.com",
		Type:          "recipe",
		Recipe:        "discuz",
	})
	if err != nil || domain != "forum.example.com" || selector != "3" {
		t.Fatalf("normalizeRecipeInput() = %q, %q, %v", domain, selector, err)
	}
	for _, input := range []SiteInput{
		{PrimaryDomain: "forum.example.com", Type: "recipe", Recipe: "unknown"},
		{PrimaryDomain: "forum.example.com", Type: "recipe", Recipe: "discuz", Aliases: []string{"www.example.com"}},
		{PrimaryDomain: "forum.example.com", Type: "php", Recipe: "discuz"},
	} {
		if _, _, err := normalizeRecipeInput(input); err == nil {
			t.Fatalf("unsafe recipe input was accepted: %#v", input)
		}
	}
}

func TestNormalizeWordPressInputUsesOnlyKejilionDomain(t *testing.T) {
	spec, err := normalizeWordPressInput(SiteInput{
		PrimaryDomain: "blog.example.com",
		Type:          "wordpress",
	})
	if err != nil || spec.Primary != "blog.example.com" || spec.Kind != contract.SiteWordPress {
		t.Fatalf("normalizeWordPressInput() = %#v, %v", spec, err)
	}
	for _, input := range []SiteInput{
		{PrimaryDomain: "blog.example.com", Type: "php"},
		{PrimaryDomain: "blog.example.com", Type: "wordpress", Aliases: []string{"www.example.com"}},
		{PrimaryDomain: "blog.example.com", Type: "wordpress", Upstream: "http://127.0.0.1:8080"},
	} {
		if _, err := normalizeWordPressInput(input); err == nil {
			t.Fatalf("non-kejilion WordPress input was accepted: %#v", input)
		}
	}
}

func TestNormalizeDirectProxyInputExtractsKejilionHostAndPort(t *testing.T) {
	spec, host, port, err := normalizeDirectProxyInput(SiteInput{
		PrimaryDomain: "proxy.example.com",
		Type:          "proxy",
		Upstream:      "http://127.0.0.1:8080",
	})
	if err != nil || spec.Kind != contract.SiteReverseProxy ||
		host != "127.0.0.1" || port != "8080" {
		t.Fatalf(
			"normalizeDirectProxyInput() = %#v, %q, %q, %v",
			spec,
			host,
			port,
			err,
		)
	}
	for _, input := range []SiteInput{
		{PrimaryDomain: "proxy.example.com", Type: "proxy", Upstream: "http://127.0.0.1"},
		{PrimaryDomain: "proxy.example.com", Type: "proxy", Upstream: "https://127.0.0.1:8443"},
		{PrimaryDomain: "proxy.example.com", Type: "proxy", Upstream: "http://[::1]:8080"},
		{
			PrimaryDomain: "proxy.example.com",
			Type:          "proxy",
			Upstream:      "http://127.0.0.1:8080",
			Aliases:       []string{"www.example.com"},
		},
	} {
		if _, _, _, err := normalizeDirectProxyInput(input); err == nil {
			t.Fatalf("non-kejilion proxy input was accepted: %#v", input)
		}
	}
}

func TestContainsAllRequiresExactWebsiteCommandTokens(t *testing.T) {
	value := `KJ_WEB_RECIPE KJ_WEB_DOMAIN KJ_WEB_PROXY_HOST KJ_WEB_PROXY_PORT ldnmp_wp "${KJ_WEB_DOMAIN:-}" ldnmp_Proxy "${KJ_WEB_DOMAIN:-}" "${KJ_WEB_PROXY_HOST:-}" "${KJ_WEB_PROXY_PORT:-}"`
	if !containsAll(value, []string{"KJ_WEB_RECIPE", `ldnmp_wp "${KJ_WEB_DOMAIN:-}"`}) {
		t.Fatal("expected WordPress command tokens to match")
	}
	if containsAll(value, []string{"KJ_WEB_PROXY_TLS"}) {
		t.Fatal("unexpected website command token matched")
	}
}

func TestDirectWebsiteInvocationsUseKejilionCommands(t *testing.T) {
	wordPress := wordPressInvocation("blog.example.com")
	if !reflect.DeepEqual(wordPress.arguments, []string{"web"}) ||
		!reflect.DeepEqual(
			wordPress.environment,
			[]string{"KJ_WEB_NONINTERACTIVE=1", "KJ_WEB_RECIPE=2", "KJ_WEB_DOMAIN=blog.example.com"},
		) {
		t.Fatalf("WordPress invocation = %#v", wordPress)
	}
	proxy := proxyInvocation("proxy.example.com", "127.0.0.1", "8080")
	if !reflect.DeepEqual(proxy.arguments, []string{"web"}) ||
		!reflect.DeepEqual(
			proxy.environment,
			[]string{
				"KJ_WEB_NONINTERACTIVE=1",
				"KJ_WEB_RECIPE=23",
				"KJ_WEB_DOMAIN=proxy.example.com",
				"KJ_WEB_PROXY_HOST=127.0.0.1",
				"KJ_WEB_PROXY_PORT=8080",
			},
		) {
		t.Fatalf("proxy invocation = %#v", proxy)
	}
}

func TestSystemdSiteArgumentsRunExactScriptWithoutShellFragments(t *testing.T) {
	job := RecipeJob{ID: "0123456789abcdef0123456789abcdef"}
	invocation := proxyInvocation("proxy.example.com", "127.0.0.1", "8080")
	arguments := systemdSiteArguments(job, "/usr/local/bin/k", invocation)
	joined := strings.Join(arguments, "\n")
	for _, expected := range []string{
		"--unit=kpanel-site-" + job.ID,
		"--wait",
		"--pipe",
		"--property=ProtectSystem=no",
		"--",
		"/bin/bash",
		"/usr/local/bin/k",
		"web",
		"--setenv=KJ_WEB_RECIPE=23",
		"--setenv=KJ_WEB_DOMAIN=proxy.example.com",
		"--setenv=KJ_WEB_PROXY_HOST=127.0.0.1",
		"--setenv=KJ_WEB_PROXY_PORT=8080",
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("systemd site arguments missing %q: %#v", expected, arguments)
		}
	}
	for _, argument := range arguments {
		if strings.ContainsAny(argument, ";&`") {
			t.Fatalf("shell fragment reached systemd invocation: %q", argument)
		}
	}
}
