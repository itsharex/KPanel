package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/kejilion/kejilion-panel/internal/contract"
	"github.com/kejilion/kejilion-panel/internal/systemmanage"
)

type systemLogTestRunner struct {
	commands []string
}

func (runner *systemLogTestRunner) Run(_ context.Context, name string, arguments ...string) ([]byte, error) {
	return runner.output(name, arguments...)
}

func (runner *systemLogTestRunner) RunResource(
	_ context.Context,
	_ int,
	_ []byte,
	name string,
	arguments ...string,
) ([]byte, []byte, error) {
	output, err := runner.output(name, arguments...)
	return output, nil, err
}

func (runner *systemLogTestRunner) LookPath(name string) (string, error) {
	if filepath.IsAbs(name) {
		return name, nil
	}
	return "/usr/bin/" + name, nil
}

func (runner *systemLogTestRunner) output(name string, arguments ...string) ([]byte, error) {
	runner.commands = append(runner.commands, strings.Join(append([]string{name}, arguments...), " "))
	switch name {
	case "du":
		return []byte("4\t/var/log\n"), nil
	case "journalctl":
		if len(arguments) > 0 && arguments[0] == "--disk-usage" {
			return []byte("Archived and active journals take up 1.0M in the file system.\n"), nil
		}
		return []byte(`{"__CURSOR":"cursor","__REALTIME_TIMESTAMP":"1785034800000000","PRIORITY":"6","_SYSTEMD_UNIT":"ssh.service","MESSAGE":"ready"}` + "\n"), nil
	default:
		return nil, nil
	}
}

func TestSystemLogRoutesRejectUnsafeQueriesBeforeHostAccess(t *testing.T) {
	server := testServer(t)
	tests := []struct {
		method string
		path   string
		status int
	}{
		{http.MethodGet, "/v1/system/logs/summary?source=system", http.StatusUnprocessableEntity},
		{http.MethodGet, "/v1/system/logs?source=service&unit=ssh.service&limit=50&priority=warning", http.StatusUnprocessableEntity},
		{http.MethodGet, "/v1/system/logs?source=system&source=login", http.StatusUnprocessableEntity},
		{http.MethodGet, "/v1/system/logs?source=security&priority=all", http.StatusUnprocessableEntity},
		{http.MethodPost, "/v1/system/logs/summary", http.StatusMethodNotAllowed},
	}
	for _, test := range tests {
		request := httptest.NewRequest(test.method, test.path, nil)
		request.Header.Set("Authorization", "Bearer "+strings.Repeat("x", 32))
		response := httptest.NewRecorder()
		server.ServeHTTP(response, request)
		if response.Code != test.status {
			t.Fatalf("%s %s status=%d body=%s", test.method, test.path, response.Code, response.Body.String())
		}
	}

	server.systemLogsGate <- struct{}{}
	request := httptest.NewRequest(http.MethodGet, "/v1/system/logs?source=system&limit=50&priority=all", nil)
	request.Header.Set("Authorization", "Bearer "+strings.Repeat("x", 32))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	<-server.systemLogsGate
	if response.Code != http.StatusTooManyRequests || !strings.Contains(response.Body.String(), "system_logs_busy") {
		t.Fatalf("busy response=%d body=%s", response.Code, response.Body.String())
	}
}

func TestSystemLogHandlerMapsDeadlineToGatewayTimeout(t *testing.T) {
	server := testServer(t)
	response := httptest.NewRecorder()
	server.writeSystemLogError(response, "request-id", context.DeadlineExceeded)
	if response.Code != http.StatusGatewayTimeout || !strings.Contains(response.Body.String(), "system_logs_timeout") {
		t.Fatalf("deadline response=%d body=%s", response.Code, response.Body.String())
	}
}

func TestSystemLogRoutesReturnStructuredBoundedData(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("system log collection is Linux-only")
	}
	server := testServer(t)
	runner := &systemLogTestRunner{}
	server.systemManager = systemmanage.NewManager(systemmanage.Config{
		LogRoot: filepath.Join(t.TempDir(), "log"), StateDir: t.TempDir(), Runner: runner,
		EffectiveUID: func() int { return 0 },
		Now:          func() time.Time { return time.Date(2026, 8, 24, 9, 0, 0, 0, time.UTC) },
	})

	request := httptest.NewRequest(http.MethodGet, "/v1/system/logs/summary", nil)
	request.Header.Set("Authorization", "Bearer "+strings.Repeat("x", 32))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("summary status=%d body=%s", response.Code, response.Body.String())
	}
	var summary contract.SystemLogSummary
	if err := json.Unmarshal(response.Body.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	if !summary.VarLog.Available || summary.VarLog.Bytes != 4096 || !summary.Journal.Available ||
		summary.Maintenance.State != "idle" {
		t.Fatalf("unexpected summary: %#v", summary)
	}

	request = httptest.NewRequest(http.MethodGet, "/v1/system/logs?source=service&limit=50&priority=all", nil)
	request.Header.Set("Authorization", "Bearer "+strings.Repeat("x", 32))
	response = httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("logs status=%d body=%s", response.Code, response.Body.String())
	}
	var snapshot contract.SystemLogSnapshot
	if err := json.Unmarshal(response.Body.Bytes(), &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.Source != "service" || len(snapshot.Entries) != 1 ||
		snapshot.Entries[0].Cursor != "cursor" || snapshot.Entries[0].Message != "ready" {
		t.Fatalf("unexpected snapshot: %#v", snapshot)
	}
}

func TestSystemLogCleanupActionReturnsPersistedTaskIdentity(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("system actions are Linux-only")
	}
	server := testServer(t)
	runner := &systemLogTestRunner{}
	manager := systemmanage.NewManager(systemmanage.Config{
		Enabled: true, StateDir: t.TempDir(), Executable: "/usr/local/libexec/kejilion-agent", Runner: runner,
		EffectiveUID: func() int { return 0 },
		Now:          func() time.Time { return time.Date(2026, 8, 24, 9, 0, 0, 0, time.UTC) },
	})
	server.systemManager = manager
	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/system/actions",
		strings.NewReader(`{"action":"log-cleanup","maintenancePolicy":"retain-3d"}`),
	)
	request.Header.Set("Authorization", "Bearer "+strings.Repeat("x", 32))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("action status=%d body=%s", response.Code, response.Body.String())
	}
	var result contract.SystemActionResult
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	status := manager.MaintenanceStatus()
	if result.TaskID == "" || result.TaskID != status.ID || result.Action != "log-cleanup" ||
		result.MaintenancePolicy != "retain-3d" || status.Action != result.Action ||
		status.Policy != result.MaintenancePolicy {
		t.Fatalf("result=%#v maintenance=%#v", result, status)
	}
}
