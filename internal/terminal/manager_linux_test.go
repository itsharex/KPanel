//go:build linux

package terminal

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestLinuxPTYLifecycle(t *testing.T) {
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
