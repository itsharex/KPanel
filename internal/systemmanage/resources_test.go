package systemmanage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

func TestHostsSnapshotUsesExactBytesAndReportsLineTruncation(t *testing.T) {
	manager, etcRoot, _, _ := testManager(t, &fakeRunner{})
	var content strings.Builder
	for index := 0; index < contract.SystemHostsMaxLines+1; index++ {
		content.WriteString("192.0.2.")
		content.WriteString(strconv.Itoa(index%250 + 1))
		content.WriteString(" host-")
		content.WriteString(strconv.Itoa(index))
		content.WriteByte('\n')
	}
	raw := content.String()
	mustWrite(t, filepath.Join(etcRoot, "hosts"), raw)
	snapshot, err := manager.Hosts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.ResourceVersion != resourceHash([]byte(raw)) ||
		snapshot.Total != contract.SystemHostsMaxLines+1 ||
		len(snapshot.Entries) != contract.SystemHostsMaxLines || !snapshot.Truncated {
		t.Fatalf("unexpected hosts snapshot: %#v", snapshot)
	}
	if snapshot.Entries[0].Line != 1 || snapshot.Entries[0].Raw != "192.0.2.1 host-0" {
		t.Fatalf("unexpected first hosts entry: %#v", snapshot.Entries[0])
	}
	if _, err := manager.systemResourceVersion(context.Background(), "hosts", contract.SystemResourceActionRequest{}); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("truncated hosts write preflight error=%v", err)
	}
}

func TestHostsSnapshotRejectsOversizedResource(t *testing.T) {
	manager, etcRoot, _, _ := testManager(t, &fakeRunner{})
	mustWrite(t, filepath.Join(etcRoot, "hosts"), strings.Repeat("x", contract.SystemHostsMaxBytes+1))
	_, err := manager.Hosts(context.Background())
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("oversized hosts error = %v", err)
	}
	capability := findCapability(manager.SystemResourceCapabilities(), "system.hosts.read")
	if capability.Enabled || capability.Reason == "" {
		t.Fatalf("oversized hosts read capability=%#v", capability)
	}
}

