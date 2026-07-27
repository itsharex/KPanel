package dockerx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDockerEnvironmentReflectsExistingDaemonConfiguration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/info" {
			http.NotFound(response, request)
			return
		}
		_, _ = response.Write([]byte(`{
			"ServerVersion":"28.1.1",
			"DockerRootDir":"/var/lib/docker",
			"Driver":"overlay2",
			"Containers":4,
			"Images":9
		}`))
	}))
	defer server.Close()
	client := testHTTPClient(server)
	client.daemonConfigPath = filepath.Join(t.TempDir(), "daemon.json")
	config := `{"log-driver":"local","ipv6":true,"fixed-cidr-v6":"fd42:6b50:616e:656c::/64","registry-mirrors":[`
	for index, mirror := range kejilionDockerMirrors {
		if index > 0 {
			config += ","
		}
		config += `"` + mirror + `"`
	}
	config += "]}\n"
	if err := os.WriteFile(client.daemonConfigPath, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	result := client.Environment(context.Background())
	if !result.Available || result.EngineVersion != "28.1.1" ||
		result.StorageDriver != "overlay2" || result.DataRoot != "/var/lib/docker" ||
		result.MirrorPreset != "cn" || !result.IPv6Enabled ||
		result.IPv6CIDR != "fd42:6b50:616e:656c::/64" ||
		result.DaemonConfig != "valid" {
		t.Fatalf("unexpected Docker environment: %#v", result)
	}
}

func TestDockerEnvironmentPreservesCustomMirrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		_, _ = response.Write([]byte(`{"ServerVersion":"28.1.1"}`))
	}))
	defer server.Close()
	client := testHTTPClient(server)
	client.daemonConfigPath = filepath.Join(t.TempDir(), "daemon.json")
	if err := os.WriteFile(
		client.daemonConfigPath,
		[]byte(`{"registry-mirrors":["https://mirror.example.invalid"]}`),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	result := client.Environment(context.Background())
	if result.MirrorPreset != "custom" ||
		len(result.RegistryMirrors) != 1 ||
		result.RegistryMirrors[0] != "https://mirror.example.invalid" {
		t.Fatalf("custom mirror status was not preserved: %#v", result)
	}
}

func TestDockerEnvironmentRejectsTrailingDaemonConfiguration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		_, _ = response.Write([]byte(`{"ServerVersion":"28.1.1"}`))
	}))
	defer server.Close()
	client := testHTTPClient(server)
	client.daemonConfigPath = filepath.Join(t.TempDir(), "daemon.json")
	if err := os.WriteFile(
		client.daemonConfigPath,
		[]byte(`{"registry-mirrors":[]} {"ipv6":true}`),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	result := client.Environment(context.Background())
	if result.DaemonConfig != "invalid" || result.DaemonWarning == "" ||
		result.IPv6Enabled || result.MirrorPreset != "official" {
		t.Fatalf("trailing daemon data was accepted: %#v", result)
	}
}
