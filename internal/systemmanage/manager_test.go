package systemmanage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kejilion/kejilion-panel/internal/contract"
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
	sysRoot := filepath.Join(root, "sys")
	runRoot := filepath.Join(root, "run")
	stateDir := filepath.Join(root, "state")
	for _, path := range []string{etcRoot, procRoot, sysRoot, runRoot, stateDir} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite(t, filepath.Join(etcRoot, "os-release"), "ID=debian\nID_LIKE=debian\n")
	mustWrite(
		t,
		filepath.Join(etcRoot, "apt", "sources.list"),
		"deb https://deb.debian.org/debian stable main\n",
	)
	manager := NewManager(Config{
		Enabled: true, EtcRoot: etcRoot, ProcRoot: procRoot, SysRoot: sysRoot, RunRoot: runRoot,
		StateDir: stateDir, SwapPath: filepath.Join(root, "swapfile"),
		Executable: "/usr/local/libexec/kejilion-agent",
		Runner:     runner, Now: func() time.Time { return time.Date(2026, 7, 26, 3, 0, 0, 0, time.UTC) },
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

func TestSetKernelTuningMatchesKejilionStreamProfileAndAcceptsScriptArtifact(t *testing.T) {
	runner := &fakeRunner{}
	manager, etcRoot, procRoot, _ := testManager(t, runner)
	mustWrite(t, filepath.Join(procRoot, "meminfo"), "MemTotal:        8388608 kB\n")
	mustWrite(
		t,
		filepath.Join(procRoot, "sys", "net", "ipv4", "tcp_available_congestion_control"),
		"reno cubic bbr\n",
	)
	mustWrite(t, filepath.Join(procRoot, "sys", "kernel", "numa_balancing"), "1\n")
	mustWrite(t, filepath.Join(procRoot, "sys", "net", "netfilter", "nf_conntrack_max"), "262144\n")
	thpPath := filepath.Join(manager.sysRoot, "kernel", "mm", "transparent_hugepage", "enabled")
	mustWrite(t, thpPath, "[always] madvise never\n")

	manualPath := filepath.Join(etcRoot, "sysctl.d", "99-kejilion-optimize.conf")
	autoPath := filepath.Join(etcRoot, "sysctl.d", "99-network-optimize.conf")
	limitsPath := filepath.Join(etcRoot, "security", "limits.conf")
	modulesPath := filepath.Join(etcRoot, "modules-load.d", "bbr.conf")
	systemSysctlPath := filepath.Join(etcRoot, "sysctl.conf")
	kpanelBBRPath := filepath.Join(etcRoot, "sysctl.d", "99-kejilion-bbr.conf")
	mustWrite(
		t,
		manualPath,
		kejilionKernelConfigMarker+"\n# 模式: 均衡优化模式 | 场景: balanced\nvm.swappiness = 30\n",
	)
	mustWrite(t, autoPath, kejilionAutoKernelMarker+"\nnet.core.somaxconn = 2048\n")
	mustWrite(
		t,
		limitsPath,
		"# user limit\nuser soft nofile 4096\n\n"+
			networkLimitsMarker+"\n* soft nofile 1048576\n* hard nofile 1048576\n"+
			"root soft nofile 1048576\nroot hard nofile 1048576\n",
	)
	mustWrite(t, modulesPath, "tcp_bbr\n")
	mustWrite(
		t,
		systemSysctlPath,
		"# keep me\nnet.ipv4.tcp_congestion_control = cubic\nvm.max_map_count = 262144\n",
	)
	mustWrite(
		t,
		kpanelBBRPath,
		kpanelBBRMarker+"\nnet.core.default_qdisc=fq_codel\nnet.ipv4.tcp_congestion_control=cubic\n",
	)

	changed, backup, message, err := manager.setKernelTuning(context.Background(), "stream")
	if err != nil || !changed || backup == "" {
		t.Fatalf("set stream profile: changed=%v backup=%q message=%q err=%v", changed, backup, message, err)
	}
	if !strings.Contains(message, "直播优化模式") || !strings.Contains(message, "内存 8192 MiB") ||
		!strings.Contains(message, "拥塞算法 bbr") {
		t.Fatalf("unexpected result message: %q", message)
	}
	config := readLimited(manualPath)
	for _, expected := range []string{
		kejilionKernelConfigMarker,
		kpanelKernelMarker,
		"# 模式: 直播优化模式 | 场景: stream",
		"net.core.default_qdisc = fq",
		"net.ipv4.tcp_congestion_control = bbr",
		"net.core.rmem_max = 67108864",
		"net.core.netdev_max_backlog = 250000",
		"net.ipv4.tcp_mem = 1048576 2097152 4194304",
		"vm.min_free_kbytes = 65536",
		"net.netfilter.nf_conntrack_max = 262144",
		"net.ipv4.udp_rmem_min = 16384",
	} {
		if !strings.Contains(config, expected) {
			t.Fatalf("generated profile missing %q:\n%s", expected, config)
		}
	}
	if regularFile(autoPath) || regularFile(kpanelBBRPath) {
		t.Fatal("conflicting automatic or standalone KPanel BBR profile remains")
	}
	if got := readLimited(limitsPath); !strings.Contains(got, "# user limit") ||
		!strings.Contains(got, kejilionLimitsMarker) ||
		strings.Contains(got, networkLimitsMarker) {
		t.Fatalf("unexpected limits configuration: %q", got)
	}
	if got := readLimited(systemSysctlPath); !strings.Contains(got, "vm.max_map_count") ||
		strings.Contains(got, "tcp_congestion_control") {
		t.Fatalf("unexpected sysctl.conf: %q", got)
	}
	if got := strings.TrimSpace(readLimited(modulesPath)); got != "tcp_bbr" {
		t.Fatalf("unexpected BBR module configuration: %q", got)
	}
	if got := strings.TrimSpace(readLimited(thpPath)); got != "never" {
		t.Fatalf("transparent hugepage mode = %q", got)
	}
	commands := strings.Join(runner.commands, "\n")
	if !strings.Contains(commands, "modprobe tcp_bbr") ||
		!strings.Contains(commands, "sysctl -w net.ipv4.udp_rmem_min=16384") {
		t.Fatalf("expected typed kernel commands were not run:\n%s", commands)
	}
	if strings.Contains(commands, "sh -c") || strings.Contains(commands, "bash -c") {
		t.Fatalf("kernel tuning invoked a shell:\n%s", commands)
	}
	manager.now = func() time.Time { return time.Date(2026, 7, 27, 3, 0, 0, 0, time.UTC) }
	changed, backup, _, err = manager.setKernelTuning(context.Background(), "stream")
	if err != nil || changed || backup != "" {
		t.Fatalf("identical profile was not idempotent: changed=%v backup=%q err=%v", changed, backup, err)
	}
}

func TestKernelProfileUsesKejilionMemoryAdaptationAndSceneExtras(t *testing.T) {
	manager, _, _, _ := testManager(t, &fakeRunner{missing: map[string]bool{"modprobe": true}})

	game := manager.buildKernelProfile(context.Background(), "game", 768)
	settings := manager.kernelSettings(game, 768)
	config := string(renderKernelConfig(game, 768, settings, manager.now()))
	for _, expected := range []string{
		"# 模式: 游戏服优化模式 | 场景: game",
		"net.core.rmem_max = 4194304",
		"net.core.somaxconn = 1024",
		"net.core.netdev_max_backlog = 1000",
		"vm.swappiness = 30",
		"vm.overcommit_memory = 0",
		"vm.min_free_kbytes = 16384",
		"net.ipv4.tcp_slow_start_after_idle = 0",
		"net.ipv4.tcp_congestion_control = cubic",
	} {
		if !strings.Contains(config, expected) {
			t.Fatalf("small-memory game profile missing %q:\n%s", expected, config)
		}
	}

	high := manager.buildKernelProfile(context.Background(), "high", 16384)
	if high.swappiness != 5 || high.minFreeKiB != 131072 {
		t.Fatalf("large-memory high profile was not adapted: %#v", high)
	}
	balanced := manager.buildKernelProfile(context.Background(), "balanced", 4096)
	if balanced.swappiness != 30 || balanced.overcommit != 0 ||
		balanced.backlog != 5000 || balanced.thpMode != "always" {
		t.Fatalf("balanced profile differs from kejilion.sh: %#v", balanced)
	}
}

func TestSetKernelTuningRejectsUnknownArtifactWithoutMutation(t *testing.T) {
	runner := &fakeRunner{}
	manager, etcRoot, procRoot, _ := testManager(t, runner)
	mustWrite(t, filepath.Join(procRoot, "meminfo"), "MemTotal: 4194304 kB\n")
	path := filepath.Join(etcRoot, "sysctl.d", "99-kejilion-optimize.conf")
	original := "# unrelated administrator configuration\nvm.swappiness = 5\n"
	mustWrite(t, path, original)

	_, _, _, err := manager.setKernelTuning(context.Background(), "web")
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	if got := readLimited(path); got != original {
		t.Fatalf("unknown artifact changed: %q", got)
	}
	if len(runner.commands) != 0 {
		t.Fatalf("commands ran before conflict gate: %#v", runner.commands)
	}
}

func TestSetKernelTuningRollsBackWhenAllSysctlSettingsFail(t *testing.T) {
	runner := &fakeRunner{}
	manager, etcRoot, procRoot, _ := testManager(t, runner)
	mustWrite(t, filepath.Join(procRoot, "meminfo"), "MemTotal: 4194304 kB\n")
	path := filepath.Join(etcRoot, "sysctl.d", "99-kejilion-optimize.conf")
	original := kejilionKernelConfigMarker +
		"\n# 模式: 均衡优化模式 | 场景: balanced\nvm.swappiness = 30\n"
	mustWrite(t, path, original)
	runner.run = func(_ context.Context, name string, arguments ...string) ([]byte, error) {
		if name == "sysctl" && len(arguments) > 0 && arguments[0] == "-w" {
			return nil, errors.New("permission denied")
		}
		return nil, nil
	}

	_, backup, _, err := manager.setKernelTuning(context.Background(), "web")
	if !errors.Is(err, ErrRolledBack) {
		t.Fatalf("expected rollback error, got %v", err)
	}
	if backup == "" {
		t.Fatal("rollback did not create a backup")
	}
	if got := readLimited(path); got != original {
		t.Fatalf("kernel profile was not restored: %q", got)
	}
}

func TestSetKernelTuningOffRestoresKejilionArtifactsAndPreservesStandaloneBBR(t *testing.T) {
	runner := &fakeRunner{}
	manager, etcRoot, _, _ := testManager(t, runner)
	manualPath := filepath.Join(etcRoot, "sysctl.d", "99-kejilion-optimize.conf")
	autoPath := filepath.Join(etcRoot, "sysctl.d", "99-network-optimize.conf")
	limitsPath := filepath.Join(etcRoot, "security", "limits.conf")
	modulesPath := filepath.Join(etcRoot, "modules-load.d", "bbr.conf")
	kpanelBBRPath := filepath.Join(etcRoot, "sysctl.d", "99-kejilion-bbr.conf")
	thpPath := filepath.Join(manager.sysRoot, "kernel", "mm", "transparent_hugepage", "enabled")
	mustWrite(
		t,
		manualPath,
		kejilionKernelConfigMarker+"\n# 模式: 网站搭建优化模式 | 场景: web\nvm.swappiness = 10\n",
	)
	mustWrite(t, autoPath, kejilionAutoKernelMarker+"\nnet.core.somaxconn = 2048\n")
	mustWrite(
		t,
		limitsPath,
		"# keep\n"+kejilionLimitsMarker+"\n* soft nofile 1048576\n* hard nofile 1048576\n"+
			"root soft nofile 1048576\nroot hard nofile 1048576\n",
	)
	mustWrite(t, modulesPath, "tcp_bbr\n")
	mustWrite(t, kpanelBBRPath, kpanelBBRMarker+"\nnet.ipv4.tcp_congestion_control=bbr\n")
	mustWrite(t, thpPath, "always madvise [never]\n")

	changed, _, message, err := manager.setKernelTuning(context.Background(), "off")
	if err != nil || !changed || !strings.Contains(message, "已还原") {
		t.Fatalf("restore defaults: changed=%v message=%q err=%v", changed, message, err)
	}
	if regularFile(manualPath) || regularFile(autoPath) || regularFile(modulesPath) {
		t.Fatal("Kejilion optimization artifacts remain after restore")
	}
	if !regularFile(kpanelBBRPath) {
		t.Fatal("independently managed KPanel BBR profile was removed")
	}
	if got := readLimited(limitsPath); !strings.Contains(got, "# keep") ||
		strings.Contains(got, kejilionLimitsMarker) {
		t.Fatalf("unexpected restored limits: %q", got)
	}
	if got := strings.TrimSpace(readLimited(thpPath)); got != "always" {
		t.Fatalf("transparent hugepage mode was not restored: %q", got)
	}
	if !slices.Contains(runner.commands, "sysctl --system") {
		t.Fatalf("system sysctl defaults were not reloaded: %#v", runner.commands)
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
	manager, etcRoot, procRoot, _ := testManager(t, runner)
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
			info, err := os.Stat(arguments[0])
			if err != nil {
				return nil, err
			}
			sizeKiB := info.Size()/1024 - 4
			mustWrite(t, filepath.Join(procRoot, "swaps"),
				"Filename Type Size Used Priority\n/dev/vda2 partition 1 0 -2\n"+
					arguments[0]+" file "+strconv.FormatInt(sizeKiB, 10)+" 0 -2\n")
		case "swapoff":
			mustWrite(t, filepath.Join(procRoot, "swaps"), "Filename Type Size Used Priority\n/dev/vda2 partition 1 0 -2\n")
		}
		return nil, nil
	}

	changed, _, _, err := manager.setSwap(context.Background(), 256)
	if err != nil || !changed {
		t.Fatalf("enable swap: changed=%v err=%v", changed, err)
	}
	swapPath := manager.swapPath
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

func TestSetSwapResizesKejilionFileAndMigratesLegacySwap(t *testing.T) {
	runner := &fakeRunner{}
	manager, etcRoot, procRoot, stateDir := testManager(t, runner)
	primaryPath := manager.swapPath
	legacyPath := filepath.Join(stateDir, "swapfile")
	externalPath := "/dev/vda2"
	mustSizedFile(t, primaryPath, 1024)
	mustSizedFile(t, legacyPath, 2048)
	mustWrite(t, filepath.Join(etcRoot, "fstab"),
		externalPath+" none swap sw 0 0\n"+
			primaryPath+" swap swap defaults 0 0\n"+
			kpanelSwapMarker+"\n"+
			legacyPath+" none swap sw 0 0\n")
	mustWrite(t, filepath.Join(procRoot, "meminfo"), "MemAvailable: 4194304 kB\n")
	active := map[string]bool{externalPath: true, primaryPath: true, legacyPath: true}
	writeSwapFixture(t, procRoot, active)
	runner.run = swapTestRunner(t, procRoot, active)

	changed, _, message, err := manager.setSwap(context.Background(), 4096)
	if err != nil || !changed {
		t.Fatalf("resize swap: changed=%v message=%q err=%v", changed, message, err)
	}
	if !strings.Contains(message, "合并旧版 KPanel Swap") {
		t.Fatalf("migration was not reported: %q", message)
	}
	info, err := os.Stat(primaryPath)
	if err != nil || !sizeWithinMiB(info.Size(), 4096) {
		t.Fatalf("primary swap size was not changed: info=%v err=%v", info, err)
	}
	if _, err := os.Lstat(legacyPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("legacy swap remains after migration: %v", err)
	}
	if !active[primaryPath] || !active[externalPath] || active[legacyPath] {
		t.Fatalf("unexpected active swap set: %#v", active)
	}
	fstab := readLimited(filepath.Join(etcRoot, "fstab"))
	if strings.Count(fstab, primaryPath+" swap swap defaults 0 0") != 1 ||
		strings.Contains(fstab, legacyPath) ||
		strings.Contains(fstab, kpanelSwapMarker) ||
		!strings.Contains(fstab, externalPath+" none swap") {
		t.Fatalf("unexpected migrated fstab: %q", fstab)
	}
	for _, command := range runner.commands {
		if strings.Contains(command, "swapoff "+externalPath) ||
			strings.Contains(command, "wipefs") {
			t.Fatalf("external swap was modified: %#v", runner.commands)
		}
	}
}

func TestSetSwapRejectsSymlinkArtifact(t *testing.T) {
	manager, _, _, _ := testManager(t, &fakeRunner{})
	target := manager.swapPath + ".target"
	mustWrite(t, target, "not swap")
	if err := os.Symlink(target, manager.swapPath); err != nil {
		t.Fatal(err)
	}
	_, _, _, err := manager.setSwap(context.Background(), 1024)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected symlink conflict, got %v", err)
	}
}

func TestSwapActiveRecognizesLegacyNamespaceAliasBySize(t *testing.T) {
	manager, _, procRoot, stateDir := testManager(t, &fakeRunner{})
	legacyPath := filepath.Join(stateDir, "swapfile")
	mustSizedFile(t, manager.swapPath, 1024)
	mustSizedFile(t, legacyPath, 2048)
	mustWrite(t, filepath.Join(procRoot, "swaps"),
		"Filename Type Size Used Priority\n"+
			manager.swapPath+" file 1048572 128 -2\n"+
			manager.swapPath+" file 2097148 0 -3\n")

	if !manager.swapActive(manager.swapPath) {
		t.Fatal("primary /swapfile was not recognized")
	}
	if !manager.swapActive(legacyPath) {
		t.Fatal("legacy swap namespace alias was not recognized by size")
	}
}

func TestSetSwapRefusesUnsafeSwapoffWhenMemoryIsLow(t *testing.T) {
	runner := &fakeRunner{}
	manager, etcRoot, procRoot, _ := testManager(t, runner)
	mustSizedFile(t, manager.swapPath, 1024)
	mustWrite(t, filepath.Join(etcRoot, "fstab"), manager.swapPath+" swap swap defaults 0 0\n")
	mustWrite(t, filepath.Join(procRoot, "swaps"),
		"Filename Type Size Used Priority\n"+manager.swapPath+" file 1048572 262144 -2\n")
	mustWrite(t, filepath.Join(procRoot, "meminfo"), "MemAvailable: 327680 kB\n")

	_, _, _, err := manager.setSwap(context.Background(), 2048)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected low-memory conflict, got %v", err)
	}
	if len(runner.commands) != 0 {
		t.Fatalf("commands ran before memory safety gate: %#v", runner.commands)
	}
}