func TestCronTreatsMissingCrontabAsExactEmptyBytes(t *testing.T) {
	runner := &fakeRunner{run: func(_ context.Context, name string, arguments ...string) ([]byte, error) {
		if name == "crontab" && len(arguments) == 1 && arguments[0] == "-l" {
			return nil, errors.New("no crontab for root")
		}
		return nil, nil
	}}
	manager, _, _, _ := testManager(t, runner)
	snapshot, err := manager.Cron(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.ResourceVersion != resourceHash([]byte{}) || snapshot.Total != 0 ||
		len(snapshot.Entries) != 0 || snapshot.Truncated {
		t.Fatalf("unexpected empty crontab: %#v", snapshot)
	}
}

func TestCronSnapshotClassifiesAndBoundsPhysicalLines(t *testing.T) {
	lines := []string{"# comment", "SHELL=/bin/bash", "0 2 * * * /usr/bin/backup --full", "@reboot /usr/bin/start", ""}
	for len(lines) < contract.SystemCronMaxLines+1 {
		lines = append(lines, "* * * * * true")
	}
	raw := strings.Join(lines, "\n") + "\n"
	runner := &fakeRunner{run: func(context.Context, string, ...string) ([]byte, error) {
		return []byte(raw), nil
	}}
	manager, _, _, _ := testManager(t, runner)
	snapshot, err := manager.Cron(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Total != len(lines) || len(snapshot.Entries) != contract.SystemCronMaxLines || !snapshot.Truncated ||
		snapshot.ResourceVersion != resourceHash([]byte(raw)) {
		t.Fatalf("unexpected bounded crontab: total=%d entries=%d truncated=%v", snapshot.Total, len(snapshot.Entries), snapshot.Truncated)
	}
	if snapshot.Entries[0].Kind != "comment" || snapshot.Entries[1].Kind != "environment" ||
		snapshot.Entries[2].Expression != "0 2 * * *" || snapshot.Entries[2].Command != "/usr/bin/backup --full" ||
		snapshot.Entries[3].Expression != "@reboot" {
		t.Fatalf("unexpected crontab parsing: %#v", snapshot.Entries[:4])
	}
	if _, err := manager.systemResourceVersion(context.Background(), "cron", contract.SystemResourceActionRequest{}); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("truncated cron write preflight error=%v", err)
	}
}

func TestNetworkSnapshotUsesAdministrativeStateAndPerEntryVersion(t *testing.T) {
	runner := &fakeRunner{run: func(_ context.Context, name string, arguments ...string) ([]byte, error) {
		if name != "ip" {
			t.Fatalf("unexpected command %s %#v", name, arguments)
		}
		return []byte(`[
			{"ifname":"eth0","addr_info":[{"family":"inet","local":"192.0.2.10","prefixlen":24}]},
			{"ifname":"lo","addr_info":[{"family":"inet","local":"127.0.0.1","prefixlen":8}]}
		]`), nil
	}}
	manager, _, _, _ := testManager(t, runner)
	networkRoot := filepath.Join(manager.sysRoot, "class", "net")
	for name, values := range map[string][2]string{
		"eth0": {"0x1002", "02:00:00:00:00:01"},
		"lo":   {"0x9", "00:00:00:00:00:00"},
	} {
		mustWrite(t, filepath.Join(networkRoot, name, "flags"), values[0]+"\n")
		mustWrite(t, filepath.Join(networkRoot, name, "address"), values[1]+"\n")
	}
	snapshot, err := manager.NetworkInterfaces(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Entries) != 2 || snapshot.Entries[0].Name != "eth0" || snapshot.Entries[0].State != "down" ||
		snapshot.Entries[1].Name != "lo" || snapshot.Entries[1].State != "up" || !snapshot.Entries[1].Loopback {
		t.Fatalf("unexpected network snapshot: %#v", snapshot)
	}
	wantVersion := resourceHash([]byte("lo|up|00:00:00:00:00:00"))
	if snapshot.Entries[1].ResourceVersion != wantVersion {
		t.Fatalf("loopback version=%q want=%q", snapshot.Entries[1].ResourceVersion, wantVersion)
	}
}

func TestFirewallSnapshotUsesExactBytesAndFirstPingRule(t *testing.T) {
	raw := strings.Join([]string{
		"# Generated by iptables-save v1.8.9 (nf_tables)",
		"*filter",
		":INPUT DROP [0:0]",
		"-A INPUT -p icmp -m icmp --icmp-type echo-request -j DROP",
		"-A INPUT -p icmp -m icmp --icmp-type echo-request -j ACCEPT",
		"-A INPUT -p tcp --syn -m limit --limit 500/sec --limit-burst 100 -j ACCEPT",
		"-A INPUT -p tcp --syn -j DROP",
		"COMMIT",
	}, "\n") + "\n"
	runner := &fakeRunner{run: func(context.Context, string, ...string) ([]byte, error) {
		return []byte(raw), nil
	}}
	manager, _, _, _ := testManager(t, runner)
	snapshot, err := manager.Firewall(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.ResourceVersion != resourceHash([]byte(raw)) || snapshot.Backend != "iptables-nft" ||
		snapshot.InputPolicy != "DROP" || snapshot.PingAllowed || !snapshot.DDoSEnabled || snapshot.Total != 4 {
		t.Fatalf("unexpected firewall snapshot: %#v", snapshot)
	}
	if snapshot.Rules[0].Line != 4 || snapshot.Rules[0].Chain != "INPUT" ||
		snapshot.Rules[0].Target != "DROP" || snapshot.Rules[0].Protocol != "icmp" {
		t.Fatalf("unexpected firewall rule: %#v", snapshot.Rules[0])
	}
}

func TestSystemResourceReceiptRejectsNoiseAndMalformedMarkers(t *testing.T) {
	version := strings.Repeat("a", 64)
	valid, err := parseSystemResourceReceipt([]byte(
		"KPANEL_SYSTEM_RESOURCE_STATUS applied\nKPANEL_SYSTEM_RESOURCE_VERSION " + version + "\n",
	))
	if err != nil || valid.Status != "applied" || valid.Version != version {
		t.Fatalf("valid receipt=%#v err=%v", valid, err)
	}
	failed, err := parseSystemResourceReceipt([]byte(
		"KPANEL_SYSTEM_RESOURCE_STATUS failed\nKPANEL_SYSTEM_RESOURCE_VERSION " + version + "\n",
	))
	if err != nil || failed.Status != "failed" {
		t.Fatalf("failed receipt=%#v err=%v", failed, err)
	}
	for _, output := range []string{
		"log noise\nKPANEL_SYSTEM_RESOURCE_STATUS applied\nKPANEL_SYSTEM_RESOURCE_VERSION " + version + "\n",
		"KPANEL_SYSTEM_RESOURCE_STATUS applied\nKPANEL_SYSTEM_RESOURCE_STATUS applied\nKPANEL_SYSTEM_RESOURCE_VERSION " + version + "\n",
		"KPANEL_SYSTEM_RESOURCE_STATUS applied\nKPANEL_SYSTEM_RESOURCE_VERSION uppercase\n",
	} {
		if _, err := parseSystemResourceReceipt([]byte(output)); err == nil {
			t.Fatalf("malformed receipt accepted: %q", output)
		}
	}
}

func TestTrustedSystemResourceProtocolMarkers(t *testing.T) {
	legacy := []byte("permission_granted=\"true\"\nKJ_SYSTEM_RESOURCE_NONINTERACTIVE=1\nkpanel_system_resource_noninteractive() { :; }\nKPANEL_SYSTEM_RESOURCE_STATUS=x\nKPANEL_SYSTEM_RESOURCE_VERSION=x\n")
	if trustedKejilionSystemResourceContent(legacy) {
		t.Fatal("legacy function marker was trusted")
	}
	current := []byte("permission_granted=\"true\"\nKJ_SYSTEM_RESOURCE_NONINTERACTIVE=1\nkpanel_system_resource_dispatch() { :; }\nKPANEL_SYSTEM_RESOURCE_STATUS=x\nKPANEL_SYSTEM_RESOURCE_VERSION=x\n")
	if !trustedKejilionSystemResourceContent(current) {
		t.Fatal("current protocol markers were rejected")
	}
}

func TestSystemResourceWriteRejectsStaleVersionBeforeScript(t *testing.T) {
	if runtime.GOOS != "linux" || os.Geteuid() != 0 {
		t.Skip("trusted root-owned script contract is Linux/root only")
	}
	runner := &fakeRunner{}
	manager, etcRoot, _, _ := testManager(t, runner)
	mustWrite(t, filepath.Join(etcRoot, "hosts"), "127.0.0.1 localhost\n")
	script := trustedResourceScript(t)
	manager.resourceScript = func() (string, error) { return script, nil }
	_, err := manager.ExecuteSystemResourceAction(context.Background(), contract.SystemResourceActionRequest{
		Action: "hosts-add", Address: "192.0.2.1", Hostnames: []string{"host.example"},
		ExpectedResourceVersion: strings.Repeat("b", 64),
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("stale version error=%v", err)
	}
	for _, command := range runner.commands {
		if strings.HasPrefix(command, "env KJ_SYSTEM_RESOURCE_NONINTERACTIVE=1") {
			t.Fatalf("stale request reached script: %s", command)
		}
	}
}

func TestSystemResourceWriteUsesFixedArgvAndVerifiesReceipt(t *testing.T) {
	if runtime.GOOS != "linux" || os.Geteuid() != 0 {
		t.Skip("trusted root-owned script contract is Linux/root only")
	}
	runner := &fakeRunner{}
	manager, etcRoot, _, _ := testManager(t, runner)
	hostsPath := filepath.Join(etcRoot, "hosts")
	original := "127.0.0.1 localhost\n"
	updated := original + "192.0.2.1 host.example # managed\n"
	mustWrite(t, hostsPath, original)
	script := trustedResourceScript(t)
	manager.resourceScript = func() (string, error) { return script, nil }
	runner.run = func(_ context.Context, name string, arguments ...string) ([]byte, error) {
		if name != "env" {
			t.Fatalf("unexpected action command=%s %#v", name, arguments)
		}
		mustWrite(t, hostsPath, updated)
		return []byte("KPANEL_SYSTEM_RESOURCE_STATUS applied\nKPANEL_SYSTEM_RESOURCE_VERSION " + resourceHash([]byte(updated)) + "\n"), nil
	}
	result, err := manager.ExecuteSystemResourceAction(context.Background(), contract.SystemResourceActionRequest{
		Action: "hosts-add", Address: "192.0.2.1", Hostnames: []string{"host.example"}, Comment: "managed",
		ExpectedResourceVersion: resourceHash([]byte(original)),
	})
	if err != nil || !result.Changed || result.ResourceVersion != resourceHash([]byte(updated)) {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	command := strings.Join(runner.commands, "\n")
	for _, expected := range []string{
		"env KJ_SYSTEM_RESOURCE_NONINTERACTIVE=1 bash " + script + " kpanel system-resource hosts add",
		resourceHash([]byte(original)) + " 192.0.2.1 host.example managed",
	} {
		if !strings.Contains(command, expected) {
			t.Fatalf("command missing %q:\n%s", expected, command)
		}
	}
	if strings.Contains(command, "sh -c") || strings.Contains(command, "bash -c") {
		t.Fatalf("action used a shell command string: %s", command)
	}
}

func TestNetworkResourceInvocationUsesSingularDomainAndAdminState(t *testing.T) {
	enabled := true
	resource, action, arguments := systemResourceInvocation(contract.SystemResourceActionRequest{
		Action: "network-interface-state", InterfaceName: "lo", Enabled: &enabled,
	})
	if resource != "network-interface" || action != "state" || strings.Join(arguments, "|") != "lo|up" {
		t.Fatalf("unexpected invocation: %q %q %#v", resource, action, arguments)
	}
}

func TestSystemResourceReadCapabilityDoesNotDependOnWriteAdapter(t *testing.T) {
	manager, etcRoot, _, _ := testManager(t, &fakeRunner{})
	mustWrite(t, filepath.Join(etcRoot, "hosts"), "127.0.0.1 localhost\n")
	manager.enabled = false
	read := findCapability(manager.SystemResourceCapabilities(), "system.hosts.read")
	write := findCapability(manager.SystemResourceCapabilities(), "system.hosts.write")
	if !read.Enabled || write.Enabled || write.Reason == "" {
		t.Fatalf("read=%#v write=%#v", read, write)
	}
	capabilities := manager.SystemResourceCapabilities()
	for _, id := range []string{
		"system.hosts.read", "system.hosts.write", "system.cron.read", "system.cron.write",
		"system.network-interfaces.read", "system.network-interfaces.write",
		"system.firewall.read", "system.firewall.write",
	} {
		if findCapability(capabilities, id).ID != id {
			t.Fatalf("capability %q missing: %#v", id, capabilities)
		}
	}
}

func TestSystemResourceCapabilitiesValidateSharedScriptOnce(t *testing.T) {
	if runtime.GOOS != "linux" || os.Geteuid() != 0 {
		t.Skip("trusted root-owned script contract is Linux/root only")
	}
	manager, etcRoot, _, _ := testManager(t, &fakeRunner{})
	mustWrite(t, filepath.Join(etcRoot, "hosts"), "127.0.0.1 localhost\n")
	script := trustedResourceScript(t)
	calls := 0
	manager.resourceScript = func() (string, error) {
		calls++
		return script, nil
	}
	_ = manager.SystemResourceCapabilities()
	if calls != 1 {
		t.Fatalf("shared script finder calls=%d want=1", calls)
	}
}

func trustedResourceScript(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "kejilion.sh")
	content := strings.Join([]string{
		"#!/usr/bin/env bash",
		"permission_granted=\"true\"",
		"KJ_SYSTEM_RESOURCE_NONINTERACTIVE=1",
		"kpanel_system_resource_dispatch() { :; }",
		"KPANEL_SYSTEM_RESOURCE_STATUS=applied",
		"KPANEL_SYSTEM_RESOURCE_VERSION=" + strings.Repeat("a", 64),
		strings.Repeat("# trusted protocol padding\n", 64),
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
