package systemmanage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	kejilionKernelConfigMarker = "# kejilion 内核调优配置"
	kejilionAutoKernelMarker   = "# 自动网络优化配置"
	kejilionLimitsMarker       = "# kejilion-optimize"
	networkLimitsMarker        = "# network-optimize"
	kpanelBBRMarker            = "# Managed by KPanel; compatible with kejilion.sh bbr_on"
)

var kernelProfileNames = map[string]string{
	"high":     "高性能优化模式",
	"balanced": "均衡优化模式",
	"web":      "网站搭建优化模式",
	"stream":   "直播优化模式",
	"game":     "游戏服优化模式",
}

type kernelProfile struct {
	key, name, thpMode                                string
	swappiness, dirtyRatio, dirtyBackground           int
	overcommit, minFreeKiB, vfsCachePressure          int
	rmemMax, wmemMax                                  int
	tcpRmem, tcpWmem                                  string
	somaxconn, backlog, synBacklog                    int
	portRange                                         string
	schedAutogroup, numa, finTimeout                  int
	keepaliveTime, keepaliveInterval, keepaliveProbes int
	congestionControl, qdisc                          string
}

type kernelSetting struct {
	key   string
	value string
}

type kernelFileState struct {
	path    string
	data    []byte
	existed bool
	mode    os.FileMode
}

