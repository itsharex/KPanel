package systemmanage

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

const f2bStatusPrefix = "KPANEL_F2B_STATUS "

func (m *Manager) SSHDefenseStatus(ctx context.Context) contract.SSHDefenseStatus {
	status := contract.SSHDefenseStatus{Jail: "sshd"}
	script, err := m.f2bScript()
	if err != nil {
		status.Message = "请更新本机 kejilion.sh 以读取 SSH 防御状态"
		return status
	}
	for _, command := range []string{"env", "bash"} {
		if _, err := m.runner.LookPath(command); err != nil {
			status.Message = "SSH 防御状态读取依赖不完整"
			return status
		}
	}
	statusContext, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	output, err := m.runner.Run(
		statusContext,
		"env",
		"KJ_F2B_NONINTERACTIVE=1",
		"LC_ALL=C.UTF-8",
		"LANG=C.UTF-8",
		"bash",
		script,
		"f2b",
		"status",
	)
	if err != nil {
		status.Message = "kejilion.sh 未能返回 SSH 防御状态"
		return status
	}
	parsed, err := parseSSHDefenseStatus(output)
	if err != nil {
		status.Message = "kejilion.sh 返回了无法识别的 SSH 防御状态"
		return status
	}
	parsed.Available = true
	switch {
	case parsed.Enabled:
		parsed.Message = "Fail2Ban SSH jail 正在防御"
	case parsed.Running:
		parsed.Message = "Fail2Ban 正在运行，但 SSH jail 未启用"
	case parsed.Installed:
		parsed.Message = "Fail2Ban 已安装但未运行"
	default:
		parsed.Message = "Fail2Ban 尚未安装"
	}
	return parsed
}

func parseSSHDefenseStatus(output []byte) (contract.SSHDefenseStatus, error) {
	for _, line := range strings.Split(string(output), "\n") {
		raw, ok := strings.CutPrefix(strings.TrimSpace(line), f2bStatusPrefix)
		if !ok {
			continue
		}
		var status contract.SSHDefenseStatus
		decoder := json.NewDecoder(strings.NewReader(raw))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&status); err != nil {
			return contract.SSHDefenseStatus{}, err
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			return contract.SSHDefenseStatus{}, errors.New("trailing SSH defense status")
		}
		if status.Jail != "sshd" && status.Jail != "alpine-sshd" {
			return contract.SSHDefenseStatus{}, errors.New("invalid SSH defense jail")
		}
		if status.Banned < 0 || status.Banned > 1_000_000 {
			return contract.SSHDefenseStatus{}, errors.New("invalid SSH defense banned count")
		}
		return status, nil
	}
	return contract.SSHDefenseStatus{}, errors.New("SSH defense status marker is missing")
}

func findKejilionF2BScript() (string, error) {
	candidates := []string{
		"/home/docker/kpanel/bin/kejilion.sh",
		"/usr/local/bin/k",
		"/usr/bin/k",
		"/root/kejilion.sh",
	}
	if path, err := exec.LookPath("k"); err == nil {
		candidates = append(candidates, path)
	}
	seen := make(map[string]bool)
	for _, candidate := range candidates {
		resolved, err := filepath.EvalSymlinks(candidate)
		if err != nil {
			continue
		}
		resolved = filepath.Clean(resolved)
		if seen[resolved] {
			continue
		}
		seen[resolved] = true
		info, err := os.Stat(resolved)
		if err != nil || !info.Mode().IsRegular() || info.Size() < 1024 || info.Size() > 4<<20 ||
			info.Mode().Perm()&0o022 != 0 || !dnsScriptOwnerTrusted(info) {
			continue
		}
		content, err := os.ReadFile(resolved)
		if err != nil {
			continue
		}
		if trustedKejilionF2BContent(content) {
			return resolved, nil
		}
	}
	return "", errors.New("a trusted kejilion.sh SSH defense command was not found")
}

func trustedKejilionF2BContent(content []byte) bool {
	value := string(content)
	return dnsScriptLicense.Match(content) &&
		strings.Contains(value, "KJ_F2B_NONINTERACTIVE") &&
		strings.Contains(value, "kpanel_protocol_active") &&
		strings.Contains(value, "kpanel_f2b_dispatch") &&
		strings.Contains(value, f2bStatusPrefix)
}
