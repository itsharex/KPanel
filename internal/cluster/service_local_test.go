package cluster

import (
	"context"
	"crypto/ed25519"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

type localNodeTestTelemetry struct {
	mu    sync.Mutex
	value contract.HostTelemetry
	err   error
	calls int
}

func (s *localNodeTestTelemetry) Telemetry(context.Context) (contract.HostTelemetry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	return cloneTelemetry(s.value), s.err
}

func (s *localNodeTestTelemetry) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

type localNodeTestRemote struct{}

func (localNodeTestRemote) Pair(
	context.Context,
	string,
	PairRequest,
) (PairResponse, error) {
	return PairResponse{}, errors.New("unexpected remote pair")
}

func (localNodeTestRemote) Summary(
	context.Context,
	string,
	string,
	string,
	ed25519.PrivateKey,
	time.Time,
) (FederationSummary, error) {
	return FederationSummary{}, errors.New("unexpected remote summary")
}

func (localNodeTestRemote) Revoke(
	context.Context,
	string,
	string,
	string,
	ed25519.PrivateKey,
	time.Time,
) error {
	return errors.New("unexpected remote revoke")
}

func TestHostsAlwaysIncludeLocalNodeWithoutPersistingIt(t *testing.T) {
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	source := &localNodeTestTelemetry{value: validLocalNodeTelemetry(now)}
	dataDir := t.TempDir()
	service, err := NewService(ServiceConfig{
		DataDir: dataDir, PanelVersion: "0.27.0", Hostname: "fallback-host",
		Telemetry: source, Remote: localNodeTestRemote{}, Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	t.Cleanup(func() { _ = service.Close() })

	inventory := service.Hosts(context.Background())
	if inventory.Total != 1 || inventory.RemoteTotal != 0 || len(inventory.Items) != 1 {
		t.Fatalf("unexpected local-only inventory: %#v", inventory)
	}
	local := inventory.Items[0]
	if local.ID != LocalHostID || !local.IsLocal || local.Name != "local-node" {
		t.Fatalf("unexpected local host: %#v", local)
	}
	if local.RemoteNodeID != inventory.NodeID || local.State != HostOnline {
		t.Fatalf("unexpected local identity/state: %#v", local)
	}
	if local.LastSnapshot == nil || local.LastSnapshot.Telemetry.OS != "Debian GNU/Linux 13" {
		t.Fatalf("local telemetry missing: %#v", local.LastSnapshot)
	}
	if len(service.store.Hosts()) != 0 {
		t.Fatal("local host must not be persisted as a remote host")
	}
	if err := service.checkpoint(); err != nil {
		t.Fatalf("checkpoint() error = %v", err)
	}
	if len(service.store.Hosts()) != 0 {
		t.Fatal("checkpoint persisted the synthetic local host")
	}
	secretEntries, err := os.ReadDir(filepath.Join(dataDir, "cluster-secrets"))
	if err != nil {
		t.Fatalf("ReadDir(cluster-secrets) error = %v", err)
	}
	if len(secretEntries) != 0 {
		t.Fatalf("local host created federation credentials: %v", secretEntries)
	}
}

func TestLocalNodeUsesCachedTelemetryAllowsRenameAndRejectsDelete(t *testing.T) {
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	source := &localNodeTestTelemetry{value: validLocalNodeTelemetry(now)}
	dataDir := t.TempDir()
	service, err := NewService(ServiceConfig{
		DataDir: dataDir, PanelVersion: "0.27.0", Hostname: "fallback-host",
		Telemetry: source, Remote: localNodeTestRemote{}, Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	t.Cleanup(func() { _ = service.Close() })

	first := service.Hosts(context.Background()).Items[0]
	second := service.Hosts(context.Background()).Items[0]
	if source.callCount() != 1 {
		t.Fatalf("local telemetry calls = %d, want 1", source.callCount())
	}
	if first.ResourceVersion != second.ResourceVersion {
		t.Fatalf("local resource version changed: %q != %q", first.ResourceVersion, second.ResourceVersion)
	}
	renamed, err := service.RenameHost(LocalHostID, UpdateHostInput{
		Name: "控制中心", ExpectedResourceVersion: first.ResourceVersion,
	})
	if err != nil {
		t.Fatalf("RenameHost(local) error = %v", err)
	}
	if !renamed.IsLocal || renamed.Name != "控制中心" ||
		renamed.ResourceVersion == first.ResourceVersion {
		t.Fatalf("unexpected renamed local host: %#v", renamed)
	}
	if got := service.Hosts(context.Background()).Items[0].Name; got != "控制中心" {
		t.Fatalf("local host name after rename = %q", got)
	}
	persisted, err := OpenStore(filepath.Join(dataDir, "cluster-state.json"))
	if err != nil {
		t.Fatalf("OpenStore() after rename error = %v", err)
	}
	if got := persisted.LocalName(); got != "控制中心" {
		t.Fatalf("persisted local name = %q", got)
	}
	if _, err := service.DeleteHost(context.Background(), LocalHostID, DeleteHostInput{
		ExpectedResourceVersion: renamed.ResourceVersion,
	}); !errors.Is(err, ErrLocalHost) {
		t.Fatalf("DeleteHost(local) error = %v, want ErrLocalHost", err)
	}
}

func TestLocalTelemetryFailureStillReturnsLocalCard(t *testing.T) {
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	source := &localNodeTestTelemetry{err: errors.New("agent unavailable")}
	service, err := NewService(ServiceConfig{
		DataDir: t.TempDir(), PanelVersion: "0.27.0", Hostname: "local-fallback",
		Telemetry: source, Remote: localNodeTestRemote{}, Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	t.Cleanup(func() { _ = service.Close() })

	local := service.Hosts(context.Background()).Items[0]
	if !local.IsLocal || local.Name != "local-fallback" {
		t.Fatalf("unexpected fallback local host: %#v", local)
	}
	if local.State != HostDegraded || local.LastErrorCode != "local_agent_unavailable" {
		t.Fatalf("unexpected local failure state: %#v", local)
	}
}

func TestValidateTelemetryRejectsStaleAndUnsafeNumbers(t *testing.T) {
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		mutate func(*contract.HostTelemetry)
	}{
		{
			name: "stale collection",
			mutate: func(value *contract.HostTelemetry) {
				value.CollectedAt = now.Add(-11 * time.Minute)
			},
		},
		{
			name: "unsafe JSON integer",
			mutate: func(value *contract.HostTelemetry) {
				value.Network.ReceivedBytes = maxSafeJSONInteger + 1
			},
		},
		{
			name: "impossible available memory",
			mutate: func(value *contract.HostTelemetry) {
				value.Memory.AvailableBytes = value.Memory.TotalBytes + 1
			},
		},
		{
			name: "control character",
			mutate: func(value *contract.HostTelemetry) {
				value.AgentVersion = "v1\nforged"
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := validLocalNodeTelemetry(now)
			test.mutate(&value)
			if err := validateTelemetry(value, now); err == nil {
				t.Fatal("validateTelemetry() accepted an unsafe value")
			}
		})
	}
}

func validLocalNodeTelemetry(now time.Time) contract.HostTelemetry {
	return contract.HostTelemetry{
		AgentVersion: "0.27.0", AgentProtocolVersion: "v1alpha1",
		Hostname: "local-node", OS: "Debian GNU/Linux 13", OSID: "debian",
		Kernel: "6.12.0", Architecture: "amd64", UptimeSeconds: 3600,
		Load: contract.LoadSummary{One: 0.1, Five: 0.2, Fifteen: 0.3},
		CPU:  contract.CPUSummary{Cores: 2, UsagePercent: 10},
		Memory: contract.MemorySummary{
			TotalBytes: 8 << 30, UsedBytes: 2 << 30, AvailableBytes: 6 << 30,
			UsagePercent: 25,
		},
		Disk: contract.DiskCapacitySummary{
			TotalBytes: 80 << 30, UsedBytes: 20 << 30, UsagePercent: 25,
		},
		Network: contract.NetworkSummary{ReceivedBytes: 1024, SentBytes: 2048},
		PublicNetwork: contract.PublicNetworkSummary{
			IPv4: "203.0.113.10", Country: "Singapore", CountryCode: "SG",
		},
		CollectedAt: now,
	}
}