func (m *Manager) setKernelTuning(ctx context.Context, profileKey string) (bool, string, string, error) {
	if profileKey != "off" {
		if _, ok := kernelProfileNames[profileKey]; !ok {
			return false, "", "", fmt.Errorf(
				"%w: profile must be high, balanced, web, stream, game, or off",
				ErrInvalidInput,
			)
		}
	}

	manualPath := filepath.Join(m.etcRoot, "sysctl.d", "99-kejilion-optimize.conf")
	autoPath := filepath.Join(m.etcRoot, "sysctl.d", "99-network-optimize.conf")
	limitsPath := filepath.Join(m.etcRoot, "security", "limits.conf")
	modulesPath := filepath.Join(m.etcRoot, "modules-load.d", "bbr.conf")
	systemSysctlPath := filepath.Join(m.etcRoot, "sysctl.conf")
	kpanelBBRPath := filepath.Join(m.etcRoot, "sysctl.d", "99-kejilion-bbr.conf")
	thpPath := filepath.Join(m.sysRoot, "kernel", "mm", "transparent_hugepage", "enabled")
	paths := []string{
		manualPath, autoPath, limitsPath, modulesPath, systemSysctlPath, kpanelBBRPath,
	}
	states, err := snapshotKernelFiles(paths)
	if err != nil {
		return false, "", "", err
	}
	manualState := states[0]
	autoState := states[1]
	bbrState := states[5]
	if manualState.existed && !recognizedManualKernelConfig(manualState.data) {
		return false, "", "", fmt.Errorf(
			"%w: %s is not a recognized kejilion.sh or KPanel kernel profile",
			ErrConflict,
			manualPath,
		)
	}
	if autoState.existed && !bytes.Contains(autoState.data, []byte(kejilionAutoKernelMarker)) {
		return false, "", "", fmt.Errorf(
			"%w: %s is not a recognized kejilion.sh automatic tuning profile",
			ErrConflict,
			autoPath,
		)
	}
	if profileKey != "off" && bbrState.existed &&
		!bytes.Contains(bbrState.data, []byte(kpanelBBRMarker)) {
		return false, "", "", fmt.Errorf(
			"%w: %s would override the selected kernel profile",
			ErrConflict,
			kpanelBBRPath,
		)
	}

	if profileKey == "off" && !manualState.existed && !autoState.existed {
		return false, "", "Kejilion 内核优化已经停用", nil
	}

	var (
		profile   kernelProfile
		settings  []kernelSetting
		config    []byte
		memoryMiB int
	)
	if profileKey != "off" {
		var memoryErr error
		memoryMiB, memoryErr = m.totalMemoryMiB()
		if memoryErr != nil {
			return false, "", "", memoryErr
		}
		profile = m.buildKernelProfile(ctx, profileKey, memoryMiB)
		settings = m.kernelSettings(profile, memoryMiB)
		config = renderKernelConfig(profile, memoryMiB, settings, m.now())
		if !autoState.existed && !bbrState.existed &&
			sameKernelSettings(manualState.data, settings) &&
			bytes.Equal(
				bytes.TrimSpace(states[2].data),
				bytes.TrimSpace(updateKernelLimits(states[2].data, true)),
			) &&
			(profile.congestionControl != "bbr" ||
				bytes.Equal(
					bytes.TrimSpace(states[4].data),
					bytes.TrimSpace(removeSysctlSetting(
						states[4].data,
						"net.ipv4.tcp_congestion_control",
					)),
				)) &&
			(profile.congestionControl != "bbr" ||
				strings.TrimSpace(string(states[3].data)) == "tcp_bbr") {
			if !regularFile(thpPath) || selectedTHPMode(readLimited(thpPath)) == profile.thpMode {
				return false, "", "内核优化配置没有变化", nil
			}
		}
	}

	backup, err := m.createBackup("kernel-"+profileKey, paths...)
	if err != nil {
		return false, "", "", err
	}
	oldTHPMode := selectedTHPMode(readLimited(thpPath))
	rollback := func(cause error) error {
		var rollbackErrors []error
		for index := len(states) - 1; index >= 0; index-- {
			state := states[index]
			if restoreErr := restoreFile(state.path, state.data, state.existed, state.mode); restoreErr != nil {
				rollbackErrors = append(rollbackErrors, restoreErr)
			}
		}
		rollbackContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if _, reloadErr := m.runner.Run(rollbackContext, "sysctl", "--system"); reloadErr != nil {
			rollbackErrors = append(rollbackErrors, reloadErr)
		}
		if oldTHPMode != "" {
			if thpErr := writeTHPMode(thpPath, oldTHPMode); thpErr != nil {
				rollbackErrors = append(rollbackErrors, thpErr)
			}
		}
		if len(rollbackErrors) > 0 {
			return fmt.Errorf("%w: %v; rollback errors: %v", ErrNeedsAttention, cause, errors.Join(rollbackErrors...))
		}
		return fmt.Errorf("%w: %v", ErrRolledBack, cause)
	}

	limits := updateKernelLimits(states[2].data, profileKey != "off")
	systemSysctl := states[4].data
	if profileKey == "off" || (profileKey != "off" && profile.congestionControl == "bbr") {
		systemSysctl = removeSysctlSetting(systemSysctl, "net.ipv4.tcp_congestion_control")
	}
	changes := map[string][]byte{
		limitsPath:       limits,
		systemSysctlPath: systemSysctl,
	}
	if profileKey == "off" {
		changes[manualPath] = nil
		changes[autoPath] = nil
		changes[modulesPath] = nil
	} else {
		changes[manualPath] = config
		changes[autoPath] = nil
		changes[kpanelBBRPath] = nil
		if profile.congestionControl == "bbr" {
			changes[modulesPath] = []byte("tcp_bbr\n")
		}
	}
	for _, path := range paths {
		target, changed := changes[path]
		if !changed {
			continue
		}
		state := kernelStateForPath(states, path)
		if len(target) == 0 && !state.existed {
			continue
		}
		if len(target) > 0 && state.existed && bytes.Equal(state.data, target) {
			continue
		}
		if writeErr := writeOrRemoveKernelFile(path, target, fileModeOr(state.mode, 0o644)); writeErr != nil {
			return false, backup, "", rollback(writeErr)
		}
	}

	if profileKey == "off" {
		if _, reloadErr := m.runner.Run(ctx, "sysctl", "--system"); reloadErr != nil {
			return false, backup, "", rollback(fmt.Errorf("reload system sysctl defaults: %w", reloadErr))
		}
		thpMessage := ""
		if regularFile(thpPath) {
			if thpErr := writeTHPMode(thpPath, "always"); thpErr != nil {
				thpMessage = "；透明大页未能恢复，请人工检查"
			}
		}
		return true, backup, "Kejilion 内核优化已还原，系统参数已重新加载" + thpMessage, nil
	}

	applied, skipped := m.applyKernelSettings(ctx, settings)
	if applied == 0 {
		return false, backup, "", rollback(errors.New("all kernel parameters were rejected"))
	}
	if regularFile(thpPath) {
		if thpErr := writeTHPMode(thpPath, profile.thpMode); thpErr != nil {
			skipped++
		}
	}
	message := fmt.Sprintf(
		"%s已生效：内存 %d MiB，拥塞算法 %s，队列 %s，应用 %d 项",
		profile.name,
		memoryMiB,
		profile.congestionControl,
		profile.qdisc,
		applied,
	)
	if skipped > 0 {
		message += fmt.Sprintf("，跳过 %d 项当前内核不支持的设置", skipped)
	}
	return true, backup, message, nil
}

