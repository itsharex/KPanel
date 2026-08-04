package panel

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/kejilion/kejilion-panel/internal/ai"
)

func TestAIProviderModelSessionCRUD(t *testing.T) {
	server, tokenPath := newTestServer(t)
	if err := server.EnableAI(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	sessionCookie, csrfCookie := bootstrapCookies(t, server, tokenPath)
	headers := map[string]string{"Content-Type": "application/json", "Origin": "http://panel.test", "X-CSRF-Token": csrfCookie.Value}
	create := authenticatedRequest(server, http.MethodPost, "/api/v1/ai/providers", []byte(`{"name":"Mock","protocol":"openai_compatible","baseUrl":"http://127.0.0.1:11434/v1","endpointScope":"private","privateConfirmed":true,"apiKey":"secret-1234","enabled":true}`), sessionCookie, csrfCookie, headers)
	if create.Code != http.StatusCreated {
		t.Fatalf("provider create=%d %s", create.Code, create.Body.String())
	}
	var provider ai.Provider
	if err := json.Unmarshal(create.Body.Bytes(), &provider); err != nil {
		t.Fatal(err)
	}
	if provider.ID == "" || !provider.APIKeySet || provider.APIKeyHint != "1234" {
		t.Fatalf("provider=%#v", provider)
	}
	if strings.Contains(create.Body.String(), "secret-1234") {
		t.Fatal("response exposed API key")
	}
	modelBody := []byte(`{"modelId":"mock-model","displayName":"Mock Model","contextWindow":8192,"toolCalling":true,"enabled":true}`)
	modelsResponse := authenticatedRequest(server, http.MethodPost, "/api/v1/ai/providers/"+provider.ID+"/models", modelBody, sessionCookie, csrfCookie, headers)
	if modelsResponse.Code != http.StatusCreated {
		t.Fatalf("model add=%d %s", modelsResponse.Code, modelsResponse.Body.String())
	}
	var models []ai.Model
	if err := json.Unmarshal(modelsResponse.Body.Bytes(), &models); err != nil || len(models) != 1 {
		t.Fatalf("models=%#v err=%v", models, err)
	}
	sessionResponse := authenticatedRequest(server, http.MethodPost, "/api/v1/ai/sessions", []byte(`{"providerId":"`+provider.ID+`","modelId":"`+models[0].ID+`"}`), sessionCookie, csrfCookie, headers)
	if sessionResponse.Code != http.StatusCreated {
		t.Fatalf("session create=%d %s", sessionResponse.Code, sessionResponse.Body.String())
	}
	list := authenticatedRequest(server, http.MethodGet, "/api/v1/ai/sessions", nil, sessionCookie, csrfCookie, nil)
	if list.Code != http.StatusOK {
		t.Fatalf("session list=%d %s", list.Code, list.Body.String())
	}
}

func TestAIToolMapsAgentVersionConflictForReplanning(t *testing.T) {
	server, _ := newTestServer(t)
	server.agent = &stubAgent{response: AgentResponse{StatusCode: http.StatusConflict, ContentType: "application/json", Body: []byte(`{"code":"resource_version_conflict"}`)}}
	tools := &panelAITools{server: server}
	execution := ai.ToolExecutionContext{UserID: "admin", SessionID: "ses", RunID: "run", ToolCallID: "call"}
	_, err := tools.Execute(context.Background(), execution, "host_system_summary", json.RawMessage(`{}`))
	if !errors.Is(err, ai.ErrToolConflict) {
		t.Fatalf("Agent conflict was not typed for replanning: %v", err)
	}
}

func TestAIToolUsesFixedAgentPathAndAudits(t *testing.T) {
	server, _ := newTestServer(t)
	agent := &stubAgent{response: AgentResponse{StatusCode: 200, ContentType: "application/json", Body: []byte(`{"ok":true}`)}}
	server.agent = agent
	tools := &panelAITools{server: server}
	execution := ai.ToolExecutionContext{UserID: "admin", SessionID: "ses-1", RunID: "run-1", ToolCallID: "call-1"}
	result, err := tools.Execute(context.Background(), execution, "host_system_summary", json.RawMessage(`{}`))
	if err != nil || result != "{\"ok\":true}" {
		t.Fatalf("result=%q err=%v", result, err)
	}
	calls := agent.snapshotCalls()
	if len(calls) != 1 || calls[0].path != "/v1/system/summary" || calls[0].method != http.MethodGet {
		t.Fatalf("calls=%#v", calls)
	}
	audits, _ := server.store.ListAudit(10, "")
	if len(audits) == 0 || audits[0].Change["sessionId"] != "ses-1" || audits[0].Change["runId"] != "run-1" || audits[0].Change["toolCallId"] != "call-1" {
		t.Fatalf("AI audit correlation missing: %#v", audits)
	}
	_, err = tools.Execute(context.Background(), execution, "host_docker_container_action", json.RawMessage(`{"containerId":"bad","action":"remove","resourceVersion":"bad"}`))
	if err == nil || len(agent.snapshotCalls()) != 1 {
		t.Fatal("invalid write reached Agent")
	}
}
