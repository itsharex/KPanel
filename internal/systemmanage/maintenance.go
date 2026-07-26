package systemmanage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

const maintenanceUnitPrefix = "kejilion-panel-maintenance-"

type maintenanceStep struct {
	stage     string
	progress  int
	command   string
	arguments []string
}

func (m *Manager) MaintenanceStatus() contract.SystemMaintenanceSummary {
	m.mu.Lock()
	defer m.mu.Unlock()
	status := m.readMaintenance()
	if status.State == "running" && status.StartedAt != nil && m.now().Sub(*status.StartedAt) > time.Hour {
		finishedAt := m.now().UTC()
		status.State = "failed"
		status.Stage = "interrupted"
		status.Progress = 100
		status.Message = "维护任务超过安全时限，需检查软件包管理器状态"
		status.FinishedAt = &finishedAt
		_ = m.writeMaintenance(status)
	}
	return status
}

func (m *Manager) startMaintenance(
	ctx context.Context,
	action string,
	policy string,
) (bool, string, error) {
	mode := ""
	switch action {
	case "update":
		if policy != "full" {
			return false, "", fmt.Errorf("%w: update policy must be full", ErrInvalidInput)
		}
		mode = "update"
	case "cleanup":
		if policy != "cache" && policy != "standard" {
			return false, "", fmt.Errorf("%w: cleanup policy must be cache or standard", ErrInvalidInput)
		}
		mode = "cleanup-" + policy
	default:
		return false, "", fmt.Errorf("%w: unknown maintenance action", ErrInvalidInput)
	}

	current := m.readMaintenance()
	if current.State == "running" {
		return false, "", fmt.Errorf("%w: another maintenance task is already running", ErrConflict)
	}
	if m.executable == "" || m.executable == "." {
		return false, "", fmt.Errorf("%w: Agent executable path is unavailable", ErrUnsupported)
	}
	if _, _, _, err := m.maintenanceSteps(mode); err != nil {
		return false, "", err
	}

	startedAt := m.now().UTC()
	status := contract.SystemMaintenanceSummary{
		ID:    idForMaintenance(startedAt),
		State: "running", Action: action, Policy: policy,
		Stage: "queued", Progress: 0, Message: "任务已进入 systemd 执行队列",
		StartedAt: &startedAt,
	}
	if err := m.writeMaintenance(status); err != nil {
		return false, "", fmt.Errorf("%w: persist maintenance state: %v", ErrUnsupported, err)
	}

	arguments := []string{
		"--unit=" + maintenanceUnitPrefix + status.ID,
		"--collect",
		"--no-block",
		"--property=Type=oneshot",
		"--property=TimeoutStartSec=45min",
		"--property=TimeoutStopSec=5min",
		"--property=User=root",
		"--property=UMask=0027",
		"--property=PrivateTmp=yes",
		"--property=ProtectHome=read-only",
		"--property=NoNewPrivileges=no",
		"--property=RestrictAddressFamilies=AF_UNIX AF_INET AF_INET6",
		"--property=Nice=10",
		"--property=CPUWeight=20",
		"--property=IOWeight=20",
		"--property=SyslogIdentifier=kpanel-maintenance",
		"--",
		m.executable,
		"maintenance-run",
		"--state-dir",
		m.stateDir,
		mode,
	}
	if _, err := m.runner.Run(ctx, "systemd-run", arguments...); err != nil {
		finishedAt := m.now().UTC()
		status.State = "failed"
		status.Stage = "launch_failed"
		status.Progress = 100
		status.Message = maintenanceErrorMessage(err)
		status.FinishedAt = &finishedAt
		_ = m.writeMaintenance(status)
		return false, "", fmt.Errorf("%w: start maintenance task: %v", ErrUnsupported, err)
	}
	return true, "系统维护任务已提交，页面将自动刷新进度", nil
}

// RunMaintenance is only called by the Agent's root-only maintenance-run
// subcommand. The mode selects one fixed command list; it never accepts shell
// fragments, package names, paths, or arbitrary arguments from the Web API.
func (m *Manager) RunMaintenance(ctx context.Context, mode string) error {
	action, policy, steps, err := m.maintenanceSteps(mode)
	if err != nil {
		return err
	}
	status := m.readMaintenance()
	if status.ID == "" || status.Action != action || status.Policy != policy {
		startedAt := m.now().UTC()
		status = contract.SystemMaintenanceSummary{
			ID:    idForMaintenance(startedAt),
			State: "running", Action: action, Policy: policy,
			StartedAt: &startedAt,
		}
	}
	status.State = "running"
	status.Stage = "starting"
	status.Progress = 1
	status.Message = "正在准备系统维护任务"
	status.FinishedAt = nil
	if err := m.writeMaintenance(status); err != nil {
		return fmt.Errorf("persist maintenance start: %w", err)
	}

	for _, step := range steps {
		status.Stage = step.stage
		status.Progress = step.progress
		status.Message = maintenanceStageMessage(step.stage)
		if err := m.writeMaintenance(status); err != nil {
			return fmt.Errorf("persist maintenance progress: %w", err)
		}
		if _, err := m.runner.Run(ctx, step.command, step.arguments...); err != nil {
			finishedAt := m.now().UTC()
			status.State = "failed"
			status.Progress = 100
			status.Message = maintenanceErrorMessage(err)
			status.FinishedAt = &finishedAt
			_ = m.writeMaintenance(status)
			return fmt.Errorf("%s: %w", step.stage, err)
		}
	}

	finishedAt := m.now().UTC()
	status.State = "succeeded"
	status.Stage = "completed"
	status.Progress = 100
	status.Message = maintenanceSuccessMessage(action, policy)
	status.FinishedAt = &finishedAt
	status.RebootRequired = regularFile(filepath.Join(m.runRoot, "reboot-required"))
	if err := m.writeMaintenance(status); err != nil {
		return fmt.Errorf("persist maintenance result: %w", err)
	}
	return nil
}

