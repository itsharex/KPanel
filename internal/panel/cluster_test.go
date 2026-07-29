package panel

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kejilion/kejilion-panel/internal/cluster"
)

func TestClusterHostsRequireSessionAndIncludeLocalHost(t *testing.T) {
	server, tokenPath := newTestServer(t)

	unauthenticated := performRequest(
		server, http.MethodGet, "/api/v1/cluster/hosts", nil, nil,
	)
	if unauthenticated.Code != http.StatusUnauthorized {
		t.Fatalf(
			"unauthenticated cluster hosts status = %d, want %d; body=%s",
			unauthenticated.Code, http.StatusUnauthorized, unauthenticated.Body.String(),
		)
	}

	sessionCookie, csrfCookie := bootstrapCookies(t, server, tokenPath)
	response := authenticatedRequest(
		server, http.MethodGet, "/api/v1/cluster/hosts", nil,
		sessionCookie, csrfCookie, nil,
	)
	if response.Code != http.StatusOK {
		t.Fatalf("cluster hosts status = %d, want %d; body=%s", response.Code, http.StatusOK, response.Body.String())
	}
	var inventory cluster.HostList
	if err := json.Unmarshal(response.Body.Bytes(), &inventory); err != nil {
		t.Fatalf("decode cluster hosts: %v", err)
	}
	if inventory.Total != 1 || inventory.RemoteTotal != 0 || len(inventory.Items) != 1 {
		t.Fatalf("unexpected local-only inventory: %#v", inventory)
	}
	local := inventory.Items[0]
	if local.ID != cluster.LocalHostID || !local.IsLocal {
		t.Fatalf("local host marker missing: %#v", local)
	}
	if local.Origin != "" {
		t.Fatalf("local host unexpectedly exposes a remote origin: %#v", local)
	}

	detail := authenticatedRequest(
		server, http.MethodGet, "/api/v1/cluster/hosts/local", nil,
		sessionCookie, csrfCookie, nil,
	)
	if detail.Code != http.StatusOK {
		t.Fatalf("local host detail status = %d, want %d; body=%s", detail.Code, http.StatusOK, detail.Body.String())
	}
	var localDetail cluster.Host
	if err := json.Unmarshal(detail.Body.Bytes(), &localDetail); err != nil {
		t.Fatalf("decode local host detail: %v", err)
	}
	if localDetail.ID != cluster.LocalHostID || !localDetail.IsLocal {
		t.Fatalf("local host detail marker missing: %#v", localDetail)
	}
}

func TestClusterMutationsRequireOriginAndCSRF(t *testing.T) {
	server, tokenPath := newTestServer(t)
	sessionCookie, csrfCookie := bootstrapCookies(t, server, tokenPath)

	missingOrigin := authenticatedRequest(
		server, http.MethodPost, "/api/v1/cluster/pairing-codes", nil,
		sessionCookie, csrfCookie, map[string]string{
			"X-CSRF-Token": csrfCookie.Value,
		},
	)
	if missingOrigin.Code != http.StatusForbidden ||
		!strings.Contains(missingOrigin.Body.String(), "origin_validation_failed") {
		t.Fatalf("missing Origin returned %d %s", missingOrigin.Code, missingOrigin.Body.String())
	}

	missingCSRF := authenticatedRequest(
		server, http.MethodPost, "/api/v1/cluster/pairing-codes", nil,
		sessionCookie, csrfCookie, map[string]string{
			"Origin": "http://panel.test",
		},
	)
	if missingCSRF.Code != http.StatusForbidden ||
		!strings.Contains(missingCSRF.Body.String(), "csrf_validation_failed") {
		t.Fatalf("missing CSRF returned %d %s", missingCSRF.Code, missingCSRF.Body.String())
	}
}

