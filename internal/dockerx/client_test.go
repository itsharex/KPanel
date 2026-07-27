package dockerx

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestOwnershipAndSafety(t *testing.T) {
	web := t.TempDir()
	apps := t.TempDir()
	state := t.TempDir()
	c := &Client{
		webRoot:   filepath.ToSlash(web),
		appRoot:   filepath.ToSlash(apps),
		stateRoot: filepath.ToSlash(state),
	}
	var raw containerInspect
	raw.ID = strings.Repeat("a", 64)
	raw.Name = "/nginx"
	raw.Config.Image = "nginx:stable"
	raw.Config.Labels = map[string]string{
		"com.docker.compose.project.working_dir": filepath.ToSlash(web),
	}
	raw.State.Status = "running"
	raw.Mounts = []dockerMount{{
		Type: "bind", Source: filepath.ToSlash(filepath.Join(web, "conf.d")),
		Destination: "/etc/nginx/conf.d", RW: true,
	}}
	if err := ensureDir(filepath.Join(web, "conf.d")); err != nil {
		t.Fatal(err)
	}
	got := c.summaryFromInspect(raw)
	if got.Ownership != "kejilion" || !contains(got.AllowedActions, "restart") {
		t.Fatalf("expected safely managed container, got %#v", got)
	}

	raw.HostConfig.Privileged = true
	got = c.summaryFromInspect(raw)
	if len(got.AllowedActions) != 0 {
		t.Fatalf("privileged container must be read-only: %#v", got)
	}
}

func TestExplicitOwnershipStillRejectsDockerSocket(t *testing.T) {
	root := t.TempDir()
	c := &Client{webRoot: filepath.ToSlash(root), appRoot: "/home/docker", stateRoot: "/var/lib/kejilion-panel"}
	var raw containerInspect
	raw.Config.Labels = map[string]string{"io.kejilion.panel.managed": "true"}
	raw.State.Status = "running"
	raw.Mounts = []dockerMount{{Type: "bind", Source: "/var/run/docker.sock", Destination: "/var/run/docker.sock"}}
	if got := c.summaryFromInspect(raw); len(got.AllowedActions) != 0 {
		t.Fatalf("Docker Socket mount must be read-only: %#v", got)
	}
}

func TestDemuxAndRedactLogs(t *testing.T) {
	payload := []byte("token=super-secret\nhttps://user:pass@example.test/path\n")
	var stream bytes.Buffer
	header := make([]byte, 8)
	header[0] = 1
	binary.BigEndian.PutUint32(header[4:], uint32(len(payload)))
	stream.Write(header)
	stream.Write(payload)
	lines := redactLines(demuxDockerStream(stream.Bytes()), 20)
	joined := strings.Join(lines, "\n")
	if strings.Contains(joined, "super-secret") || strings.Contains(joined, "user:pass") {
		t.Fatalf("secret was not redacted: %s", joined)
	}
}

func TestRedactJSONAndPrefixedSecretKeys(t *testing.T) {
	input := []byte(`{"password":"json-secret","OPENAI_API_KEY":"sk-secret","safe":"visible"}`)
	joined := strings.Join(redactLines(input, 20), "\n")
	if strings.Contains(joined, "json-secret") || strings.Contains(joined, "sk-secret") {
		t.Fatalf("JSON secret was not redacted: %s", joined)
	}
	if !strings.Contains(joined, `"safe":"visible"`) {
		t.Fatalf("non-secret field was unexpectedly changed: %s", joined)
	}
}

func TestContainerLogsRejectsExternalContainer(t *testing.T) {
	id := strings.Repeat("a", 64)
	logRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/containers/" + id + "/json":
			_ = json.NewEncoder(w).Encode(containerInspect{
				ID: id,
			})
		case "/containers/" + id + "/logs":
			logRequests++
			_, _ = w.Write([]byte("must not be read"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := testHTTPClient(server)
	_, err := client.ContainerLogs(context.Background(), id, 20)
	if !errors.Is(err, ErrReadOnlyContainer) {
		t.Fatalf("ContainerLogs() error = %v, want ErrReadOnlyContainer", err)
	}
	if logRequests != 0 {
		t.Fatalf("external container log endpoint was called %d times", logRequests)
	}
}

func TestClientDoesNotSocketActivateDockerByDefault(t *testing.T) {
	client := New(filepath.Join(t.TempDir(), "docker.sock"), t.TempDir(), t.TempDir())
	client.ConfigureDaemonAccess(filepath.Join(t.TempDir(), "missing.pid"), false)
	if err := client.Ping(context.Background()); !errors.Is(err, ErrDockerNotRunning) {
		t.Fatalf("Ping() error = %v, want ErrDockerNotRunning", err)
	}
}

func TestLifecycleSerializesAndRejectsStaleRestart(t *testing.T) {
	id := strings.Repeat("b", 64)
	var stateMu sync.Mutex
	restartCount := 0
	startedAt := "2026-07-25T00:00:00Z"
	postCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/containers/" + id + "/json":
			stateMu.Lock()
			raw := managedInspect(id, startedAt, restartCount)
			stateMu.Unlock()
			_ = json.NewEncoder(w).Encode(raw)
		case "/containers/" + id + "/restart":
			time.Sleep(20 * time.Millisecond)
			stateMu.Lock()
			postCount++
			restartCount++
			startedAt = "2026-07-25T00:00:01Z"
			stateMu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := testHTTPClient(server)
	initial := managedInspect(id, startedAt, restartCount)
	expected := client.summaryFromInspect(initial).ResourceVersion
	errs := make(chan error, 2)
	for range 2 {
		go func() {
			_, err := client.Lifecycle(context.Background(), id, "restart", expected)
			errs <- err
		}()
	}
	first, second := <-errs, <-errs
	if !((first == nil && errors.Is(second, ErrResourceConflict)) ||
		(second == nil && errors.Is(first, ErrResourceConflict))) {
		t.Fatalf("concurrent restart errors = (%v, %v), want one success and one conflict", first, second)
	}
	stateMu.Lock()
	defer stateMu.Unlock()
	if postCount != 1 {
		t.Fatalf("restart endpoint called %d times, want 1", postCount)
	}
}

func testHTTPClient(server *httptest.Server) *Client {
	root := filepath.ToSlash(server.URL)
	return &Client{
		httpClient: server.Client(),
		baseURL:    server.URL,
		webRoot:    root,
		appRoot:    root + "/apps",
		stateRoot:  root + "/state",
		now:        time.Now,
	}
}

func managedInspect(id, startedAt string, restartCount int) containerInspect {
	var raw containerInspect
	raw.ID = id
	raw.Name = "/managed"
	raw.Config.Image = "example:latest"
	raw.Config.Labels = map[string]string{"io.kejilion.panel.managed": "true"}
	raw.State.Status = "running"
	raw.State.Running = true
	raw.State.StartedAt = startedAt
	raw.RestartCount = restartCount
	raw.NetworkSettings.Networks = map[string]dockerNetworkEndpoint{}
	raw.NetworkSettings.Ports = map[string][]struct {
		HostIP   string `json:"HostIp"`
		HostPort string `json:"HostPort"`
	}{}
	return raw
}

func ensureDir(path string) error {
	return osMkdirAll(path)
}
