package panel

import (
	"encoding/json"
	"testing"
)

func TestAIToolApprovalBoundary(t *testing.T) {
	tools := &panelAITools{}
	tests := []struct {
		name      string
		arguments string
		approval  bool
	}{
		{name: "host_system_summary", arguments: `{}`, approval: false},
		{name: "host_diagnostic_start", arguments: `{"checkId":"system"}`, approval: false},
		{name: "host_app_action", arguments: `{"action":"install"}`, approval: false},
		{name: "host_app_action", arguments: `{"action":"restart"}`, approval: false},
		{name: "host_app_action", arguments: `{"action":"uninstall"}`, approval: true},
		{name: "host_docker_container_action", arguments: `{"action":"stop"}`, approval: false},
		{name: "host_docker_container_action", arguments: `{"action":"remove"}`, approval: true},
		{name: "host_site_change", arguments: `{"operation":"create"}`, approval: false},
		{name: "host_site_change", arguments: `{"operation":"delete"}`, approval: true},
		{name: "host_system_action", arguments: `{"action":"reboot"}`, approval: true},
		{name: "host_docker_exec", arguments: `{}`, approval: true},
		{name: "host_job_input", arguments: `{}`, approval: true},
		{name: "unknown", arguments: `{}`, approval: true},
		{name: "host_app_action", arguments: `{`, approval: true},
	}
	for _, test := range tests {
		t.Run(test.name+test.arguments, func(t *testing.T) {
			if got := tools.RequiresApproval(test.name, json.RawMessage(test.arguments)); got != test.approval {
				t.Fatalf("RequiresApproval()=%v want=%v", got, test.approval)
			}
		})
	}
}
