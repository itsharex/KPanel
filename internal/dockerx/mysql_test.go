package dockerx

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestDropSiteDatabaseUsesFixedManagedContainerAndSecretEnvironment(t *testing.T) {
	containerID := strings.Repeat("a", 64)
	execID := strings.Repeat("b", 64)
	database := "blog_example_com"
	password := "root password with shell $ characters"
	raw := managedInspect(containerID, "2026-07-25T00:00:00Z", 0)
	raw.Name = "/mysql"
	raw.Config.Env = []string{"MYSQL_ROOT_PASSWORD=" + password}

	var sequence []string
	var create struct {
		Cmd []string `json:"Cmd"`
		Env []string `json:"Env"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sequence = append(sequence, r.Method+" "+r.URL.Path)
		switch r.Method + " " + r.URL.Path {
		case "GET /containers/mysql/json":
			_ = json.NewEncoder(w).Encode(raw)
		case "POST /containers/" + containerID + "/exec":
			if err := json.NewDecoder(r.Body).Decode(&create); err != nil {
				t.Fatal(err)
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"Id": execID})
		case "POST /exec/" + execID + "/start":
			_, _ = w.Write(dockerMux([]byte(database + "\n")))
		case "GET /exec/" + execID + "/json":
			_ = json.NewEncoder(w).Encode(nginxExecState{ID: execID, ExitCode: 0})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := testHTTPClient(server)
	dropped, err := client.DropSiteDatabase(context.Background(), "blog.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if !dropped {
		t.Fatal("DropSiteDatabase() did not report the existing database")
	}
	wantSequence := []string{
		"GET /containers/mysql/json",
		"POST /containers/" + containerID + "/exec",
		"POST /exec/" + execID + "/start",
		"GET /exec/" + execID + "/json",
	}
	if !reflect.DeepEqual(sequence, wantSequence) {
		t.Fatalf("Docker API sequence = %#v, want %#v", sequence, wantSequence)
	}
	if len(create.Cmd) != 7 || create.Cmd[0] != "mysql" ||
		create.Cmd[1] != "-u" || create.Cmd[2] != "root" ||
		!strings.Contains(create.Cmd[6], "DROP DATABASE IF EXISTS `"+database+"`") {
		t.Fatalf("unexpected fixed database command: %#v", create.Cmd)
	}
	if strings.Contains(strings.Join(create.Cmd, "\x00"), password) {
		t.Fatalf("database credential leaked into command arguments: %#v", create.Cmd)
	}
	if !reflect.DeepEqual(create.Env, []string{"MYSQL_PWD=" + password}) {
		t.Fatalf("credential environment changed: %#v", create.Env)
	}
}

func TestSiteDatabaseNameMatchesKejilionWebDelete(t *testing.T) {
	tests := map[string]string{
		"example.com":     "example_com",
		"www.example.com": "www_example_com",
		"a-b.example.com": "a_b_example_com",
	}
	for domain, want := range tests {
		if got := siteDatabaseName(domain); got != want {
			t.Fatalf("siteDatabaseName(%q) = %q, want %q", domain, got, want)
		}
	}
}

func TestDropSiteDatabaseTreatsMissingFixedContainerAsNoDatabase(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method == http.MethodGet && r.URL.Path == "/containers/mysql/json" {
			http.Error(w, `{"message":"No such container: mysql"}`, http.StatusNotFound)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := testHTTPClient(server)
	dropped, err := client.DropSiteDatabase(context.Background(), "static.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if dropped {
		t.Fatal("missing fixed MySQL container was reported as a dropped database")
	}
	if requests != 1 {
		t.Fatalf("missing MySQL container triggered %d Docker requests, want 1", requests)
	}
}