func TestSetSwapRollsBackOriginalFileWhenActivationFails(t *testing.T) {
	runner := &fakeRunner{}
	manager, etcRoot, procRoot, _ := testManager(t, runner)
	oldFstab := manager.swapPath + " swap swap defaults 0 0\n"
	mustSizedFile(t, manager.swapPath, 1024)
	mustWrite(t, filepath.Join(etcRoot, "fstab"), oldFstab)
	mustWrite(t, filepath.Join(procRoot, "meminfo"), "MemAvailable: 4194304 kB\n")
	active := map[string]bool{manager.swapPath: true}
	writeSwapFixture(t, procRoot, active)
	swaponCalls := 0
	baseRunner := swapTestRunner(t, procRoot, active)
	runner.run = func(ctx context.Context, name string, arguments ...string) ([]byte, error) {
		if name == "swapon" {
			swaponCalls++
			if swaponCalls == 1 {
				return nil, errors.New("simulated activation failure")
			}
		}
		return baseRunner(ctx, name, arguments...)
	}

	_, _, _, err := manager.setSwap(context.Background(), 2048)
	if !errors.Is(err, ErrRolledBack) {
		t.Fatalf("expected rollback error, got %v", err)
	}
	info, statErr := os.Stat(manager.swapPath)
	if statErr != nil || !sizeWithinMiB(info.Size(), 1024) {
		t.Fatalf("original swapfile was not restored: info=%v err=%v", info, statErr)
	}
	if got := readLimited(filepath.Join(etcRoot, "fstab")); got != oldFstab {
		t.Fatalf("fstab was not restored: %q", got)
	}
	if !active[manager.swapPath] {
		t.Fatalf("original swap was not re-enabled: %#v", active)
	}
	if leftovers, _ := filepath.Glob(manager.swapPath + ".kpanel-previous-*"); len(leftovers) != 0 {
		t.Fatalf("rollback files remain: %#v", leftovers)
	}
}

