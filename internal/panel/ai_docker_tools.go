package panel

import (
	"encoding/json"
	"strings"

	"github.com/kejilion/kejilion-panel/internal/dockerx"
)

type dockerMaintenanceActionHint struct {
	Actions     []string
	ReadBefore  string
	Description string
}

var dockerMaintenanceActionHints = []dockerMaintenanceActionHint{
	{Actions: []string{"container_create"}, Description: "创建容器时按需提供 image、name、ports、mounts、environment 等字段"},
	{Actions: []string{"compose_deploy"}, Description: "使用用户提供的 Compose YAML 在 /home/docker 下创建并启动新项目"},
	{Actions: []string{"container_access"}, ReadBefore: "host_docker_containers", Description: "修改容器外部访问时使用当前容器状态"},
	{Actions: []string{"image_pull"}, Description: "拉取模型选择的完整镜像引用"},
	{Actions: []string{"image_remove"}, ReadBefore: "host_docker_images", Description: "精确删除模型和用户选定的镜像，不得改用 image_prune"},
	{Actions: []string{"network_create"}, Description: "创建用户需要的网络"},
	{Actions: []string{"network_remove"}, ReadBefore: "host_docker_networks", Description: "删除模型和用户选定的网络"},
	{Actions: []string{"network_connect", "network_disconnect"}, ReadBefore: "host_docker_networks and host_docker_containers", Description: "按真实网络和容器状态修改网络成员"},
	{Actions: []string{"volume_create"}, Description: "创建用户需要的卷"},
	{Actions: []string{"volume_remove"}, ReadBefore: "host_docker_volumes", Description: "删除模型和用户选定的卷"},
	{Actions: []string{"prune", "container_prune", "image_prune", "network_prune", "volume_prune"}, Description: "清理未使用资源；image_prune 只清理悬空镜像，不能代替 image_remove"},
	{Actions: []string{"backup_create"}, Description: "创建 Docker 备份"},
	{Actions: []string{"backup_restore"}, ReadBefore: "host_docker_backups", Description: "恢复模型和用户选定的备份"},
	{Actions: []string{"backup_migrate"}, ReadBefore: "host_docker_backups", Description: "迁移模型和用户选定的备份，目标由用户提供"},
	{Actions: []string{"daemon_mirror"}, ReadBefore: "host_docker_environment", Description: "在 cn 和 official 镜像源之间选择"},
	{Actions: []string{"daemon_ipv6"}, ReadBefore: "host_docker_environment", Description: "按用户需求启用或停用 Docker IPv6"},
}

var dockerMaintenanceToolSchema = buildDockerMaintenanceToolSchema()

func buildDockerMaintenanceToolSchema() json.RawMessage {
	var document map[string]any
	if err := json.Unmarshal(dockerMaintenanceToolBaseSchema, &document); err != nil {
		panic("invalid Docker maintenance tool schema: " + err.Error())
	}
	properties := document["properties"].(map[string]any)
	properties["action"] = map[string]any{
		"type":        "string",
		"enum":        dockerx.MaintenanceActions(),
		"description": dockerMaintenanceActionGuide(),
	}
	properties["image"] = map[string]any{
		"type":        "string",
		"description": "镜像引用；用于 container_create 或 image_pull",
	}
	properties["target"] = map[string]any{
		"type":        "string",
		"description": "从对应只读工具返回值复制的资源 id/name；删除镜像优先使用完整 image id",
	}
	properties["expectedResourceVersion"] = map[string]any{
		"type":        "string",
		"description": "对应只读工具刚返回的 resourceVersion，不得猜测或复用旧值",
	}
	properties["containerResourceVersion"] = map[string]any{
		"type":        "string",
		"description": "host_docker_containers 刚返回的容器 resourceVersion",
	}
	properties["preset"] = map[string]any{"type": "string", "enum": []string{"cn", "official"}}
	properties["migrationPort"] = map[string]any{"type": "integer"}
	encoded, err := json.Marshal(document)
	if err != nil {
		panic("encode Docker maintenance tool schema: " + err.Error())
	}
	return encoded
}

func dockerMaintenanceActionGuide() string {
	items := make([]string, 0, len(dockerMaintenanceActionHints))
	for _, hint := range dockerMaintenanceActionHints {
		item := strings.Join(hint.Actions, "/") + ": " + hint.Description
		if hint.ReadBefore != "" {
			item += "，先调用 " + hint.ReadBefore
		}
		items = append(items, item)
	}
	return "选择一个 Docker 维护动作并根据真实状态自主填写相关字段；KPanel 只固定动作入口、鉴权、审批和审计，底层技术前置条件由 Agent 校验并作为可纠正结果返回。" + strings.Join(items, "；")
}
