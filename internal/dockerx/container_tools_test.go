package dockerx

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestContainerStatsReturnsBoundedSingleSample(t *testing.T) {
	id := strings.Repeat("a", 64)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/containers/" + id + "/json":
			_ = json.NewEncoder(response).Encode(managedInspect(id, "2026-01-01T00:00:00Z", 0))
		case "/containers/" + id + "/stats":
			if request.URL.Query().Get("stream") != "false" || request.URL.Query().Get("one-shot") != "true" {
				t.Fatalf("unexpected stats query: %s", request.URL.RawQuery)
			}
			_, _ = response.Write([]byte(`{
				"cpu_stats":{"cpu_usage":{"total_usage":300,"percpu_usage":[150,150]},"system_cpu_usage":3000,"online_cpus":2},
				"precpu_stats":{"cpu_usage":{"total_usage":200},"system_cpu_usage":2000},
				"memory_stats":{"usage":1000,"limit":2000,"stats":{"inactive_file":100}},
				"networks":{"eth0":{"rx_bytes":11,"tx_bytes":13}},
				"blkio_stats":{"io_service_bytes_recursive":[{"op":"Read","value":17},{"op":"Write","value":19}]},
				"pids_stats":{"current":7}
			}`))
		default:
			http.NotFound(response, request)
		}
	}))
	defer server.Close()

	client := testHTTPClient(server)
	stats, err := client.ContainerStats(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if stats.CPUPercent != 20 || stats.MemoryBytes != 900 || stats.MemoryPercent != 45 ||
		stats.NetworkRx != 11 || stats.NetworkTx != 13 ||
		stats.BlockRead != 17 || stats.BlockWrite != 19 || stats.PIDs != 7 {
		t.Fatalf("unexpected container stats: %#v", stats)
	}
}

func TestContainerExecUsesFixedShellAndDoesNotReturnCommand(t *testing.T) {
	id := strings.Repeat("b", 64)
	execID := strings.Repeat("c", 64)
	command := "printf hello"
	var createPayload struct {
		Cmd []string `json:"Cmd"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch {
		case request.Method == http.MethodGet && request.URL.Path == "/containers/"+id+"/json":
			_ = json.NewEncoder(response).Encode(managedInspect(id, "2026-01-01T00:00:00Z", 0))
		case request.Method == http.MethodPost && request.URL.Path == "/containers/"+id+"/exec":
			if err := json.NewDecoder(request.Body).Decode(&createPayload); err != nil {
				t.Fatal(err)
			}
			_, _ = response.Write([]byte(`{"Id":"` + execID + `"}`))
		case request.Method == http.MethodPost && request.URL.Path == "/exec/"+execID+"/start":
			_, _ = response.Write([]byte("hello\n"))
		case request.Method == http.MethodGet && request.URL.Path == "/exec/"+execID+"/json":
			_, _ = response.Write([]byte(`{"ID":"` + execID + `","Running":false,"ExitCode":0}`))
		default:
			http.NotFound(response, request)
		}
	}))
	defer server.Close()

	client := testHTTPClient(server)
	inspect := managedInspect(id, "2026-01-01T00:00:00Z", 0)
	version := client.summaryFromInspect(inspect).ResourceVersion
	result, err := client.ContainerExec(context.Background(), id, ContainerExecInput{
		ResourceVersion: version,
		Command:         command,
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(createPayload.Cmd, "\x00") != "/bin/sh\x00-lc\x00"+command {
		t.Fatalf("exec command = %#v", createPayload.Cmd)
	}
	if result.Output != "hello" || result.ExitCode != 0 ||
		strings.Contains(result.Output, command) {
		t.Fatalf("unexpected exec result: %#v", result)
	}
}

func TestContainerExecRejectsMultilineCommandBeforeDocker(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("invalid command reached Docker")
	}))
	defer server.Close()
	client := testHTTPClient(server)
	_, err := client.ContainerExec(
		context.Background(),
		strings.Repeat("a", 64),
		ContainerExecInput{ResourceVersion: "sha256:" + strings.Repeat("0", 64), Command: "id\nuname"},
	)
	if err != ErrUnsafeOrInvalidAction {
		t.Fatalf("invalid command error = %v", err)
	}
}

func TestManagedContainerCreateIsStructuredAndRollsBackStartFailure(t *testing.T) {
	createdID := strings.Repeat("d", 64)
	var payload struct {
		Image        string            `json:"Image"`
		Labels       map[string]string `json:"Labels"`
		ExposedPorts map[string]any    `json:"ExposedPorts"`
		HostConfig   struct {
			PortBindings  map[string][]map[string]string `json:"PortBindings"`
			RestartPolicy struct {
				Name string `json:"Name"`
			} `json:"RestartPolicy"`
		} `json:"HostConfig"`
	}
	var rollback bool
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/containers/create":
			if request.URL.Query().Get("name") != "demo" {
				t.Fatalf("create name = %q", request.URL.Query().Get("name"))
			}
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			_, _ = response.Write([]byte(`{"Id":"` + createdID + `"}`))
		case request.Method == http.MethodPost && request.URL.Path == "/containers/"+createdID+"/start":
			response.WriteHeader(http.StatusInternalServerError)
			_, _ = response.Write([]byte(`{"message":"start failed"}`))
		case request.Method == http.MethodDelete && request.URL.Path == "/containers/"+createdID:
			rollback = request.URL.Query().Get("force") == "1"
			response.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(response, request)
		}
	}))
	defer server.Close()

	client := testHTTPClient(server)
	err := client.createManagedContainer(context.Background(), MaintenanceInput{
		Action: "container_create", Name: "demo", Image: "nginx:alpine",
		RestartPolicy: "unless-stopped",
		Ports:         []ContainerCreatePort{{PrivatePort: 80, PublicPort: 8080, Protocol: "tcp", HostIP: "127.0.0.1"}},
	})
	if err == nil || !strings.Contains(err.Error(), "rolled back") || !rollback {
		t.Fatalf("create rollback result: err=%v rollback=%v", err, rollback)
	}
	if payload.Image != "nginx:alpine" ||
		payload.Labels["io.kejilion.panel.managed"] != "true" ||
		payload.HostConfig.RestartPolicy.Name != "unless-stopped" ||
		payload.HostConfig.PortBindings["80/tcp"][0]["HostIp"] != "127.0.0.1" {
		t.Fatalf("unexpected create payload: %#v", payload)
	}
}