func TestRunSwapViaSystemdUsesFixedHelper(t *testing.T) {
	runner := &fakeRunner{}
	manager, _, _, stateDir := testManager(t, runner)
	requestContext, cancel := context.WithCancel(context.Background())
	cancel()
	runner.run = func(ctx context.Context, name string, _ ...string) ([]byte, error) {
		if name != "systemd-run" {
			t.Fatalf("unexpected command %q", name)
		}
		if err := ctx.Err(); err != nil {
			t.Fatalf("host transaction inherited caller cancellation: %v", err)
		}
		return []byte(`{"changed":true,"backupPath":"/backup","message":"ok"}`), nil
	}

	changed, backup, message, err := manager.runSwapViaSystemd(requestContext, 2048)
	if err != nil || !changed || backup != "/backup" || message != "ok" {
		t.Fatalf("unexpected helper result: changed=%v backup=%q message=%q err=%v", changed, backup, message, err)
	}
	if len(runner.commands) != 1 {
		t.Fatalf("commands = %#v", runner.commands)
	}
	command := runner.commands[0]
	for _, expected := range []string{
		"systemd-run",
		"--wait",
		"--pipe",
		manager.executable + " swap-run --state-dir " + stateDir,
		"--swap-path " + manager.swapPath,
		"--size-mib 2048",
	} {
		if !strings.Contains(command, expected) {
			t.Fatalf("swap launcher missing %q: %q", expected, command)
		}
	}
	if strings.Contains(command, "sh -c") || strings.Contains(command, "bash -c") ||
		strings.Contains(command, "wipefs") {
		t.Fatalf("swap launcher used an unsafe command: %q", command)
	}
}