func TestClusterLocalHostCanBeRenamed(t *testing.T) {
	server, tokenPath := newTestServer(t)
	sessionCookie, csrfCookie := bootstrapCookies(t, server, tokenPath)

	list := authenticatedRequest(
		server, http.MethodGet, "/api/v1/cluster/hosts", nil,
		sessionCookie, csrfCookie, nil,
	)
	if list.Code != http.StatusOK {
		t.Fatalf("cluster hosts status = %d; body=%s", list.Code, list.Body.String())
	}
	var inventory cluster.HostList
	if err := json.Unmarshal(list.Body.Bytes(), &inventory); err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(cluster.UpdateHostInput{
		Name: "控制中心", ExpectedResourceVersion: inventory.Items[0].ResourceVersion,
	})
	if err != nil {
		t.Fatal(err)
	}
	response := authenticatedRequest(
		server, http.MethodPatch, "/api/v1/cluster/hosts/local", body,
		sessionCookie, csrfCookie, map[string]string{
			"Content-Type": "application/json",
			"Origin":       "http://panel.test",
			"X-CSRF-Token": csrfCookie.Value,
		},
	)
	if response.Code != http.StatusOK {
		t.Fatalf("local rename returned %d %s", response.Code, response.Body.String())
	}
	var renamed cluster.Host
	if err := json.Unmarshal(response.Body.Bytes(), &renamed); err != nil {
		t.Fatal(err)
	}
	if !renamed.IsLocal || renamed.Name != "控制中心" {
		t.Fatalf("unexpected renamed local host: %#v", renamed)
	}
}

func TestClusterLocalHostCannotBeDeleted(t *testing.T) {
	server, tokenPath := newTestServer(t)
	sessionCookie, csrfCookie := bootstrapCookies(t, server, tokenPath)

	list := authenticatedRequest(
		server, http.MethodGet, "/api/v1/cluster/hosts", nil,
		sessionCookie, csrfCookie, nil,
	)
	if list.Code != http.StatusOK {
		t.Fatalf("cluster hosts status = %d; body=%s", list.Code, list.Body.String())
	}
	var inventory cluster.HostList
	if err := json.Unmarshal(list.Body.Bytes(), &inventory); err != nil {
		t.Fatal(err)
	}
	if len(inventory.Items) == 0 {
		t.Fatal("local host is missing")
	}
	body, err := json.Marshal(cluster.DeleteHostInput{
		ExpectedResourceVersion: inventory.Items[0].ResourceVersion,
	})
	if err != nil {
		t.Fatal(err)
	}
	response := authenticatedRequest(
		server, http.MethodDelete, "/api/v1/cluster/hosts/local", body,
		sessionCookie, csrfCookie, map[string]string{
			"Content-Type": "application/json",
			"Origin":       "http://panel.test",
			"X-CSRF-Token": csrfCookie.Value,
		},
	)
	if response.Code != http.StatusConflict ||
		!strings.Contains(response.Body.String(), `"cluster_local_host"`) {
		t.Fatalf("local delete returned %d %s", response.Code, response.Body.String())
	}
}

func TestClusterPairingCodeSecretIsNotAudited(t *testing.T) {
	server, tokenPath := newTestServer(t)
	sessionCookie, csrfCookie := bootstrapCookies(t, server, tokenPath)

	response := authenticatedRequest(
		server, http.MethodPost, "/api/v1/cluster/pairing-codes/v2", nil,
		sessionCookie, csrfCookie, map[string]string{
			"Origin":       "http://panel.test",
			"X-CSRF-Token": csrfCookie.Value,
		},
	)
	if response.Code != http.StatusCreated {
		t.Fatalf("pairing code status = %d, want %d; body=%s", response.Code, http.StatusCreated, response.Body.String())
	}
	var code cluster.PairingCode
	if err := json.Unmarshal(response.Body.Bytes(), &code); err != nil {
		t.Fatalf("decode pairing code: %v", err)
	}
	if code.Code == "" {
		t.Fatal("pairing code response is empty")
	}
	if !strings.HasPrefix(code.Code, "kp2.") {
		t.Fatalf("pairing code does not use the encrypted v2 protocol: %q", code.Code)
	}

	events, _ := server.store.ListAudit(200, "")
	serialized, err := json.Marshal(events)
	if err != nil {
		t.Fatalf("marshal audit events: %v", err)
	}
	if strings.Contains(string(serialized), code.Code) {
		t.Fatalf("pairing code leaked into audit events: %s", serialized)
	}
	intent, success := false, false
	for _, event := range events {
		if event.Action != "cluster.pairing-code.create" {
			continue
		}
		if event.Change != nil {
			t.Fatalf("pairing code audit contains change data: %#v", event)
		}
		intent = intent || event.Result == "intent"
		success = success || event.Result == "success"
	}
	if !intent || !success {
		t.Fatalf("pairing code audit intent/success missing: %#v", events)
	}
}