func (m *Manager) buildKernelProfile(ctx context.Context, key string, memoryMiB int) kernelProfile {
	profile := kernelProfile{
		key: key, name: kernelProfileNames[key],
		keepaliveProbes: 5,
	}
	switch key {
	case "high", "stream", "game":
		profile.swappiness = 10
		profile.dirtyRatio = 15
		profile.dirtyBackground = 5
		profile.overcommit = 1
		profile.vfsCachePressure = 50
		profile.rmemMax = 67108864
		profile.wmemMax = 67108864
		profile.tcpRmem = "4096 262144 67108864"
		profile.tcpWmem = "4096 262144 67108864"
		profile.somaxconn = 8192
		profile.backlog = 250000
		profile.synBacklog = 8192
		profile.portRange = "1024 65535"
		profile.schedAutogroup = 0
		profile.thpMode = "never"
		profile.numa = 0
		profile.finTimeout = 10
		profile.keepaliveTime = 300
		profile.keepaliveInterval = 30
	case "web":
		profile.swappiness = 10
		profile.dirtyRatio = 20
		profile.dirtyBackground = 10
		profile.overcommit = 1
		profile.vfsCachePressure = 50
		profile.rmemMax = 33554432
		profile.wmemMax = 33554432
		profile.tcpRmem = "4096 131072 33554432"
		profile.tcpWmem = "4096 131072 33554432"
		profile.somaxconn = 16384
		profile.backlog = 10000
		profile.synBacklog = 16384
		profile.portRange = "1024 65535"
		profile.schedAutogroup = 0
		profile.thpMode = "never"
		profile.numa = 0
		profile.finTimeout = 15
		profile.keepaliveTime = 600
		profile.keepaliveInterval = 60
	case "balanced":
		profile.swappiness = 30
		profile.dirtyRatio = 20
		profile.dirtyBackground = 10
		profile.overcommit = 0
		profile.vfsCachePressure = 75
		profile.rmemMax = 16777216
		profile.wmemMax = 16777216
		profile.tcpRmem = "4096 87380 16777216"
		profile.tcpWmem = "4096 65536 16777216"
		profile.somaxconn = 4096
		profile.backlog = 5000
		profile.synBacklog = 4096
		profile.portRange = "1024 49151"
		profile.schedAutogroup = 1
		profile.thpMode = "always"
		profile.numa = 1
		profile.finTimeout = 30
		profile.keepaliveTime = 600
		profile.keepaliveInterval = 60
	}

	switch {
	case memoryMiB >= 16384:
		profile.minFreeKiB = 131072
		if key != "balanced" {
			profile.swappiness = 5
		}
	case memoryMiB >= 4096:
		profile.minFreeKiB = 65536
	case memoryMiB >= 1024:
		profile.minFreeKiB = 32768
		if key != "balanced" {
			profile.rmemMax = 16777216
			profile.wmemMax = 16777216
			profile.tcpRmem = "4096 87380 16777216"
			profile.tcpWmem = "4096 65536 16777216"
		}
	default:
		profile.minFreeKiB = 16384
		profile.swappiness = 30
		profile.overcommit = 0
		profile.rmemMax = 4194304
		profile.wmemMax = 4194304
		profile.tcpRmem = "4096 32768 4194304"
		profile.tcpWmem = "4096 32768 4194304"
		profile.somaxconn = 1024
		profile.backlog = 1000
	}

	if _, lookErr := m.runner.LookPath("modprobe"); lookErr == nil {
		_, _ = m.runner.Run(ctx, "modprobe", "tcp_bbr")
	}
	available := strings.Fields(m.procValue("sys/net/ipv4/tcp_available_congestion_control"))
	if slices.Contains(available, "bbr") {
		profile.congestionControl = "bbr"
		profile.qdisc = "fq"
	} else {
		profile.congestionControl = "cubic"
		profile.qdisc = "fq_codel"
	}
	return profile
}

