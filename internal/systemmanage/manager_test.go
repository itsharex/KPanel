package systemmanage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

type fakeRunner struct {
	run      func(context.Context, string, ...string) ([]byte, error)
	missing  map[string]bool
	commands []string
}

func (runner *fakeRunner) Run(ctx context.Context, name string, arguments ...string) ([]byte, error) {
	runner.commands = append(runner.commands, strings.Join(append([]string{name}, arguments...), " "))
	if runner.run != nil {
		return runner.run(ctx, name, arguments...)
	}
	return nil, nil
}

func (runner *fakeRunner) LookPath(name string) (string, error) {
	if runner.missing[name] {
		return "", errors.New("missing")
	}
	return "/usr/bin/" + name, nil
}

func testManager(t *testing.T, runner Runner) (*Manager, string, string, string) {
	t.Helper()
	root := t.TempDir()
	etcRoot := filepath.Join(root, "etc")
	procRoot := filepath.Join(root, "proc")
	stateDir := filepath.Join(root, "state")
	for _, path := range []string{etcRoot, procRoot, stateDir} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	manager := NewManager(Config{
		Enabled: true, EtcRoot: etcRoot, ProcRoot: procRoot, StateDir: stateDir,
		Runner: runner, Now: func() time.Time { return time.Date(2026, 7, 26, 3, 0, 0, 0, time.UTC) },
	})
	return manager, etcRoot, procRoot, stateDir
}

func TestSetHostnameUpdatesHostsAndCreatesBackup(t *testing.T) {
	runner := &fakeRunner{}
	manager, etcRoot, _, _ := testManager(t, runner)
	mustWrite(t, filepath.Join(etcRoot, "hostname"), "old-host\n")
	mustWrite(t, filepath.Join(etcRoot, "hosts"), "127.0.0.1\tlocalhost\n127.0.1.1\told-host alias\n")
	runner.run = func(_ context.Context, name string, arguments ...string) ([]byte, error) {
		if name == "hostname" {
			return []byte("new-host\n"), nil
		}
		return nil, nil
	}

	changed, backup, message, err := manager.setHostname(context.Background(), "new-host")
	if err != nil {
		t.Fatal(err)
	}
	if !changed || backup == "" || message == "" {
		t.Fatalf("unexpected result: changed=%v backup=%q message=%q", changed, backup, message)
	}
	if got := readLimited(filepath.Join(etcRoot, "hostname")); got != "new-host\n" {
		t.Fatalf("hostname = %q", got)
	}
	hosts := readLimited(filepath.Join(etcRoot, "hosts"))
	if !strings.Contains(hosts, "127.0.1.1\tnew-host alias") || strings.Contains(hosts, "old-host") {
		t.Fatalf("hosts were not updated safely: %q", hosts)
	}
	if !regularFile(filepath.Join(backup, "manifest.tsv")) {
		t.Fatalf("backup manifest missing at %s", backup)
	}
}

func TestSetHostnameRollsBackWhenVerificationFails(t *testing.T) {
	runner := &fakeRunner{}
	manager, etcRoot, _, _ := testManager(t, runner)
	mustWrite(t, filepath.Join(etcRoot, "hostname"), "old-host\n")
	mustWrite(t, filepath.Join(etcRoot, "hosts"), "127.0.1.1\told-host\n")
	runner.run = func(_ context.Context, name string, arguments ...string) ([]byte, error) {
		if name == "hostname" {
			return []byte("unexpected-host\n"), nil
		}
		return nil, nil
	}

	_, _, _, err := manager.setHostname(context.Background(), "new-host")
	if !errors.Is(err, ErrRolledBack) {
		t.Fatalf("expected rollback error, got %v", err)
	}
	if got := readLimited(filepath.Join(etcRoot, "hostname")); got != "old-host\n" {
		t.Fatalf("hostname rollback failed: %q", got)
	}
	if got := readLimited(filepath.Join(etcRoot, "hosts")); got != "127.0.1.1\told-host\n" {
		t.Fatalf("hosts rollback failed: %q", got)
	}
}

