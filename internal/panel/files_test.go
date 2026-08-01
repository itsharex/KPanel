package panel

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
)

type streamAgentCall struct {
	method   string
	path     string
	rawQuery string
	headers  http.Header
	body     []byte
}

type fileStubAgent struct {
	*stubAgent
	mu             sync.Mutex
	streamCalls    []streamAgentCall
	streamStatus   int
	streamHeaders  http.Header
	streamResponse []byte
}

func (agent *fileStubAgent) OpenStream(
	_ context.Context,
	method, path, rawQuery, _ string,
	body io.Reader,
	headers http.Header,
	_ int64,
) (*http.Response, error) {
	content, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	agent.mu.Lock()
	agent.streamCalls = append(agent.streamCalls, streamAgentCall{
		method: method, path: path, rawQuery: rawQuery, headers: headers.Clone(), body: content,
	})
	agent.mu.Unlock()
	status := agent.streamStatus
	if status == 0 {
		status = http.StatusOK
	}
	responseHeaders := agent.streamHeaders.Clone()
	if responseHeaders == nil {
		responseHeaders = make(http.Header)
	}
	return &http.Response{
		StatusCode: status,
		Header:     responseHeaders,
		Body:       io.NopCloser(bytes.NewReader(agent.streamResponse)),
	}, nil
}

func (agent *fileStubAgent) snapshotStreamCalls() []streamAgentCall {
	agent.mu.Lock()
	defer agent.mu.Unlock()
	return append([]streamAgentCall(nil), agent.streamCalls...)
}

func TestFileListRequiresSessionAndForwardsStrictQuery(t *testing.T) {
	server, tokenPath := newTestServer(t)
	sessionCookie, csrfCookie := bootstrapCookies(t, server, tokenPath)
	agent := &fileStubAgent{stubAgent: &stubAgent{response: AgentResponse{
		StatusCode:  http.StatusOK,
		ContentType: "application/json",
		Body:        []byte(`{"path":"/","entries":[],"truncated":false,"readAt":"2026-07-30T00:00:00Z"}`),
	}}}
	server.agent = agent

	unauthenticated := performRequest(server, http.MethodGet, "/api/v1/files?path=%2F", nil, nil)
	if unauthenticated.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d", unauthenticated.Code)
	}
	if len(agent.snapshotCalls()) != 0 {
		t.Fatal("Agent called before authentication")
	}

	response := authenticatedRequest(
		server, http.MethodGet, "/api/v1/files?path=%2F&limit=100&offset=0&search=log", nil,
		sessionCookie, csrfCookie, nil,
	)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"path":"/"`) {
		t.Fatalf("list response = %d %s", response.Code, response.Body.String())
	}
	calls := agent.snapshotCalls()
	if len(calls) != 1 || calls[0].path != "/v1/files" ||
		calls[0].rawQuery != "path=%2F&limit=100&offset=0&search=log" {
		t.Fatalf("unexpected Agent calls: %#v", calls)
	}

	invalid := authenticatedRequest(
		server, http.MethodGet, "/api/v1/files?path=%2F&unknown=true", nil,
		sessionCookie, csrfCookie, nil,
	)
	if invalid.Code != http.StatusBadRequest || len(agent.snapshotCalls()) != 1 {
		t.Fatalf("invalid query = %d calls=%#v", invalid.Code, agent.snapshotCalls())
	}
}

func TestFileTrashListRequiresSessionAndForwardsToAgent(t *testing.T) {
	server, tokenPath := newTestServer(t)
	sessionCookie, csrfCookie := bootstrapCookies(t, server, tokenPath)
	agent := &fileStubAgent{stubAgent: &stubAgent{response: AgentResponse{
		StatusCode: http.StatusOK, ContentType: "application/json",
		Body: []byte(`{"entries":[],"total":0,"readAt":"2026-07-30T00:00:00Z"}`),
	}}}
	server.agent = agent

	unauthenticated := performRequest(server, http.MethodGet, "/api/v1/files/trash", nil, nil)
	if unauthenticated.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d", unauthenticated.Code)
	}
	response := authenticatedRequest(
		server, http.MethodGet, "/api/v1/files/trash", nil, sessionCookie, csrfCookie, nil,
	)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"total":0`) {
		t.Fatalf("trash response = %d %s", response.Code, response.Body.String())
	}
	calls := agent.snapshotCalls()
	if len(calls) != 1 || calls[0].path != "/v1/files/trash" || calls[0].rawQuery != "" {
		t.Fatalf("unexpected Agent calls: %#v", calls)
	}
}

