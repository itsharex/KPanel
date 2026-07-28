package appmarket

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestAppJobTerminalReturnsRawChunks(t *testing.T) {
	root := t.TempDir()
	registry := &appJobRegistry{
		stateDir: filepath.Join(root, "jobs"),
		jobs:     make(map[string]appJobRecord),
	}
	if err := ensureAppJobDirectory(registry.stateDir); err != nil {
		t.Fatal(err)
	}
	const id = "0123456789abcdef0123456789abcdef"
	record := appJobRecord{
		AppJob: AppJob{
			ID: id, AppID: "builtin-4", AppName: "test", Action: "install",
			Interactive: true, InputOpen: true, Status: "running", Stage: "interactive",
			CreatedAt: time.Now().UTC(), Logs: []string{},
		},
		Adapter:  "kejilion",
		Selector: "4",
	}
	if err := registry.put(record); err != nil {
		t.Fatal(err)
	}
	raw := []byte("\x1b[32m原生终端\x1b[0m\r\n")
	if err := os.WriteFile(registry.logPath(id), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	service := &Service{jobs: registry}
	chunk, err := service.AppJobTerminal(id, 0)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := base64.StdEncoding.DecodeString(chunk.DataBase64)
	if err != nil {
		t.Fatal(err)
	}
	if string(decoded) != string(raw) || chunk.NextOffset != int64(len(raw)) ||
		!chunk.InputOpen || chunk.Finished {
		t.Fatalf("terminal chunk = %#v raw=%q", chunk, decoded)
	}
}

func TestFinishedTerminalDrainsEveryChunkBeforeStopping(t *testing.T) {
	root := t.TempDir()
	registry := &appJobRegistry{
		stateDir: filepath.Join(root, "jobs"),
		jobs:     make(map[string]appJobRecord),
	}
	if err := ensureAppJobDirectory(registry.stateDir); err != nil {
		t.Fatal(err)
	}
	const id = "abcdef0123456789abcdef0123456789"
	finished := time.Now().UTC()
	record := appJobRecord{
		AppJob: AppJob{
			ID: id, AppID: "builtin-4", AppName: "test", Action: "install",
			Interactive: true, Status: "succeeded", Stage: "completed",
			CreatedAt: finished, FinishedAt: &finished, Logs: []string{},
		},
		Adapter:  "kejilion",
		Selector: "4",
	}
	if err := registry.put(record); err != nil {
		t.Fatal(err)
	}
	raw := make([]byte, maxTerminalChunkBytes+1)
	if err := os.WriteFile(registry.logPath(id), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	service := &Service{jobs: registry}
	first, err := service.AppJobTerminal(id, 0)
	if err != nil {
		t.Fatal(err)
	}
	if first.Finished || first.NextOffset != maxTerminalChunkBytes {
		t.Fatalf("first terminal chunk stopped early: %#v", first)
	}
	last, err := service.AppJobTerminal(id, first.NextOffset)
	if err != nil {
		t.Fatal(err)
	}
	if !last.Finished || last.NextOffset != int64(len(raw)) {
		t.Fatalf("last terminal chunk did not finish: %#v", last)
	}
}

func TestTerminalInputValidationRejectsSecretsFromInvalidJobs(t *testing.T) {
	service := &Service{}
	if err := service.WriteAppJobInput("not-an-id", "password\n"); err == nil {
		t.Fatal("invalid job accepted terminal input")
	}
}

func TestStripTerminalControlsKeepsProgressPayload(t *testing.T) {
	got := stripTerminalControls("\x1b[2K\rKPANEL_PROGRESS 35 正在安装\x1b[0m\r\n")
	if got != "\rKPANEL_PROGRESS 35 正在安装\r\n" {
		t.Fatalf("stripped terminal output = %q", got)
	}
}

func TestInteractiveInstallEnvironmentCarriesPanelSelectedPort(t *testing.T) {
	install := interactiveAppJobEnvironment(appJobRecord{
		AppJob:   AppJob{Action: "install"},
		HostPort: 18081,
	})
	if !slices.Contains(install, "KJ_APP_PORT=18081") {
		t.Fatalf("install environment = %#v", install)
	}
	update := interactiveAppJobEnvironment(appJobRecord{
		AppJob:   AppJob{Action: "update"},
		HostPort: 18081,
	})
	if slices.Contains(update, "KJ_APP_PORT=18081") {
		t.Fatalf("update environment leaked install port: %#v", update)
	}
}
