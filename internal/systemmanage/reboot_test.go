package systemmanage

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

func TestScheduleRebootUsesDelayedFixedSystemdUnit(t *testing.T) {
	var command string
	var arguments []string
	runner := &fakeRunner{
		run: func(_ context.Context, name string, values ...string) ([]byte, error) {
			command = name
			arguments = append([]string(nil), values...)
			return nil, nil
		},
	}
	manager, _, _, _ := testManager(t, runner)
	changed, message, err := manager.scheduleReboot(
		context.Background(),
		contract.SystemActionRequest{Action: "reboot", Confirmation: "REBOOT"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !changed || !strings.Contains(message, "15 秒") {
		t.Fatalf("unexpected result: changed=%v message=%q", changed, message)
	}
	if command != "systemd-run" {
		t.Fatalf("command = %q, want systemd-run", command)
	}
	for _, expected := range []string{
		"--collect",
		"--no-block",
		"--on-active=15s",
		"--timer-property=AccuracySec=1s",
		"--property=NoNewPrivileges=yes",
		"--property=ProtectSystem=strict",
		"--",
		"/usr/bin/systemctl",
		"--no-wall",
		"reboot",
	} {
		if !slices.Contains(arguments, expected) {
			t.Fatalf("reboot arguments missing %q: %#v", expected, arguments)
		}
	}
	if len(arguments) == 0 || arguments[0] != "--unit="+rebootUnitName {
		t.Fatalf("unsafe or missing transient unit name: %#v", arguments)
	}
	if _, _, err := manager.scheduleReboot(
		context.Background(),
		contract.SystemActionRequest{Action: "reboot", Confirmation: "REBOOT"},
	); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate schedule error = %v, want ErrConflict", err)
	}
}

func TestScheduleRebootRequiresExactIsolatedConfirmation(t *testing.T) {
	manager, _, _, _ := testManager(t, &fakeRunner{})
	tests := []contract.SystemActionRequest{
		{Action: "reboot"},
		{Action: "reboot", Confirmation: "reboot"},
		{Action: "reboot", Confirmation: "REBOOT", Hostname: "unexpected"},
	}
	for _, input := range tests {
		if _, _, err := manager.scheduleReboot(context.Background(), input); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("input %#v error = %v, want ErrInvalidInput", input, err)
		}
	}
}

func TestScheduleRebootRejectsRunningMaintenance(t *testing.T) {
	manager, _, _, _ := testManager(t, &fakeRunner{})
	if err := manager.writeMaintenance(contract.SystemMaintenanceSummary{State: "running"}); err != nil {
		t.Fatal(err)
	}
	_, _, err := manager.scheduleReboot(
		context.Background(),
		contract.SystemActionRequest{Action: "reboot", Confirmation: "REBOOT"},
	)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("scheduleReboot() error = %v, want ErrConflict", err)
	}
}

func TestRebootCapabilityRequiresSystemdTools(t *testing.T) {
	manager, _, _, _ := testManager(t, &fakeRunner{missing: map[string]bool{"systemd-run": true}})
	capability := findCapability(manager.Capabilities(), "system.reboot.write")
	if capability.Enabled || !strings.Contains(capability.Reason, "systemd-run") {
		t.Fatalf("unexpected reboot capability: %#v", capability)
	}
}
