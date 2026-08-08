package panel

import (
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/kejilion/kejilion-panel/internal/ai"
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
		{name: "host_nginx_reload", arguments: `{"reason":"configuration was validated"}`, approval: false},
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
	normalized, err := normalizeToolArguments(json.RawMessage(`{"reason":"配置修改后验证通过"}`))
	if err != nil || string(normalized) != `{}` {
		t.Fatalf("normalize nginx reload arguments=%s err=%v", normalized, err)
	}
	method, path, target, body, err = tools.prepareWrite("host_nginx_reload", normalized)
	if err != nil || method != "POST" || path != "/v1/nginx/reload" || target != "nginx" || string(body) != `{}` {
		t.Fatalf("nginx reload method=%q path=%q target=%q body=%s err=%v", method, path, target, body, err)
	}
	if _, _, _, _, err := tools.prepareWrite("host_nginx_reload", json.RawMessage(`{"unknown":true}`)); err == nil {
		t.Fatal("unknown nginx reload field was accepted")
	}
	if summary := safeArgumentSummary(json.RawMessage(`{"reason":"configuration contains a secret"}`)); summary["reason"] != "[REDACTED]" {
		t.Fatalf("tool reason was retained in audit summary: %#v", summary)
	}
}

func TestAllAIToolSchemasExposeUniversalReasonMetadata(t *testing.T) {
	definitions := (&panelAITools{}).Definitions()
	if len(definitions) == 0 {
		t.Fatal("AI tool registry is empty")
	}
	for _, definition := range definitions {
		t.Run(definition.Name, func(t *testing.T) {
			var schema map[string]any
			if err := json.Unmarshal(definition.Schema, &schema); err != nil {
				t.Fatal(err)
			}
			if schema["type"] != "object" || schema["additionalProperties"] != false {
				t.Fatalf("tool schema boundary changed: %#v", schema)
			}
			properties, _ := schema["properties"].(map[string]any)
			reason, _ := properties["reason"].(map[string]any)
			if reason["type"] != "string" || reason["maxLength"] != float64(toolReasonMaxRunes) {
				t.Fatalf("universal reason metadata missing: %#v", reason)
			}
		})
	}
}

func TestToolArgumentNormalizationIsGeneralAndFailClosed(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{name: "empty", raw: `{}`, want: `{}`},
		{name: "read tool metadata", raw: `{"reason":"inspect host"}`, want: `{}`},
		{name: "write tool metadata", raw: `{"action":"cleanup","reason":"disk is full"}`, want: `{"action":"cleanup"}`},
		{name: "unicode limit", raw: `{"reason":"` + strings.Repeat("修", toolReasonMaxRunes) + `"}`, want: `{}`},
		{name: "wrong metadata type", raw: `{"reason":{"text":"why"}}`, wantErr: true},
		{name: "oversized metadata", raw: `{"reason":"` + strings.Repeat("修", toolReasonMaxRunes+1) + `"}`, wantErr: true},
		{name: "non object", raw: `[]`, wantErr: true},
		{name: "null", raw: `null`, wantErr: true},
		{name: "trailing JSON", raw: `{} {}`, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := normalizeToolArguments(json.RawMessage(test.raw))
			if (err != nil) != test.wantErr {
				t.Fatalf("normalize error=%v wantErr=%v", err, test.wantErr)
			}
			if !test.wantErr && string(got) != test.want {
				t.Fatalf("normalized=%s want=%s", got, test.want)
			}
		})
	}
}

func TestUniversalReasonMetadataDoesNotRelaxBusinessArguments(t *testing.T) {
	tools := &panelAITools{}
	if err := tools.DryRun("host_system_summary", json.RawMessage(`{"reason":"inspect"}`)); err != nil {
		t.Fatalf("read tool reason was rejected: %v", err)
	}
	if err := tools.DryRun("host_system_summary", json.RawMessage(`{"reason":"inspect","unknown":true}`)); err == nil {
		t.Fatal("unknown read tool field was accepted")
	}
	if err := tools.DryRun("host_system_action", json.RawMessage(`{"action":"cleanup","maintenancePolicy":"cache","reason":"free disk"}`)); err != nil {
		t.Fatalf("write tool reason was rejected: %v", err)
	}
	if err := tools.DryRun("host_system_action", json.RawMessage(`{"action":"cleanup","maintenancePolicy":"cache","unknown":true}`)); err == nil {
		t.Fatal("unknown write tool field was accepted")
	}
}

