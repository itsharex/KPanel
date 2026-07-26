package panel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
)

type agentCall struct {
	method   string
	path     string
	rawQuery string
	request  string
	body     []byte
}

type stubAgent struct {
	mu       sync.Mutex
	calls    []agentCall
	response AgentResponse
	err      error
}

func (agent *stubAgent) Get(
	_ context.Context,
	path string,
	rawQuery string,
	requestID string,
) (AgentResponse, error) {
	return agent.Do(context.Background(), http.MethodGet, path, rawQuery, requestID, nil)
}

func (agent *stubAgent) Do(
	_ context.Context,
	method string,
	path string,
	rawQuery string,
	requestID string,
	body []byte,
) (AgentResponse, error) {
	agent.mu.Lock()
	defer agent.mu.Unlock()
	agent.calls = append(agent.calls, agentCall{
		method: method, path: path, rawQuery: rawQuery, request: requestID,
		body: append([]byte(nil), body...),
	})
	return agent.response, agent.err
}

func (agent *stubAgent) snapshotCalls() []agentCall {
	agent.mu.Lock()
	defer agent.mu.Unlock()
	return append([]agentCall(nil), agent.calls...)
}

func TestSiteCreateProxiesAgentProblemAndAuditsSafeMetadata(t *testing.T) {
	server, tokenPath := newTestServer(t)
	sessionCookie, csrfCookie := bootstrapCookies(t, server, tokenPath)
	agentProblem := []byte(`{"title":"Conflict","status":409,"code":"site_conflict","requestId":"agent-request"}`)
	agent := &stubAgent{response: AgentResponse{
		StatusCode: http.StatusConflict, ContentType: "application/problem+json", Body: agentProblem,
	}}
	server.agent = agent

	body := []byte(`{
		"primaryDomain":"Example.COM",
		"aliases":["www.Example.COM"],
		"type":"proxy",
		"upstream":"https://private-upstream.example",
		"enabled":true
	}`)
	response := authenticatedSiteRequest(
		server, sessionCookie, csrfCookie, http.MethodPost, "/api/v1/sites", body, true,
	)
	if response.Code != http.StatusConflict {
		t.Fatalf("unexpected status: %d %s", response.Code, response.Body.String())
	}
	if !bytes.Equal(response.Body.Bytes(), agentProblem) {
		t.Fatalf("Agent Problem body changed: got %q want %q", response.Body.Bytes(), agentProblem)
	}
	if got := response.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Fatalf("Agent content type changed: %q", got)
	}

	calls := agent.snapshotCalls()
	if len(calls) != 1 {
		t.Fatalf("expected one Agent call, got %d", len(calls))
	}
	if calls[0].method != http.MethodPost || calls[0].path != "/v1/sites" ||
		calls[0].rawQuery != "" || calls[0].request == "" {
		t.Fatalf("unexpected Agent call: %#v", calls[0])
	}
	var forwarded map[string]any
	if err := json.Unmarshal(calls[0].body, &forwarded); err != nil {
		t.Fatal(err)
	}
	if forwarded["primaryDomain"] != "example.com" ||
		forwarded["upstream"] != "https://private-upstream.example" {
		t.Fatalf("unexpected forwarded body: %#v", forwarded)
	}
	if _, present := forwarded["expectedResourceVersion"]; present {
		t.Fatalf("create forwarded an expectedResourceVersion: %#v", forwarded)
	}

	events, _ := server.store.ListAudit(20, "")
	var siteEvents int
	for _, event := range events {
		if event.Action != "site.create" {
			continue
		}
		siteEvents++
		if event.Result != "intent" && event.Result != "failure" {
			t.Fatalf("unexpected site audit result: %q", event.Result)
		}
		expectedKeys := []string{"domain", "kind", "resourceVersion"}
		keys := make([]string, 0, len(event.Change))
		for key := range event.Change {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		if !reflect.DeepEqual(keys, expectedKeys) {
			t.Fatalf("unsafe audit change keys: %#v", keys)
		}
		encoded, err := json.Marshal(event.Change)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(encoded, []byte("private-upstream")) ||
			bytes.Contains(encoded, []byte("www.example.com")) {
			t.Fatalf("audit leaked sensitive site fields: %s", encoded)
		}
	}
	if siteEvents != 2 {
		t.Fatalf("expected intent and failure audits, got %d", siteEvents)
	}
}

