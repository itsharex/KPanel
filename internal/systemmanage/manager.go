package systemmanage

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

var (
	ErrDisabled       = errors.New("system write executor is disabled")
	ErrInvalidInput   = errors.New("invalid system action input")
	ErrUnsupported    = errors.New("system action is unsupported on this host")
	ErrConflict       = errors.New("system configuration changed or conflicts")
	ErrRolledBack     = errors.New("system action failed and was rolled back")
	ErrNeedsAttention = errors.New("system action needs manual attention")
)

const (
	kpanelIPPreferenceMarker = "# KPanel managed IPv4 precedence"
	kpanelKernelMarker       = "# KPanel managed kernel profile"
	kpanelSwapMarker         = "# KPanel managed swap"
)

type Runner interface {
	Run(context.Context, string, ...string) ([]byte, error)
	LookPath(string) (string, error)
}

type commandRunner struct{}

func (commandRunner) Run(ctx context.Context, name string, arguments ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, name, arguments...)
	command.Env = append(
		os.Environ(),
		"LC_ALL=C",
		"LANG=C",
		"DEBIAN_FRONTEND=noninteractive",
		"NEEDRESTART_MODE=a",
		"APT_LISTCHANGES_FRONTEND=none",
	)
	output, err := command.CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(string(output))
		if len(detail) > 300 {
			detail = detail[:300]
		}
		if detail != "" {
			return nil, fmt.Errorf("%s: %w", detail, err)
		}
		return nil, err
	}
	return output, nil
}

func (commandRunner) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

type Config struct {
	Enabled    bool
	EtcRoot    string
	ProcRoot   string
	RunRoot    string
	StateDir   string
	Executable string
	Now        func() time.Time
	Runner     Runner
}

type Manager struct {
	enabled    bool
	etcRoot    string
	procRoot   string
	runRoot    string
	stateDir   string
	executable string
	now        func() time.Time
	runner     Runner
	mu         sync.Mutex
}

