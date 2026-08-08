package panel

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/kejilion/kejilion-panel/internal/dockerx"
)

type dockerMaintenanceActionSpec struct {
	Actions             []string
	Required            []string
	RequiredWhenEnabled []string
	ReadBefore          string
	Description         string
}

var dockerMaintenanceActionSpecs = []dockerMaintenanceActionSpec{
	{Actions: []string{"container_create"}, Required: []string{"image", "name"}, Description: "创建容器；image 和 name 必填"},
	{Actions: []string{"container_access"}, Required: []string{"target", "expectedResourceVersion", "enabled"}, ReadBefore: "host_docker_containers", Description: "修改容器外部访问；使用最新容器 id 和 resourceVersion"},
	{Actions: []string{"image_pull"}, Required: []string{"image"}, Description: "拉取镜像；image 必须是完整镜像引用"},
	{Actions: []string{"image_remove"}, Required: []string{"target", "expectedResourceVersion"}, ReadBefore: "host_docker_images", Description: "删除指定镜像；使用最新镜像 id 和 resourceVersion，不得改用 image_prune"},
	{Actions: []string{"network_create"}, Required: []string{"name"}, Description: "创建网络；name 必填"},
	{Actions: []string{"network_remove"}, Required: []string{"target", "expectedResourceVersion"}, ReadBefore: "host_docker_networks", Description: "删除网络；使用最新网络 id/name 和 resourceVersion"},
	{Actions: []string{"network_connect", "network_disconnect"}, Required: []string{"target", "expectedResourceVersion", "containerId", "containerResourceVersion"}, ReadBefore: "host_docker_networks and host_docker_containers", Description: "修改网络成员；同时使用最新网络和容器 resourceVersion"},
	{Actions: []string{"volume_create"}, Required: []string{"name"}, Description: "创建卷；name 必填"},
	{Actions: []string{"volume_remove"}, Required: []string{"target", "expectedResourceVersion"}, ReadBefore: "host_docker_volumes", Description: "删除卷；使用最新卷 name 和 resourceVersion"},
	{Actions: []string{"prune", "container_prune", "image_prune", "network_prune", "volume_prune"}, Description: "清理未使用资源；image_prune 只清理悬空镜像，不能代替 image_remove"},
	{Actions: []string{"backup_create"}, Description: "创建 Docker 备份"},
	{Actions: []string{"backup_restore"}, Required: []string{"backupId"}, ReadBefore: "host_docker_backups", Description: "恢复已有备份；使用备份列表中的 backupId"},
	{Actions: []string{"backup_migrate"}, Required: []string{"backupId", "migrationHost", "migrationUser", "migrationPort"}, ReadBefore: "host_docker_backups", Description: "迁移已有备份；目标主机、用户和端口必须由用户明确提供"},
	{Actions: []string{"daemon_mirror"}, Required: []string{"preset"}, ReadBefore: "host_docker_environment", Description: "修改镜像源；preset 只能为 cn 或 official"},
	{Actions: []string{"daemon_ipv6"}, Required: []string{"enabled"}, RequiredWhenEnabled: []string{"ipv6Cidr"}, ReadBefore: "host_docker_environment", Description: "修改 Docker IPv6；启用时还必须提供 ipv6Cidr"},
}

var dockerMaintenanceToolSchema = buildDockerMaintenanceToolSchema()