func mustSizedFile(t *testing.T, path string, sizeMiB int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(int64(sizeMiB) * 1024 * 1024); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func writeSwapFixture(t *testing.T, procRoot string, active map[string]bool) {
	t.Helper()
	data := "Filename Type Size Used Priority\n"
	for path, enabled := range active {
		if enabled {
			sizeKiB := int64(1048572)
			if info, err := os.Stat(path); err == nil && info.Size() >= 4*1024 {
				sizeKiB = info.Size()/1024 - 4
			}
			data += path + " file " + strconv.FormatInt(sizeKiB, 10) + " 0 -2\n"
		}
	}
	mustWrite(t, filepath.Join(procRoot, "swaps"), data)
}

func swapTestRunner(
	t *testing.T,
	procRoot string,
	active map[string]bool,
) func(context.Context, string, ...string) ([]byte, error) {
	t.Helper()
	return func(_ context.Context, name string, arguments ...string) ([]byte, error) {
		switch name {
		case "fallocate":
			size, _ := strconv.Atoi(strings.TrimSuffix(arguments[1], "M"))
			mustSizedFile(t, arguments[2], size)
		case "swapon":
			active[arguments[0]] = true
			writeSwapFixture(t, procRoot, active)
		case "swapoff":
			delete(active, arguments[0])
			writeSwapFixture(t, procRoot, active)
		}
		return nil, nil
	}
}

func TestStartMaintenanceUsesFixedSystemdUnit(t *testing.T) {
	runner := &fakeRunner{}
	manager, _, _, stateDir := testManager(t, runner)

	changed, message, err := manager.startMaintenance(context.Background(), "update", "full")
	if err != nil || !changed || message == "" {
		t.Fatalf("start maintenance: changed=%v message=%q err=%v", changed, message, err)
	}
	if len(runner.commands) != 1 {
		t.Fatalf("commands = %#v", runner.commands)
	}
	command := runner.commands[0]
	for _, expected := range []string{
		"systemd-run",
		"--unit=kejilion-panel-maintenance-",
		manager.executable + " maintenance-run --state-dir " + stateDir + " update",
	} {
		if !strings.Contains(command, expected) {
			t.Fatalf("maintenance launcher missing %q: %q", expected, command)
		}
	}
	if strings.Contains(command, "sh -c") || strings.Contains(command, "bash -c") {
		t.Fatalf("maintenance launcher used a shell: %q", command)
	}
	status := manager.MaintenanceStatus()
	if status.State != "running" || status.Action != "update" || status.Policy != "full" {
		t.Fatalf("unexpected maintenance state: %#v", status)
	}
	if _, _, err := manager.startMaintenance(context.Background(), "cleanup", "cache"); !errors.Is(err, ErrConflict) {
		t.Fatalf("second maintenance task should conflict, got %v", err)
	}
	if len(runner.commands) != 1 {
		t.Fatalf("conflicting task reached runner: %#v", runner.commands)
	}
}

func TestRunMaintenanceUpdateUsesSafeAPTSequence(t *testing.T) {
	runner := &fakeRunner{}
	manager, _, _, _ := testManager(t, runner)
	if err := manager.RunMaintenance(context.Background(), "update"); err != nil {
		t.Fatal(err)
	}
	commands := strings.Join(runner.commands, "\n")
	for _, expected := range []string{
		"dpkg --force-confold --configure -a",
		"apt-get -o Dpkg::Lock::Timeout=120 update",
		"apt-get -o Dpkg::Lock::Timeout=120 -y -o Dpkg::Options::=--force-confold full-upgrade",
	} {
		if !strings.Contains(commands, expected) {
			t.Fatalf("update sequence missing %q:\n%s", expected, commands)
		}
	}
	for _, forbidden := range []string{"pkill", "rm -f", "/var/lib/dpkg/lock", "sh -c"} {
		if strings.Contains(commands, forbidden) {
			t.Fatalf("unsafe command %q found:\n%s", forbidden, commands)
		}
	}
	status := manager.MaintenanceStatus()
	if status.State != "succeeded" || status.Progress != 100 || status.Action != "update" {
		t.Fatalf("unexpected final update state: %#v", status)
	}
}

func TestRunMaintenanceStandardCleanupPreservesLogsAndDocker(t *testing.T) {
	runner := &fakeRunner{}
	manager, _, _, _ := testManager(t, runner)
	if err := manager.RunMaintenance(context.Background(), "cleanup-standard"); err != nil {
		t.Fatal(err)
	}
	commands := strings.Join(runner.commands, "\n")
	for _, expected := range []string{
		"apt-get -o Dpkg::Lock::Timeout=120 -y autoremove --purge",
		"apt-get -o Dpkg::Lock::Timeout=120 clean",
		"journalctl --vacuum-time=7d",
		"journalctl --vacuum-size=500M",
	} {
		if !strings.Contains(commands, expected) {
			t.Fatalf("cleanup sequence missing %q:\n%s", expected, commands)
		}
	}
	for _, forbidden := range []string{"--vacuum-time=1s", "docker", "rm -rf", "/var/log/*", "/tmp/*"} {
		if strings.Contains(commands, forbidden) {
			t.Fatalf("unsafe cleanup %q found:\n%s", forbidden, commands)
		}
	}
	status := manager.MaintenanceStatus()
	if status.State != "succeeded" || status.Policy != "standard" {
		t.Fatalf("unexpected final cleanup state: %#v", status)
	}
}

func TestRunMaintenanceFailureIsPersisted(t *testing.T) {
	runner := &fakeRunner{
		run: func(_ context.Context, name string, arguments ...string) ([]byte, error) {
			if name == "apt-get" && arguments[len(arguments)-1] == "update" {
				return nil, errors.New("repository unavailable")
			}
			return nil, nil
		},
	}
	manager, _, _, _ := testManager(t, runner)
	if err := manager.RunMaintenance(context.Background(), "update"); err == nil {
		t.Fatal("expected update failure")
	}
	status := manager.MaintenanceStatus()
	if status.State != "failed" || status.FinishedAt == nil ||
		!strings.Contains(status.Message, "repository unavailable") {
		t.Fatalf("failure was not persisted: %#v", status)
	}
}

func TestDetectPackageManagerFamilies(t *testing.T) {
	tests := []struct {
		name        string
		release     string
		missing     map[string]bool
		wantKind    packageManagerKind
		wantCommand string
	}{
		{name: "ubuntu", release: "ID=ubuntu\nID_LIKE=debian\n", wantKind: packageManagerAPT, wantCommand: "apt-get"},
		{name: "rocky", release: "ID=rocky\nID_LIKE=\"rhel centos fedora\"\n", wantKind: packageManagerDNF, wantCommand: "dnf"},
		{
			name: "centos yum fallback", release: "ID=centos\nID_LIKE=rhel\n",
			missing:  map[string]bool{"dnf": true, "dnf5": true},
			wantKind: packageManagerYUM, wantCommand: "yum",
		},
		{name: "arch", release: "ID=arch\n", wantKind: packageManagerPacman, wantCommand: "pacman"},
		{name: "manjaro", release: "ID=manjaro\nID_LIKE=arch\n", wantKind: packageManagerPacman, wantCommand: "pacman"},
		{name: "opensuse", release: "ID=opensuse-tumbleweed\nID_LIKE=\"suse opensuse\"\n", wantKind: packageManagerZypper, wantCommand: "zypper"},
		{name: "alpine protected", release: "ID=alpine\n", wantKind: packageManagerUnknown},
		{name: "unknown protected", release: "ID=customlinux\n", wantKind: packageManagerUnknown},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manager, etcRoot, _, _ := testManager(t, &fakeRunner{missing: test.missing})
			mustWrite(t, filepath.Join(etcRoot, "os-release"), test.release)
			got := manager.detectPackageManager()
			if got.kind != test.wantKind || got.command != test.wantCommand {
				t.Fatalf(
					"detectPackageManager() = kind %q command %q reason %q, want kind %q command %q",
					got.kind,
					got.command,
					got.reason,
					test.wantKind,
					test.wantCommand,
				)
			}
		})
	}
}

