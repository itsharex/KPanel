package dockerx

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestComposeDeploymentPersistsProjectAndVerifiesContainer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/containers/json" {
			_, _ = response.Write([]byte(`[]`))
			return
		}
		http.NotFound(response, request)
	}))
	defer server.Close()
	client := testHTTPClient(server)
	client.appRoot = t.TempDir()
	var calls []string
	client.composeCommand = func(_ context.Context, arguments ...string) ([]byte, error) {
		calls = append(calls, strings.Join(arguments, " "))
		switch {
		case containsArgumentSequence(arguments, "config", "--services"):
			return []byte("app\n"), nil
		case containsArgumentSequence(arguments, "up", "--detach"):
			return []byte("started\n"), nil
		case containsArgumentSequence(arguments, "ps", "--all", "--quiet"):
			return []byte(strings.Repeat("a", 64) + "\n"), nil
		default:
			return nil, errors.New("unexpected Compose command")
		}
	}
	input := MaintenanceInput{
		Action: "compose_deploy", Name: "demo",
		Compose: "services:\n  app:\n    image: nginx:alpine\n",
	}
	if err := client.deployComposeProject(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(client.appRoot, "demo", "docker-compose.yml"))
	if err != nil || string(data) != input.Compose {
		t.Fatalf("persisted Compose source = %q, %v", data, err)
	}
	joined := strings.Join(calls, "\n")
	if !strings.Contains(joined, "--project-name demo config --services") ||
		!strings.Contains(joined, "--project-name demo up --detach") ||
		!strings.Contains(joined, "--project-name demo ps --all --quiet") {
		t.Fatalf("Compose calls = %q", joined)
	}
}

func TestComposeDeploymentRollsBackFailedStart(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/containers/json" {
			_, _ = response.Write([]byte(`[]`))
			return
		}
		http.NotFound(response, request)
	}))
	defer server.Close()
	client := testHTTPClient(server)
	client.appRoot = t.TempDir()
	rolledBack := false
	client.composeCommand = func(_ context.Context, arguments ...string) ([]byte, error) {
		switch {
		case containsArgumentSequence(arguments, "config", "--services"):
			return []byte("app\n"), nil
		case containsArgumentSequence(arguments, "up", "--detach"):
			return []byte("port is already allocated"), errors.New("exit status 1")
		case containsArgumentSequence(arguments, "down", "--remove-orphans"):
			rolledBack = true
			return nil, nil
		default:
			return nil, errors.New("unexpected Compose command")
		}
	}
	err := client.deployComposeProject(context.Background(), MaintenanceInput{
		Action: "compose_deploy", Name: "demo",
		Compose: "services:\n  app:\n    image: nginx:alpine\n",
	})
	if err == nil || !strings.Contains(err.Error(), "rolled back") || !rolledBack {
		t.Fatalf("rollback result: err=%v rolledBack=%v", err, rolledBack)
	}
	if _, statErr := os.Stat(filepath.Join(client.appRoot, "demo")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("Compose project directory remains after rollback: %v", statErr)
	}
}

func TestComposeDeploymentKeepsProjectFilesWhenRollbackNeedsAttention(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/containers/json" {
			_, _ = response.Write([]byte(`[]`))
			return
		}
		http.NotFound(response, request)
	}))
	defer server.Close()
	client := testHTTPClient(server)
	client.appRoot = t.TempDir()
	client.composeCommand = func(_ context.Context, arguments ...string) ([]byte, error) {
		switch {
		case containsArgumentSequence(arguments, "config", "--services"):
			return []byte("app\n"), nil
		case containsArgumentSequence(arguments, "up", "--detach"):
			return []byte("start failed"), errors.New("exit status 1")
		case containsArgumentSequence(arguments, "down", "--remove-orphans"):
			return []byte("daemon unavailable"), errors.New("exit status 1")
		default:
			return nil, errors.New("unexpected Compose command")
		}
	}
	err := client.deployComposeProject(context.Background(), MaintenanceInput{
		Action: "compose_deploy", Name: "demo",
		Compose: "services:\n  app:\n    image: nginx:alpine\n",
	})
	if err == nil || !strings.Contains(err.Error(), "needs attention") {
		t.Fatalf("rollback attention error = %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(client.appRoot, "demo", "docker-compose.yml")); statErr != nil {
		t.Fatalf("Compose source needed for recovery was removed: %v", statErr)
	}
}

func TestComposeDeploymentRejectsExistingProjectBeforeWrite(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/containers/json" {
			_, _ = response.Write([]byte(`[]`))
			return
		}
		http.NotFound(response, request)
	}))
	defer server.Close()
	client := testHTTPClient(server)
	client.appRoot = t.TempDir()
	if err := os.Mkdir(filepath.Join(client.appRoot, "demo"), 0o750); err != nil {
		t.Fatal(err)
	}
	err := client.validateComposeDeploymentInput(context.Background(), MaintenanceInput{
		Action: "compose_deploy", Name: "demo", Compose: "services:\n  app:\n    image: nginx\n",
	})
	if !errors.Is(err, ErrResourceConflict) {
		t.Fatalf("existing project error = %v", err)
	}
}

func containsArgumentSequence(arguments []string, sequence ...string) bool {
	for index := 0; index+len(sequence) <= len(arguments); index++ {
		if strings.Join(arguments[index:index+len(sequence)], "\x00") == strings.Join(sequence, "\x00") {
			return true
		}
	}
	return false
}
