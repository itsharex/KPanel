package dockerx

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExactDeclarativeContainerAccessModes(t *testing.T) {
	spec := DeclarativeAppSpec{
		Token: "it-tools", ContainerName: "it-tools",
		Image: "corentinth/it-tools:latest", ContainerPort: 80,
	}
	var raw containerInspect
	raw.Name = "/it-tools"
	raw.Config.Image = spec.Image
	raw.HostConfig.RestartPolicy.Name = "always"
	raw.NetworkSettings.Ports = map[string][]struct {
		HostIP   string `json:"HostIp"`
		HostPort string `json:"HostPort"`
	}{
		"80/tcp": {{HostIP: "0.0.0.0", HostPort: "8064"}},
	}
	port, access, ok := exactDeclarativeContainer(raw, spec)
	if !ok || port != 8064 || access != "direct" {
		t.Fatalf("direct container mismatch: port=%d access=%s ok=%v", port, access, ok)
	}
	bindings := raw.NetworkSettings.Ports["80/tcp"]
	bindings[0].HostIP = "127.0.0.1"
	raw.NetworkSettings.Ports["80/tcp"] = bindings
	_, access, ok = exactDeclarativeContainer(raw, spec)
	if !ok || access != "domain_only" {
		t.Fatalf("domain-only container mismatch: access=%s ok=%v", access, ok)
	}
	raw.HostConfig.Privileged = true
	if _, _, ok := exactDeclarativeContainer(raw, spec); ok {
		t.Fatal("privileged container was accepted as declarative")
	}
}

func TestInstallDeclarativeAppRollsBackWhenVerificationFails(t *testing.T) {
	id := strings.Repeat("a", 64)
	deleted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/containers/it-tools/json":
			http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		case r.Method == http.MethodPost && r.URL.Path == "/images/create":
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && r.URL.Path == "/containers/create":
			_, _ = w.Write([]byte(`{"Id":"` + id + `"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/containers/"+id+"/start":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/containers/"+id+"/json":
			http.Error(w, `{"message":"temporary inspect failure"}`, http.StatusInternalServerError)
		case r.Method == http.MethodPost && r.URL.Path == "/containers/"+id+"/stop":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && r.URL.Path == "/containers/"+id:
			deleted = true
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := testHTTPClient(server)
	_, err := client.InstallDeclarativeApp(
		context.Background(),
		DeclarativeAppSpec{
			Token: "it-tools", ContainerName: "it-tools",
			Image: "corentinth/it-tools:latest", ContainerPort: 80,
		},
		8064,
		"direct",
	)
	if !errors.Is(err, ErrAppRolledBack) {
		t.Fatalf("InstallDeclarativeApp() error = %v, want ErrAppRolledBack", err)
	}
	if !deleted {
		t.Fatal("failed installation verification left the created container behind")
	}
}