func (m *Manager) maintenanceSteps(
	mode string,
) (string, string, []maintenanceStep, error) {
	support := m.detectPackageManager()
	if !support.available() {
		reason := support.reason
		if reason == "" {
			reason = "当前发行版不支持安全系统维护"
		}
		return "", "", nil, fmt.Errorf("%w: %s", ErrUnsupported, reason)
	}
	if len(m.packageSourceFiles(support.kind)) == 0 {
		return "", "", nil, fmt.Errorf(
			"%w: %s 软件源配置不可用",
			ErrUnsupported,
			support.displayName(),
		)
	}

	aptOptions := []string{"-o", "Dpkg::Lock::Timeout=120"}
	var action, policy string
	var steps []maintenanceStep
	switch support.kind {
	case packageManagerAPT:
		action, policy, steps = aptMaintenanceSteps(mode, aptOptions)
	case packageManagerDNF:
		action, policy, steps = rpmMaintenanceSteps(mode, support.command)
	case packageManagerYUM:
		action, policy, steps = rpmMaintenanceSteps(mode, support.command)
	case packageManagerPacman:
		action, policy, steps = pacmanMaintenanceSteps(mode)
	case packageManagerZypper:
		action, policy, steps = zypperMaintenanceSteps(mode)
	}
	if action == "" {
		return "", "", nil, fmt.Errorf("%w: unknown maintenance mode", ErrInvalidInput)
	}
	return action, policy, steps, nil
}

func aptMaintenanceSteps(mode string, aptOptions []string) (string, string, []maintenanceStep) {
	switch mode {
	case "update":
		return "update", "full", []maintenanceStep{
			{stage: "dpkg_configure", progress: 10, command: "dpkg", arguments: []string{"--force-confold", "--configure", "-a"}},
			{stage: "package_index", progress: 35, command: "apt-get", arguments: append(slicesClone(aptOptions), "update")},
			{
				stage: "full_upgrade", progress: 60, command: "apt-get",
				arguments: append(slicesClone(aptOptions), "-y", "-o", "Dpkg::Options::=--force-confold", "full-upgrade"),
			},
		}
	case "cleanup-cache":
		return "cleanup", "cache", []maintenanceStep{
			{stage: "package_cache", progress: 35, command: "apt-get", arguments: append(slicesClone(aptOptions), "clean")},
			{stage: "obsolete_cache", progress: 70, command: "apt-get", arguments: append(slicesClone(aptOptions), "autoclean")},
		}
	case "cleanup-standard":
		return "cleanup", "standard", append([]maintenanceStep{
			{
				stage: "unused_packages", progress: 15, command: "apt-get",
				arguments: append(slicesClone(aptOptions), "-y", "autoremove", "--purge"),
			},
			{stage: "package_cache", progress: 40, command: "apt-get", arguments: append(slicesClone(aptOptions), "clean")},
			{stage: "obsolete_cache", progress: 55, command: "apt-get", arguments: append(slicesClone(aptOptions), "autoclean")},
		}, journalCleanupSteps()...)
	default:
		return "", "", nil
	}
}

func rpmMaintenanceSteps(mode, command string) (string, string, []maintenanceStep) {
	switch mode {
	case "update":
		return "update", "full", []maintenanceStep{
			{stage: "full_upgrade", progress: 40, command: command, arguments: []string{"-y", "update"}},
		}
	case "cleanup-cache":
		return "cleanup", "cache", []maintenanceStep{
			{stage: "package_cache", progress: 50, command: command, arguments: []string{"clean", "all"}},
		}
	case "cleanup-standard":
		return "cleanup", "standard", append([]maintenanceStep{
			{stage: "unused_packages", progress: 15, command: command, arguments: []string{"-y", "autoremove"}},
			{stage: "package_cache", progress: 40, command: command, arguments: []string{"clean", "all"}},
			{stage: "package_index", progress: 60, command: command, arguments: []string{"-y", "makecache"}},
		}, journalCleanupSteps()...)
	default:
		return "", "", nil
	}
}