func TestZeroArgumentReadToolsTolerateOnlyBooleanPlaceholder(t *testing.T) {
	tools := &panelAITools{}
	for _, definition := range tools.Definitions() {
		if !zeroArgumentReadTool(definition.Name) {
			continue
		}
		t.Run(definition.Name, func(t *testing.T) {
			if err := tools.DryRun(definition.Name, json.RawMessage(`{"_":true}`)); err != nil {
				t.Fatalf("boolean placeholder was rejected: %v", err)
			}
		})
	}
	if err := tools.DryRun("host_system_summary", json.RawMessage(`{"_":"true"}`)); err == nil {
		t.Fatal("non-boolean placeholder was accepted")
	}
	if err := tools.DryRun("host_file_read", json.RawMessage(`{"path":"/tmp/test","_":true}`)); err == nil {
		t.Fatal("placeholder relaxed a read tool with business arguments")
	}
	if err := tools.DryRun("host_nginx_reload", json.RawMessage(`{"_":true}`)); err == nil {
		t.Fatal("placeholder relaxed a write tool")
	}
}

func TestRecoverableAgentToolStatusBoundary(t *testing.T) {
	for _, status := range []int{400, 403, 404, 413, 422} {
		if !recoverableAgentToolStatus(status) {
			t.Errorf("business status %d was not recoverable", status)
		}
	}
	for _, status := range []int{401, 408, 409, 412, 429, 500, 503} {
		if recoverableAgentToolStatus(status) {
			t.Errorf("infrastructure/control status %d was recoverable", status)
		}
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
	branches, _ := schema["anyOf"].([]any)
	if len(branches) == 0 {
		t.Fatal("maintenance schema has no action-specific anyOf branches")
	}
	for _, spec := range dockerMaintenanceActionSpecs {
		for _, expectedAction := range spec.Actions {
			found := false
			for _, rawBranch := range branches {
				branch, _ := rawBranch.(map[string]any)
				branchProperties, _ := branch["properties"].(map[string]any)
				actionProperty, _ := branchProperties["action"].(map[string]any)
				branchActions, _ := actionProperty["enum"].([]any)
				for _, action := range branchActions {
					if action == expectedAction {
						found = true
					}
				}
			}
			if !found {
				t.Fatalf("maintenance action %q has no conditional schema branch", expectedAction)
			}
		}
	}
	validFields := map[string]bool{}
	inputType := reflect.TypeOf(dockerx.MaintenanceInput{})
	for index := 0; index < inputType.NumField(); index++ {
		name := strings.Split(inputType.Field(index).Tag.Get("json"), ",")[0]
		validFields[name] = true
	}
	for name := range properties {
		if name == "reason" {
			continue
		}
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
	if _, _, _, _, err := tools.prepareWrite("host_docker_task", json.RawMessage(`{"action":"image_remove","image":"example/app:old"}`)); err == nil ||
		!strings.Contains(err.Error(), "target, expectedResourceVersion") ||
		!strings.Contains(err.Error(), "host_docker_images") {
		t.Fatalf("image removal contract error = %v", err)
	}
}

func TestDockerMaintenanceRequirementsCoverEveryAction(t *testing.T) {
	version := "sha256:" + strings.Repeat("0", 64)
	containerID := strings.Repeat("a", 64)
	inputs := map[string]map[string]any{
		"container_create":   {"image": "nginx:alpine", "name": "web"},
		"container_access":   {"target": containerID, "expectedResourceVersion": version, "enabled": false},
		"image_pull":         {"image": "nginx:alpine"},
		"image_remove":       {"target": strings.Repeat("b", 64), "expectedResourceVersion": version},
		"network_create":     {"name": "frontend"},
		"network_remove":     {"target": "frontend", "expectedResourceVersion": version},
		"network_connect":    {"target": "frontend", "expectedResourceVersion": version, "containerId": containerID, "containerResourceVersion": version},
		"network_disconnect": {"target": "frontend", "expectedResourceVersion": version, "containerId": containerID, "containerResourceVersion": version},
		"volume_create":      {"name": "data"},
		"volume_remove":      {"target": "data", "expectedResourceVersion": version},
		"prune":              {},
		"container_prune":    {},
		"image_prune":        {},
		"network_prune":      {},
		"volume_prune":       {},
		"backup_create":      {},
		"backup_restore":     {"backupId": "docker-20260809T010203Z-abcdef01.tar.gz"},
		"backup_migrate":     {"backupId": "docker-20260809T010203Z-abcdef01.tar.gz", "migrationHost": "backup.example.com", "migrationUser": "root", "migrationPort": 22},
		"daemon_mirror":      {"preset": "cn"},
		"daemon_ipv6":        {"enabled": true, "ipv6Cidr": "fd42:6b50:616e:656c::/64"},
	}
	tools := &panelAITools{}
	for _, action := range dockerx.MaintenanceActions() {
		arguments, ok := inputs[action]
		if !ok {
			t.Fatalf("test contract missing action %q", action)
		}
		arguments["action"] = action
		raw, err := json.Marshal(arguments)
		if err != nil {
			t.Fatal(err)
		}
		if _, _, _, _, err := tools.prepareWrite("host_docker_task", raw); err != nil {
			t.Fatalf("valid %s arguments were rejected: %v", action, err)
		}
		spec, ok := dockerMaintenanceSpec(action)
		if !ok {
			t.Fatalf("action %q has no requirements spec", action)
		}
		for _, required := range spec.Required {
			copyArguments := make(map[string]any, len(arguments))
			for key, value := range arguments {
				copyArguments[key] = value
			}
			delete(copyArguments, required)
			raw, _ := json.Marshal(copyArguments)
			if _, _, _, _, err := tools.prepareWrite("host_docker_task", raw); err == nil {
				t.Fatalf("action %q accepted missing required field %q", action, required)
			}
		}
	}
	withoutCIDR, _ := json.Marshal(map[string]any{"action": "daemon_ipv6", "enabled": true})
	if _, _, _, _, err := tools.prepareWrite("host_docker_task", withoutCIDR); err == nil {
		t.Fatal("daemon_ipv6 enabled without ipv6Cidr was accepted")
	}
	disabledIPv6, _ := json.Marshal(map[string]any{"action": "daemon_ipv6", "enabled": false})
	if _, _, _, _, err := tools.prepareWrite("host_docker_task", disabledIPv6); err != nil {
		t.Fatalf("daemon_ipv6 disable without ipv6Cidr was rejected: %v", err)
	}
	staleShape, _ := json.Marshal(map[string]any{"action": "image_remove", "target": strings.Repeat("b", 64), "expectedResourceVersion": "stale"})
	if _, _, _, _, err := tools.prepareWrite("host_docker_task", staleShape); err == nil ||
		!strings.Contains(err.Error(), "current expectedResourceVersion") {
		t.Fatalf("malformed image resourceVersion error = %v", err)
	}
}

func TestDockerAgentProblemsAreReplannedByMeaning(t *testing.T) {
	if err := classifyAgentToolProblem("host_docker_task", contract.Problem{Code: "docker_task_invalid"}); !errors.Is(err, ai.ErrToolArguments) {
		t.Fatalf("docker_task_invalid classification = %v", err)
	}
	if err := classifyAgentToolProblem("host_docker_task", contract.Problem{Code: "docker_resource_not_found"}); !errors.Is(err, ai.ErrToolConflict) {
		t.Fatalf("docker_resource_not_found classification = %v", err)
	}
	if err := classifyAgentToolProblem("host_file_write", contract.Problem{Code: "docker_task_invalid"}); err != nil {
		t.Fatalf("unrelated tool problem was reclassified: %v", err)
	}
}