func (m *Manager) kernelSettings(profile kernelProfile, memoryMiB int) []kernelSetting {
	middleRmem := strings.Fields(profile.tcpRmem)[1]
	middleWmem := strings.Fields(profile.tcpWmem)[1]
	settings := []kernelSetting{
		{"net.core.default_qdisc", profile.qdisc},
		{"net.ipv4.tcp_congestion_control", profile.congestionControl},
		{"net.core.rmem_max", strconv.Itoa(profile.rmemMax)},
		{"net.core.wmem_max", strconv.Itoa(profile.wmemMax)},
		{"net.core.rmem_default", middleRmem},
		{"net.core.wmem_default", middleWmem},
		{"net.ipv4.tcp_rmem", profile.tcpRmem},
		{"net.ipv4.tcp_wmem", profile.tcpWmem},
		{"net.core.somaxconn", strconv.Itoa(profile.somaxconn)},
		{"net.core.netdev_max_backlog", strconv.Itoa(profile.backlog)},
		{"net.ipv4.tcp_max_syn_backlog", strconv.Itoa(profile.synBacklog)},
		{"net.ipv4.tcp_fastopen", "3"},
		{"net.ipv4.tcp_tw_reuse", "1"},
		{"net.ipv4.tcp_fin_timeout", strconv.Itoa(profile.finTimeout)},
		{"net.ipv4.tcp_keepalive_time", strconv.Itoa(profile.keepaliveTime)},
		{"net.ipv4.tcp_keepalive_intvl", strconv.Itoa(profile.keepaliveInterval)},
		{"net.ipv4.tcp_keepalive_probes", strconv.Itoa(profile.keepaliveProbes)},
		{"net.ipv4.tcp_max_tw_buckets", "65536"},
		{"net.ipv4.tcp_syncookies", "1"},
		{"net.ipv4.tcp_synack_retries", "2"},
		{"net.ipv4.tcp_syn_retries", "3"},
		{"net.ipv4.tcp_mtu_probing", "1"},
		{"net.ipv4.tcp_sack", "1"},
		{"net.ipv4.tcp_timestamps", "1"},
		{"net.ipv4.tcp_window_scaling", "1"},
		{"net.ipv4.ip_local_port_range", profile.portRange},
		{
			"net.ipv4.tcp_mem",
			fmt.Sprintf(
				"%d %d %d",
				int64(memoryMiB)*128,
				int64(memoryMiB)*256,
				int64(memoryMiB)*512,
			),
		},
		{"net.ipv4.tcp_max_orphans", "32768"},
		{"vm.swappiness", strconv.Itoa(profile.swappiness)},
		{"vm.dirty_ratio", strconv.Itoa(profile.dirtyRatio)},
		{"vm.dirty_background_ratio", strconv.Itoa(profile.dirtyBackground)},
		{"vm.overcommit_memory", strconv.Itoa(profile.overcommit)},
		{"vm.min_free_kbytes", strconv.Itoa(profile.minFreeKiB)},
		{"vm.vfs_cache_pressure", strconv.Itoa(profile.vfsCachePressure)},
		{"kernel.sched_autogroup_enabled", strconv.Itoa(profile.schedAutogroup)},
	}
	if regularFile(filepath.Join(m.procRoot, "sys", "kernel", "numa_balancing")) {
		settings = append(settings, kernelSetting{"kernel.numa_balancing", strconv.Itoa(profile.numa)})
	}
	settings = append(settings,
		kernelSetting{"net.ipv4.conf.all.rp_filter", "1"},
		kernelSetting{"net.ipv4.conf.default.rp_filter", "1"},
		kernelSetting{"net.ipv4.icmp_echo_ignore_broadcasts", "1"},
		kernelSetting{"net.ipv4.icmp_ignore_bogus_error_responses", "1"},
		kernelSetting{"net.ipv4.conf.all.accept_redirects", "0"},
		kernelSetting{"net.ipv4.conf.default.accept_redirects", "0"},
		kernelSetting{"net.ipv4.conf.all.send_redirects", "0"},
		kernelSetting{"net.ipv4.conf.default.send_redirects", "0"},
		kernelSetting{"net.ipv6.conf.all.accept_redirects", "0"},
		kernelSetting{"net.ipv6.conf.default.accept_redirects", "0"},
		kernelSetting{"fs.file-max", "1048576"},
		kernelSetting{"fs.nr_open", "1048576"},
	)
	if regularFile(filepath.Join(m.procRoot, "sys", "net", "netfilter", "nf_conntrack_max")) {
		settings = append(settings,
			kernelSetting{"net.netfilter.nf_conntrack_max", strconv.Itoa(profile.somaxconn * 32)},
			kernelSetting{"net.netfilter.nf_conntrack_tcp_timeout_established", "7200"},
			kernelSetting{"net.netfilter.nf_conntrack_tcp_timeout_time_wait", "30"},
			kernelSetting{"net.netfilter.nf_conntrack_tcp_timeout_close_wait", "15"},
			kernelSetting{"net.netfilter.nf_conntrack_tcp_timeout_fin_wait", "15"},
		)
	}
	if profile.key == "stream" || profile.key == "game" {
		settings = append(settings,
			kernelSetting{"net.ipv4.udp_rmem_min", "16384"},
			kernelSetting{"net.ipv4.udp_wmem_min", "16384"},
			kernelSetting{"net.ipv4.tcp_notsent_lowat", "16384"},
		)
	}
	if profile.key == "game" {
		settings = append(settings, kernelSetting{"net.ipv4.tcp_slow_start_after_idle", "0"})
	}
	return settings
}