func TestRunMaintenanceUpdateUsesSafeDNFSequence(t *testing.T) {
	runner := &fakeRunner{}
	manager, etcRoot, _, _ := testManager(t, runner)
	mustWrite(t, filepath.Join(etcRoot, "os-release"), "ID=rocky\nID_LIKE=\"rhel centos fedora\"\n")
	mustWrite(
		t,
		filepath.Join(etcRoot, "yum.repos.d", "rocky.repo"),
		"[baseos]\nbaseurl=https://dl.rockylinux.org/pub/rocky/$releasever/BaseOS/$basearch/os/\n",
	)

	if err := manager.RunMaintenance(context.Background(), "update"); err != nil {
		t.Fatal(err)
	}
	commands := strings.Join(runner.commands, "\n")
	for _, expected := range []string{
		"dnf -y update",
	} {
		if !strings.Contains(commands, expected) {
			t.Fatalf("DNF update sequence missing %q:\n%s", expected, commands)
		}
	}
}

func TestRunMaintenanceStandardCleanupUsesSafePacmanSequence(t *testing.T) {
	runner := &fakeRunner{}
	manager, etcRoot, _, _ := testManager(t, runner)
	mustWrite(t, filepath.Join(etcRoot, "os-release"), "ID=arch\n")
	mustWrite(
		t,
		filepath.Join(etcRoot, "pacman.d", "mirrorlist"),
		"Server = https://geo.mirror.pkgbuild.com/$repo/os/$arch\n",
	)

	if err := manager.RunMaintenance(context.Background(), "cleanup-standard"); err != nil {
		t.Fatal(err)
	}
	commands := strings.Join(runner.commands, "\n")
	for _, expected := range []string{
		"pacman -Scc --noconfirm",
		"journalctl --vacuum-time=7d",
		"journalctl --vacuum-size=500M",
	} {
		if !strings.Contains(commands, expected) {
			t.Fatalf("Pacman cleanup sequence missing %q:\n%s", expected, commands)
		}
	}
	for _, forbidden := range []string{"sh -c", "pacman -Rns", "rm -rf"} {
		if strings.Contains(commands, forbidden) {
			t.Fatalf("unsafe Pacman cleanup %q found:\n%s", forbidden, commands)
		}
	}
}

