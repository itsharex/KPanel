package systemmanage

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

const networkOperationsOutputLimit = 5 << 20

var (
	networkOperationsProtocolV1Pattern = regexp.MustCompile(`(?m)^KPANEL_NETWORK_OPERATIONS_PROTOCOL_VERSION="1"\r?$`)
	networkOperationsPIDPattern        = regexp.MustCompile(`(?:^|,)pid=([0-9]+)(?:,|\))`)
	networkOperationsProcessPattern    = regexp.MustCompile(`(?:^|\s)users:\(\("([^"]{1,128})"`)
	networkOperationsRecoveryPattern   = regexp.MustCompile(`^/var/lib/kejilion-panel/system/recovery/system-resource/[0-9]{8}T[0-9]{6}Z-traffic-shutdown\.[A-Za-z0-9]{6}$`)
)

func (m *Manager) NetworkOperationsCapabilities() []contract.Capability {
	commonErr := m.networkOperationsCommonAvailability()
	portErr := m.networkOperationDependencies(commonErr, "ss", "awk", "wc", "sha256sum", "od", "tr", "mktemp")
	trafficErr := m.networkOperationDependencies(commonErr, "awk", "crontab", "grep", "sed", "sha256sum", "stat", "wc", "mktemp", "cmp")
	capability := func(id, method string, err error) contract.Capability {
		if err != nil {
			return contract.Capability{ID: id, Enabled: false, Reason: resourceCapabilityReason(err)}
		}
		return contract.Capability{ID: id, Enabled: true, Methods: []string{method}}
	}
	writeErr := trafficErr
	writeErr = m.networkOperationDependencies(writeErr, "flock", "cp", "mv", "chmod", "chown")
	if writeErr == nil && !m.enabled {
		writeErr = fmt.Errorf("%w: host system writes are disabled", ErrDisabled)
	}
	return []contract.Capability{
		capability("system.port-usage.read", "GET", portErr),
		capability("system.traffic-shutdown.read", "GET", trafficErr),
		capability("system.traffic-shutdown.write", "POST", writeErr),
	}
}

func (m *Manager) networkOperationDependencies(current error, commands ...string) error {
	if current != nil {
		return current
	}
	for _, command := range commands {
		if _, err := m.runner.LookPath(command); err != nil {
			return fmt.Errorf("%w: %s is unavailable", ErrUnsupported, command)
		}
	}
	return nil
}

func (m *Manager) networkOperationsCommonAvailability() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("%w: network operations are only available on Linux", ErrUnsupported)
	}
	if m.effectiveUID() != 0 {
		return fmt.Errorf("%w: Agent must run as root for network operations", ErrUnsupported)
	}
	for _, command := range []string{"env", "bash"} {
		if _, err := m.runner.LookPath(command); err != nil {
			return fmt.Errorf("%w: %s is unavailable", ErrUnsupported, command)
		}
	}
	_, err := m.networkOperationsScriptPath()
	return err
}

func (m *Manager) networkOperationsScriptPath() (string, error) {
	path, err := m.systemResourceScriptPath()
	if err != nil {
		return "", err
	}
	content, err := readResourceFile(path, resourceScriptMaxBytes)
	if err != nil || !trustedKejilionNetworkOperationsContent(content) {
		return "", fmt.Errorf("%w: trusted kejilion.sh network-operations protocol v1 was not found", ErrUnsupported)
	}
	return path, nil
}

func trustedKejilionNetworkOperationsContent(content []byte) bool {
	value := string(content)
	return trustedKejilionSystemResourceContent(content) &&
		networkOperationsProtocolV1Pattern.Match(content) &&
		strings.Contains(value, "KJ_NETWORK_OPERATIONS_NONINTERACTIVE") &&
		strings.Contains(value, "kpanel_network_operations_dispatch") &&
		strings.Contains(value, "KPANEL_NETWORK_OPERATIONS_STATUS") &&
		strings.Contains(value, "KPANEL_NETWORK_OPERATIONS_VERSION")
}

func (m *Manager) runNetworkOperation(ctx context.Context, limit int, arguments ...string) ([]byte, error) {
	script, err := m.networkOperationsScriptPath()
	if err != nil {
		return nil, err
	}
	commandArguments := []string{
		"KJ_NETWORK_OPERATIONS_NONINTERACTIVE=1", "bash", script,
		"kpanel", "network-operations",
	}
	commandArguments = append(commandArguments, arguments...)
	output, _, runErr := m.runResourceCommand(ctx, limit, "env", commandArguments...)
	return output, runErr
}