func renderKernelConfig(
	profile kernelProfile,
	memoryMiB int,
	settings []kernelSetting,
	generatedAt time.Time,
) []byte {
	groups := []struct {
		title string
		keys  []string
	}{
		{"TCP 拥塞控制", []string{"net.core.default_qdisc", "net.ipv4.tcp_congestion_control"}},
		{"TCP 缓冲区", []string{
			"net.core.rmem_max", "net.core.wmem_max", "net.core.rmem_default",
			"net.core.wmem_default", "net.ipv4.tcp_rmem", "net.ipv4.tcp_wmem",
		}},
		{"连接队列", []string{"net.core.somaxconn", "net.core.netdev_max_backlog", "net.ipv4.tcp_max_syn_backlog"}},
		{"TCP 连接优化", []string{
			"net.ipv4.tcp_fastopen", "net.ipv4.tcp_tw_reuse", "net.ipv4.tcp_fin_timeout",
			"net.ipv4.tcp_keepalive_time", "net.ipv4.tcp_keepalive_intvl",
			"net.ipv4.tcp_keepalive_probes", "net.ipv4.tcp_max_tw_buckets",
			"net.ipv4.tcp_syncookies", "net.ipv4.tcp_synack_retries", "net.ipv4.tcp_syn_retries",
			"net.ipv4.tcp_mtu_probing", "net.ipv4.tcp_sack", "net.ipv4.tcp_timestamps",
			"net.ipv4.tcp_window_scaling",
		}},
		{"端口与内存", []string{"net.ipv4.ip_local_port_range", "net.ipv4.tcp_mem", "net.ipv4.tcp_max_orphans"}},
		{"虚拟内存", []string{
			"vm.swappiness", "vm.dirty_ratio", "vm.dirty_background_ratio",
			"vm.overcommit_memory", "vm.min_free_kbytes", "vm.vfs_cache_pressure",
		}},
		{"CPU/内核调度", []string{"kernel.sched_autogroup_enabled", "kernel.numa_balancing"}},
		{"安全防护", []string{
			"net.ipv4.conf.all.rp_filter", "net.ipv4.conf.default.rp_filter",
			"net.ipv4.icmp_echo_ignore_broadcasts", "net.ipv4.icmp_ignore_bogus_error_responses",
			"net.ipv4.conf.all.accept_redirects", "net.ipv4.conf.default.accept_redirects",
			"net.ipv4.conf.all.send_redirects", "net.ipv4.conf.default.send_redirects",
			"net.ipv6.conf.all.accept_redirects", "net.ipv6.conf.default.accept_redirects",
		}},
		{"文件描述符", []string{"fs.file-max", "fs.nr_open"}},
		{"连接跟踪", []string{
			"net.netfilter.nf_conntrack_max", "net.netfilter.nf_conntrack_tcp_timeout_established",
			"net.netfilter.nf_conntrack_tcp_timeout_time_wait",
			"net.netfilter.nf_conntrack_tcp_timeout_close_wait",
			"net.netfilter.nf_conntrack_tcp_timeout_fin_wait",
		}},
		{"直播/游戏附加优化", []string{
			"net.ipv4.udp_rmem_min", "net.ipv4.udp_wmem_min", "net.ipv4.tcp_notsent_lowat",
			"net.ipv4.tcp_slow_start_after_idle",
		}},
	}
	byKey := make(map[string]string, len(settings))
	for _, setting := range settings {
		byKey[setting.key] = setting.value
	}
	var output strings.Builder
	fmt.Fprintf(&output, "%s\n%s\n", kejilionKernelConfigMarker, kpanelKernelMarker)
	fmt.Fprintf(&output, "# 模式: %s | 场景: %s\n", profile.name, profile.key)
	fmt.Fprintf(
		&output,
		"# 内存: %dMB | 生成时间: %s\n",
		memoryMiB,
		generatedAt.Format("2006-01-02 15:04:05"),
	)
	for _, group := range groups {
		var lines []string
		for _, key := range group.keys {
			if value, exists := byKey[key]; exists {
				lines = append(lines, key+" = "+value)
			}
		}
		if len(lines) == 0 {
			continue
		}
		fmt.Fprintf(&output, "\n# ── %s ──\n%s\n", group.title, strings.Join(lines, "\n"))
	}
	return []byte(output.String())
}