func TestSiteWritesRejectUnsafeRequestsBeforeAgentCall(t *testing.T) {
	server, tokenPath := newTestServer(t)
	sessionCookie, csrfCookie := bootstrapCookies(t, server, tokenPath)
	agent := &stubAgent{response: AgentResponse{
		StatusCode: http.StatusOK, ContentType: "application/json", Body: []byte(`{"ok":true}`),
	}}
	server.agent = agent
	version := "sha256:" + strings.Repeat("b", 64)
	id := strings.Repeat("a", 32)
	createBody := []byte(`{"primaryDomain":"example.com","aliases":[],"type":"static","enabled":true}`)
	updateBody := []byte(`{"primaryDomain":"example.com","aliases":[],"type":"static","enabled":true,"expectedResourceVersion":"` + version + `"}`)

	unauthenticated := httptest.NewRequest(http.MethodPost, "/api/v1/sites", bytes.NewReader(createBody))
	unauthenticated.Host = "panel.test"
	unauthenticated.Header.Set("Content-Type", "application/json")
	unauthenticated.Header.Set("Origin", "http://panel.test")
	unauthenticatedResponse := httptest.NewRecorder()
	server.ServeHTTP(unauthenticatedResponse, unauthenticated)
	if unauthenticatedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated write returned %d", unauthenticatedResponse.Code)
	}

	crossOrigin := newAuthenticatedSiteRequest(
		sessionCookie, csrfCookie, http.MethodPost, "/api/v1/sites", createBody, true,
	)
	crossOrigin.Header.Set("Origin", "https://evil.example")
	crossOriginResponse := httptest.NewRecorder()
	server.ServeHTTP(crossOriginResponse, crossOrigin)
	if crossOriginResponse.Code != http.StatusForbidden {
		t.Fatalf("cross-origin write returned %d", crossOriginResponse.Code)
	}

	tests := []struct {
		name       string
		method     string
		path       string
		body       []byte
		csrf       bool
		rawPath    string
		wantStatus int
	}{
		{
			name: "missing csrf", method: http.MethodPost, path: "/api/v1/sites",
			body: createBody, wantStatus: http.StatusForbidden,
		},
		{
			name: "unknown field", method: http.MethodPost, path: "/api/v1/sites",
			body: []byte(`{"primaryDomain":"example.com","type":"static","command":"rm"}`),
			csrf: true, wantStatus: http.StatusBadRequest,
		},
		{
			name: "create resource version", method: http.MethodPost, path: "/api/v1/sites",
			body: updateBody, csrf: true, wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "unsupported site type", method: http.MethodPost, path: "/api/v1/sites",
			body: []byte(`{"primaryDomain":"example.com","type":"shell"}`),
			csrf: true, wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "too many aliases", method: http.MethodPost, path: "/api/v1/sites",
			body: []byte(`{"primaryDomain":"example.com","aliases":[
				"a01.example.com","a02.example.com","a03.example.com","a04.example.com",
				"a05.example.com","a06.example.com","a07.example.com","a08.example.com",
				"a09.example.com","a10.example.com","a11.example.com","a12.example.com",
				"a13.example.com","a14.example.com","a15.example.com","a16.example.com",
				"a17.example.com","a18.example.com","a19.example.com","a20.example.com",
				"a21.example.com"
			],"type":"static"}`),
			csrf: true, wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "wrong enabled type", method: http.MethodPost, path: "/api/v1/sites",
			body: []byte(`{"primaryDomain":"example.com","type":"static","enabled":"yes"}`),
			csrf: true, wantStatus: http.StatusBadRequest,
		},
		{
			name: "patch invalid version", method: http.MethodPatch, path: "/api/v1/sites/" + id,
			body: []byte(`{"primaryDomain":"example.com","type":"static","expectedResourceVersion":"bad"}`),
			csrf: true, wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "patch requires full form", method: http.MethodPatch, path: "/api/v1/sites/" + id,
			body: []byte(`{"type":"static","expectedResourceVersion":"` + version + `"}`),
			csrf: true, wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "uppercase id", method: http.MethodPatch, path: "/api/v1/sites/" + strings.Repeat("A", 32),
			body: updateBody, csrf: true, wantStatus: http.StatusNotFound,
		},
		{
			name: "trailing segment", method: http.MethodPatch, path: "/api/v1/sites/" + id + "/extra",
			body: updateBody, csrf: true, wantStatus: http.StatusNotFound,
		},
		{
			name: "write query", method: http.MethodPatch, path: "/api/v1/sites/" + id + "?force=true",
			body: updateBody, csrf: true, wantStatus: http.StatusNotFound,
		},
		{
			name: "encoded path", method: http.MethodPatch, path: "/api/v1/sites/" + id,
			rawPath: "/api/v1/sites/%61" + id[1:], body: updateBody, csrf: true,
			wantStatus: http.StatusNotFound,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := newAuthenticatedSiteRequest(
				sessionCookie, csrfCookie, test.method, test.path, test.body, test.csrf,
			)
			if test.rawPath != "" {
				request.URL.RawPath = test.rawPath
			}
			response := httptest.NewRecorder()
			server.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("got %d, want %d: %s", response.Code, test.wantStatus, response.Body.String())
			}
		})
	}
	if calls := agent.snapshotCalls(); len(calls) != 0 {
		t.Fatalf("unsafe requests reached Agent: %#v", calls)
	}

	success := authenticatedSiteRequest(
		server, sessionCookie, csrfCookie, http.MethodPatch, "/api/v1/sites/"+id, updateBody, true,
	)
	if success.Code != http.StatusOK {
		t.Fatalf("valid update failed: %d %s", success.Code, success.Body.String())
	}
	calls := agent.snapshotCalls()
	if len(calls) != 1 || calls[0].method != http.MethodPatch ||
		calls[0].path != "/v1/sites/"+id {
		t.Fatalf("valid update was not routed exactly: %#v", calls)
	}
}