func (m *Manager) PortUsage(ctx context.Context) (contract.PortUsageSnapshot, error) {
	if err := m.networkOperationsCommonAvailability(); err != nil {
		return contract.PortUsageSnapshot{}, err
	}
	if err := m.networkOperationDependencies(nil, "ss", "awk", "wc", "sha256sum", "od", "tr", "mktemp"); err != nil {
		return contract.PortUsageSnapshot{}, err
	}
	readContext, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	output, runErr := m.runNetworkOperation(readContext, networkOperationsOutputLimit, "port-usage", "list")
	snapshot, parseErr := parsePortUsageOutput(output)
	if parseErr != nil {
		return contract.PortUsageSnapshot{}, fmt.Errorf("%w: invalid port-usage protocol output: %v", ErrNeedsAttention, parseErr)
	}
	if runErr != nil || snapshot.ResourceVersion == "" {
		return contract.PortUsageSnapshot{}, fmt.Errorf("%w: kejilion.sh could not read port usage: %v", ErrUnsupported, runErr)
	}
	snapshot.ObservedAt = m.now().UTC()
	return snapshot, nil
}

func parsePortUsageOutput(output []byte) (contract.PortUsageSnapshot, error) {
	var snapshot contract.PortUsageSnapshot
	seen := make(map[string]bool)
	status := ""
	for _, line := range resourceLines(output) {
		if line == "" {
			continue
		}
		if value, ok := receiptValue(line, "KPANEL_NETWORK_OPERATIONS_PORT_HEX"); ok {
			if len(snapshot.Entries) >= contract.NetworkOperationMaxPorts || len(value) == 0 || len(value) > 8192 || len(value)%2 != 0 {
				return contract.PortUsageSnapshot{}, errors.New("port entry count or size is invalid")
			}
			raw, err := hex.DecodeString(value)
			if err != nil || len(raw) > 4096 || strings.ContainsAny(string(raw), "\x00\r\n") {
				return contract.PortUsageSnapshot{}, errors.New("port entry encoding is invalid")
			}
			entry, err := parsePortUsageLine(string(raw))
			if err != nil {
				return contract.PortUsageSnapshot{}, err
			}
			snapshot.Entries = append(snapshot.Entries, entry)
			continue
		}
		matched := false
		for key, target := range map[string]*string{
			"KPANEL_NETWORK_OPERATIONS_STATUS":  &status,
			"KPANEL_NETWORK_OPERATIONS_VERSION": &snapshot.ResourceVersion,
		} {
			if value, ok := receiptValue(line, key); ok {
				if seen[key] {
					return contract.PortUsageSnapshot{}, fmt.Errorf("duplicate %s", key)
				}
				seen[key], matched, *target = true, true, value
				break
			}
		}
		if matched {
			continue
		}
		if value, ok := receiptValue(line, "KPANEL_NETWORK_OPERATIONS_TOTAL"); ok {
			if seen["total"] {
				return contract.PortUsageSnapshot{}, errors.New("duplicate total")
			}
			seen["total"] = true
			total, err := strconv.Atoi(value)
			if err != nil || total < 0 || total > 4096 {
				return contract.PortUsageSnapshot{}, errors.New("total is invalid")
			}
			snapshot.Total = total
			continue
		}
		if value, ok := receiptValue(line, "KPANEL_NETWORK_OPERATIONS_TRUNCATED"); ok {
			if seen["truncated"] || (value != "true" && value != "false") {
				return contract.PortUsageSnapshot{}, errors.New("truncated is invalid")
			}
			seen["truncated"] = true
			snapshot.Truncated = value == "true"
			continue
		}
		return contract.PortUsageSnapshot{}, errors.New("unexpected protocol output")
	}
	if status != "ok" || !resourceVersionPattern.MatchString(snapshot.ResourceVersion) || !seen["total"] || !seen["truncated"] {
		return contract.PortUsageSnapshot{}, errors.New("required port snapshot fields are missing")
	}
	if snapshot.Total < len(snapshot.Entries) || snapshot.Truncated != (snapshot.Total > len(snapshot.Entries)) ||
		(!snapshot.Truncated && snapshot.Total != len(snapshot.Entries)) {
		return contract.PortUsageSnapshot{}, errors.New("port snapshot bounds are inconsistent")
	}
	return snapshot, nil
}

