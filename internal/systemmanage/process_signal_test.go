package systemmanage

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

func TestProcessSignalRejectsStaleIdentityWithoutSignaling(t *testing.T) {
	manager, _, procRoot, _ := testManager(t, &fakeRunner{})
	mustWrite(t, filepath.Join(procRoot, "42", "stat"), "42 (worker) S 1 0 0 0 0 0 0 0 0 0 10 5 0 0 0 0 0 0 999\n")
	signaled := false
	manager.processSignaler = func(int, string) error {
		signaled = true
		return nil
	}

	_, _, err := manager.signalProcess(context.Background(), contract.SystemActionRequest{
		Action: "process-signal", PID: 42, StartTimeTicks: 998, Signal: "term",
	})
	if !errors.Is(err, ErrConflict) || signaled {
		t.Fatalf("signalProcess() err=%v signaled=%v", err, signaled)
	}
}

func TestProcessSignalRejectsUnknownSignal(t *testing.T) {
	manager, _, procRoot, _ := testManager(t, &fakeRunner{})
	mustWrite(t, filepath.Join(procRoot, "42", "stat"), "42 (worker) S 1 0 0 0 0 0 0 0 0 0 10 5 0 0 0 0 0 0 999\n")
	manager.processSignaler = func(int, string) error {
		t.Fatal("invalid signal reached the system call")
		return nil
	}

	_, _, err := manager.signalProcess(context.Background(), contract.SystemActionRequest{
		Action: "process-signal", PID: 42, StartTimeTicks: 999, Signal: "stop",
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("signalProcess() err=%v, want ErrInvalidInput", err)
	}
}

func TestProcessSignalDeliversFixedSignalAfterIdentityCheck(t *testing.T) {
	manager, _, procRoot, _ := testManager(t, &fakeRunner{})
	mustWrite(t, filepath.Join(procRoot, "42", "stat"), "42 (worker) S 1 0 0 0 0 0 0 0 0 0 10 5 0 0 0 0 0 0 999\n")
	gotPID, gotSignal := 0, ""
	manager.processSignaler = func(pid int, signal string) error {
		gotPID, gotSignal = pid, signal
		return nil
	}

	changed, message, err := manager.signalProcess(context.Background(), contract.SystemActionRequest{
		Action: "process-signal", PID: 42, StartTimeTicks: 999, Signal: "kill",
	})
	if err != nil || !changed || gotPID != 42 || gotSignal != "kill" || message == "" {
		t.Fatalf("signalProcess() changed=%v pid=%d signal=%q message=%q err=%v", changed, gotPID, gotSignal, message, err)
	}
}