func NewManager(config Config) *Manager {
	if config.EtcRoot == "" {
		config.EtcRoot = "/etc"
	}
	if config.ProcRoot == "" {
		config.ProcRoot = "/proc"
	}
	if config.RunRoot == "" {
		config.RunRoot = "/var/run"
	}
	if config.StateDir == "" {
		config.StateDir = "/var/lib/kejilion-panel/system"
	}
	if config.Executable == "" {
		config.Executable = "/usr/local/libexec/kejilion-agent"
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.Runner == nil {
		config.Runner = commandRunner{}
	}
	return &Manager{
		enabled: config.Enabled, etcRoot: filepath.Clean(config.EtcRoot),
		procRoot: filepath.Clean(config.ProcRoot), runRoot: filepath.Clean(config.RunRoot),
		stateDir: filepath.Clean(config.StateDir), executable: filepath.Clean(config.Executable),
		now: config.Now, runner: config.Runner,
	}
}

func (m *Manager) Capabilities() []contract.Capability {
	disabledReason := ""
	if !m.enabled {
		disabledReason = "宿主机系统写入开关未启用"
	} else if runtime.GOOS != "linux" {
		disabledReason = "系统写入仅支持 Linux"
	} else if os.Geteuid() != 0 {
		disabledReason = "Agent 必须以受限 root 服务运行"
	}
	capability := func(id string, supported bool, reason string) contract.Capability {
		if disabledReason != "" {
			return contract.Capability{ID: id, Enabled: false, Reason: disabledReason}
		}
		if !supported {
			return contract.Capability{ID: id, Enabled: false, Reason: reason}
		}
		return contract.Capability{ID: id, Enabled: true, Methods: []string{"POST"}}
	}
	_, hostnamectlErr := m.runner.LookPath("hostnamectl")
	_, sshdErr := m.runner.LookPath("sshd")
	_, ssErr := m.runner.LookPath("ss")
	_, timedatectlErr := m.runner.LookPath("timedatectl")
	_, systemctlErr := m.runner.LookPath("systemctl")
	_, mkswapErr := m.runner.LookPath("mkswap")
	_, swaponErr := m.runner.LookPath("swapon")
	_, swapoffErr := m.runner.LookPath("swapoff")
	_, fallocateErr := m.runner.LookPath("fallocate")
	_, sysctlErr := m.runner.LookPath("sysctl")
	_, aptErr := m.runner.LookPath("apt-get")
	_, dpkgErr := m.runner.LookPath("dpkg")
	_, journalctlErr := m.runner.LookPath("journalctl")
	_, systemdRunErr := m.runner.LookPath("systemd-run")
	_, modprobeErr := m.runner.LookPath("modprobe")

	resolved := m.resolvedSupported()
	sshConfig := regularFile(filepath.Join(m.etcRoot, "ssh", "sshd_config"))
	aptSources := len(m.aptSourceFiles()) > 0
	return []contract.Capability{
		capability("system.hostname.write", hostnamectlErr == nil, "hostnamectl 不可用"),
		capability("system.ssh-port.write", sshdErr == nil && ssErr == nil && systemctlErr == nil && sshConfig, "OpenSSH 服务或配置不可用"),
		capability("system.dns.write", resolved && systemctlErr == nil, "当前仅安全接管 systemd-resolved"),
		capability("system.timezone.write", timedatectlErr == nil, "timedatectl 不可用"),
		capability("system.swap.write", mkswapErr == nil && swaponErr == nil && swapoffErr == nil && fallocateErr == nil, "Swap 工具不完整"),
		capability("system.mirror.write", aptErr == nil && aptSources, "当前仅支持 Debian/Ubuntu APT 软件源"),
		capability("system.ip-preference.write", true, ""),
		capability("system.kernel-tuning.write", sysctlErr == nil, "sysctl 不可用"),
		capability("system.bbr.write", sysctlErr == nil && modprobeErr == nil, "内核调优工具不完整"),
		capability("system.update.write", aptErr == nil && dpkgErr == nil && systemdRunErr == nil && aptSources, "当前仅支持由 systemd 托管的 Debian/Ubuntu APT 更新"),
		capability("system.cleanup.write", aptErr == nil && journalctlErr == nil && systemdRunErr == nil && aptSources, "当前仅支持由 systemd 托管的 Debian/Ubuntu APT 清理"),
		{ID: "system.reinstall", Enabled: false, Reason: "重装系统必须使用带外控制台，Web 端保持锁定"},
	}
}

func (m *Manager) Execute(ctx context.Context, input contract.SystemActionRequest) (contract.SystemActionResult, error) {
	if !m.enabled {
		return contract.SystemActionResult{}, ErrDisabled
	}
	if runtime.GOOS != "linux" || os.Geteuid() != 0 {
		return contract.SystemActionResult{}, ErrUnsupported
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	result := contract.SystemActionResult{
		Action: input.Action, Status: "succeeded", AppliedAt: m.now().UTC(),
	}
	var err error
	switch input.Action {
	case "hostname":
		result.Changed, result.BackupPath, result.Message, err = m.setHostname(ctx, input.Hostname)
	case "ssh-port":
		result.Changed, result.BackupPath, result.Message, err = m.addSSHPort(ctx, input.Port)
	case "dns":
		result.Changed, result.BackupPath, result.Message, err = m.setDNS(ctx, input.Servers)
	case "timezone":
		result.Changed, result.Message, err = m.setTimezone(ctx, input.Timezone)
	case "swap":
		result.Changed, result.BackupPath, result.Message, err = m.setSwap(ctx, input.SwapSizeMiB)
	case "mirror":
		result.Changed, result.BackupPath, result.Message, err = m.setMirror(ctx, input.MirrorPreset)
	case "ip-preference":
		result.Changed, result.BackupPath, result.Message, err = m.setIPPreference(input.Preference)
	case "kernel-tuning":
		result.Changed, result.BackupPath, result.Message, err = m.setKernelTuning(ctx, input.Profile)
	case "bbr":
		if input.Enabled == nil {
			err = fmt.Errorf("%w: enabled is required", ErrInvalidInput)
			break
		}
		result.Changed, result.BackupPath, result.Message, err = m.setBBR(ctx, *input.Enabled)
	case "update":
		result.Changed, result.Message, err = m.startMaintenance(ctx, input.Action, input.MaintenancePolicy)
		if err == nil {
			result.Status = "accepted"
		}
	case "cleanup":
		result.Changed, result.Message, err = m.startMaintenance(ctx, input.Action, input.MaintenancePolicy)
		if err == nil {
			result.Status = "accepted"
		}
	default:
		err = fmt.Errorf("%w: unknown action", ErrInvalidInput)
	}
	if err != nil {
		result.Status = "failed"
		return result, err
	}
	return result, nil
}

func (m *Manager) setHostname(ctx context.Context, value string) (bool, string, string, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if !validHostname(value) {
		return false, "", "", fmt.Errorf("%w: hostname must be a valid DNS hostname", ErrInvalidInput)
	}
	hostnamePath := filepath.Join(m.etcRoot, "hostname")
	hostsPath := filepath.Join(m.etcRoot, "hosts")
	oldHostname := strings.TrimSpace(readLimited(hostnamePath))
	if oldHostname == value {
		return false, "", "主机名已经是 " + value, nil
	}
	backup, err := m.createBackup("hostname", hostnamePath, hostsPath)
	if err != nil {
		return false, "", "", err
	}
	oldHosts, err := os.ReadFile(hostsPath)
	if err != nil {
		return false, backup, "", fmt.Errorf("%w: read hosts: %v", ErrUnsupported, err)
	}
	newHosts := updateHosts(oldHosts, oldHostname, value)
	if err := writeAtomic(hostnamePath, []byte(value+"\n"), 0o644); err != nil {
		return false, backup, "", err
	}
	if err := writeAtomic(hostsPath, newHosts, 0o644); err != nil {
		_ = writeAtomic(hostnamePath, []byte(oldHostname+"\n"), 0o644)
		return false, backup, "", fmt.Errorf("%w: %v", ErrRolledBack, err)
	}
	if _, err := m.runner.Run(ctx, "hostnamectl", "set-hostname", value); err != nil {
		_ = writeAtomic(hostnamePath, []byte(oldHostname+"\n"), 0o644)
		_ = writeAtomic(hostsPath, oldHosts, 0o644)
		_, _ = m.runner.Run(ctx, "hostnamectl", "set-hostname", oldHostname)
		return false, backup, "", fmt.Errorf("%w: hostnamectl: %v", ErrRolledBack, err)
	}
	output, err := m.runner.Run(ctx, "hostname")
	if err != nil || strings.TrimSpace(string(output)) != value {
		_ = writeAtomic(hostnamePath, []byte(oldHostname+"\n"), 0o644)
		_ = writeAtomic(hostsPath, oldHosts, 0o644)
		_, rollbackErr := m.runner.Run(ctx, "hostnamectl", "set-hostname", oldHostname)
		if rollbackErr != nil {
			return false, backup, "", fmt.Errorf("%w: hostname verification failed and rollback command failed", ErrNeedsAttention)
		}
		return false, backup, "", fmt.Errorf("%w: hostname verification failed", ErrRolledBack)
	}
	return true, backup, "主机名已更新并回读验证", nil
}

func (m *Manager) addSSHPort(ctx context.Context, port uint16) (bool, string, string, error) {
	if port == 0 {
		return false, "", "", fmt.Errorf("%w: port must be between 1 and 65535", ErrInvalidInput)
	}
	current := m.configuredSSHPorts()
	if slices.Contains(current, port) {
		return false, "", "SSH 已监听该配置端口", nil
	}
	if used, _ := m.portListening(ctx, port); used {
		return false, "", "", fmt.Errorf("%w: port %d is already in use", ErrConflict, port)
	}
	configPath := filepath.Join(m.etcRoot, "ssh", "sshd_config.d", "00-kpanel-ports.conf")
	backup, err := m.createBackup("ssh-port", configPath)
	if err != nil {
		return false, "", "", err
	}
	old, existed, mode, err := snapshotFile(configPath)
	if err != nil {
		return false, backup, "", err
	}
	ports := append(append([]uint16{}, current...), port)
	slices.Sort(ports)
	ports = slices.Compact(ports)
	var config strings.Builder
	config.WriteString("# Managed by KPanel. Existing ports are retained to prevent lockout.\n")
	for _, item := range ports {
		fmt.Fprintf(&config, "Port %d\n", item)
	}
	if err := writeAtomic(configPath, []byte(config.String()), 0o640); err != nil {
		return false, backup, "", err
	}
	rollback := func() error {
		if err := restoreFile(configPath, old, existed, mode); err != nil {
			return err
		}
		_, err := m.reloadSSH(ctx)
		return err
	}
	if _, err := m.runner.Run(ctx, "sshd", "-t", "-f", filepath.Join(m.etcRoot, "ssh", "sshd_config")); err != nil {
		_ = rollback()
		return false, backup, "", fmt.Errorf("%w: sshd configuration test: %v", ErrRolledBack, err)
	}
	firewallRollback, err := m.openFirewallPort(ctx, port)
	if err != nil {
		_ = rollback()
		return false, backup, "", fmt.Errorf("%w: firewall: %v", ErrRolledBack, err)
	}
	if _, err := m.reloadSSH(ctx); err != nil {
		_ = firewallRollback()
		_ = rollback()
		return false, backup, "", fmt.Errorf("%w: reload SSH: %v", ErrRolledBack, err)
	}
	listening := false
	for attempt := 0; attempt < 10; attempt++ {
		if listening, _ = m.portListening(ctx, port); listening {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if !listening {
		_ = firewallRollback()
		if err := rollback(); err != nil {
			return false, backup, "", fmt.Errorf("%w: new SSH port did not listen and rollback failed: %v", ErrNeedsAttention, err)
		}
		return false, backup, "", fmt.Errorf("%w: new SSH port did not listen", ErrRolledBack)
	}
	return true, backup, fmt.Sprintf("已安全添加 SSH 端口 %d，原端口继续保留", port), nil
}

func (m *Manager) setDNS(ctx context.Context, servers []string) (bool, string, string, error) {
	if !m.resolvedSupported() {
		return false, "", "", fmt.Errorf("%w: only systemd-resolved is supported", ErrUnsupported)
	}
	if len(servers) < 1 || len(servers) > 4 {
		return false, "", "", fmt.Errorf("%w: one to four DNS servers are required", ErrInvalidInput)
	}
	normalized := make([]string, 0, len(servers))
	seen := make(map[string]bool)
	for _, raw := range servers {
		ip := net.ParseIP(strings.TrimSpace(raw))
		if ip == nil || ip.IsUnspecified() || ip.IsMulticast() {
			return false, "", "", fmt.Errorf("%w: invalid DNS address %q", ErrInvalidInput, raw)
		}
		value := ip.String()
		if !seen[value] {
			seen[value] = true
			normalized = append(normalized, value)
		}
	}
	configPath := filepath.Join(m.etcRoot, "systemd", "resolved.conf.d", "90-kpanel.conf")
	config := []byte("[Resolve]\nDNS=" + strings.Join(normalized, " ") + "\nFallbackDNS=\n")
	if bytes.Equal(bytes.TrimSpace([]byte(readLimited(configPath))), bytes.TrimSpace(config)) {
		return false, "", "DNS 配置没有变化", nil
	}
	backup, err := m.createBackup("dns", configPath)
	if err != nil {
		return false, "", "", err
	}
	old, existed, mode, err := snapshotFile(configPath)
	if err != nil {
		return false, backup, "", err
	}
	if err := writeAtomic(configPath, config, 0o644); err != nil {
		return false, backup, "", err
	}
	if _, err := m.runner.Run(ctx, "systemctl", "reload-or-restart", "systemd-resolved.service"); err != nil {
		_ = restoreFile(configPath, old, existed, mode)
		_, rollbackErr := m.runner.Run(ctx, "systemctl", "reload-or-restart", "systemd-resolved.service")
		if rollbackErr != nil {
			return false, backup, "", fmt.Errorf("%w: DNS reload and rollback reload failed", ErrNeedsAttention)
		}
		return false, backup, "", fmt.Errorf("%w: reload resolver: %v", ErrRolledBack, err)
	}
	return true, backup, "DNS 已更新，systemd-resolved 已重新加载", nil
}

func (m *Manager) setTimezone(ctx context.Context, zone string) (bool, string, error) {
	zone = strings.TrimSpace(zone)
	if zone == "" || strings.HasPrefix(zone, "/") || strings.Contains(zone, "..") || strings.ContainsAny(zone, "\x00\r\n") {
		return false, "", fmt.Errorf("%w: invalid timezone", ErrInvalidInput)
	}
	zoneRoot := filepath.Join(m.etcRoot, "..", "usr", "share", "zoneinfo")
	if m.etcRoot == "/etc" {
		zoneRoot = "/usr/share/zoneinfo"
	}
	candidate := filepath.Clean(filepath.Join(zoneRoot, filepath.FromSlash(zone)))
	relative, err := filepath.Rel(filepath.Clean(zoneRoot), candidate)
	if err != nil || strings.HasPrefix(relative, "..") || !regularFileFollow(candidate) {
		return false, "", fmt.Errorf("%w: timezone is not present in the IANA database", ErrInvalidInput)
	}
	oldOutput, err := m.runner.Run(ctx, "timedatectl", "show", "--property=Timezone", "--value")
	if err != nil {
		return false, "", fmt.Errorf("%w: read current timezone: %v", ErrUnsupported, err)
	}
	old := strings.TrimSpace(string(oldOutput))
	if old == zone {
		return false, "系统时区没有变化", nil
	}
	if _, err := m.runner.Run(ctx, "timedatectl", "set-timezone", zone); err != nil {
		return false, "", fmt.Errorf("%w: set timezone: %v", ErrRolledBack, err)
	}
	currentOutput, err := m.runner.Run(ctx, "timedatectl", "show", "--property=Timezone", "--value")
	if err != nil || strings.TrimSpace(string(currentOutput)) != zone {
		_, rollbackErr := m.runner.Run(ctx, "timedatectl", "set-timezone", old)
		if rollbackErr != nil {
			return false, "", fmt.Errorf("%w: timezone verification failed and rollback failed", ErrNeedsAttention)
		}
		return false, "", fmt.Errorf("%w: timezone verification failed", ErrRolledBack)
	}
	return true, "系统时区已更新并回读验证", nil
}

func (m *Manager) setIPPreference(preference string) (bool, string, string, error) {
	if preference != "ipv4" && preference != "system_default" {
		return false, "", "", fmt.Errorf("%w: preference must be ipv4 or system_default", ErrInvalidInput)
	}
	path := filepath.Join(m.etcRoot, "gai.conf")
	old, existed, mode, err := snapshotFile(path)
	if err != nil {
		return false, "", "", err
	}
	newData := updateIPPreference(old, preference)
	if bytes.Equal(old, newData) {
		return false, "", "IP 优先级配置没有变化", nil
	}
	backup, err := m.createBackup("ip-preference", path)
	if err != nil {
		return false, "", "", err
	}
	if mode == 0 {
		mode = 0o644
	}
	if err := writeAtomic(path, newData, mode); err != nil {
		_ = restoreFile(path, old, existed, mode)
		return false, backup, "", err
	}
	if preference == "ipv4" {
		return true, backup, "已设置 IPv4 优先；kejilion.sh 可识别同一规则", nil
	}
	return true, backup, "已恢复系统默认地址优先级", nil
}

func (m *Manager) setSwap(ctx context.Context, sizeMiB int) (bool, string, string, error) {
	if sizeMiB != 0 && (sizeMiB < 256 || sizeMiB > 65536) {
		return false, "", "", fmt.Errorf("%w: swapSizeMiB must be 0 or between 256 and 65536", ErrInvalidInput)
	}
	swapPath := filepath.Join(m.stateDir, "swapfile")
	fstabPath := filepath.Join(m.etcRoot, "fstab")
	active := m.swapActive(swapPath)
	if sizeMiB == 0 {
		if !active && !regularFile(swapPath) {
			return false, "", "KPanel 专属 Swap 已经停用", nil
		}
		backup, err := m.createBackup("swap-disable", fstabPath)
		if err != nil {
			return false, "", "", err
		}
		oldFstab, existed, mode, err := snapshotFile(fstabPath)
		if err != nil {
			return false, backup, "", err
		}
		newFstab := updateFstabSwap(oldFstab, swapPath, false)
		if err := writeAtomic(fstabPath, newFstab, fileModeOr(mode, 0o644)); err != nil {
			return false, backup, "", err
		}
		if active {
			if _, err := m.runner.Run(ctx, "swapoff", swapPath); err != nil {
				_ = restoreFile(fstabPath, oldFstab, existed, mode)
				return false, backup, "", fmt.Errorf("%w: swapoff: %v", ErrRolledBack, err)
			}
		}
		if err := os.Remove(swapPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			_ = restoreFile(fstabPath, oldFstab, existed, mode)
			if active {
				_, _ = m.runner.Run(ctx, "swapon", swapPath)
			}
			return false, backup, "", fmt.Errorf("%w: remove managed swapfile: %v", ErrNeedsAttention, err)
		}
		return true, backup, "KPanel 专属 Swap 已停用；其他 Swap 未受影响", nil
	}

	if active {
		info, err := os.Stat(swapPath)
		if err == nil && sizeWithinMiB(info.Size(), sizeMiB) {
			return false, "", "KPanel 专属 Swap 大小没有变化", nil
		}
		return false, "", "", fmt.Errorf("%w: disable the existing KPanel swap before resizing", ErrConflict)
	}
	if err := os.MkdirAll(m.stateDir, 0o700); err != nil {
		return false, "", "", fmt.Errorf("%w: create system state directory: %v", ErrUnsupported, err)
	}
	backup, err := m.createBackup("swap-enable", fstabPath)
	if err != nil {
		return false, "", "", err
	}
	oldFstab, fstabExisted, fstabMode, err := snapshotFile(fstabPath)
	if err != nil {
		return false, backup, "", err
	}
	stalePath := ""
	if regularFile(swapPath) {
		stalePath = swapPath + ".previous"
		_ = os.Remove(stalePath)
		if err := os.Rename(swapPath, stalePath); err != nil {
			return false, backup, "", fmt.Errorf("%w: preserve stale managed swapfile: %v", ErrConflict, err)
		}
	}
	tempPath := swapPath + ".new"
	_ = os.Remove(tempPath)
	cleanup := func() {
		_ = os.Remove(tempPath)
		if stalePath != "" && !regularFile(swapPath) {
			_ = os.Rename(stalePath, swapPath)
		}
	}
	if _, err := m.runner.Run(ctx, "fallocate", "-l", strconv.Itoa(sizeMiB)+"M", tempPath); err != nil {
		cleanup()
		return false, backup, "", fmt.Errorf("%w: allocate swapfile: %v", ErrRolledBack, err)
	}
	if err := os.Chmod(tempPath, 0o600); err != nil {
		cleanup()
		return false, backup, "", fmt.Errorf("%w: protect swapfile: %v", ErrRolledBack, err)
	}
	if _, err := m.runner.Run(ctx, "mkswap", tempPath); err != nil {
		cleanup()
		return false, backup, "", fmt.Errorf("%w: mkswap: %v", ErrRolledBack, err)
	}
	if err := os.Rename(tempPath, swapPath); err != nil {
		cleanup()
		return false, backup, "", fmt.Errorf("%w: publish swapfile: %v", ErrRolledBack, err)
	}
	newFstab := updateFstabSwap(oldFstab, swapPath, true)
	if err := writeAtomic(fstabPath, newFstab, fileModeOr(fstabMode, 0o644)); err != nil {
		_ = os.Remove(swapPath)
		cleanup()
		return false, backup, "", fmt.Errorf("%w: update fstab: %v", ErrRolledBack, err)
	}
	if _, err := m.runner.Run(ctx, "swapon", swapPath); err != nil {
		_ = restoreFile(fstabPath, oldFstab, fstabExisted, fstabMode)
		_ = os.Remove(swapPath)
		cleanup()
		return false, backup, "", fmt.Errorf("%w: swapon: %v", ErrRolledBack, err)
	}
	if !m.swapActive(swapPath) {
		_, _ = m.runner.Run(ctx, "swapoff", swapPath)
		_ = restoreFile(fstabPath, oldFstab, fstabExisted, fstabMode)
		_ = os.Remove(swapPath)
		cleanup()
		return false, backup, "", fmt.Errorf("%w: swap activation could not be verified", ErrRolledBack)
	}
	if stalePath != "" {
		_ = os.Remove(stalePath)
	}
	return true, backup, fmt.Sprintf("已启用 %d MiB KPanel 专属 Swap，未修改现有 Swap", sizeMiB), nil
}

func (m *Manager) setMirror(ctx context.Context, preset string) (bool, string, string, error) {
	if preset != "official" && preset != "aliyun" {
		return false, "", "", fmt.Errorf("%w: mirrorPreset must be official or aliyun", ErrInvalidInput)
	}
	osID := strings.ToLower(osReleaseValue(filepath.Join(m.etcRoot, "os-release"), "ID"))
	if osID != "debian" && osID != "ubuntu" {
		return false, "", "", fmt.Errorf("%w: only Debian and Ubuntu are supported", ErrUnsupported)
	}
	files := m.aptSourceFiles()
	if len(files) == 0 {
		return false, "", "", fmt.Errorf("%w: no APT source files were found", ErrUnsupported)
	}
	type sourceChange struct {
		path    string
		old     []byte
		new     []byte
		mode    os.FileMode
		existed bool
	}
	var changes []sourceChange
	for _, path := range files {
		old, existed, mode, err := snapshotFile(path)
		if err != nil {
			return false, "", "", err
		}
		rewritten := rewriteAPTSource(old, osID, preset)
		if !bytes.Equal(old, rewritten) {
			changes = append(changes, sourceChange{path: path, old: old, new: rewritten, mode: mode, existed: existed})
		}
	}
	if len(changes) == 0 {
		return false, "", "软件源已经使用所选线路", nil
	}
	paths := make([]string, 0, len(changes))
	for _, change := range changes {
		paths = append(paths, change.path)
	}
	backup, err := m.createBackup("mirror-"+preset, paths...)
	if err != nil {
		return false, "", "", err
	}
	restore := func() error {
		var restoreErrors []error
		for _, change := range changes {
			if err := restoreFile(change.path, change.old, change.existed, change.mode); err != nil {
				restoreErrors = append(restoreErrors, err)
			}
		}
		return errors.Join(restoreErrors...)
	}
	for _, change := range changes {
		if err := writeAtomic(change.path, change.new, fileModeOr(change.mode, 0o644)); err != nil {
			_ = restore()
			return false, backup, "", fmt.Errorf("%w: write APT source: %v", ErrRolledBack, err)
		}
	}
	aptState := filepath.Join(m.stateDir, "apt-validation")
	aptLists := filepath.Join(aptState, "lists")
	aptCache := filepath.Join(aptState, "cache")
	for _, path := range []string{filepath.Join(aptLists, "partial"), filepath.Join(aptCache, "archives", "partial")} {
		if err := os.MkdirAll(path, 0o700); err != nil {
			_ = restore()
			return false, backup, "", fmt.Errorf("%w: prepare isolated APT validation state: %v", ErrRolledBack, err)
		}
	}
	if _, err := m.runner.Run(
		ctx, "apt-get", "-o", "Acquire::Retries=1",
		"-o", "Acquire::http::Timeout=12", "-o", "Acquire::https::Timeout=12",
		"-o", "Dir::State::Lists="+aptLists,
		"-o", "Dir::Cache="+aptCache,
		"-o", "APT::Get::List-Cleanup=1",
		"update",
	); err != nil {
		if rollbackErr := restore(); rollbackErr != nil {
			return false, backup, "", fmt.Errorf("%w: APT validation failed and rollback failed: %v", ErrNeedsAttention, rollbackErr)
		}
		return false, backup, "", fmt.Errorf("%w: APT validation failed: %v", ErrRolledBack, err)
	}
	label := "发行版官方源"
	if preset == "aliyun" {
		label = "阿里云镜像源"
	}
	return true, backup, "已切换为" + label + "并通过 apt-get update 验证；第三方源未修改", nil
}

func (m *Manager) setKernelTuning(ctx context.Context, profile string) (bool, string, string, error) {
	if profile != "balanced" && profile != "web" && profile != "off" {
		return false, "", "", fmt.Errorf("%w: profile must be balanced, web, or off", ErrInvalidInput)
	}
	path := filepath.Join(m.etcRoot, "sysctl.d", "99-kejilion-optimize.conf")
	old, existed, mode, err := snapshotFile(path)
	if err != nil {
		return false, "", "", err
	}
	if existed && !bytes.Contains(old, []byte(kpanelKernelMarker)) {
		return false, "", "", fmt.Errorf("%w: existing kejilion.sh tuning is externally managed; restore it from the script before replacing it", ErrConflict)
	}
	if profile == "off" && !existed {
		return false, "", "KPanel 内核优化已经停用", nil
	}
	var config []byte
	switch profile {
	case "balanced":
		config = []byte(kpanelKernelMarker + "\n# 模式: 均衡优化模式 | 场景: balanced\n" +
			"net.core.somaxconn = 4096\n" +
			"net.ipv4.tcp_max_syn_backlog = 4096\n" +
			"net.ipv4.tcp_fastopen = 3\n" +
			"net.ipv4.tcp_fin_timeout = 30\n" +
			"net.ipv4.tcp_keepalive_time = 600\n" +
			"net.ipv4.tcp_syncookies = 1\n" +
			"vm.swappiness = 10\n" +
			"fs.file-max = 1048576\n")
	case "web":
		config = []byte(kpanelKernelMarker + "\n# 模式: 网站搭建优化模式 | 场景: web\n" +
			"net.core.somaxconn = 16384\n" +
			"net.core.netdev_max_backlog = 16384\n" +
			"net.ipv4.tcp_max_syn_backlog = 8192\n" +
			"net.ipv4.tcp_fastopen = 3\n" +
			"net.ipv4.tcp_tw_reuse = 1\n" +
			"net.ipv4.tcp_fin_timeout = 20\n" +
			"net.ipv4.tcp_keepalive_time = 300\n" +
			"net.ipv4.tcp_syncookies = 1\n" +
			"vm.swappiness = 10\n" +
			"fs.file-max = 1048576\n")
	}
	if profile != "off" && bytes.Equal(bytes.TrimSpace(old), bytes.TrimSpace(config)) {
		return false, "", "内核优化配置没有变化", nil
	}
	backup, err := m.createBackup("kernel-"+profile, path)
	if err != nil {
		return false, "", "", err
	}
	if profile == "off" {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return false, backup, "", err
		}
		if _, err := m.runner.Run(ctx, "sysctl", "--system"); err != nil {
			_ = restoreFile(path, old, existed, mode)
			_, _ = m.runner.Run(ctx, "sysctl", "-p", path)
			return false, backup, "", fmt.Errorf("%w: restore system sysctl defaults: %v", ErrRolledBack, err)
		}
		return true, backup, "KPanel 内核优化已停用并重新加载系统参数", nil
	}
	if err := writeAtomic(path, config, 0o644); err != nil {
		return false, backup, "", err
	}
	if _, err := m.runner.Run(ctx, "sysctl", "-p", path); err != nil {
		_ = restoreFile(path, old, existed, mode)
		if existed {
			_, _ = m.runner.Run(ctx, "sysctl", "-p", path)
		} else {
			_, _ = m.runner.Run(ctx, "sysctl", "--system")
		}
		return false, backup, "", fmt.Errorf("%w: apply kernel profile: %v", ErrRolledBack, err)
	}
	return true, backup, "内核优化配置已写入 kejilion.sh 同一识别路径并生效", nil
}

func (m *Manager) setBBR(ctx context.Context, enabled bool) (bool, string, string, error) {
	path := filepath.Join(m.etcRoot, "sysctl.d", "99-kejilion-bbr.conf")
	old, existed, mode, err := snapshotFile(path)
	if err != nil {
		return false, "", "", err
	}
	targetCC, targetQDisc := "cubic", "fq_codel"
	if enabled {
		if _, err := m.runner.Run(ctx, "modprobe", "tcp_bbr"); err != nil {
			return false, "", "", fmt.Errorf("%w: load tcp_bbr: %v", ErrUnsupported, err)
		}
		available := strings.Fields(m.procValue("sys/net/ipv4/tcp_available_congestion_control"))
		if !slices.Contains(available, "bbr") {
			return false, "", "", fmt.Errorf("%w: the running kernel does not expose BBR", ErrUnsupported)
		}
		targetCC, targetQDisc = "bbr", "fq"
	}
	config := []byte("# Managed by KPanel; compatible with kejilion.sh bbr_on\n" +
		"net.core.default_qdisc=" + targetQDisc + "\n" +
		"net.ipv4.tcp_congestion_control=" + targetCC + "\n")
	currentCC := strings.TrimSpace(m.procValue("sys/net/ipv4/tcp_congestion_control"))
	currentQDisc := strings.TrimSpace(m.procValue("sys/net/core/default_qdisc"))
	if currentCC == targetCC && currentQDisc == targetQDisc && bytes.Equal(bytes.TrimSpace(old), bytes.TrimSpace(config)) {
		return false, "", "BBR 配置没有变化", nil
	}
	backup, err := m.createBackup("bbr", path)
	if err != nil {
		return false, "", "", err
	}
	if err := writeAtomic(path, config, 0o644); err != nil {
		return false, backup, "", err
	}
	if _, err := m.runner.Run(ctx, "sysctl", "-p", path); err != nil {
		_ = restoreFile(path, old, existed, mode)
		if existed {
			_, _ = m.runner.Run(ctx, "sysctl", "-p", path)
		} else {
			_, _ = m.runner.Run(ctx, "sysctl", "--system")
		}
		return false, backup, "", fmt.Errorf("%w: apply BBR configuration: %v", ErrRolledBack, err)
	}
	if strings.TrimSpace(m.procValue("sys/net/ipv4/tcp_congestion_control")) != targetCC {
		_ = restoreFile(path, old, existed, mode)
		_, _ = m.runner.Run(ctx, "sysctl", "--system")
		return false, backup, "", fmt.Errorf("%w: congestion control verification failed", ErrRolledBack)
	}
	if enabled {
		return true, backup, "BBR 已启用并回读验证", nil
	}
	return true, backup, "BBR 已停用，已恢复 cubic/fq_codel", nil
}

func (m *Manager) createBackup(action string, paths ...string) (string, error) {
	var nonce [4]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", fmt.Errorf("generate backup identifier: %w", err)
	}
	name := m.now().UTC().Format("20060102T150405Z") + "-" + safeName(action) + "-" + hex.EncodeToString(nonce[:])
	root := filepath.Join(m.stateDir, "backups", name)
	if err := os.MkdirAll(root, 0o700); err != nil {
		return "", fmt.Errorf("create system backup: %w", err)
	}
	var manifest strings.Builder
	for index, path := range paths {
		fmt.Fprintf(&manifest, "%d\t%s\n", index, path)
		data, existed, mode, err := snapshotFile(path)
		if err != nil {
			return root, err
		}
		if !existed {
			if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("%02d.absent", index)), nil, 0o600); err != nil {
				return root, err
			}
			continue
		}
		backupPath := filepath.Join(root, fmt.Sprintf("%02d-%s", index, safeName(filepath.Base(path))))
		if err := os.WriteFile(backupPath, data, fileModeOr(mode, 0o600)); err != nil {
			return root, err
		}
	}
	if err := os.WriteFile(filepath.Join(root, "manifest.tsv"), []byte(manifest.String()), 0o600); err != nil {
		return root, err
	}
	return root, nil
}

