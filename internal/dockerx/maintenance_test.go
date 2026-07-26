package dockerx

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestImagePullRunsAsPersistentBackgroundJob(t *testing.T) {
	requested := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/images/create" {
			http.NotFound(response, request)
			return
		}
		requested <- request.URL.RawQuery
		_, _ = response.Write([]byte("{\"status\":\"Pull complete\"}\n"))
	}))
	defer server.Close()

	client := testHTTPClient(server)
	if err := client.ConfigureJobs(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	job, err := client.StartMaintenance(context.Background(), MaintenanceInput{
		Action: "image_pull", Image: "nginx:alpine",
	})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case query := <-requested:
		if query != "fromImage=nginx&tag=alpine" {
			t.Fatalf("pull query = %q", query)
		}
	case <-time.After(time.Second):
		t.Fatal("image pull did not start")
	}
	job = waitForDockerJob(t, client, job.ID)
	if job.Status != "succeeded" || job.Progress != 100 {
		t.Fatalf("unexpected completed job: %#v", job)
	}
}

func TestPruneRequiresConfirmationAndUsesFixedEndpoints(t *testing.T) {
	var mu sync.Mutex
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		mu.Lock()
		paths = append(paths, request.URL.RequestURI())
		mu.Unlock()
		_ = json.NewEncoder(response).Encode(map[string]any{})
	}))
	defer server.Close()

	client := testHTTPClient(server)
	if err := client.ConfigureJobs(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if _, err := client.StartMaintenance(context.Background(), MaintenanceInput{
		Action: "prune", Confirmation: "wrong",
	}); err != ErrInvalidDockerJob {
		t.Fatalf("invalid confirmation error = %v", err)
	}
	job, err := client.StartMaintenance(context.Background(), MaintenanceInput{
		Action: "prune", Confirmation: "PRUNE",
	})
	if err != nil {
		t.Fatal(err)
	}
	job = waitForDockerJob(t, client, job.ID)
	if job.Status != "succeeded" {
		t.Fatalf("prune failed: %#v", job)
	}
	want := []string{
		"/containers/prune",
		"/images/prune",
		"/networks/prune",
		"/volumes/prune",
		"/build/prune?all=1",
	}
	mu.Lock()
	defer mu.Unlock()
	if len(paths) != len(want) {
		t.Fatalf("prune paths = %#v", paths)
	}
	for index := range want {
		if paths[index] != want[index] {
			t.Fatalf("prune path %d = %q, want %q", index, paths[index], want[index])
		}
	}
}

func TestMaintenanceRejectsUnsafeNamesAndProtectsPanelResources(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	client := testHTTPClient(server)
	if err := client.ConfigureJobs(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if validImageReference("nginx;touch /tmp/x") || validImageReference("../nginx") ||
		!validImageReference("ghcr.io/example/app:v1") {
		t.Fatal("image reference validation did not enforce the allowlist")
	}
	if _, err := client.StartMaintenance(context.Background(), MaintenanceInput{
		Action: "network_create", Name: "bad/name",
	}); err == nil {
		t.Fatal("unsafe network name was accepted")
	}
}

func TestDockerBackupExcludesKPanelAndCreatesPrivateArchive(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	client := testHTTPClient(server)
	client.appRoot = t.TempDir()
	client.stateRoot = t.TempDir()
	if err := os.MkdirAll(filepath.Join(client.appRoot, "example", "data"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(client.appRoot, "example", "compose.yml"), []byte("services: {}\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(client.appRoot, "kpanel", "secrets"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(client.appRoot, "kpanel", "secrets", "token"), []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := client.ConfigureJobs(filepath.Join(client.stateRoot, "jobs")); err != nil {
		t.Fatal(err)
	}
	job, err := client.StartMaintenance(context.Background(), MaintenanceInput{Action: "backup_create"})
	if err != nil {
		t.Fatal(err)
	}
	job = waitForDockerJob(t, client, job.ID)
	if job.Status != "succeeded" || job.ResultPath == "" {
		t.Fatalf("backup job failed: %#v", job)
	}
	info, err := os.Stat(job.ResultPath)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("backup permissions = %v, %v", info, err)
	}
	file, err := os.Open(job.ResultPath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer gzipReader.Close()
	reader := tar.NewReader(gzipReader)
	var names []string
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, header.Name)
	}
	joined := strings.Join(names, "\n")
	if !strings.Contains(joined, "docker/example/compose.yml") || strings.Contains(joined, "kpanel") {
		t.Fatalf("unexpected backup contents:\n%s", joined)
	}
}

func TestDockerDaemonSettingsPreserveExistingKeys(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	client := testHTTPClient(server)
	client.daemonConfigPath = filepath.Join(t.TempDir(), "docker", "daemon.json")
	if err := os.MkdirAll(filepath.Dir(client.daemonConfigPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		client.daemonConfigPath,
		[]byte("{\"log-driver\":\"local\",\"storage-driver\":\"overlay2\"}\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	restarts := 0
	client.restartDocker = func(context.Context) error {
		restarts++
		return nil
	}

	if err := client.updateDaemonMirrors(context.Background(), "cn"); err != nil {
		t.Fatal(err)
	}
	if err := client.updateDaemonIPv6(context.Background(), true, "fd42:6b50:616e:656c::/64"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(client.daemonConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	if config["log-driver"] != "local" || config["storage-driver"] != "overlay2" ||
		config["ipv6"] != true || config["fixed-cidr-v6"] != "fd42:6b50:616e:656c::/64" {
		t.Fatalf("existing daemon keys were not preserved: %#v", config)
	}
	mirrors, ok := config["registry-mirrors"].([]any)
	if !ok || len(mirrors) != len(kejilionDockerMirrors) || restarts != 2 {
		t.Fatalf("mirror/restart result = %#v, restarts=%d", config["registry-mirrors"], restarts)
	}
}

func TestDockerDaemonRestartFailureRollsBack(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	client := testHTTPClient(server)
	client.daemonConfigPath = filepath.Join(t.TempDir(), "daemon.json")
	original := []byte("{\"log-driver\":\"json-file\"}\n")
	if err := os.WriteFile(client.daemonConfigPath, original, 0o644); err != nil {
		t.Fatal(err)
	}
	restarts := 0
	client.restartDocker = func(context.Context) error {
		restarts++
		if restarts == 1 {
			return errors.New("restart failed")
		}
		return nil
	}
	if err := client.updateDaemonMirrors(context.Background(), "cn"); err == nil ||
		!strings.Contains(err.Error(), "previous configuration restored") {
		t.Fatalf("rollback error = %v", err)
	}
	data, err := os.ReadFile(client.daemonConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(original) || restarts != 2 {
		t.Fatalf("daemon config was not rolled back: %q, restarts=%d", data, restarts)
	}
}

func TestDockerIPv6CIDRValidation(t *testing.T) {
	if !validDockerIPv6CIDR("fd42:6b50:616e:656c::/64") {
		t.Fatal("valid IPv6 /64 was rejected")
	}
	for _, value := range []string{
		"2001:db8:1::/64",
		"fd42:6b50:616e:656c::/48",
		"127.0.0.1/64",
		"not-a-cidr",
	} {
		if validDockerIPv6CIDR(value) {
			t.Fatalf("unsafe IPv6 CIDR was accepted: %s", value)
		}
	}
}

func waitForDockerJob(t *testing.T, client *Client, id string) MaintenanceJob {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		job, err := client.MaintenanceJob(id)
		if err != nil {
			t.Fatal(err)
		}
		if job.Status != "queued" && job.Status != "running" {
			return job
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("Docker job did not finish")
	return MaintenanceJob{}
}
