package panel

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/kejilion/kejilion-panel/internal/dockerx"
)

func TestAIToolApprovalBoundary(t *testing.T) {
	tools := &panelAITools{}
	tests := []struct {
		name      string
		arguments string
		approval  bool
	}{
		{name: "host_system_summary", arguments: `{}`, approval: false},
		{name: "host_docker_resource_usage", arguments: `{}`, approval: false},
		{name: "host_file_read", arguments: `{"path":"/tmp/test"}`, approval: false},
		{name: "host_file_write", arguments: `{}`, approval: false},
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

func TestDockerMaintenanceToolSchemaMatchesAgentContract(t *testing.T) {
	tools := (&panelAITools{}).Definitions()
	var schema map[string]any
	for _, definition := range tools {
		if definition.Name == "host_docker_task" {
			if err := json.Unmarshal(definition.Schema, &schema); err != nil {
				t.Fatal(err)
			}
			break
		}
	}
	if schema == nil {
		t.Fatal("host_docker_task schema missing")
	}
	required, _ := schema["required"].([]any)
	if len(required) != 1 || required[0] != "action" || schema["additionalProperties"] != false {
		t.Fatalf("maintenance schema contract=%#v", schema)
	}
	properties, _ := schema["properties"].(map[string]any)
	if _, ok := properties["type"]; ok {
		t.Fatal("legacy type discriminator remains in schema")
	}
	action, _ := properties["action"].(map[string]any)
	values, _ := action["enum"].([]any)
	gotActions := make([]string, 0, len(values))
	for _, value := range values {
		gotActions = append(gotActions, value.(string))
	}
	wantActions := dockerx.MaintenanceActions()
	sort.Strings(gotActions)
	sort.Strings(wantActions)
	if !reflect.DeepEqual(gotActions, wantActions) {
		t.Fatalf("maintenance action enum=%#v want=%#v", gotActions, wantActions)
	}
	validFields := map[string]bool{}
	inputType := reflect.TypeOf(dockerx.MaintenanceInput{})
	for index := 0; index < inputType.NumField(); index++ {
		name := strings.Split(inputType.Field(index).Tag.Get("json"), ",")[0]
		validFields[name] = true
	}
	for name := range properties {
		if !validFields[name] {
			t.Fatalf("schema property %q is not accepted by MaintenanceInput", name)
		}
	}
}

func TestDockerMaintenanceArgumentsAreStrictAndCanonical(t *testing.T) {
	tools := &panelAITools{}
	if _, _, _, _, err := tools.prepareWrite("host_docker_task", json.RawMessage(`{"type":"summary"}`)); err == nil {
		t.Fatal("legacy type argument was accepted")
	}
	method, path, target, body, err := tools.prepareWrite("host_docker_task", json.RawMessage(`{"action":"backup_create"}`))
	if err != nil || method != "POST" || path != "/v1/docker/tasks" || target != "backup_create" || string(body) != `{"action":"backup_create"}` {
		t.Fatalf("maintenance request method=%q path=%q target=%q body=%s err=%v", method, path, target, body, err)
	}
}