func (m *Manager) configuredSSHPorts() []uint16 {
	files := []string{filepath.Join(m.etcRoot, "ssh", "sshd_config")}
	fragments, _ := filepath.Glob(filepath.Join(m.etcRoot, "ssh", "sshd_config.d", "*.conf"))
	slices.Sort(fragments)
	files = append(files, fragments...)
	var ports []uint16
	for _, path := range files {
		scanner := bufio.NewScanner(strings.NewReader(readLimited(path)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) != 2 || !strings.EqualFold(fields[0], "Port") {
				continue
			}
			value, err := strconv.ParseUint(fields[1], 10, 16)
			if err == nil && value > 0 {
				ports = append(ports, uint16(value))
			}
		}
	}
	if len(ports) == 0 {
		return []uint16{22}
	}
	slices.Sort(ports)
	return slices.Compact(ports)
}

func (m *Manager) reloadSSH(ctx context.Context) ([]byte, error) {
	if output, err := m.runner.Run(ctx, "systemctl", "reload", "ssh.service"); err == nil {
		return output, nil
	}
	return m.runner.Run(ctx, "systemctl", "reload", "sshd.service")
}

func (m *Manager) portListening(ctx context.Context, port uint16) (bool, error) {
	output, err := m.runner.Run(ctx, "ss", "-H", "-ltn")
	if err != nil {
		return false, err
	}
	suffix := ":" + strconv.Itoa(int(port))
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 4 && strings.HasSuffix(fields[3], suffix) {
			return true, nil
		}
	}
	return false, nil
}

