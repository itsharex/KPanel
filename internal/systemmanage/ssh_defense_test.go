package systemmanage

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseSSHDefenseStatus(t *testing.T) {
	status, err := parseSSHDefenseStatus([]byte(
		"KPANEL_F2B_PROTOCOL 1\n" +
			`KPANEL_F2B_STATUS {"installed":true,"running":true,"enabled":true,"autostart":true,"jail":"sshd","banned":3}` +
			"\n",
	))
	if err != nil {
		t.Fatal(err)
	}
	if !status.Installed || !status.Running || !status.Enabled || !status.Autostart ||
		status.Jail != "sshd" || status.Banned != 3 {
		t.Fatalf("SSH defense status = %#v", status)
	}

	for name, output := range map[string]string{
		"missing marker": `{"enabled":true}`,
		"unknown field":  `KPANEL_F2B_STATUS {"jail":"sshd","extra":true}`,
		"unknown jail":   `KPANEL_F2B_STATUS {"jail":"custom"}`,
		"negative count": `KPANEL_F2B_STATUS {"jail":"sshd","banned":-1}`,
		"trailing JSON":  `KPANEL_F2B_STATUS {"jail":"sshd"} {}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseSSHDefenseStatus([]byte(output)); err == nil {
				t.Fatalf("invalid status accepted: %q", output)
			}
		})
	}
}

func TestSSHDefenseStatusUsesFixedKejilionProtocol(t *testing.T) {
	runner := &fakeRunner{
		run: func(_ context.Context, name string, arguments ...string) ([]byte, error) {
			if name != "env" {
				t.Fatalf("SSH defense status command = %s %#v", name, arguments)
			}
			return []byte(
				"KPANEL_F2B_PROTOCOL 1\n" +
					`KPANEL_F2B_STATUS {"installed":true,"running":true,"enabled":true,"autostart":true,"jail":"sshd","banned":2}` +
					"\n",
			), nil
		},
	}
	manager, _, _, _ := testManager(t, runner)
	manager.f2bScript = func() (string, error) {
		return "/home/docker/kpanel/bin/kejilion.sh", nil
	}
	status := manager.SSHDefenseStatus(context.Background())
	if !status.Available || !status.Enabled || status.Banned != 2 {
		t.Fatalf("SSH defense status = %#v", status)
	}
	command := strings.Join(runner.commands, "\n")
	for _, required := range []string{
		"env KJ_F2B_NONINTERACTIVE=1",
		"bash /home/docker/kpanel/bin/kejilion.sh f2b status",
	} {
		if !strings.Contains(command, required) {
			t.Fatalf("SSH defense command missing %q: %s", required, command)
		}
	}
	for _, forbidden := range []string{"sh -c", "bash -c", "fail2ban-client"} {
		if strings.Contains(command, forbidden) {
			t.Fatalf("SSH defense status bypassed the script protocol: %s", command)
		}
	}
}

func TestSSHDefenseStatusReportsUnavailableProtocol(t *testing.T) {
	manager, _, _, _ := testManager(t, &fakeRunner{})
	manager.f2bScript = func() (string, error) {
		return "", errors.New("old script")
	}
	status := manager.SSHDefenseStatus(context.Background())
	if status.Available || status.Message == "" {
		t.Fatalf("unavailable SSH defense status = %#v", status)
	}
}

func TestSSHDefenseRunsAsPersistentFixedMaintenanceTask(t *testing.T) {
	runner := &fakeRunner{}
	manager, _, _, stateDir := testManager(t, runner)
	manager.executable = filepath.Join(stateDir, "kejilion-agent")
	manager.f2bScript = func() (string, error) {
		return "/home/docker/kpanel/bin/kejilion.sh", nil
	}

	changed, message, err := manager.startMaintenance(
		context.Background(),
		"ssh-defense",
		"enable",
	)
	if err != nil || !changed || message == "" {
		t.Fatalf("start SSH defense: changed=%v message=%q err=%v", changed, message, err)
	}
	if err := manager.RunMaintenance(context.Background(), "ssh-defense-enable"); err != nil {
		t.Fatal(err)
	}
	status := manager.MaintenanceStatus()
	if status.Action != "ssh-defense" || status.Policy != "enable" ||
		status.State != "succeeded" || status.Progress != 100 {
		t.Fatalf("SSH defense maintenance state = %#v", status)
	}
	command := strings.Join(runner.commands, "\n")
	for _, required := range []string{
		"--unit=kejilion-panel-maintenance-",
		manager.executable + " maintenance-run --state-dir " + stateDir + " ssh-defense-enable",
		"env KJ_F2B_NONINTERACTIVE=1 LC_ALL=C.UTF-8 LANG=C.UTF-8 bash " +
			"/home/docker/kpanel/bin/kejilion.sh f2b enable",
	} {
		if !strings.Contains(command, required) {
			t.Fatalf("SSH defense task missing %q:\n%s", required, command)
		}
	}
	for _, forbidden := range []string{"sh -c", "bash -c"} {
		if strings.Contains(command, forbidden) {
			t.Fatalf("SSH defense task used a shell fragment %q:\n%s", forbidden, command)
		}
	}
}
