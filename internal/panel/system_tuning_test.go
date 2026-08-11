package panel

import (
	"net/http"
	"testing"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

func TestSystemTuningActionRequiresCSRFAndProxiesTypedRequest(t *testing.T) {
	server, tokenPath := newTestServer(t)
	sessionCookie, csrfCookie := bootstrapCookies(t, server, tokenPath)
	agent := &stubAgent{response: AgentResponse{StatusCode: http.StatusAccepted, ContentType: "application/json", Body: []byte(`{"status":"accepted"}`)}}
	server.agent = agent
	body := []byte(`{"action":"apply","items":["bbr","kernel-auto"],"expectedResourceVersion":"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}`)
	withoutCSRF := authenticatedSiteRequest(server, sessionCookie, csrfCookie, http.MethodPost, "/api/v1/system/system-tuning/actions", body, false)
	if withoutCSRF.Code == http.StatusAccepted {
		t.Fatal("request without CSRF was accepted")
	}
	response := authenticatedSiteRequest(server, sessionCookie, csrfCookie, http.MethodPost, "/api/v1/system/system-tuning/actions", body, true)
	if response.Code < 200 || response.Code >= 300 {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	calls := agent.snapshotCalls()
	if len(calls) != 1 || calls[0].path != "/v1/system/system-tuning/actions" {
		t.Fatalf("agent calls = %#v", calls)
	}
}

func TestSystemTuningActionRejectsUnknownItem(t *testing.T) {
	request := contract.SystemTuningActionRequest{Action: "apply", Items: []string{"shell"}, ExpectedResourceVersion: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}
	if field, _ := contract.ValidateSystemTuningAction(&request); field != "items" {
		t.Fatalf("field = %q", field)
	}
}