func (m *Manager) openFirewallPort(ctx context.Context, port uint16) (func() error, error) {
	rule := strconv.Itoa(int(port)) + "/tcp"
	if _, err := m.runner.LookPath("ufw"); err == nil {
		output, statusErr := m.runner.Run(ctx, "ufw", "status")
		if statusErr == nil && strings.Contains(strings.ToLower(string(output)), "status: active") {
			if _, err := m.runner.Run(ctx, "ufw", "allow", rule); err != nil {
				return func() error { return nil }, err
			}
			return func() error {
				_, err := m.runner.Run(ctx, "ufw", "--force", "delete", "allow", rule)
				return err
			}, nil
		}
	}
	if _, err := m.runner.LookPath("firewall-cmd"); err == nil {
		if _, stateErr := m.runner.Run(ctx, "firewall-cmd", "--state"); stateErr == nil {
			if _, err := m.runner.Run(ctx, "firewall-cmd", "--add-port="+rule); err != nil {
				return func() error { return nil }, err
			}
			if _, err := m.runner.Run(ctx, "firewall-cmd", "--permanent", "--add-port="+rule); err != nil {
				_, _ = m.runner.Run(ctx, "firewall-cmd", "--remove-port="+rule)
				return func() error { return nil }, err
			}
			return func() error {
				_, first := m.runner.Run(ctx, "firewall-cmd", "--remove-port="+rule)
				_, second := m.runner.Run(ctx, "firewall-cmd", "--permanent", "--remove-port="+rule)
				return errors.Join(first, second)
			}, nil
		}
	}
	if _, err := m.runner.LookPath("iptables"); err == nil {
		output, statusErr := m.runner.Run(ctx, "iptables", "-S", "INPUT")
		if statusErr == nil && strings.Contains(string(output), "-P INPUT DROP") {
			if _, allowedErr := m.runner.Run(
				ctx, "iptables", "-C", "INPUT", "-p", "tcp",
				"--dport", strconv.Itoa(int(port)), "-j", "ACCEPT",
			); allowedErr == nil {
				return func() error { return nil }, nil
			}
			return func() error { return nil },
				fmt.Errorf("unmanaged iptables INPUT policy is DROP; add a persistent allow rule first")
		}
	}
	return func() error { return nil }, nil
}