func TestIPPreferencePreservesUnrelatedConfiguration(t *testing.T) {
	manager, etcRoot, _, _ := testManager(t, &fakeRunner{})
	path := filepath.Join(etcRoot, "gai.conf")
	mustWrite(t, path, "# user setting\nlabel 2002::/16 2\n")

	changed, _, _, err := manager.setIPPreference("ipv4")
	if err != nil || !changed {
		t.Fatalf("enable IPv4 preference: changed=%v err=%v", changed, err)
	}
	enabled := readLimited(path)
	if !strings.Contains(enabled, "# user setting") ||
		!strings.Contains(enabled, "precedence ::ffff:0:0/96  100") {
		t.Fatalf("unexpected enabled configuration: %q", enabled)
	}
	changed, _, _, err = manager.setIPPreference("system_default")
	if err != nil || !changed {
		t.Fatalf("restore preference: changed=%v err=%v", changed, err)
	}
	restored := readLimited(path)
	if !strings.Contains(restored, "# user setting") ||
		strings.Contains(restored, "::ffff:0:0/96") {
		t.Fatalf("unrelated gai.conf content was not preserved: %q", restored)
	}
}

func TestRewriteAPTSourceLeavesThirdPartyRepositoriesUntouched(t *testing.T) {
	input := []byte(
		"deb https://deb.debian.org/debian bookworm main\n" +
			"deb https://deb.debian.org/debian-security bookworm-security main\n" +
			"deb https://download.docker.com/linux/debian bookworm stable\n",
	)
	aliyun := string(rewriteAPTSource(input, "debian", "aliyun"))
	if strings.Count(aliyun, "https://mirrors.aliyun.com/") != 2 {
		t.Fatalf("distribution repositories were not rewritten: %q", aliyun)
	}
	if !strings.Contains(aliyun, "https://download.docker.com/linux/debian") {
		t.Fatalf("third-party repository changed: %q", aliyun)
	}
	official := string(rewriteAPTSource([]byte(aliyun), "debian", "official"))
	if strings.Count(official, "https://deb.debian.org/") != 2 {
		t.Fatalf("official repositories were not restored: %q", official)
	}
}

func TestSetSwapOnlyManagesKPanelSwap(t *testing.T) {
	runner := &fakeRunner{}
	manager, etcRoot, procRoot, stateDir := testManager(t, runner)
	mustWrite(t, filepath.Join(etcRoot, "fstab"), "/dev/vda2 none swap sw 0 0\n")
	mustWrite(t, filepath.Join(procRoot, "swaps"), "Filename Type Size Used Priority\n/dev/vda2 partition 1 0 -2\n")
	runner.run = func(_ context.Context, name string, arguments ...string) ([]byte, error) {
		switch name {
		case "fallocate":
			size, _ := strconv.Atoi(strings.TrimSuffix(arguments[1], "M"))
			file, err := os.Create(arguments[2])
			if err != nil {
				return nil, err
			}
			err = file.Truncate(int64(size) * 1024 * 1024)
			closeErr := file.Close()
			return nil, errors.Join(err, closeErr)
		case "swapon":
			mustWrite(t, filepath.Join(procRoot, "swaps"),
				"Filename Type Size Used Priority\n/dev/vda2 partition 1 0 -2\n"+arguments[0]+" file 1 0 -2\n")
		case "swapoff":
			mustWrite(t, filepath.Join(procRoot, "swaps"), "Filename Type Size Used Priority\n/dev/vda2 partition 1 0 -2\n")
		}
		return nil, nil
	}

	changed, _, _, err := manager.setSwap(context.Background(), 256)
	if err != nil || !changed {
		t.Fatalf("enable swap: changed=%v err=%v", changed, err)
	}
	swapPath := filepath.Join(stateDir, "swapfile")
	fstab := readLimited(filepath.Join(etcRoot, "fstab"))
	if !strings.Contains(fstab, "/dev/vda2 none swap") || !strings.Contains(fstab, swapPath) {
		t.Fatalf("fstab does not preserve external swap: %q", fstab)
	}
	changed, _, _, err = manager.setSwap(context.Background(), 0)
	if err != nil || !changed {
		t.Fatalf("disable swap: changed=%v err=%v", changed, err)
	}
	fstab = readLimited(filepath.Join(etcRoot, "fstab"))
	if !strings.Contains(fstab, "/dev/vda2 none swap") || strings.Contains(fstab, swapPath) {
		t.Fatalf("disable touched external swap or retained managed swap: %q", fstab)
	}
}

func mustWrite(t *testing.T, path, value string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatal(err)
	}
}