func TestRunMaintenanceUpdateUsesSafeZypperSequence(t *testing.T) {
	runner := &fakeRunner{}
	manager, etcRoot, _, _ := testManager(t, runner)
	mustWrite(t, filepath.Join(etcRoot, "os-release"), "ID=opensuse-leap\nID_LIKE=\"suse opensuse\"\n")
	mustWrite(
		t,
		filepath.Join(etcRoot, "zypp", "repos.d", "repo-oss.repo"),
		"[repo-oss]\nbaseurl=https://download.opensuse.org/distribution/leap/$releasever/repo/oss/\n",
	)

	if err := manager.RunMaintenance(context.Background(), "update"); err != nil {
		t.Fatal(err)
	}
	commands := strings.Join(runner.commands, "\n")
	for _, expected := range []string{
		"zypper --non-interactive refresh",
		"zypper --non-interactive update",
	} {
		if !strings.Contains(commands, expected) {
			t.Fatalf("Zypper update sequence missing %q:\n%s", expected, commands)
		}
	}
}

func TestRunMaintenanceRejectsAlpineAndUnknownDistributions(t *testing.T) {
	for _, release := range []string{"ID=alpine\n", "ID=customlinux\n"} {
		t.Run(strings.TrimSpace(release), func(t *testing.T) {
			manager, etcRoot, _, _ := testManager(t, &fakeRunner{})
			mustWrite(t, filepath.Join(etcRoot, "os-release"), release)
			if err := manager.RunMaintenance(context.Background(), "update"); !errors.Is(err, ErrUnsupported) {
				t.Fatalf("expected unsupported error, got %v", err)
			}
		})
	}
}

