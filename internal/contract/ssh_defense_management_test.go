package contract

import "testing"

func TestValidateSSHDefenseAction(t *testing.T) {
	version := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	tests := []struct {
		name    string
		request SSHDefenseActionRequest
		field   string
	}{
		{name: "enable", request: SSHDefenseActionRequest{Action: "enable", ExpectedResourceVersion: version}},
		{name: "strict profile", request: SSHDefenseActionRequest{Action: "set-profile", ExpectedResourceVersion: version, Profile: "strict"}},
		{name: "trusted cidr", request: SSHDefenseActionRequest{Action: "add-trusted", ExpectedResourceVersion: version, Address: "203.0.113.0/24"}},
		{name: "unban ipv6", request: SSHDefenseActionRequest{Action: "unban", ExpectedResourceVersion: version, Address: "2001:db8::8"}},
		{name: "invalid version", request: SSHDefenseActionRequest{Action: "enable", ExpectedResourceVersion: "old"}, field: "expectedResourceVersion"},
		{name: "unknown profile", request: SSHDefenseActionRequest{Action: "set-profile", ExpectedResourceVersion: version, Profile: "maximum"}, field: "profile"},
		{name: "unban cidr", request: SSHDefenseActionRequest{Action: "unban", ExpectedResourceVersion: version, Address: "203.0.113.0/24"}, field: "address"},
		{name: "extra value", request: SSHDefenseActionRequest{Action: "disable", ExpectedResourceVersion: version, Address: "203.0.113.8"}, field: "action"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			field, _ := ValidateSSHDefenseAction(&test.request)
			if field != test.field {
				t.Fatalf("field = %q, want %q", field, test.field)
			}
		})
	}
}