func (m *Manager) resolvedSupported() bool {
	path := filepath.Join(m.etcRoot, "resolv.conf")
	if target, err := os.Readlink(path); err == nil && strings.Contains(strings.ToLower(target), "systemd/resolve") {
		return true
	}
	return regularFile(filepath.Join(m.etcRoot, "systemd", "resolved.conf"))
}

func (m *Manager) aptSourceFiles() []string {
	var files []string
	primary := filepath.Join(m.etcRoot, "apt", "sources.list")
	if regularFile(primary) {
		files = append(files, primary)
	}
	for _, pattern := range []string{
		filepath.Join(m.etcRoot, "apt", "sources.list.d", "*.list"),
		filepath.Join(m.etcRoot, "apt", "sources.list.d", "*.sources"),
	} {
		matches, _ := filepath.Glob(pattern)
		slices.Sort(matches)
		for _, path := range matches {
			if regularFile(path) {
				files = append(files, path)
			}
		}
	}
	return files
}

func (m *Manager) swapActive(path string) bool {
	for _, line := range strings.Split(m.procValue("swaps"), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 1 && fields[0] == path {
			return true
		}
	}
	return false
}

func (m *Manager) procValue(relative string) string {
	data, _ := os.ReadFile(filepath.Join(m.procRoot, filepath.FromSlash(relative)))
	return string(data)
}