func TestCapabilitiesEnableMaintenanceButNotMirrorForRocky(t *testing.T) {
	manager, etcRoot, _, _ := testManager(t, &fakeRunner{})
	mustWrite(t, filepath.Join(etcRoot, "os-release"), "ID=rocky\nID_LIKE=\"rhel centos fedora\"\n")
	mustWrite(
		t,
		filepath.Join(etcRoot, "yum.repos.d", "rocky.repo"),
		"[baseos]\nbaseurl=https://dl.rockylinux.org/pub/rocky/$releasever/BaseOS/$basearch/os/\n",
	)
	capabilities := manager.Capabilities()
	for _, id := range []string{"system.update.write", "system.cleanup.write"} {
		capability := findCapability(capabilities, id)
		if !capability.Enabled {
			t.Fatalf("%s unexpectedly disabled: %s", id, capability.Reason)
		}
	}
	mirror := findCapability(capabilities, "system.mirror.write")
	if mirror.Enabled || !strings.Contains(mirror.Reason, "Debian/Ubuntu") {
		t.Fatalf("unexpected Rocky mirror capability: %#v", mirror)
	}
}

func findCapability(capabilities []contract.Capability, id string) contract.Capability {
	for _, capability := range capabilities {
		if capability.ID == id {
			return capability
		}
	}
	return contract.Capability{}
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
