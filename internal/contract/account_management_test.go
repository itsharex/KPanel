package contract

import (
	"strings"
	"testing"
)

func TestValidateAccountManagementActionKeepsSecretsAndActionsTyped(t *testing.T) {
	version := strings.Repeat("a", 64)
	removeHome := true
	passwordAuthentication := false
	tests := []struct {
		name    string
		request AccountManagementActionRequest
		field   string
	}{
		{name: "create password", request: AccountManagementActionRequest{Action: "create", ExpectedResourceVersion: version, Username: "operator", Role: "administrator", Credential: "password", Secret: "correct horse battery staple"}},
		{name: "create key admin must be explicit", request: AccountManagementActionRequest{Action: "create", ExpectedResourceVersion: version, Username: "operator", Role: "administrator", Credential: "key", Secret: "ssh-ed25519 AAAA test"}, field: "role"},
		{name: "password newline", request: AccountManagementActionRequest{Action: "set-password", ExpectedResourceVersion: version, Username: "root", Secret: "password\nnext"}, field: "secret"},
		{name: "key id", request: AccountManagementActionRequest{Action: "delete-key", ExpectedResourceVersion: version, Username: "operator", KeyID: strings.Repeat("b", 64)}},
		{name: "policy", request: AccountManagementActionRequest{Action: "set-ssh-policy", ExpectedResourceVersion: version, PasswordAuthentication: &passwordAuthentication, RootLogin: "key-only"}},
		{name: "delete root", request: AccountManagementActionRequest{Action: "delete", ExpectedResourceVersion: version, Username: "root", RemoveHome: &removeHome}, field: "username"},
		{name: "delete requires decision", request: AccountManagementActionRequest{Action: "delete", ExpectedResourceVersion: version, Username: "operator"}, field: "removeHome"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			field, _ := ValidateAccountManagementAction(&test.request)
			if field != test.field {
				t.Fatalf("field=%q want=%q", field, test.field)
			}
		})
	}
}