func parsePortUsageLine(raw string) (contract.PortUsageEntry, error) {
	fields := strings.Fields(raw)
	if len(fields) < 6 {
		return contract.PortUsageEntry{}, errors.New("ss record has too few fields")
	}
	localAddress, localPort := splitSSEndpoint(fields[4])
	peerAddress, peerPort := splitSSEndpoint(fields[5])
	entry := contract.PortUsageEntry{
		Protocol: strings.ToLower(fields[0]), State: fields[1],
		LocalAddress: localAddress, LocalPort: localPort,
		PeerAddress: peerAddress, PeerPort: peerPort, Raw: raw,
	}
	if len(fields) > 6 {
		processDetails := strings.Join(fields[6:], " ")
		if match := networkOperationsProcessPattern.FindStringSubmatch(processDetails); len(match) == 2 {
			entry.Process = match[1]
		}
		if match := networkOperationsPIDPattern.FindStringSubmatch(processDetails); len(match) == 2 {
			entry.PID, _ = strconv.Atoi(match[1])
		}
	}
	return entry, nil
}

func splitSSEndpoint(value string) (string, string) {
	if host, port, err := net.SplitHostPort(value); err == nil {
		return strings.Trim(host, "[]"), port
	}
	index := strings.LastIndexByte(value, ':')
	if index < 0 {
		return strings.Trim(value, "[]"), ""
	}
	return strings.Trim(value[:index], "[]"), value[index+1:]
}

func (m *Manager) TrafficShutdown(ctx context.Context) (contract.TrafficShutdownSnapshot, error) {
	if err := m.networkOperationsCommonAvailability(); err != nil {
		return contract.TrafficShutdownSnapshot{}, err
	}
	if err := m.networkOperationDependencies(nil, "awk", "crontab", "grep", "sed", "sha256sum", "stat", "wc", "mktemp", "cmp"); err != nil {
		return contract.TrafficShutdownSnapshot{}, err
	}
	readContext, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	output, runErr := m.runNetworkOperation(readContext, resourceReceiptOutputLimit, "traffic-shutdown", "status")
	snapshot, parseErr := parseTrafficShutdownOutput(output)
	if parseErr != nil {
		return contract.TrafficShutdownSnapshot{}, fmt.Errorf("%w: invalid traffic-shutdown protocol output: %v", ErrNeedsAttention, parseErr)
	}
	if runErr != nil {
		return contract.TrafficShutdownSnapshot{}, fmt.Errorf("%w: kejilion.sh could not read traffic-shutdown state: %v", ErrUnsupported, runErr)
	}
	snapshot.ObservedAt = m.now().UTC()
	return snapshot, nil
}

