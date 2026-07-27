//go:build linux

package appmarket

import (
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLinuxTerminalProcessSupportsInteractiveRead(t *testing.T) {
	command := exec.Command(
		"/bin/bash",
		"-c",
		"printf 'name: '; IFS= read -r value; printf '\\r\\nhello %s\\r\\n' \"$value\"",
	)
	process, err := startTerminalProcess(command, 24, 80)
	if err != nil {
		t.Fatal(err)
	}
	defer process.Close()
	if _, err := process.Write([]byte("kpanel\n")); err != nil {
		t.Fatal(err)
	}
	output, readErr := io.ReadAll(process)
	if readErr != nil && !isTerminalEnd(readErr) {
		t.Fatal(readErr)
	}
	if err := process.Wait(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(output), "hello kpanel") {
		t.Fatalf("terminal output = %q", output)
	}
}

func TestLinuxTerminalInputFIFOTransfersBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "terminal.input")
	if err := createTerminalInput(path); err != nil {
		t.Fatal(err)
	}
	defer removeTerminalInput(path)
	reader, err := openTerminalInput(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	want := []byte("8998\r")
	if err := writeTerminalInput(path, want); err != nil {
		t.Fatal(err)
	}
	got := make([]byte, len(want))
	if _, err := io.ReadFull(reader, got); err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("FIFO input = %q, want %q", got, want)
	}
}
