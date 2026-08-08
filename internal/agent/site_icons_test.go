package agent

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kejilion/kejilion-panel/internal/sites"
)

type stubSiteIconProvider struct {
	icon            sites.SiteIcon
	err             error
	appearance      sites.SiteAppearance
	appearanceErr   error
	calls           []string
	appearanceCalls []string
}

func (provider *stubSiteIconProvider) Get(_ context.Context, id string) (sites.SiteIcon, error) {
	provider.calls = append(provider.calls, id)
	return provider.icon, provider.err
}

func (provider *stubSiteIconProvider) Appearance(_ context.Context, id string) (sites.SiteAppearance, error) {
	provider.appearanceCalls = append(provider.appearanceCalls, id)
	return provider.appearance, provider.appearanceErr
}

func TestSiteAppearanceEndpointReturnsHomepageName(t *testing.T) {
	server := testServer(t)
	id := strings.Repeat("d", 32)
	provider := &stubSiteIconProvider{appearance: sites.SiteAppearance{Name: "科技狮网站"}}
	server.siteIcons = provider

	request := httptest.NewRequest(http.MethodGet, "/v1/sites/"+id+"/appearance", nil)
	request.Header.Set("Authorization", "Bearer "+strings.Repeat("x", 32))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"name":"科技狮网站"`) {
		t.Fatalf("site appearance response = %d %s", response.Code, response.Body.String())
	}
	if len(provider.appearanceCalls) != 1 || provider.appearanceCalls[0] != id {
		t.Fatalf("site appearance provider calls = %#v", provider.appearanceCalls)
	}
}

func TestSiteIconEndpointReturnsValidatedBitmap(t *testing.T) {
	server := testServer(t)
	id := strings.Repeat("a", 32)
	body := []byte("\x89PNG\r\n\x1a\nvalidated-by-service")
	provider := &stubSiteIconProvider{
		icon: sites.SiteIcon{ContentType: "image/png", Data: body},
	}
	server.siteIcons = provider

	request := httptest.NewRequest(http.MethodGet, "/v1/sites/"+id+"/icon", nil)
	request.Header.Set("Authorization", "Bearer "+strings.Repeat("x", 32))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK || !bytes.Equal(response.Body.Bytes(), body) {
		t.Fatalf("site icon response = %d %q", response.Code, response.Body.Bytes())
	}
	if response.Header().Get("Content-Type") != "image/png" ||
		response.Header().Get("Content-Length") != "28" ||
		response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("site icon headers = %#v", response.Header())
	}
	if len(provider.calls) != 1 || provider.calls[0] != id {
		t.Fatalf("site icon provider calls = %#v", provider.calls)
	}
}

func TestSiteIconEndpointRejectsMalformedRequestsBeforeProvider(t *testing.T) {
	id := strings.Repeat("b", 32)
	for name, test := range map[string]struct {
		method string
		target string
		status int
	}{
		"missing authentication": {
			method: http.MethodGet, target: "/v1/sites/" + id + "/icon",
			status: http.StatusUnauthorized,
		},
		"uppercase id": {
			method: http.MethodGet, target: "/v1/sites/" + strings.Repeat("B", 32) + "/icon",
			status: http.StatusNotFound,
		},
		"short id": {
			method: http.MethodGet, target: "/v1/sites/" + strings.Repeat("b", 31) + "/icon",
			status: http.StatusNotFound,
		},
		"query": {
			method: http.MethodGet, target: "/v1/sites/" + id + "/icon?refresh=true",
			status: http.StatusBadRequest,
		},
		"wrong method": {
			method: http.MethodPost, target: "/v1/sites/" + id + "/icon",
			status: http.StatusMethodNotAllowed,
		},
		"trailing segment": {
			method: http.MethodGet, target: "/v1/sites/" + id + "/icon/extra",
			status: http.StatusNotFound,
		},
		"appearance query": {
			method: http.MethodGet, target: "/v1/sites/" + id + "/appearance?refresh=true",
			status: http.StatusBadRequest,
		},
		"appearance wrong method": {
			method: http.MethodPost, target: "/v1/sites/" + id + "/appearance",
			status: http.StatusMethodNotAllowed,
		},
	} {
		t.Run(name, func(t *testing.T) {
			server := testServer(t)
			provider := &stubSiteIconProvider{}
			server.siteIcons = provider
			request := httptest.NewRequest(test.method, test.target, nil)
			if name != "missing authentication" {
				request.Header.Set("Authorization", "Bearer "+strings.Repeat("x", 32))
			}
			response := httptest.NewRecorder()
			server.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, test.status, response.Body.String())
			}
			if len(provider.calls) != 0 {
				t.Fatalf("provider called for malformed request: %#v", provider.calls)
			}
			if len(provider.appearanceCalls) != 0 {
				t.Fatalf("appearance provider called for malformed request: %#v", provider.appearanceCalls)
			}
		})
	}
}

func TestSiteIconEndpointMapsProviderFailuresAndRejectsUnsafeMedia(t *testing.T) {
	id := strings.Repeat("c", 32)
	for name, test := range map[string]struct {
		provider *stubSiteIconProvider
		status   int
		code     string
	}{
		"not found": {
			provider: &stubSiteIconProvider{err: sites.ErrSiteIconNotFound},
			status:   http.StatusNotFound,
			code:     "site_icon_not_found",
		},
		"unavailable": {
			provider: &stubSiteIconProvider{err: sites.ErrSiteIconUnavailable},
			status:   http.StatusServiceUnavailable,
			code:     "site_icon_unavailable",
		},
		"invalid content type": {
			provider: &stubSiteIconProvider{
				icon: sites.SiteIcon{ContentType: "image/svg+xml", Data: []byte("<svg/>")},
			},
			status: http.StatusBadGateway,
			code:   "invalid_site_icon",
		},
		"oversized": {
			provider: &stubSiteIconProvider{
				icon: sites.SiteIcon{
					ContentType: "image/png",
					Data:        bytes.Repeat([]byte("x"), (256<<10)+1),
				},
			},
			status: http.StatusBadGateway,
			code:   "invalid_site_icon",
		},
		"wrapped failure": {
			provider: &stubSiteIconProvider{err: errors.New("network failed")},
			status:   http.StatusServiceUnavailable,
			code:     "site_icon_unavailable",
		},
	} {
		t.Run(name, func(t *testing.T) {
			server := testServer(t)
			server.siteIcons = test.provider
			request := httptest.NewRequest(http.MethodGet, "/v1/sites/"+id+"/icon", nil)
			request.Header.Set("Authorization", "Bearer "+strings.Repeat("x", 32))
			response := httptest.NewRecorder()
			server.ServeHTTP(response, request)
			if response.Code != test.status ||
				!strings.Contains(response.Body.String(), `"code":"`+test.code+`"`) {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
		})
	}
}