func TestSiteWriteCatalogValidationAndForwarding(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "static", body: `{"primaryDomain":"static.example.com","type":"static"}`},
		{name: "php", body: `{"primaryDomain":"php.example.com","type":"php","phpVersion":"7.4"}`},
		{name: "private proxy", body: `{"primaryDomain":"proxy.example.com","type":"proxy","upstream":"http://127.0.0.1:3000"}`},
		{name: "domain proxy", body: `{"primaryDomain":"edge.example.com","type":"proxy_domain","upstream":"https://origin.example.net"}`},
		{name: "load balance", body: `{"primaryDomain":"balanced.example.com","type":"load_balance","upstreams":["http://10.0.0.1:80","http://10.0.0.2:80"]}`},
		{name: "redirect", body: `{"primaryDomain":"old.example.com","type":"redirect","redirectTarget":"https://new.example.com","redirectCode":308}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var input siteWriteInput
			if err := json.Unmarshal([]byte(test.body), &input); err != nil {
				t.Fatal(err)
			}
			if field, detail := validateSiteWriteInput(&input, true); field != "" {
				t.Fatalf("valid catalog request rejected at %s: %s", field, detail)
			}
			encoded, err := json.Marshal(input.agentPayload())
			if err != nil {
				t.Fatal(err)
			}
			var original, forwarded map[string]any
			if err := json.Unmarshal([]byte(test.body), &original); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(encoded, &forwarded); err != nil {
				t.Fatal(err)
			}
			for key, value := range original {
				if !reflect.DeepEqual(forwarded[key], value) {
					t.Fatalf("field %s changed: got %#v want %#v; body=%s", key, forwarded[key], value, encoded)
				}
			}
		})
	}
}

func TestSiteTransportFailureReturnsServiceUnavailable(t *testing.T) {
	server, tokenPath := newTestServer(t)
	sessionCookie, csrfCookie := bootstrapCookies(t, server, tokenPath)
	server.agent = &stubAgent{err: errors.New("socket unavailable")}

	response := authenticatedSiteRequest(
		server,
		sessionCookie,
		csrfCookie,
		http.MethodPost,
		"/api/v1/sites",
		[]byte(`{"primaryDomain":"example.com","aliases":[],"type":"static","enabled":true}`),
		true,
	)
	if response.Code != http.StatusServiceUnavailable ||
		!strings.Contains(response.Body.String(), `"code":"agent_unavailable"`) {
		t.Fatalf("unexpected transport failure: %d %s", response.Code, response.Body.String())
	}
}

func TestSiteWriteFailsClosedWhenIntentAuditCannotPersist(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("cannot remove an open lock file on Windows")
	}
	server, tokenPath := newTestServer(t)
	sessionCookie, csrfCookie := bootstrapCookies(t, server, tokenPath)
	agent := &stubAgent{response: AgentResponse{
		StatusCode: http.StatusOK, ContentType: "application/json", Body: []byte(`{"ok":true}`),
	}}
	server.agent = agent

	if err := os.RemoveAll(server.config.DataDir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		server.config.DataDir,
		[]byte("block store directory recreation"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	response := authenticatedSiteRequest(
		server,
		sessionCookie,
		csrfCookie,
		http.MethodPost,
		"/api/v1/sites",
		[]byte(`{"primaryDomain":"example.com","aliases":[],"type":"static","enabled":true}`),
		true,
	)
	if response.Code != http.StatusServiceUnavailable ||
		!strings.Contains(response.Body.String(), `"code":"audit_unavailable"`) {
		t.Fatalf("site write did not fail closed: %d %s", response.Code, response.Body.String())
	}
	if calls := agent.snapshotCalls(); len(calls) != 0 {
		t.Fatalf("Agent was called after intent persistence failed: %#v", calls)
	}
}

func authenticatedSiteRequest(
	server *Server,
	sessionCookie *http.Cookie,
	csrfCookie *http.Cookie,
	method string,
	path string,
	body []byte,
	includeCSRF bool,
) *httptest.ResponseRecorder {
	request := newAuthenticatedSiteRequest(
		sessionCookie, csrfCookie, method, path, body, includeCSRF,
	)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	return response
}

func newAuthenticatedSiteRequest(
	sessionCookie *http.Cookie,
	csrfCookie *http.Cookie,
	method string,
	path string,
	body []byte,
	includeCSRF bool,
) *http.Request {
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	request.Host = "panel.test"
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://panel.test")
	if includeCSRF {
		request.Header.Set("X-CSRF-Token", csrfCookie.Value)
	}
	request.AddCookie(sessionCookie)
	request.AddCookie(csrfCookie)
	return request
}