func (m *Manager) applyKernelSettings(ctx context.Context, settings []kernelSetting) (int, int) {
	applied := 0
	skipped := 0
	for _, setting := range settings {
		if _, err := m.runner.Run(ctx, "sysctl", "-w", setting.key+"="+setting.value); err != nil {
			skipped++
			continue
		}
		applied++
	}
	return applied, skipped
}

func (m *Manager) totalMemoryMiB() (int, error) {
	for _, line := range strings.Split(m.procValue("meminfo"), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "MemTotal:" {
			continue
		}
		memoryKiB, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil || memoryKiB < 1024 {
			break
		}
		memoryMiB := memoryKiB / 1024
		maxInt := uint64(^uint(0) >> 1)
		if memoryMiB > maxInt {
			break
		}
		return int(memoryMiB), nil
	}
	return 0, fmt.Errorf("%w: cannot determine MemTotal from /proc/meminfo", ErrUnsupported)
}

func recognizedManualKernelConfig(data []byte) bool {
	if bytes.Contains(data, []byte(kpanelKernelMarker)) {
		return true
	}
	if !bytes.Contains(data, []byte(kejilionKernelConfigMarker)) {
		return false
	}
	for key, name := range kernelProfileNames {
		if bytes.Contains(data, []byte("# 模式: "+name+" | 场景: "+key)) {
			return true
		}
	}
	return false
}

