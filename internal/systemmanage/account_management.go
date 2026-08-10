package systemmanage

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

const accountManagementOutputLimit = 2 << 20

var (
	accountManagementProtocolV1Pattern = regexp.MustCompile(`(?m)^KPANEL_ACCOUNT_MANAGEMENT_PROTOCOL_VERSION="1"\r?$`)
	accountManagementRecoveryPattern   = regexp.MustCompile(`^/var/lib/kejilion-panel/system/recovery/system-resource/[0-9]{8}T[0-9]{6}Z-account-management\.[A-Za-z0-9]{6}$`)
	accountManagementUsernamePattern   = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}$`)
)

func (m *Manager) AccountManagementCapabilities() []contract.Capability {
	readErr := m.accountManagementAvailability(false)
	writeErr := m.accountManagementAvailability(true)
	capability := func(id, method string, err error) contract.Capability {
		if err != nil {
			return contract.Capability{ID: id, Enabled: false, Reason: resourceCapabilityReason(err)}
		}
		return contract.Capability{ID: id, Enabled: true, Methods: []string{method}}
	}
	return []contract.Capability{
		capability("system.accounts.read", "GET", readErr),
		capability("system.accounts.write", "POST", writeErr),
	}
}

func (m *Manager) accountManagementAvailability(write bool) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("%w: account management is only available on Linux", ErrUnsupported)
	}
	if m.effectiveUID() != 0 {
		return fmt.Errorf("%w: Agent must run as root for account management", ErrUnsupported)
	}
	commands := []string{
		"env", "bash", "awk", "cat", "cut", "getent", "grep", "id", "mktemp", "od", "passwd", "runuser",
		"sha256sum", "ssh-keygen", "sshd", "stat", "tr", "wc",
	}
	if write {
		if !m.enabled {
			return fmt.Errorf("%w: host system writes are disabled", ErrDisabled)
		}
		commands = append(commands,
			"chpasswd", "cmp", "flock", "gpasswd", "head", "mkdir", "tail", "useradd", "userdel", "usermod", "visudo",
			"cp", "mv", "chmod", "chown",
		)
	}
	for _, command := range commands {
		if _, err := m.runner.LookPath(command); err != nil {
			return fmt.Errorf("%w: %s is unavailable", ErrUnsupported, command)
		}
	}
	_, err := m.accountManagementScriptPath()
	return err
}

func (m *Manager) accountManagementScriptPath() (string, error) {
	path, err := m.systemResourceScriptPath()
	if err != nil {
		return "", err
	}
	content, err := readResourceFile(path, resourceScriptMaxBytes)
	if err != nil || !trustedKejilionAccountManagementContent(content) {
		return "", fmt.Errorf("%w: trusted kejilion.sh account-management protocol v1 was not found", ErrUnsupported)
	}
	return path, nil
}

func trustedKejilionAccountManagementContent(content []byte) bool {
	value := string(content)
	return trustedKejilionSystemResourceContent(content) &&
		accountManagementProtocolV1Pattern.Match(content) &&
		strings.Contains(value, "KJ_ACCOUNT_MANAGEMENT_NONINTERACTIVE") &&
		strings.Contains(value, "kpanel_account_dispatch") &&
		strings.Contains(value, "KPANEL_ACCOUNT_MANAGEMENT_STATUS") &&
		strings.Contains(value, "--secret-stdin")
}

func (m *Manager) runAccountManagement(
	ctx context.Context,
	input []byte,
	arguments ...string,
) ([]byte, error) {
	script, err := m.accountManagementScriptPath()
	if err != nil {
		return nil, err
	}
	commandArguments := []string{
		"KJ_ACCOUNT_MANAGEMENT_NONINTERACTIVE=1", "bash", script,
		"kpanel", "account-management",
	}
	commandArguments = append(commandArguments, arguments...)
	output, _, runErr := m.runResourceCommandInput(
		ctx, accountManagementOutputLimit, input, "env", commandArguments...,
	)
	return output, runErr
}

func (m *Manager) Accounts(ctx context.Context) (contract.AccountManagementSnapshot, error) {
	if err := m.accountManagementAvailability(false); err != nil {
		return contract.AccountManagementSnapshot{}, err
	}
	readContext, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	output, runErr := m.runAccountManagement(readContext, nil, "status")
	snapshot, parseErr := parseAccountManagementSnapshot(output)
	if parseErr != nil {
		return contract.AccountManagementSnapshot{}, fmt.Errorf("%w: invalid account-management protocol output: %v", ErrNeedsAttention, parseErr)
	}
	if runErr != nil {
		return contract.AccountManagementSnapshot{}, fmt.Errorf("%w: kejilion.sh could not read accounts: %v", ErrUnsupported, runErr)
	}
	snapshot.ObservedAt = m.now().UTC()
	return snapshot, nil
}

func parseAccountManagementSnapshot(output []byte) (contract.AccountManagementSnapshot, error) {
	var snapshot contract.AccountManagementSnapshot
	values := make(map[string]string)
	accounts := make(map[string]contract.SystemAccount)
	keyCounts := make(map[string]int)
	keys := make(map[string][]contract.SSHAuthorizedKey)
	for _, line := range resourceLines(output) {
		if line == "" {
			continue
		}
		if value, ok := receiptValue(line, "KPANEL_ACCOUNT_MANAGEMENT_ACCOUNT_HEX"); ok {
			if len(accounts) >= contract.AccountManagementMaxAccounts {
				return snapshot, errors.New("account count exceeds protocol limit")
			}
			record, err := decodeAccountManagementHex(value, 8192)
			if err != nil {
				return snapshot, err
			}
			fields := strings.Split(record, "\t")
			if len(fields) != 10 || !accountManagementUsernamePattern.MatchString(fields[0]) {
				return snapshot, errors.New("account record is invalid")
			}
			if _, exists := accounts[fields[0]]; exists {
				return snapshot, errors.New("duplicate account")
			}
			uid, uidErr := strconv.Atoi(fields[1])
			gid, gidErr := strconv.Atoi(fields[2])
			count, countErr := strconv.Atoi(fields[9])
			if uidErr != nil || gidErr != nil || uid < 0 || gid < 0 || countErr != nil || count < 0 || count > contract.AccountManagementMaxKeys {
				return snapshot, errors.New("account numeric fields are invalid")
			}
			if fields[5] != "root" && fields[5] != "human" && fields[5] != "system" {
				return snapshot, errors.New("account kind is invalid")
			}
			if fields[6] != "enabled" && fields[6] != "locked" && fields[6] != "unset" && fields[6] != "unknown" {
				return snapshot, errors.New("password status is invalid")
			}
			if fields[7] != "root" && fields[7] != "standard" && fields[7] != "administrator" && fields[7] != "passwordless-admin" {
				return snapshot, errors.New("account role is invalid")
			}
			groups := []string{}
			if fields[8] != "" {
				groups = strings.Split(fields[8], ",")
				if len(groups) > 128 {
					return snapshot, errors.New("account group count is invalid")
				}
			}
			accounts[fields[0]] = contract.SystemAccount{
				Username: fields[0], UID: uid, GID: gid, Home: fields[3], Shell: fields[4],
				Kind: fields[5], PasswordStatus: fields[6], Role: fields[7], Groups: groups,
			}
			keyCounts[fields[0]] = count
			continue
		}
		if value, ok := receiptValue(line, "KPANEL_ACCOUNT_MANAGEMENT_KEY_HEX"); ok {
			record, err := decodeAccountManagementHex(value, 16384)
			if err != nil {
				return snapshot, err
			}
			fields := strings.Split(record, "\t")
			if len(fields) != 5 || !accountManagementUsernamePattern.MatchString(fields[0]) || !resourceVersionPattern.MatchString(fields[1]) {
				return snapshot, errors.New("SSH key record is invalid")
			}
			if len(keys[fields[0]]) >= contract.AccountManagementMaxKeys || len(fields[2]) > 128 || len(fields[3]) > 256 || len(fields[4]) > 128 {
				return snapshot, errors.New("SSH key record exceeds protocol bounds")
			}
			keys[fields[0]] = append(keys[fields[0]], contract.SSHAuthorizedKey{
				ID: fields[1], Type: fields[2], Fingerprint: fields[3], Comment: fields[4],
			})
			continue
		}
		matched := false
		for _, key := range []string{
			"KPANEL_ACCOUNT_MANAGEMENT_STATUS", "KPANEL_ACCOUNT_MANAGEMENT_VERSION",
			"KPANEL_ACCOUNT_MANAGEMENT_PASSWORD_AUTH", "KPANEL_ACCOUNT_MANAGEMENT_PUBKEY_AUTH",
			"KPANEL_ACCOUNT_MANAGEMENT_ROOT_LOGIN", "KPANEL_ACCOUNT_MANAGEMENT_TOTAL",
			"KPANEL_ACCOUNT_MANAGEMENT_TRUNCATED",
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
	if values["KPANEL_ACCOUNT_MANAGEMENT_STATUS"] != "ok" || !resourceVersionPattern.MatchString(values["KPANEL_ACCOUNT_MANAGEMENT_VERSION"]) {
		return snapshot, errors.New("status or version is invalid")
	}
	if values["KPANEL_ACCOUNT_MANAGEMENT_PASSWORD_AUTH"] != "yes" && values["KPANEL_ACCOUNT_MANAGEMENT_PASSWORD_AUTH"] != "no" {
		return snapshot, errors.New("password authentication state is invalid")
	}
	if values["KPANEL_ACCOUNT_MANAGEMENT_PUBKEY_AUTH"] != "yes" && values["KPANEL_ACCOUNT_MANAGEMENT_PUBKEY_AUTH"] != "no" {
		return snapshot, errors.New("public key authentication state is invalid")
	}
	rootLogin := values["KPANEL_ACCOUNT_MANAGEMENT_ROOT_LOGIN"]
	if rootLogin != "enabled" && rootLogin != "key-only" && rootLogin != "disabled" && rootLogin != "custom" {
		return snapshot, errors.New("root login state is invalid")
	}
	total, err := strconv.Atoi(values["KPANEL_ACCOUNT_MANAGEMENT_TOTAL"])
	if err != nil || total < 0 || total > 8192 {
		return snapshot, errors.New("account total is invalid")
	}
	truncatedValue := values["KPANEL_ACCOUNT_MANAGEMENT_TRUNCATED"]
	if truncatedValue != "true" && truncatedValue != "false" {
		return snapshot, errors.New("account truncated state is invalid")
	}
	usernames := make([]string, 0, len(accounts))
	for username := range accounts {
		usernames = append(usernames, username)
	}
	sort.Slice(usernames, func(i, j int) bool {
		left, right := accounts[usernames[i]], accounts[usernames[j]]
		if left.UID != right.UID {
			return left.UID < right.UID
		}
		return left.Username < right.Username
	})
	for _, username := range usernames {
		account := accounts[username]
		account.SSHKeys = keys[username]
		if len(account.SSHKeys) != keyCounts[username] {
			return snapshot, errors.New("account SSH key count does not match key records")
		}
		snapshot.Accounts = append(snapshot.Accounts, account)
	}
	for username := range keys {
		if _, exists := accounts[username]; !exists {
			return snapshot, errors.New("SSH key references an unknown account")
		}
	}
	truncated := truncatedValue == "true"
	if total < len(snapshot.Accounts) || truncated != (total > len(snapshot.Accounts)) || (!truncated && total != len(snapshot.Accounts)) {
		return snapshot, errors.New("account snapshot bounds are inconsistent")
	}
	snapshot.ResourceVersion = values["KPANEL_ACCOUNT_MANAGEMENT_VERSION"]
	snapshot.Total = total
	snapshot.Truncated = truncated
	snapshot.SSHPolicy = contract.SSHLoginPolicy{
		PasswordAuthentication:  values["KPANEL_ACCOUNT_MANAGEMENT_PASSWORD_AUTH"] == "yes",
		PublicKeyAuthentication: values["KPANEL_ACCOUNT_MANAGEMENT_PUBKEY_AUTH"] == "yes",
		RootLogin:               rootLogin,
	}
	return snapshot, nil
}

func decodeAccountManagementHex(value string, maximum int) (string, error) {
	if value == "" || len(value) > maximum || len(value)%2 != 0 {
		return "", errors.New("hex record size is invalid")
	}
	decoded, err := hex.DecodeString(value)
	if err != nil || strings.ContainsAny(string(decoded), "\x00\r\n") {
		return "", errors.New("hex record encoding is invalid")
	}
	return string(decoded), nil
}

type accountManagementReceipt struct {
	Status  string
	Version string
	Backup  string
}

func parseAccountManagementReceipt(output []byte) (accountManagementReceipt, error) {
	var receipt accountManagementReceipt
	seen := make(map[string]bool)
	for _, line := range resourceLines(output) {
		if line == "" {
			continue
		}
		matched := false
		for key, target := range map[string]*string{
			"KPANEL_ACCOUNT_MANAGEMENT_STATUS":  &receipt.Status,
			"KPANEL_ACCOUNT_MANAGEMENT_VERSION": &receipt.Version,
			"KPANEL_ACCOUNT_MANAGEMENT_BACKUP":  &receipt.Backup,
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
			return receipt, errors.New("unexpected receipt output")
		}
	}
	if receipt.Status == "" {
		return receipt, errors.New("receipt status is missing")
	}
	if receipt.Status != "needs-attention" && !resourceVersionPattern.MatchString(receipt.Version) {
		return receipt, errors.New("receipt version is invalid")
	}
	if receipt.Backup != "" && !accountManagementRecoveryPattern.MatchString(receipt.Backup) {
		return receipt, errors.New("receipt recovery path is invalid")
	}
	return receipt, nil
}

func accountManagementInvocation(request contract.AccountManagementActionRequest) ([]string, []byte) {
	secretInput := func() []byte { return []byte(request.Secret + "\n") }
	switch request.Action {
	case "create":
		return []string{"create", request.ExpectedResourceVersion, request.Username, request.Role, request.Credential, "--secret-stdin"}, secretInput()
	case "set-password":
		return []string{"set-password", request.ExpectedResourceVersion, request.Username, "--secret-stdin"}, secretInput()
	case "add-key":
		return []string{"add-key", request.ExpectedResourceVersion, request.Username, "--secret-stdin"}, secretInput()
	case "delete-key":
		return []string{"delete-key", request.ExpectedResourceVersion, request.Username, request.KeyID}, nil
	case "set-role":
		return []string{"set-role", request.ExpectedResourceVersion, request.Username, request.Role}, nil
	case "set-ssh-policy":
		password := "disabled"
		if request.PasswordAuthentication != nil && *request.PasswordAuthentication {
			password = "enabled"
		}
		return []string{"set-ssh-policy", request.ExpectedResourceVersion, password, request.RootLogin}, nil
	case "disable-root":
		return []string{"disable-root", request.ExpectedResourceVersion}, nil
	case "create-admin-disable-root":
		return []string{"create-admin-disable-root", request.ExpectedResourceVersion, request.Username, request.Credential, "--secret-stdin"}, secretInput()
	case "delete":
		removeHome := "false"
		if request.RemoveHome != nil && *request.RemoveHome {
			removeHome = "true"
		}
		return []string{"delete", request.ExpectedResourceVersion, request.Username, removeHome}, nil
	default:
		return nil, nil
	}
}

func (m *Manager) ExecuteAccountManagementAction(
	ctx context.Context,
	request contract.AccountManagementActionRequest,
) (contract.AccountManagementActionResult, error) {
	if field, detail := contract.ValidateAccountManagementAction(&request); field != "" {
		return contract.AccountManagementActionResult{}, fmt.Errorf("%w: %s: %s", ErrInvalidInput, field, detail)
	}
	if err := m.accountManagementAvailability(true); err != nil {
		return contract.AccountManagementActionResult{}, err
	}
	lockContext, cancelLock := context.WithTimeout(ctx, resourceWriterLockTimeout)
	if !lockSystemResource(lockContext, &m.mu) {
		cancelLock()
		return contract.AccountManagementActionResult{}, fmt.Errorf("%w: timed out waiting for the account management writer", ErrConflict)
	}
	cancelLock()
	defer m.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return contract.AccountManagementActionResult{}, err
	}
	transactionContext, cancelTransaction := context.WithTimeout(context.WithoutCancel(ctx), resourceActionTimeout)
	defer cancelTransaction()
	current, err := m.Accounts(transactionContext)
	if err != nil {
		return contract.AccountManagementActionResult{}, err
	}
	if current.ResourceVersion != request.ExpectedResourceVersion {
		return contract.AccountManagementActionResult{}, fmt.Errorf("%w: expected resource version is stale", ErrConflict)
	}
	arguments, input := accountManagementInvocation(request)
	output, runErr := m.runAccountManagement(transactionContext, input, arguments...)
	receipt, parseErr := parseAccountManagementReceipt(output)
	if parseErr != nil {
		return contract.AccountManagementActionResult{}, fmt.Errorf("%w: account-management receipt is invalid: %v", ErrNeedsAttention, parseErr)
	}
	switch receipt.Status {
	case "conflict":
		return contract.AccountManagementActionResult{}, fmt.Errorf("%w: script detected an account resource conflict", ErrConflict)
	case "failed":
		return contract.AccountManagementActionResult{}, fmt.Errorf("%w: kejilion.sh reported a completed rollback", ErrRolledBack)
	case "rollback-failed":
		detail := "kejilion.sh reported account rollback failure"
		if receipt.Backup != "" {
			detail += "; recovery backup: " + receipt.Backup
		}
		return contract.AccountManagementActionResult{}, fmt.Errorf("%w: %s", ErrRollbackFailed, detail)
	case "needs-attention":
		detail := "kejilion.sh could not verify final account state"
		if receipt.Backup != "" {
			detail += "; recovery backup: " + receipt.Backup
		}
		return contract.AccountManagementActionResult{}, fmt.Errorf("%w: %s", ErrNeedsAttention, detail)
	case "applied", "unchanged":
		if runErr != nil {
			return contract.AccountManagementActionResult{}, fmt.Errorf("%w: script exited after a success receipt: %v", ErrNeedsAttention, runErr)
		}
	default:
		return contract.AccountManagementActionResult{}, fmt.Errorf("%w: unsupported account receipt status", ErrNeedsAttention)
	}
	actual, err := m.Accounts(transactionContext)
	if err != nil {
		return contract.AccountManagementActionResult{}, fmt.Errorf("%w: account post-write readback failed: %v", ErrNeedsAttention, err)
	}
	if actual.ResourceVersion != receipt.Version {
		return contract.AccountManagementActionResult{}, fmt.Errorf("%w: account resource version does not match the script receipt", ErrNeedsAttention)
	}
	changed := receipt.Status == "applied"
	status, message := "succeeded", "account action applied by kejilion.sh"
	if !changed {
		status, message = "unchanged", "account state already matched the requested state"
	}
	return contract.AccountManagementActionResult{
		Action: request.Action, Status: status, Changed: changed, Message: message,
		BackupPath: receipt.Backup, ResourceVersion: actual.ResourceVersion, AppliedAt: m.now().UTC(),
	}, nil
}
