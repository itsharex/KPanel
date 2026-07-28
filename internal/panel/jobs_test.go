package panel

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kejilion/kejilion-panel/internal/appmarket"
	"github.com/kejilion/kejilion-panel/internal/contract"
	"github.com/kejilion/kejilion-panel/internal/store"
	"github.com/kejilion/kejilion-panel/internal/webenv"
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

func TestApplicationJobsMapToManagementJobs(t *testing.T) {
	now := time.Date(2026, time.July, 26, 8, 0, 0, 0, time.UTC)
	finished := now.Add(time.Minute)
	jobs := jobsFromAppJobs([]appmarket.AppJob{
		{
			ID: strings.Repeat("a", 32), AppID: "builtin-4", AppName: "Nginx Proxy Manager",
			Action: "install", Status: "failed", Stage: "failed", Progress: 100,
			Message: "port conflict", CreatedAt: now, FinishedAt: &finished,
		},
		{
			ID: strings.Repeat("b", 32), AppID: "builtin-114", AppName: "OpenClaw",
			Action: "manage", Status: "cancelled", Stage: "cancelled", Progress: 100,
			Message: "ended by administrator", CreatedAt: now, FinishedAt: &finished,
		},
	})
	if len(jobs) != 2 || jobs[0].Action != "app.install" ||
		jobs[0].State != contract.JobFailedNeedsAttention ||
		jobs[0].TargetID != "builtin-4" || jobs[0].Error == nil ||
		jobs[0].Error.Detail != "port conflict" ||
		jobs[1].State != contract.JobCancelled || jobs[1].Error != nil {
		t.Fatalf("application job mapping = %#v", jobs)
	}
}

func TestWebEnvironmentJobsMapToManagementJobs(t *testing.T) {
	now := time.Date(2026, time.July, 28, 8, 0, 0, 0, time.UTC)
	finished := now.Add(time.Minute)
	jobs := jobsFromWebEnvironment([]webenv.Job{
		{
			ID: strings.Repeat("b", 32), Action: "restore", Target: "web_20260728080000.tar.gz",
			Status: "needs_attention", Stage: "receipt_missing", Progress: 100,
			Message: "completion receipt missing", CreatedAt: now, FinishedAt: &finished,
		},
	})
	if len(jobs) != 1 || jobs[0].Action != "web.environment.restore" ||
		jobs[0].State != contract.JobFailedNeedsAttention ||
		jobs[0].TargetKind != "web_environment" || jobs[0].Error == nil ||
		jobs[0].Error.Detail != "completion receipt missing" {
		t.Fatalf("web environment job mapping = %#v", jobs)
	}
}
