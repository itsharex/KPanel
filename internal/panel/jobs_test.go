package panel

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kejilion/kejilion-panel/internal/contract"
	"github.com/kejilion/kejilion-panel/internal/store"
)

func TestJobsBuildsManagementViewWithoutAuditChange(t *testing.T) {
	server, tokenPath := newTestServer(t)
	sessionCookie, _ := bootstrapCookies(t, server, tokenPath)
	base := time.Date(2026, time.July, 25, 7, 0, 0, 0, time.UTC)
	events := []store.AuditEvent{
		{
			ID: "audit-1", OccurredAt: base, Action: "site.create",
			TargetKind: "site", TargetID: "example.com", Result: "intent",
			RequestID: "request-success", Change: map[string]any{"upstream": "secret.internal"},
		},
		{
			ID: "audit-2", OccurredAt: base.Add(time.Second), Action: "site.create",
			TargetKind: "site", TargetID: "example.com", Result: "success",
			RequestID: "request-success", Change: map[string]any{"upstream": "secret.internal"},
		},
		{
			ID: "audit-3", OccurredAt: base.Add(2 * time.Second), Action: "docker.restart",
			TargetKind: "container", TargetID: "container-1", Result: "intent",
			RequestID: "request-running", Change: map[string]any{"resourceVersion": "secret-version"},
		},
		{
			ID: "audit-4", OccurredAt: base.Add(3 * time.Second), Action: "site.update",
			TargetKind: "site", TargetID: "site-1", Result: "denied",
			RequestID: "request-failed", Change: map[string]any{"upstream": "do-not-expose"},
		},
		{
			ID: "audit-5", OccurredAt: base.Add(4 * time.Second), Action: "auth.login",
			TargetKind: "user", TargetID: "admin", Result: "failure", RequestID: "ignored",
		},
	}
	for _, event := range events {
		if err := server.store.AppendAudit(event, 10_000); err != nil {
			t.Fatal(err)
		}
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/jobs?limit=3", nil)
	request.Host = "panel.test"
	request.AddCookie(sessionCookie)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("jobs failed: %d %s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "secret") ||
		strings.Contains(response.Body.String(), "do-not-expose") ||
		strings.Contains(response.Body.String(), `"change"`) {
		t.Fatalf("jobs exposed audit change: %s", response.Body.String())
	}
	var page contract.PageResult[contract.Job]
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 3 {
		t.Fatalf("expected three management jobs, got %#v", page.Items)
	}
	if page.Items[0].ID != "request-failed" ||
		page.Items[0].State != contract.JobFailedNeedsAttention ||
		page.Items[1].ID != "request-running" ||
		page.Items[1].State != contract.JobRunning ||
		page.Items[2].ID != "request-success" ||
		page.Items[2].State != contract.JobSucceeded {
		t.Fatalf("unexpected merged jobs: %#v", page.Items)
	}
	if page.Items[2].StartedAt == nil || page.Items[2].FinishedAt == nil ||
		page.Items[2].Progress != 100 {
		t.Fatalf("successful job lacks timing/progress: %#v", page.Items[2])
	}
}

func TestJobsRequiresSessionAndStrictLimit(t *testing.T) {
	server, tokenPath := newTestServer(t)

	unauthenticated := performRequest(server, http.MethodGet, "/api/v1/jobs", nil, nil)
	if unauthenticated.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated jobs returned %d", unauthenticated.Code)
	}

	sessionCookie, _ := bootstrapCookies(t, server, tokenPath)
	for _, path := range []string{
		"/api/v1/jobs?limit=0",
		"/api/v1/jobs?limit=101",
		"/api/v1/jobs?limit=1&limit=2",
		"/api/v1/jobs?cursor=unexpected",
	} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.Host = "panel.test"
		request.AddCookie(sessionCookie)
		response := httptest.NewRecorder()
		server.ServeHTTP(response, request)
		if response.Code != http.StatusUnprocessableEntity {
			t.Errorf("%s returned %d: %s", path, response.Code, response.Body.String())
		}
	}
}