func TestFileContentStreamsRangeAndUploadRequiresCSRF(t *testing.T) {
	server, tokenPath := newTestServer(t)
	sessionCookie, csrfCookie := bootstrapCookies(t, server, tokenPath)
	agent := &fileStubAgent{
		stubAgent:      &stubAgent{},
		streamStatus:   http.StatusPartialContent,
		streamResponse: []byte("hello"),
		streamHeaders: http.Header{
			"Content-Type":  []string{"text/plain"},
			"Content-Range": []string{"bytes 0-4/5"},
			"ETag":          []string{`"sha256:test"`},
		},
	}
	server.agent = agent

	download := authenticatedRequest(
		server, http.MethodGet, "/api/v1/files/content?path=%2Fhello.txt&disposition=inline",
		nil, sessionCookie, csrfCookie, map[string]string{"Range": "bytes=0-4"},
	)
	if download.Code != http.StatusPartialContent || download.Body.String() != "hello" {
		t.Fatalf("download = %d %q", download.Code, download.Body.String())
	}
	if download.Header().Get("Content-Security-Policy") == "" ||
		download.Header().Get("X-Content-Type-Options") != "nosniff" ||
		download.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("download security headers = %#v", download.Header())
	}
	streamCalls := agent.snapshotStreamCalls()
	if len(streamCalls) != 1 || streamCalls[0].headers.Get("Range") != "bytes=0-4" {
		t.Fatalf("range not forwarded: %#v", streamCalls)
	}

	agent.streamStatus = http.StatusCreated
	agent.streamResponse = []byte(`{"name":"upload.txt","path":"/upload.txt"}`)
	withoutCSRF := authenticatedRequest(
		server, http.MethodPost, "/api/v1/files/upload?path=%2F&name=upload.txt",
		[]byte("payload"), sessionCookie, csrfCookie,
		map[string]string{"Content-Type": "application/octet-stream", "Origin": "http://panel.test"},
	)
	if withoutCSRF.Code != http.StatusForbidden {
		t.Fatalf("upload without CSRF = %d", withoutCSRF.Code)
	}

	upload := authenticatedRequest(
		server, http.MethodPost, "/api/v1/files/upload?path=%2F&name=upload.txt",
		[]byte("payload"), sessionCookie, csrfCookie,
		map[string]string{
			"Content-Type": "application/octet-stream",
			"Origin":       "http://panel.test",
			"X-CSRF-Token": csrfCookie.Value,
		},
	)
	if upload.Code != http.StatusCreated {
		t.Fatalf("upload = %d %s", upload.Code, upload.Body.String())
	}
	streamCalls = agent.snapshotStreamCalls()
	if len(streamCalls) != 2 || streamCalls[1].method != http.MethodPost ||
		string(streamCalls[1].body) != "payload" {
		t.Fatalf("upload stream calls = %#v", streamCalls)
	}
}

func TestFileActionUsesFixedEnumAndWritesAudit(t *testing.T) {
	server, tokenPath := newTestServer(t)
	sessionCookie, csrfCookie := bootstrapCookies(t, server, tokenPath)
	agent := &fileStubAgent{stubAgent: &stubAgent{response: AgentResponse{
		StatusCode:  http.StatusOK,
		ContentType: "application/json",
		Body:        []byte(`{"action":"trash","succeeded":[{"path":"/old.txt"}]}`),
	}}}
	server.agent = agent
	headers := map[string]string{
		"Content-Type": "application/json",
		"Origin":       "http://panel.test",
		"X-CSRF-Token": csrfCookie.Value,
	}

	rejected := authenticatedRequest(
		server, http.MethodPost, "/api/v1/files/actions",
		[]byte(`{"action":"shell","sources":["/old.txt"]}`),
		sessionCookie, csrfCookie, headers,
	)
	if rejected.Code != http.StatusUnprocessableEntity || len(agent.snapshotCalls()) != 0 {
		t.Fatalf("unsupported action = %d calls=%#v", rejected.Code, agent.snapshotCalls())
	}

	response := authenticatedRequest(
		server, http.MethodPost, "/api/v1/files/actions",
		[]byte(`{"action":"trash","sources":["/old.txt"]}`),
		sessionCookie, csrfCookie, headers,
	)
	if response.Code != http.StatusOK {
		t.Fatalf("trash action = %d %s", response.Code, response.Body.String())
	}
	events, _ := server.store.ListAudit(20, "")
	var intent, success bool
	for _, event := range events {
		if event.Action != "file.trash" {
			continue
		}
		intent = intent || event.Result == "intent"
		success = success || event.Result == "success"
	}
	if !intent || !success {
		t.Fatalf("file audit events missing: %#v", events)
	}
}

func TestFileActionAuditsPartialFailure(t *testing.T) {
	server, tokenPath := newTestServer(t)
	sessionCookie, csrfCookie := bootstrapCookies(t, server, tokenPath)
	agent := &fileStubAgent{stubAgent: &stubAgent{response: AgentResponse{
		StatusCode:  http.StatusMultiStatus,
		ContentType: "application/json",
		Body: []byte(
			`{"action":"trash","succeeded":[{"path":"/one.txt"}],` +
				`"failed":[{"path":"/missing.txt","detail":"文件不存在"}]}`,
		),
	}}}
	server.agent = agent
	response := authenticatedRequest(
		server, http.MethodPost, "/api/v1/files/actions",
		[]byte(`{"action":"trash","sources":["/one.txt","/missing.txt"]}`),
		sessionCookie, csrfCookie, map[string]string{
			"Content-Type": "application/json",
			"Origin":       "http://panel.test",
			"X-CSRF-Token": csrfCookie.Value,
		},
	)
	if response.Code != http.StatusMultiStatus {
		t.Fatalf("partial response = %d %s", response.Code, response.Body.String())
	}
	events, _ := server.store.ListAudit(20, "")
	for _, event := range events {
		if event.Action == "file.trash" && event.Result == "partial_failure" {
			return
		}
	}
	t.Fatalf("partial failure audit missing: %#v", events)
}