func TestLegacyPairingCodeAPIStillIssuesV1Code(t *testing.T) {
	server, tokenPath := newTestServer(t)
	sessionCookie, csrfCookie := bootstrapCookies(t, server, tokenPath)
	response := authenticatedRequest(
		server,
		http.MethodPost,
		"/api/v1/cluster/pairing-codes",
		nil,
		sessionCookie,
		csrfCookie,
		map[string]string{
			"Origin":       "http://panel.test",
			"X-CSRF-Token": csrfCookie.Value,
		},
	)
	if response.Code != http.StatusCreated {
		t.Fatalf("legacy pairing code status = %d; body=%s", response.Code, response.Body.String())
	}
	var code cluster.PairingCode
	if err := json.Unmarshal(response.Body.Bytes(), &code); err != nil {
		t.Fatalf("decode legacy pairing code: %v", err)
	}
	if len(code.Code) != 81 || strings.HasPrefix(code.Code, "kp2.") {
		t.Fatalf("legacy pairing endpoint returned an incompatible code: %q", code.Code)
	}
}

func TestFederationV2BypassesPublicHostCheckButStillAuthenticates(t *testing.T) {
	server, _ := newTestServer(t)
	request := httptest.NewRequest(
		http.MethodPost,
		"http://8.8.8.8:1801/api/v2/federation/pair",
		strings.NewReader(`{}`),
	)
	request.Host = "8.8.8.8:1801"
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf(
			"unauthenticated v2 federation status = %d, want %d; body=%s",
			response.Code,
			http.StatusUnauthorized,
			response.Body.String(),
		)
	}
	if strings.Contains(response.Body.String(), "host_header_rejected") {
		t.Fatalf("v2 federation request was rejected by the browser Host policy: %s", response.Body.String())
	}
}

func TestFederationV2RejectsWrongMethodAndQuery(t *testing.T) {
	server, _ := newTestServer(t)
	for _, test := range []struct {
		name   string
		method string
		target string
		status int
	}{
		{
			name:   "wrong method",
			method: http.MethodGet,
			target: "http://8.8.8.8:1801/api/v2/federation/pair",
			status: http.StatusMisdirectedRequest,
		},
		{
			name:   "query",
			method: http.MethodPost,
			target: "http://8.8.8.8:1801/api/v2/federation/pair?unexpected=1",
			status: http.StatusBadRequest,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.target, strings.NewReader(`{}`))
			request.Host = "8.8.8.8:1801"
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()

			server.ServeHTTP(response, request)

			if response.Code != test.status {
				t.Fatalf(
					"invalid v2 federation status = %d, want %d; body=%s",
					response.Code,
					test.status,
					response.Body.String(),
				)
			}
		})
	}
}

func TestFederationV2RejectsOversizedEnvelope(t *testing.T) {
	server, _ := newTestServer(t)
	request := httptest.NewRequest(
		http.MethodPost,
		"http://8.8.8.8:1801/api/v2/federation/pair",
		strings.NewReader(
			`{"message":"`+
				strings.Repeat("x", cluster.MaxFederationV2Bytes+1)+
				`"}`,
		),
	)
	request.Host = "8.8.8.8:1801"
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf(
			"oversized v2 federation status = %d, want %d; body=%s",
			response.Code,
			http.StatusRequestEntityTooLarge,
			response.Body.String(),
		)
	}
}
