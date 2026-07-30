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

const bbrv3StatusPrefix = "KPANEL_BBRV3_STATUS "
const bbrv3ResultPrefix = "KPANEL_BBRV3_RESULT "

type bbrv3Result struct {
	Action         string `json:"action"`
	Changed        bool   `json:"changed"`
	RebootRequired bool   `json:"rebootRequired"`
}

func (m *Manager) BBRv3Status(ctx context.Context) contract.BBRv3Summary {
	status := contract.BBRv3Summary{}
	script, err := m.bbrv3Script()
	if err != nil {
		status.Reason = "protocol_unavailable"
		return status
	}
	for _, command := range []string{"env", "bash"} {
		if _, err := m.runner.LookPath(command); err != nil {
			status.Reason = "missing_dependencies"
			return status
		}
	}
	statusContext, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	output, err := m.runner.Run(
		statusContext,
		"env",
		"KJ_BBRV3_NONINTERACTIVE=1",
		"LC_ALL=C.UTF-8",
		"LANG=C.UTF-8",
		"bash",
		script,
		"bbrv3",
		"status",
	)
	if err != nil {
		status.Reason = "status_failed"
		return status
	}
	parsed, err := parseBBRv3Status(output)
	if err != nil {
		status.Reason = "invalid_protocol"
		return status
	}
	parsed.Available = true
	return parsed
}

func parseBBRv3Status(output []byte) (contract.BBRv3Summary, error) {
	for _, line := range strings.Split(string(output), "\n") {
		raw, ok := strings.CutPrefix(strings.TrimSpace(line), bbrv3StatusPrefix)
		if !ok {
			continue
		}
		var status contract.BBRv3Summary
		decoder := json.NewDecoder(strings.NewReader(raw))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&status); err != nil {
			return contract.BBRv3Summary{}, err
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			return contract.BBRv3Summary{}, errors.New("trailing BBRv3 status")
		}
		if len(status.Architecture) > 32 || len(status.OS) > 32 || len(status.Codename) > 32 ||
			len(status.RunningKernel) > 128 || len(status.InstalledKernel) > 128 ||
			len(status.CongestionControl) > 32 ||
			len(status.DefaultQDisc) > 32 || len(status.Reason) > 64 {
			return contract.BBRv3Summary{}, errors.New("BBRv3 status field is too long")
		}
		switch status.Reason {
		case "", "unsupported_release", "arm64_external_installer_untrusted",
			"unsupported_distribution", "missing_dependencies":
		default:
			return contract.BBRv3Summary{}, errors.New("invalid BBRv3 reason")
		}
		return status, nil
	}
	return contract.BBRv3Summary{}, errors.New("BBRv3 status marker is missing")
}

func verifyBBRv3ActionOutput(output []byte, expectedAction string) (bool, error) {
	var result *bbrv3Result
	for _, line := range strings.Split(string(output), "\n") {
		raw, ok := strings.CutPrefix(strings.TrimSpace(line), bbrv3ResultPrefix)
		if !ok {
			continue
		}
		var parsed bbrv3Result
		decoder := json.NewDecoder(strings.NewReader(raw))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&parsed); err != nil {
			return false, err
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			return false, errors.New("trailing BBRv3 result")
		}
		result = &parsed
	}
	if result == nil {
		return false, errors.New("BBRv3 completion marker is missing")
	}
	if result.Action != expectedAction {
		return false, errors.New("BBRv3 completion action does not match the request")
	}
	status, err := parseBBRv3Status(output)
	if err != nil {
		return false, err
	}
	switch expectedAction {
	case "install", "update":
		if !status.Installed {
			return false, errors.New("BBRv3 package verification failed")
		}
	case "uninstall":
		if status.Installed {
			return false, errors.New("BBRv3 package removal verification failed")
		}
	default:
		return false, errors.New("invalid BBRv3 completion action")
	}
	if result.RebootRequired != status.RebootRequired {
		return false, errors.New("BBRv3 reboot state does not match the verified status")
	}
	return status.RebootRequired, nil
}

func findKejilionBBRv3Script() (string, error) {
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
		if trustedKejilionBBRv3Content(content) {
			return resolved, nil
		}
	}
	return "", errors.New("a trusted kejilion.sh BBRv3 command was not found")
}

func trustedKejilionBBRv3Content(content []byte) bool {
	value := string(content)
	return dnsScriptLicense.Match(content) &&
		strings.Contains(value, "KJ_BBRV3_NONINTERACTIVE") &&
		strings.Contains(value, "kpanel_protocol_active") &&
		strings.Contains(value, "kpanel_bbrv3_dispatch") &&
		strings.Contains(value, bbrv3StatusPrefix)
}