func parseTrafficShutdownOutput(output []byte) (contract.TrafficShutdownSnapshot, error) {
	var snapshot contract.TrafficShutdownSnapshot
	values := make(map[string]string)
	for _, line := range resourceLines(output) {
		if line == "" {
			continue
		}
		matched := false
		for _, key := range []string{
			"KPANEL_NETWORK_OPERATIONS_STATUS", "KPANEL_NETWORK_OPERATIONS_VERSION",
			"KPANEL_NETWORK_OPERATIONS_ENABLED", "KPANEL_NETWORK_OPERATIONS_HEALTH",
			"KPANEL_NETWORK_OPERATIONS_RX_BYTES", "KPANEL_NETWORK_OPERATIONS_TX_BYTES",
			"KPANEL_NETWORK_OPERATIONS_RX_THRESHOLD_GIB", "KPANEL_NETWORK_OPERATIONS_TX_THRESHOLD_GIB",
			"KPANEL_NETWORK_OPERATIONS_RESET_DAY",
		} {
			if value, ok := receiptValue(line, key); ok {
				if _, exists := values[key]; exists {
					return snapshot, fmt.Errorf("duplicate %s", key)
				}
				values[key], matched = value, true
				break
			}
		}
		if !matched {
			return snapshot, errors.New("unexpected protocol output")
		}
	}
	if values["KPANEL_NETWORK_OPERATIONS_STATUS"] != "ok" || !resourceVersionPattern.MatchString(values["KPANEL_NETWORK_OPERATIONS_VERSION"]) {
		return snapshot, errors.New("status or version is invalid")
	}
	enabledValue := values["KPANEL_NETWORK_OPERATIONS_ENABLED"]
	if enabledValue != "true" && enabledValue != "false" {
		return snapshot, errors.New("enabled is invalid")
	}
	snapshot.Enabled = enabledValue == "true"
	snapshot.Health = values["KPANEL_NETWORK_OPERATIONS_HEALTH"]
	if snapshot.Health != "disabled" && snapshot.Health != "ready" && snapshot.Health != "inconsistent" {
		return snapshot, errors.New("health is invalid")
	}
	parseUint := func(key string) (uint64, error) {
		return strconv.ParseUint(values[key], 10, 64)
	}
	var err error
	if snapshot.RXBytes, err = parseUint("KPANEL_NETWORK_OPERATIONS_RX_BYTES"); err != nil {
		return snapshot, errors.New("rxBytes is invalid")
	}
	if snapshot.TXBytes, err = parseUint("KPANEL_NETWORK_OPERATIONS_TX_BYTES"); err != nil {
		return snapshot, errors.New("txBytes is invalid")
	}
	if snapshot.RXThresholdGiB, err = parseUint("KPANEL_NETWORK_OPERATIONS_RX_THRESHOLD_GIB"); err != nil {
		return snapshot, errors.New("rxThresholdGiB is invalid")
	}
	if snapshot.TXThresholdGiB, err = parseUint("KPANEL_NETWORK_OPERATIONS_TX_THRESHOLD_GIB"); err != nil {
		return snapshot, errors.New("txThresholdGiB is invalid")
	}
	if snapshot.ResetDay, err = strconv.Atoi(values["KPANEL_NETWORK_OPERATIONS_RESET_DAY"]); err != nil {
		return snapshot, errors.New("resetDay is invalid")
	}
	if snapshot.RXThresholdGiB > contract.TrafficShutdownMaxThresholdGiB ||
		snapshot.TXThresholdGiB > contract.TrafficShutdownMaxThresholdGiB ||
		snapshot.ResetDay < 0 || snapshot.ResetDay > 31 {
		return snapshot, errors.New("traffic shutdown configuration is outside protocol bounds")
	}
	snapshot.ResourceVersion = values["KPANEL_NETWORK_OPERATIONS_VERSION"]
	if snapshot.Health == "ready" && (!snapshot.Enabled || snapshot.RXThresholdGiB == 0 || snapshot.TXThresholdGiB == 0 || snapshot.ResetDay < 1 || snapshot.ResetDay > 31) {
		return snapshot, errors.New("ready state is inconsistent")
	}
	if snapshot.Health == "disabled" && (snapshot.Enabled || snapshot.RXThresholdGiB != 0 || snapshot.TXThresholdGiB != 0 || snapshot.ResetDay != 0) {
		return snapshot, errors.New("disabled state is inconsistent")
	}
	return snapshot, nil
}