func validHostname(value string) bool {
	if len(value) < 1 || len(value) > 253 || strings.HasSuffix(value, ".") {
		return false
	}
	for _, label := range strings.Split(value, ".") {
		if len(label) < 1 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, char := range label {
			if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '-' {
				return false
			}
		}
	}
	return true
}

func updateHosts(data []byte, oldHostname, newHostname string) []byte {
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	found := false
	for index, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if fields[0] != "127.0.0.1" && fields[0] != "127.0.1.1" && fields[0] != "::1" {
			continue
		}
		aliases := make([]string, 0, len(fields))
		if fields[0] == "127.0.1.1" {
			aliases = append(aliases, newHostname)
			found = true
		}
		for _, alias := range fields[1:] {
			if alias != oldHostname && alias != newHostname {
				aliases = append(aliases, alias)
			}
		}
		lines[index] = fields[0]
		if len(aliases) > 0 {
			lines[index] += "\t" + strings.Join(aliases, " ")
		}
	}
	if !found {
		lines = append(lines, "127.0.1.1\t"+newHostname)
	}
	return []byte(strings.TrimRight(strings.Join(lines, "\n"), "\n") + "\n")
}

func updateIPPreference(data []byte, preference string) []byte {
	var kept []string
	for _, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == kpanelIPPreferenceMarker ||
			regexp.MustCompile(`^precedence\s+::ffff:0:0/96\s+100(?:\s*#.*)?$`).MatchString(trimmed) {
			continue
		}
		kept = append(kept, line)
	}
	result := strings.TrimRight(strings.Join(kept, "\n"), "\n")
	if preference == "ipv4" {
		if result != "" {
			result += "\n\n"
		}
		result += kpanelIPPreferenceMarker + "\nprecedence ::ffff:0:0/96  100"
	}
	if result == "" {
		return nil
	}
	return []byte(result + "\n")
}

