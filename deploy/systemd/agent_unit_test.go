package systemd_test

import (
	"os"
	"strings"
	"testing"
)

func TestMainAgentKeepsPrivateDevicesAndNoMountSyscallClass(t *testing.T) {
	data, err := os.ReadFile("kejilion-agent.service")
	if err != nil {
		t.Fatal(err)
	}
	unit := string(data)
	if !strings.Contains(unit, "\nPrivateDevices=true\n") {
		t.Fatal("main Agent service must retain PrivateDevices=true")
	}
	for _, line := range strings.Split(unit, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "SystemCallFilter=") && strings.Contains(line, "@mount") {
			t.Fatalf("main Agent service must not add @mount: %q", line)
		}
	}
}