func sameKernelSettings(data []byte, settings []kernelSetting) bool {
	actual := make(map[string]string)
	for _, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			return false
		}
		actual[strings.TrimSpace(key)] = strings.Join(strings.Fields(value), " ")
	}
	if len(actual) != len(settings) {
		return false
	}
	for _, setting := range settings {
		if actual[setting.key] != strings.Join(strings.Fields(setting.value), " ") {
			return false
		}
	}
	return true
}

func snapshotKernelFiles(paths []string) ([]kernelFileState, error) {
	states := make([]kernelFileState, 0, len(paths))
	for _, path := range paths {
		data, existed, mode, err := snapshotFile(path)
		if err != nil {
			return nil, err
		}
		states = append(states, kernelFileState{
			path: path, data: data, existed: existed, mode: mode,
		})
	}
	return states, nil
}

func kernelStateForPath(states []kernelFileState, path string) kernelFileState {
	for _, state := range states {
		if state.path == path {
			return state
		}
	}
	return kernelFileState{path: path}
}

func writeOrRemoveKernelFile(path string, data []byte, mode os.FileMode) error {
	if len(data) == 0 {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	return writeAtomic(path, data, mode)
}

func updateKernelLimits(data []byte, enabled bool) []byte {
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	managedLines := map[string]bool{
		"* soft nofile 1048576":    true,
		"* hard nofile 1048576":    true,
		"root soft nofile 1048576": true,
		"root hard nofile 1048576": true,
	}
	var kept []string
	skipManaged := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == kejilionLimitsMarker || trimmed == networkLimitsMarker {
			skipManaged = true
			continue
		}
		if skipManaged && managedLines[trimmed] {
			continue
		}
		skipManaged = false
		kept = append(kept, line)
	}
	result := strings.TrimSpace(strings.Join(kept, "\n"))
	if enabled {
		if result != "" {
			result += "\n\n"
		}
		result += kejilionLimitsMarker + "\n" +
			"* soft nofile 1048576\n" +
			"* hard nofile 1048576\n" +
			"root soft nofile 1048576\n" +
			"root hard nofile 1048576"
	}
	if result == "" {
		return nil
	}
	return []byte(result + "\n")
}

func removeSysctlSetting(data []byte, key string) []byte {
	pattern := regexp.MustCompile(`^\s*` + regexp.QuoteMeta(key) + `\s*=`)
	var kept []string
	for _, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		if pattern.MatchString(line) {
			continue
		}
		kept = append(kept, line)
	}
	result := strings.TrimRight(strings.Join(kept, "\n"), "\n")
	if result == "" {
		return nil
	}
	return []byte(result + "\n")
}

func selectedTHPMode(data string) string {
	match := regexp.MustCompile(`\[([a-z]+)\]`).FindStringSubmatch(data)
	if len(match) == 2 {
		return match[1]
	}
	value := strings.TrimSpace(data)
	if value == "always" || value == "madvise" || value == "never" {
		return value
	}
	return ""
}

func writeTHPMode(path, mode string) error {
	if mode != "always" && mode != "madvise" && mode != "never" {
		return fmt.Errorf("%w: invalid transparent hugepage mode", ErrInvalidInput)
	}
	return os.WriteFile(path, []byte(mode+"\n"), 0o644)
}