func updateFstabSwap(data []byte, path string, enabled bool) []byte {
	var lines []string
	for _, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		fields := strings.Fields(trimmed)
		if trimmed == kpanelSwapMarker || (len(fields) >= 3 && fields[0] == path && fields[2] == "swap") {
			continue
		}
		lines = append(lines, line)
	}
	result := strings.TrimRight(strings.Join(lines, "\n"), "\n")
	if enabled {
		if result != "" {
			result += "\n"
		}
		result += kpanelSwapMarker + "\n" + path + " none swap sw 0 0"
	}
	if result == "" {
		return nil
	}
	return []byte(result + "\n")
}

var aptURLPattern = regexp.MustCompile(`https?://[A-Za-z0-9.-]+/(?:debian-security|debian|ubuntu-ports|ubuntu)(?:/[^ \t\r\n]*)?`)

func rewriteAPTSource(data []byte, osID, preset string) []byte {
	return aptURLPattern.ReplaceAllFunc(data, func(raw []byte) []byte {
		value := string(raw)
		parsedHostStart := strings.Index(value, "://")
		if parsedHostStart < 0 {
			return raw
		}
		hostStart := parsedHostStart + 3
		slash := strings.Index(value[hostStart:], "/")
		if slash < 0 {
			return raw
		}
		slash += hostStart
		host := strings.ToLower(value[hostStart:slash])
		path := value[slash:]
		allowedHosts := map[string]bool{
			"deb.debian.org": true, "security.debian.org": true, "ftp.debian.org": true,
			"archive.ubuntu.com": true, "security.ubuntu.com": true, "ports.ubuntu.com": true,
			"mirrors.aliyun.com": true, "mirrors.cloud.tencent.com": true,
		}
		if !allowedHosts[host] {
			return raw
		}
		if osID == "debian" && !strings.HasPrefix(path, "/debian") {
			return raw
		}
		if osID == "ubuntu" && !strings.HasPrefix(path, "/ubuntu") {
			return raw
		}
		targetHost := "mirrors.aliyun.com"
		if preset == "official" {
			if osID == "debian" {
				targetHost = "deb.debian.org"
			} else if strings.HasPrefix(path, "/ubuntu-ports") {
				targetHost = "ports.ubuntu.com"
			} else {
				targetHost = "archive.ubuntu.com"
			}
		}
		return []byte("https://" + targetHost + path)
	})
}

