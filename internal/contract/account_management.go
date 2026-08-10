package contract

import (
	"regexp"
	"strings"
	"time"
)

const (
	AccountManagementMaxAccounts = 256
	AccountManagementMaxKeys     = 32
)

var (
	accountUsernamePattern = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}$`)
	accountKeyIDPattern    = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

type SSHAuthorizedKey struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Fingerprint string `json:"fingerprint"`
	Comment     string `json:"comment,omitempty"`
}

type SystemAccount struct {
	Username       string             `json:"username"`
	UID            int                `json:"uid"`
	GID            int                `json:"gid"`
	Home           string             `json:"home"`
	Shell          string             `json:"shell"`
	Kind           string             `json:"kind"`
	PasswordStatus string             `json:"passwordStatus"`
	Role           string             `json:"role"`
	Groups         []string           `json:"groups"`
	SSHKeys        []SSHAuthorizedKey `json:"sshKeys"`
}

type SSHLoginPolicy struct {
	PasswordAuthentication  bool   `json:"passwordAuthentication"`
	PublicKeyAuthentication bool   `json:"publicKeyAuthentication"`
	RootLogin               string `json:"rootLogin"`
}

type AccountManagementSnapshot struct {
	ResourceVersion string          `json:"resourceVersion"`
	Accounts        []SystemAccount `json:"accounts"`
	Total           int             `json:"total"`
	Truncated       bool            `json:"truncated"`
	SSHPolicy       SSHLoginPolicy  `json:"sshPolicy"`
	ObservedAt      time.Time       `json:"observedAt"`
}

type AccountManagementActionRequest struct {
	Action                  string `json:"action"`
	ExpectedResourceVersion string `json:"expectedResourceVersion"`
	Username                string `json:"username,omitempty"`
	Role                    string `json:"role,omitempty"`
	Credential              string `json:"credential,omitempty"`
	Secret                  string `json:"secret,omitempty"`
	KeyID                   string `json:"keyId,omitempty"`
	PasswordAuthentication  *bool  `json:"passwordAuthentication,omitempty"`
	RootLogin               string `json:"rootLogin,omitempty"`
	RemoveHome              *bool  `json:"removeHome,omitempty"`
}

func ValidateAccountManagementAction(request *AccountManagementActionRequest) (string, string) {
	if request == nil {
		return "request", "request is required"
	}
	request.Action = strings.TrimSpace(request.Action)
	request.Username = strings.TrimSpace(request.Username)
	request.Role = strings.TrimSpace(request.Role)
	request.Credential = strings.TrimSpace(request.Credential)
	request.KeyID = strings.TrimSpace(request.KeyID)
	request.RootLogin = strings.TrimSpace(request.RootLogin)
	if !accountKeyIDPattern.MatchString(request.ExpectedResourceVersion) {
		return "expectedResourceVersion", "expectedResourceVersion must be 64 lowercase hexadecimal characters"
	}
	validUsername := func() (string, string) {
		if !accountUsernamePattern.MatchString(request.Username) || strings.HasSuffix(request.Username, "$") {
			return "username", "username must be a valid Linux account name with at most 32 characters"
		}
		return "", ""
	}
	validRole := func() (string, string) {
		if request.Role != "standard" && request.Role != "administrator" && request.Role != "passwordless-admin" {
			return "role", "role must be standard, administrator, or passwordless-admin"
		}
		return "", ""
	}
	validCredential := func() (string, string) {
		if request.Credential != "password" && request.Credential != "key" {
			return "credential", "credential must be password or key"
		}
		if strings.ContainsAny(request.Secret, "\x00\r\n") {
			return "secret", "secret must be a single line"
		}
		if request.Credential == "password" {
			if len(request.Secret) < 8 || len(request.Secret) > 256 {
				return "secret", "password must contain 8 to 256 bytes"
			}
			return "", ""
		}
		if len(request.Secret) == 0 || len(request.Secret) > 4096 || !validPublicKeyPrefix(request.Secret) {
			return "secret", "public key must be one supported OpenSSH public key line"
		}
		return "", ""
	}
	noSecretFields := func() (string, string) {
		if request.Credential != "" || request.Secret != "" || request.KeyID != "" {
			return "action", "credential, secret, and keyId are not allowed for this action"
		}
		return "", ""
	}
	noPolicyFields := func() (string, string) {
		if request.PasswordAuthentication != nil || request.RootLogin != "" || request.RemoveHome != nil {
			return "action", "SSH policy and removeHome fields are not allowed for this action"
		}
		return "", ""
	}

	switch request.Action {
	case "create":
		if field, detail := validUsername(); field != "" {
			return field, detail
		}
		if request.Username == "root" {
			return "username", "root already exists and cannot be created"
		}
		if field, detail := validRole(); field != "" {
			return field, detail
		}
		if field, detail := validCredential(); field != "" {
			return field, detail
		}
		if request.Credential == "key" && request.Role == "administrator" {
			return "role", "key-only administrator must use passwordless-admin because sudo has no account password"
		}
		if field, detail := noPolicyFields(); field != "" {
			return field, detail
		}
	case "set-password":
		if field, detail := validUsername(); field != "" {
			return field, detail
		}
		if request.Role != "" || request.Credential != "" || request.KeyID != "" {
			return "action", "role, credential, and keyId are not allowed for set-password"
		}
		if len(request.Secret) < 8 || len(request.Secret) > 256 || strings.ContainsAny(request.Secret, "\x00\r\n") {
			return "secret", "password must be a single line containing 8 to 256 bytes"
		}
		if field, detail := noPolicyFields(); field != "" {
			return field, detail
		}
	case "add-key":
		if field, detail := validUsername(); field != "" {
			return field, detail
		}
		if request.Role != "" || request.Credential != "" || request.KeyID != "" {
			return "action", "role, credential, and keyId are not allowed for add-key"
		}
		if len(request.Secret) == 0 || len(request.Secret) > 4096 || strings.ContainsAny(request.Secret, "\x00\r\n") || !validPublicKeyPrefix(request.Secret) {
			return "secret", "public key must be one supported OpenSSH public key line"
		}
		if field, detail := noPolicyFields(); field != "" {
			return field, detail
		}
	case "delete-key":
		if field, detail := validUsername(); field != "" {
			return field, detail
		}
		if !accountKeyIDPattern.MatchString(request.KeyID) {
			return "keyId", "keyId must be 64 lowercase hexadecimal characters"
		}
		if request.Role != "" || request.Credential != "" || request.Secret != "" {
			return "action", "role, credential, and secret are not allowed for delete-key"
		}
		if field, detail := noPolicyFields(); field != "" {
			return field, detail
		}
	case "set-role":
		if field, detail := validUsername(); field != "" {
			return field, detail
		}
		if request.Username == "root" {
			return "username", "root role is fixed"
		}
		if field, detail := validRole(); field != "" {
			return field, detail
		}
		if field, detail := noSecretFields(); field != "" {
			return field, detail
		}
		if field, detail := noPolicyFields(); field != "" {
			return field, detail
		}
	case "set-ssh-policy":
		if request.Username != "" || request.Role != "" {
			return "action", "username and role are not allowed for set-ssh-policy"
		}
		if field, detail := noSecretFields(); field != "" {
			return field, detail
		}
		if request.PasswordAuthentication == nil {
			return "passwordAuthentication", "passwordAuthentication is required"
		}
		if request.RootLogin != "enabled" && request.RootLogin != "key-only" && request.RootLogin != "disabled" {
			return "rootLogin", "rootLogin must be enabled, key-only, or disabled"
		}
		if request.RemoveHome != nil {
			return "removeHome", "removeHome is not allowed for set-ssh-policy"
		}
	case "disable-root":
		if request.Username != "" || request.Role != "" {
			return "action", "username and role are not allowed for disable-root"
		}
		if field, detail := noSecretFields(); field != "" {
			return field, detail
		}
		if field, detail := noPolicyFields(); field != "" {
			return field, detail
		}
	case "create-admin-disable-root":
		if field, detail := validUsername(); field != "" {
			return field, detail
		}
		if request.Username == "root" {
			return "username", "replacement administrator cannot be root"
		}
		if request.Role != "" || request.KeyID != "" {
			return "action", "role and keyId are not allowed for create-admin-disable-root"
		}
		if field, detail := validCredential(); field != "" {
			return field, detail
		}
		if field, detail := noPolicyFields(); field != "" {
			return field, detail
		}
	case "delete":
		if field, detail := validUsername(); field != "" {
			return field, detail
		}
		if request.Username == "root" {
			return "username", "root cannot be deleted; use disable-root"
		}
		if request.Role != "" {
			return "role", "role is not allowed for delete"
		}
		if field, detail := noSecretFields(); field != "" {
			return field, detail
		}
		if request.PasswordAuthentication != nil || request.RootLogin != "" {
			return "action", "SSH policy fields are not allowed for delete"
		}
		if request.RemoveHome == nil {
			return "removeHome", "removeHome is required"
		}
	default:
		return "action", "unsupported account management action"
	}
	return "", ""
}

func validPublicKeyPrefix(value string) bool {
	for _, prefix := range []string{
		"ssh-rsa ", "ssh-ed25519 ", "ecdsa-sha2-", "sk-ssh-ed25519@openssh.com ", "sk-ecdsa-sha2-nistp256@openssh.com ",
	} {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

type AccountManagementActionResult struct {
	Action          string    `json:"action"`
	Status          string    `json:"status"`
	Changed         bool      `json:"changed"`
	Message         string    `json:"message"`
	BackupPath      string    `json:"backupPath,omitempty"`
	ResourceVersion string    `json:"resourceVersion"`
	AppliedAt       time.Time `json:"appliedAt"`
}
