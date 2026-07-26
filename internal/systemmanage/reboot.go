package systemmanage

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

const (
	rebootConfirmation = "REBOOT"
	rebootDelay        = 15 * time.Second
	rebootUnitName     = "kejilion-panel-reboot"
)

func (m *Manager) scheduleReboot(
	ctx context.Context,
	input contract.SystemActionRequest,
) (bool, string, error) {
	if !validRebootRequest(input) {
		return false, "", fmt.Errorf(
			"%w: confirmation must be exactly %s and no unrelated fields are allowed",
			ErrInvalidInput,
			rebootConfirmation,
		)
	}
	if m.readMaintenance().State == "running" {
		return false, "", fmt.Errorf(
			"%w: a system maintenance task is still running",
			ErrConflict,
		)
	}
	if m.rebootScheduled {
		return false, "", fmt.Errorf("%w: a reboot task is already scheduled", ErrConflict)
	}
	if _, err := m.runner.LookPath("systemd-run"); err != nil {
		return false, "", fmt.Errorf("%w: systemd-run is unavailable", ErrUnsupported)
	}
	systemctlPath, err := m.runner.LookPath("systemctl")
	if err != nil || !filepath.IsAbs(systemctlPath) || filepath.Base(systemctlPath) != "systemctl" {
		return false, "", fmt.Errorf("%w: systemctl is unavailable", ErrUnsupported)
	}
	arguments := []string{
		"--unit=" + rebootUnitName,
		"--collect",
		"--no-block",
		"--on-active=" + rebootDelay.String(),
		"--timer-property=AccuracySec=1s",
		"--property=Type=oneshot",
		"--property=TimeoutStartSec=2min",
		"--property=User=root",
		"--property=UMask=0077",
		"--property=NoNewPrivileges=yes",
		"--property=PrivateTmp=yes",
		"--property=ProtectHome=yes",
		"--property=ProtectSystem=strict",
		"--property=RestrictAddressFamilies=AF_UNIX",
		"--property=SyslogIdentifier=kpanel-reboot",
		"--",
		filepath.Clean(systemctlPath),
		"--no-wall",
		"reboot",
	}
	if _, err := m.runner.Run(ctx, "systemd-run", arguments...); err != nil {
		return false, "", fmt.Errorf("%w: schedule reboot: %v", ErrUnsupported, err)
	}
	m.rebootScheduled = true
	return true, fmt.Sprintf(
		"重启任务已安全排队，服务器将在约 %d 秒后离线；正常情况下 KPanel 会随系统启动恢复",
		int(rebootDelay/time.Second),
	), nil
}

func validRebootRequest(input contract.SystemActionRequest) bool {
	return input.Action == "reboot" &&
		input.Confirmation == rebootConfirmation &&
		input.Hostname == "" &&
		input.Port == 0 &&
		len(input.Servers) == 0 &&
		input.Timezone == "" &&
		input.SwapSizeMiB == 0 &&
		input.MirrorPreset == "" &&
		input.Preference == "" &&
		input.Profile == "" &&
		input.MaintenancePolicy == "" &&
		input.Enabled == nil
}