func pacmanMaintenanceSteps(mode string) (string, string, []maintenanceStep) {
	switch mode {
	case "update":
		return "update", "full", []maintenanceStep{
			{stage: "full_upgrade", progress: 40, command: "pacman", arguments: []string{"-Syu", "--noconfirm"}},
		}
	case "cleanup-cache":
		return "cleanup", "cache", []maintenanceStep{
			{stage: "package_cache", progress: 50, command: "pacman", arguments: []string{"-Scc", "--noconfirm"}},
		}
	case "cleanup-standard":
		return "cleanup", "standard", append([]maintenanceStep{
			{stage: "package_cache", progress: 50, command: "pacman", arguments: []string{"-Scc", "--noconfirm"}},
		}, journalCleanupSteps()...)
	default:
		return "", "", nil
	}
}

func zypperMaintenanceSteps(mode string) (string, string, []maintenanceStep) {
	switch mode {
	case "update":
		return "update", "full", []maintenanceStep{
			{stage: "package_index", progress: 30, command: "zypper", arguments: []string{"--non-interactive", "refresh"}},
			{stage: "full_upgrade", progress: 60, command: "zypper", arguments: []string{"--non-interactive", "update"}},
		}
	case "cleanup-cache":
		return "cleanup", "cache", []maintenanceStep{
			{stage: "package_cache", progress: 50, command: "zypper", arguments: []string{"--non-interactive", "clean", "--all"}},
		}
	case "cleanup-standard":
		return "cleanup", "standard", append([]maintenanceStep{
			{stage: "package_cache", progress: 45, command: "zypper", arguments: []string{"--non-interactive", "clean", "--all"}},
			{stage: "package_index", progress: 60, command: "zypper", arguments: []string{"--non-interactive", "refresh"}},
		}, journalCleanupSteps()...)
	default:
		return "", "", nil
	}
}

func journalCleanupSteps() []maintenanceStep {
	return []maintenanceStep{
		{stage: "journal_rotate", progress: 70, command: "journalctl", arguments: []string{"--rotate"}},
		{stage: "journal_time", progress: 80, command: "journalctl", arguments: []string{"--vacuum-time=7d"}},
		{stage: "journal_size", progress: 90, command: "journalctl", arguments: []string{"--vacuum-size=500M"}},
	}
}

func (m *Manager) maintenanceStatePath() string {
	return filepath.Join(m.stateDir, "maintenance-state.json")
}

func (m *Manager) readMaintenance() contract.SystemMaintenanceSummary {
	status := contract.SystemMaintenanceSummary{State: "idle"}
	data, err := os.ReadFile(m.maintenanceStatePath())
	if err != nil || len(data) > 64<<10 || json.Unmarshal(data, &status) != nil {
		return contract.SystemMaintenanceSummary{State: "idle"}
	}
	switch status.State {
	case "idle", "running", "succeeded", "failed":
	default:
		return contract.SystemMaintenanceSummary{State: "idle"}
	}
	return status
}

func (m *Manager) writeMaintenance(status contract.SystemMaintenanceSummary) error {
	if err := os.MkdirAll(m.stateDir, 0o750); err != nil {
		return err
	}
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeAtomic(m.maintenanceStatePath(), data, 0o600)
}

func idForMaintenance(now time.Time) string {
	var nonce [4]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return now.Format("20060102T150405.000000000Z")
	}
	return now.Format("20060102T150405Z") + "-" + hex.EncodeToString(nonce[:])
}

func maintenanceStageMessage(stage string) string {
	switch stage {
	case "dpkg_configure":
		return "正在完成未结束的软件包配置"
	case "package_index":
		return "正在刷新软件包索引"
	case "full_upgrade":
		return "正在更新系统软件包"
	case "unused_packages":
		return "正在移除不再使用的依赖"
	case "package_cache":
		return "正在清理软件包缓存"
	case "obsolete_cache":
		return "正在清理过期软件包缓存"
	case "journal_rotate":
		return "正在轮转 systemd journal"
	case "journal_time":
		return "正在保留最近 7 天 journal"
	case "journal_size":
		return "正在限制 journal 最大 500 MiB"
	default:
		return "正在执行系统维护"
	}
}

func maintenanceSuccessMessage(action, policy string) string {
	if action == "update" {
		return "系统更新已完成；如内核或核心组件变化，请按提示安排重启"
	}
	if policy == "cache" {
		return "软件包缓存清理已完成"
	}
	return "系统支持的无用依赖、软件包缓存和旧 journal 已安全清理"
}

func maintenanceErrorMessage(err error) string {
	message := strings.TrimSpace(err.Error())
	if len(message) > 240 {
		message = message[:240]
	}
	if message == "" {
		return "系统维护任务失败，请检查 Agent 日志"
	}
	return "任务失败：" + message
}

func slicesClone(values []string) []string {
	return append([]string(nil), values...)
}