func osReleaseValue(path, key string) string {
	for _, line := range strings.Split(readLimited(path), "\n") {
		name, value, ok := strings.Cut(line, "=")
		if ok && name == key {
			return strings.Trim(strings.TrimSpace(value), `"'`)
		}
	}
	return ""
}

func snapshotFile(path string) ([]byte, bool, os.FileMode, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, 0, nil
	}
	if err != nil {
		return nil, false, 0, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, false, 0, fmt.Errorf("%w: %s is not a regular file", ErrConflict, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false, 0, err
	}
	if len(data) > 8<<20 {
		return nil, false, 0, fmt.Errorf("%w: configuration file %s exceeds 8 MiB", ErrConflict, path)
	}
	return data, true, info.Mode().Perm(), nil
}

func restoreFile(path string, data []byte, existed bool, mode os.FileMode) error {
	if !existed {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	return writeAtomic(path, data, fileModeOr(mode, 0o600))
}

func writeAtomic(path string, data []byte, mode os.FileMode) error {
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: refusing to replace non-regular file %s", ErrConflict, path)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(parent, "."+filepath.Base(path)+".kpanel-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(mode.Perm()); err != nil {
		temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return renameReplace(tempPath, path)
}

func readLimited(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	data, _ := io.ReadAll(io.LimitReader(file, 8<<20))
	return string(data)
}

func regularFile(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0
}

func regularFileFollow(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func fileModeOr(value, fallback os.FileMode) os.FileMode {
	if value == 0 {
		return fallback
	}
	return value.Perm()
}

func sizeWithinMiB(size int64, wanted int) bool {
	return size >= int64(wanted)*1024*1024 && size < int64(wanted+1)*1024*1024
}

func safeName(value string) string {
	value = strings.ToLower(value)
	var out strings.Builder
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' || char == '_' {
			out.WriteRune(char)
		} else {
			out.WriteByte('_')
		}
	}
	if out.Len() == 0 {
		return "item"
	}
	return out.String()
}
