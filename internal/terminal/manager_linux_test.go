//go:build linux

package terminal

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLinuxTransientPTYEscapesAgentReadOnlyMountNamespace(t *testing.T) {
	if os.Getenv("KPANEL_TEST_TRANSIENT_TERMINAL") != "1" {
		t.Skip("set KPANEL_TEST_TRANSIENT_TERMINAL=1 inside a systemd service")
	}
	if os.Getenv("INVOCATION_ID") == "" {
		t.Fatal("transient terminal test must run inside a systemd service")
	}
	manager := New(Config{IdleTimeout: time.Minute, Lifetime: time.Minute})
	t.Cleanup(manager.CloseAll)
	snapshot, err := manager.Open("transient-linux-test", 24, 100)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	command := "for path in /usr/local /var/lib /var/cache /etc; do test -w \"$path\" || exit 42; done; printf 'KPANEL_SYSTEM_WRITE_READY\\n'\r"
	if err := manager.Input("transient-linux-test", snapshot.ID, []byte(command)); err != nil {
		t.Fatalf("Input() error = %v", err)
	}

	offset := snapshot.Offset
	deadline := time.Now().Add(10 * time.Second)
	var output strings.Builder
	for !strings.Contains(output.String(), "KPANEL_SYSTEM_WRITE_READY") && time.Now().Before(deadline) {
		chunk, outputErr := manager.Output(context.Background(), "transient-linux-test", snapshot.ID, offset, 500*time.Millisecond)
		if outputErr != nil {
			t.Fatalf("Output() error = %v", outputErr)
		}
		offset = chunk.NextOffset
		output.Write(chunk.Data)
	}
	if !strings.Contains(output.String(), "KPANEL_SYSTEM_WRITE_READY") {
		t.Fatalf("transient PTY did not obtain a writable system mount: %q", output.String())
	}
}

func TestLinuxPTYLifecycle(t *testing.T) {
	t.Setenv("INVOCATION_ID", "")
	manager := New(Config{IdleTimeout: time.Minute, Lifetime: time.Minute})
	t.Cleanup(manager.CloseAll)
	snapshot, err := manager.Open("linux-test", 24, 80)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := manager.Resize("linux-test", snapshot.ID, 30, 100); err != nil {
		t.Fatalf("Resize() error = %v", err)
	}
	if err := manager.Input("linux-test", snapshot.ID, []byte("printf 'KPANEL_PTY_READY\\n'\r")); err != nil {
		t.Fatalf("Input() error = %v", err)
	}

	offset := snapshot.Offset
	deadline := time.Now().Add(5 * time.Second)
	var output strings.Builder
	for !strings.Contains(output.String(), "KPANEL_PTY_READY") && time.Now().Before(deadline) {
		chunk, outputErr := manager.Output(context.Background(), "linux-test", snapshot.ID, offset, 250*time.Millisecond)
		if outputErr != nil {
			t.Fatalf("Output() error = %v", outputErr)
		}
		offset = chunk.NextOffset
		output.Write(chunk.Data)
	}
	if !strings.Contains(output.String(), "KPANEL_PTY_READY") {
		t.Fatalf("PTY output did not contain marker: %q", output.String())
	}
	if err := manager.Close("linux-test", snapshot.ID); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}
