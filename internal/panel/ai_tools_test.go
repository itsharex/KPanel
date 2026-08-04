package panel

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/kejilion/kejilion-panel/internal/contract"
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
		{name: "host_system_processes", arguments: `{}`, approval: false},
		{name: "host_system_storage_usage", arguments: `{"path":"/var"}`, approval: false},
		{name: "host_file_read", arguments: `{"path":"/tmp/test"}`, approval: false},
		{name: "host_file_tail", arguments: `{"path":"/var/log/test"}`, approval: false},
		{name: "host_file_write", arguments: `{}`, approval: false},
		{name: "host_file_trash", arguments: `{}`, approval: false},
		{name: "host_nginx_test", arguments: `{}`, approval: false},
		{name: "host_nginx_reload", arguments: `{}`, approval: false},
		{name: "host_diagnostic_start", arguments: `{"checkId":"system"}`, approval: false},
		{name: "host_app_action", arguments: `{"action":"install"}`, approval: false},
		{name: "host_app_action", arguments: `{"action":"restart"}`, approval: false},
		{name: "host_app_action", arguments: `{"action":"uninstall"}`, approval: true},
		{name: "host_docker_container_action", arguments: `{"action":"stop"}`, approval: false},
		{name: "host_docker_container_action", arguments: `{"action":"remove"}`, approval: true},
		{name: "host_site_change", arguments: `{"operation":"create"}`, approval: false},
		{name: "host_site_change", arguments: `{"operation":"delete"}`, approval: true},
		{name: "host_system_action", arguments: `{"action":"reboot"}`, approval: true},
		{name: "host_system_action", arguments: `{"action":"cleanup","maintenancePolicy":"cache"}`, approval: false},
		{name: "host_system_action", arguments: `{"action":"cleanup","maintenancePolicy":"standard"}`, approval: true},
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

func TestOperationsToolArgumentsAreStrictAndRecoverable(t *testing.T) {
	tools := &panelAITools{}
	if _, _, _, _, err := tools.prepareWrite("host_system_action", json.RawMessage(`{"action":"cleanup","maintenancePolicy":"cache","unknown":true}`)); err == nil {
		t.Fatal("unknown system action field was accepted")
	}
	method, path, target, body, err := tools.prepareWrite("host_file_trash", json.RawMessage(`{"sources":["/tmp/old.log"],"expectedResourceVersions":{"/tmp/old.log":"sha256:0000000000000000000000000000000000000000000000000000000000000000"}}`))
	if err != nil || method != "POST" || path != "/v1/files/actions" || target != "/tmp/old.log" || !strings.Contains(string(body), `"action":"trash"`) {
		t.Fatalf("trash request method=%q path=%q target=%q body=%s err=%v", method, path, target, body, err)
	}
	if _, _, _, _, err := tools.prepareWrite("host_file_trash", json.RawMessage(`{"sources":["/tmp/old.log"],"expectedResourceVersions":{}}`)); err == nil {
		t.Fatal("trash without a current resourceVersion was accepted")
	}
}

func TestAIFileBoundarySeparatesOperationsFromSecretsAndCore(t *testing.T) {
	for _, path := range []string{"/home/web/log/nginx/error.log", "/home/web/conf.d/example.conf", "/tmp/cleanup.log"} {
		if !aiFileReadable(path) || !aiFileMutable(path) {
			t.Fatalf("ordinary operations path was blocked: %s", path)
		}
	}
	for _, path := range []string{"/etc/shadow", "/root/.ssh/id_ed25519", "/home/app/.env", "/proc/1/environ", "/etc/ssl/private/site.key", "/home/docker/kpanel/.env"} {
		if aiFileReadable(path) {
			t.Fatalf("sensitive content was readable by AI: %s", path)
		}
	}
	for _, path := range []string{"/etc/passwd", "/etc/ssh/sshd_config", "/usr/bin/bash", "/var/lib/docker/config.json", "/home/docker/kpanel/docker-compose.yml"} {
		if aiFileMutable(path) {
			t.Fatalf("core path was mutable by AI: %s", path)
		}
	}
}

func TestAIStorageAnalysisUsesFixedRoots(t *testing.T) {
	for _, path := range []string{"/", "/var", "/var/log", "/home/docker"} {
		if !aiStorageRoot(path) {
			t.Fatalf("fixed storage root was rejected: %s", path)
		}
	}
	for _, path := range []string{"/etc", "/var/../etc", "/home/user/private"} {
		if aiStorageRoot(path) {
			t.Fatalf("arbitrary storage root was accepted: %s", path)
		}
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

func TestSystemActionToolSchemaMatchesPanelContract(t *testing.T) {
	var schema map[string]any
	if err := json.Unmarshal(systemActionToolSchema, &schema); err != nil {
		t.Fatal(err)
	}
	if schema["additionalProperties"] != false {
		t.Fatal("system action schema must reject unknown properties")
	}
	properties, _ := schema["properties"].(map[string]any)
	validFields := map[string]bool{}
	typeInfo := reflect.TypeOf(contract.SystemActionRequest{})
	for index := 0; index < typeInfo.NumField(); index++ {
		name := strings.Split(typeInfo.Field(index).Tag.Get("json"), ",")[0]
		validFields[name] = true
	}
	for name := range properties {
		if !validFields[name] {
			t.Fatalf("system schema property %q is not accepted by SystemActionRequest", name)
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