func (m *Manager) ExecuteTrafficShutdownAction(ctx context.Context, request contract.TrafficShutdownActionRequest) (contract.TrafficShutdownActionResult, error) {
	if field, detail := contract.ValidateTrafficShutdownAction(&request); field != "" {
		return contract.TrafficShutdownActionResult{}, fmt.Errorf("%w: %s: %s", ErrInvalidInput, field, detail)
	}
	if !m.enabled {
		return contract.TrafficShutdownActionResult{}, ErrDisabled
	}
	if err := m.networkOperationsCommonAvailability(); err != nil {
		return contract.TrafficShutdownActionResult{}, err
	}
	if err := m.networkOperationDependencies(nil, "awk", "crontab", "grep", "sed", "sha256sum", "stat", "wc", "mktemp", "cmp", "flock", "cp", "mv", "chmod", "chown"); err != nil {
		return contract.TrafficShutdownActionResult{}, err
	}
	lockContext, cancelLock := context.WithTimeout(ctx, resourceWriterLockTimeout)
	if !lockSystemResource(lockContext, &m.mu) {
		cancelLock()
		return contract.TrafficShutdownActionResult{}, fmt.Errorf("%w: timed out waiting for the network operation writer", ErrConflict)
	}
	cancelLock()
	defer m.mu.Unlock()
	transactionContext, cancelTransaction := context.WithTimeout(context.WithoutCancel(ctx), resourceActionTimeout)
	defer cancelTransaction()
	current, err := m.TrafficShutdown(transactionContext)
	if err != nil {
		return contract.TrafficShutdownActionResult{}, err
	}
	if current.ResourceVersion != request.ExpectedResourceVersion {
		return contract.TrafficShutdownActionResult{}, fmt.Errorf("%w: expected resource version is stale", ErrConflict)
	}
	arguments := []string{"traffic-shutdown", request.Action, request.ExpectedResourceVersion}
	if request.Action == "enable" {
		arguments = append(arguments,
			strconv.FormatUint(*request.RXThresholdGiB, 10),
			strconv.FormatUint(*request.TXThresholdGiB, 10),
			strconv.Itoa(*request.ResetDay),
		)
	}
	output, runErr := m.runNetworkOperation(transactionContext, resourceReceiptOutputLimit, arguments...)
	receipt, parseErr := parseNetworkOperationsReceipt(output)
	if parseErr != nil {
		return contract.TrafficShutdownActionResult{}, fmt.Errorf("%w: traffic-shutdown receipt is invalid: %v", ErrNeedsAttention, parseErr)
	}
	switch receipt.Status {
	case "conflict":
		return contract.TrafficShutdownActionResult{}, fmt.Errorf("%w: script detected a locked resource conflict", ErrConflict)
	case "failed":
		return contract.TrafficShutdownActionResult{}, fmt.Errorf("%w: kejilion.sh reported a completed rollback", ErrRolledBack)
	case "rollback-failed":
		detail := "kejilion.sh reported rollback failure"
		if receipt.Backup != "" {
			detail += "; recovery backup: " + receipt.Backup
		}
		return contract.TrafficShutdownActionResult{}, fmt.Errorf("%w: %s", ErrRollbackFailed, detail)
	case "applied", "unchanged":
		if runErr != nil {
			return contract.TrafficShutdownActionResult{}, fmt.Errorf("%w: script exited after a success receipt: %v", ErrNeedsAttention, runErr)
		}
	default:
		return contract.TrafficShutdownActionResult{}, fmt.Errorf("%w: unsupported script receipt status", ErrNeedsAttention)
	}
	actual, err := m.TrafficShutdown(transactionContext)
	if err != nil || actual.ResourceVersion != receipt.Version {
		return contract.TrafficShutdownActionResult{}, fmt.Errorf("%w: post-write traffic-shutdown readback did not match receipt", ErrNeedsAttention)
	}
	changed := receipt.Status == "applied"
	status, message := "succeeded", "traffic shutdown configuration applied by kejilion.sh"
	if !changed {
		status, message = "unchanged", "traffic shutdown configuration already matched the requested state"
	}
	return contract.TrafficShutdownActionResult{
		Action: request.Action, Status: status, Changed: changed, Message: message,
		BackupPath: receipt.Backup, ResourceVersion: actual.ResourceVersion, AppliedAt: m.now().UTC(),
	}, nil
}

type networkOperationsReceipt struct {
	Status, Version, Backup string
}

func parseNetworkOperationsReceipt(output []byte) (networkOperationsReceipt, error) {
	var receipt networkOperationsReceipt
	seen := make(map[string]bool)
	for _, line := range resourceLines(output) {
		if line == "" {
			continue
		}
		matched := false
		for key, target := range map[string]*string{
			"KPANEL_NETWORK_OPERATIONS_STATUS":  &receipt.Status,
			"KPANEL_NETWORK_OPERATIONS_VERSION": &receipt.Version,
			"KPANEL_NETWORK_OPERATIONS_BACKUP":  &receipt.Backup,
		} {
			if value, ok := receiptValue(line, key); ok {
				if seen[key] {
					return receipt, fmt.Errorf("duplicate %s", key)
				}
				seen[key], matched, *target = true, true, value
				break
			}
		}
		if !matched {
			return receipt, errors.New("unexpected non-empty output")
		}
	}
	if receipt.Status == "" || !resourceVersionPattern.MatchString(receipt.Version) {
		return receipt, errors.New("status or version is missing")
	}
	if receipt.Backup != "" && (receipt.Status != "rollback-failed" || len(receipt.Backup) > 4096 ||
		!filepath.IsAbs(receipt.Backup) || filepath.Clean(receipt.Backup) != receipt.Backup ||
		strings.ContainsAny(receipt.Backup, "\x00\r\n") || !networkOperationsRecoveryPattern.MatchString(receipt.Backup)) {
		return receipt, errors.New("backup path is malformed")
	}
	return receipt, nil
}