func buildDockerMaintenanceToolSchema() json.RawMessage {
	var document map[string]any
	if err := json.Unmarshal(dockerMaintenanceToolBaseSchema, &document); err != nil {
		panic("invalid Docker maintenance tool schema: " + err.Error())
	}
	properties := document["properties"].(map[string]any)
	actions := make([]string, 0, len(dockerx.MaintenanceActions()))
	branches := make([]any, 0, len(dockerMaintenanceActionSpecs)+1)
	for _, spec := range dockerMaintenanceActionSpecs {
		actions = append(actions, spec.Actions...)
		if len(spec.RequiredWhenEnabled) == 0 {
			branches = append(branches, dockerMaintenanceSchemaBranch(spec, nil))
			continue
		}
		disabled, enabled := false, true
		branches = append(branches, dockerMaintenanceSchemaBranch(spec, &disabled), dockerMaintenanceSchemaBranch(spec, &enabled))
	}
	properties["action"] = map[string]any{
		"type":        "string",
		"enum":        actions,
		"description": "Docker 维护动作；不同动作必须满足 anyOf 中对应的必填字段",
	}
	properties["image"] = map[string]any{
		"type": "string", "minLength": 1, "maxLength": 383,
		"description": "镜像引用；用于 container_create 或 image_pull",
	}
	properties["target"] = map[string]any{
		"type": "string", "minLength": 1, "maxLength": 383,
		"description": "从对应只读工具返回值复制的资源 id/name；删除镜像优先使用完整 image id",
	}
	properties["expectedResourceVersion"] = map[string]any{
		"type": "string", "pattern": "^sha256:[a-f0-9]{64}$",
		"description": "对应只读工具刚返回的 resourceVersion，不得猜测或复用旧值",
	}
	properties["containerResourceVersion"] = map[string]any{
		"type": "string", "pattern": "^sha256:[a-f0-9]{64}$",
	}
	properties["preset"] = map[string]any{"enum": []string{"cn", "official"}}
	properties["migrationPort"] = map[string]any{"type": "integer", "minimum": 1, "maximum": 65535}
	document["anyOf"] = branches
	encoded, err := json.Marshal(document)
	if err != nil {
		panic("encode Docker maintenance tool schema: " + err.Error())
	}
	return encoded
}

func dockerMaintenanceSchemaBranch(spec dockerMaintenanceActionSpec, enabled *bool) map[string]any {
	required := append([]string{"action"}, spec.Required...)
	properties := map[string]any{"action": map[string]any{"enum": spec.Actions}}
	description := spec.Description
	if spec.ReadBefore != "" {
		description += "; 先调用 " + spec.ReadBefore
	}
	if enabled != nil {
		properties["enabled"] = map[string]any{"enum": []bool{*enabled}}
		if *enabled {
			required = append(required, spec.RequiredWhenEnabled...)
		}
	}
	return map[string]any{"description": description, "properties": properties, "required": required}
}

func validateDockerMaintenanceRequirements(raw json.RawMessage, action string) error {
	var arguments map[string]json.RawMessage
	if err := json.Unmarshal(raw, &arguments); err != nil {
		return err
	}
	spec, ok := dockerMaintenanceSpec(action)
	if !ok {
		return errors.New("unsupported Docker maintenance action")
	}
	required := append([]string(nil), spec.Required...)
	if len(spec.RequiredWhenEnabled) > 0 {
		var enabled bool
		if value, exists := arguments["enabled"]; exists && json.Unmarshal(value, &enabled) == nil && enabled {
			required = append(required, spec.RequiredWhenEnabled...)
		}
	}
	missing := make([]string, 0, len(required))
	for _, field := range required {
		if !meaningfulDockerToolArgument(arguments[field]) {
			missing = append(missing, field)
		}
	}
	if len(missing) > 0 {
		message := "Docker action " + action + " requires non-empty arguments: " + joinToolFields(missing)
		if spec.ReadBefore != "" {
			message += "; read current values with " + spec.ReadBefore + " before retrying"
		}
		return errors.New(message)
	}
	for _, field := range []string{"expectedResourceVersion", "containerResourceVersion"} {
		if value, required := arguments[field]; required {
			var version string
			if json.Unmarshal(value, &version) != nil || !resourceVersionPattern.MatchString(version) {
				return errors.New("Docker action " + action + " requires a current " + field + " from the matching read tool")
			}
		}
	}
	return nil
}

func dockerMaintenanceSpec(action string) (dockerMaintenanceActionSpec, bool) {
	for _, spec := range dockerMaintenanceActionSpecs {
		for _, candidate := range spec.Actions {
			if action == candidate {
				return spec, true
			}
		}
	}
	return dockerMaintenanceActionSpec{}, false
}

func meaningfulDockerToolArgument(value json.RawMessage) bool {
	if len(value) == 0 || string(value) == "null" {
		return false
	}
	var text string
	if json.Unmarshal(value, &text) == nil {
		return strings.TrimSpace(text) != ""
	}
	return true
}

func joinToolFields(fields []string) string {
	return strings.Join(fields, ", ")
}
