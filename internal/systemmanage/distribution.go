package systemmanage

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
)

type packageManagerKind string

const (
	packageManagerAPT     packageManagerKind = "apt"
	packageManagerDNF     packageManagerKind = "dnf"
	packageManagerYUM     packageManagerKind = "yum"
	packageManagerPacman  packageManagerKind = "pacman"
	packageManagerZypper  packageManagerKind = "zypper"
	packageManagerUnknown packageManagerKind = ""
)

type packageManagerSupport struct {
	kind    packageManagerKind
	command string
	osID    string
	reason  string
}

func (support packageManagerSupport) available() bool {
	return support.kind != packageManagerUnknown && support.command != ""
}

func (support packageManagerSupport) displayName() string {
	switch support.kind {
	case packageManagerAPT:
		return "APT"
	case packageManagerDNF:
		return "DNF"
	case packageManagerYUM:
		return "YUM"
	case packageManagerPacman:
		return "Pacman"
	case packageManagerZypper:
		return "Zypper"
	default:
		return "未知软件包管理器"
	}
}

func (m *Manager) detectPackageManager() packageManagerSupport {
	releasePath := filepath.Join(m.etcRoot, "os-release")
	osID := strings.ToLower(osReleaseValue(releasePath, "ID"))
	idLike := strings.Fields(strings.ToLower(osReleaseValue(releasePath, "ID_LIKE")))
	matches := func(ids ...string) bool {
		return slices.Contains(ids, osID) || slices.ContainsFunc(idLike, func(value string) bool {
			return slices.Contains(ids, value)
		})
	}
	find := func(names ...string) string {
		for _, name := range names {
			if _, err := m.runner.LookPath(name); err == nil {
				return name
			}
		}
		return ""
	}

	switch {
	case matches("debian", "ubuntu"):
		if find("apt-get") == "" || find("dpkg") == "" {
			return packageManagerSupport{osID: osID, reason: "APT/dpkg 工具不完整"}
		}
		return packageManagerSupport{kind: packageManagerAPT, command: "apt-get", osID: osID}
	case matches(
		"rhel", "centos", "rocky", "almalinux", "fedora", "ol", "oraclelinux",
		"amzn", "amazon", "openeuler", "euleros",
	):
		switch command := find("dnf", "dnf5", "yum"); command {
		case "dnf", "dnf5":
			return packageManagerSupport{kind: packageManagerDNF, command: command, osID: osID}
		case "yum":
			return packageManagerSupport{kind: packageManagerYUM, command: command, osID: osID}
		default:
			return packageManagerSupport{osID: osID, reason: "DNF/YUM 工具不可用"}
		}
	case matches("arch", "manjaro"):
		if command := find("pacman"); command != "" {
			return packageManagerSupport{kind: packageManagerPacman, command: command, osID: osID}
		}
		return packageManagerSupport{osID: osID, reason: "Pacman 工具不可用"}
	case matches("suse", "opensuse", "opensuse-leap", "opensuse-tumbleweed", "sles"):
		if command := find("zypper"); command != "" {
			return packageManagerSupport{kind: packageManagerZypper, command: command, osID: osID}
		}
		return packageManagerSupport{osID: osID, reason: "Zypper 工具不可用"}
	case matches("alpine"):
		return packageManagerSupport{
			osID:   osID,
			reason: "Alpine/OpenRC 暂不支持宿主机写入；当前 Agent 需要 systemd",
		}
	case osID == "":
		return packageManagerSupport{reason: "无法识别宿主机发行版"}
	default:
		return packageManagerSupport{
			osID:   osID,
			reason: fmt.Sprintf("发行版 %s 尚未加入安全维护白名单", osID),
		}
	}
}

func (m *Manager) packageSourceFiles(kind packageManagerKind) []string {
	var patterns []string
	switch kind {
	case packageManagerAPT:
		return m.aptSourceFiles()
	case packageManagerDNF, packageManagerYUM:
		patterns = []string{filepath.Join(m.etcRoot, "yum.repos.d", "*.repo")}
	case packageManagerPacman:
		patterns = []string{filepath.Join(m.etcRoot, "pacman.d", "mirrorlist")}
	case packageManagerZypper:
		patterns = []string{filepath.Join(m.etcRoot, "zypp", "repos.d", "*.repo")}
	default:
		return nil
	}

	var files []string
	for _, pattern := range patterns {
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
