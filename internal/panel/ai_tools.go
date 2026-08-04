package panel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kejilion/kejilion-panel/internal/ai"
	"github.com/kejilion/kejilion-panel/internal/contract"
	"github.com/kejilion/kejilion-panel/internal/store"
)

type panelAITools struct{ server *Server }

func (t *panelAITools) DryRun(name string, arguments json.RawMessage) error {
	for _, definition := range t.Definitions() {
		if definition.Name == name {
			if definition.ReadOnly {
				return nil
			}
			_, _, _, _, err := t.prepareWrite(name, arguments)
			return err
		}
	}
	return errors.New("unknown KPanel tool")
}

func (t *panelAITools) Definitions() []ai.ToolDefinition {
	readSchema := json.RawMessage(`{"type":"object","additionalProperties":false}`)
	return []ai.ToolDefinition{
		{Name: "host_system_summary", Description: "读取宿主机系统概览与 resourceVersion", Schema: readSchema, ReadOnly: true},
		{Name: "host_public_network", Description: "读取宿主机公网网络信息", Schema: readSchema, ReadOnly: true},
		{Name: "host_sites_list", Description: "读取网站列表与 resourceVersion", Schema: readSchema, ReadOnly: true},
		{Name: "host_apps_list", Description: "读取应用列表、状态与 resourceVersion", Schema: readSchema, ReadOnly: true},
		{Name: "host_diagnostics_list", Description: "读取可用诊断检查", Schema: readSchema, ReadOnly: true},
		{Name: "host_docker_summary", Description: "读取 Docker 概览", Schema: readSchema, ReadOnly: true},
		{Name: "host_docker_containers", Description: "读取 Docker 容器与 resourceVersion", Schema: readSchema, ReadOnly: true},
		{Name: "host_docker_images", Description: "读取 Docker 镜像", Schema: readSchema, ReadOnly: true},
		{Name: "host_docker_networks", Description: "读取 Docker 网络", Schema: readSchema, ReadOnly: true},
		{Name: "host_docker_volumes", Description: "读取 Docker 卷", Schema: readSchema, ReadOnly: true},
		{Name: "host_docker_jobs", Description: "读取 Docker 后台任务", Schema: readSchema, ReadOnly: true},
		{Name: "host_system_action", Description: "执行 KPanel 已注册的结构化系统操作", Schema: json.RawMessage(`{"type":"object","required":["action"],"properties":{"action":{"type":"string"},"hostname":{"type":"string"},"port":{"type":"integer"},"servers":{"type":"array","items":{"type":"string"}},"hostsOperation":{"type":"string"},"hostsEntry":{"type":"string"},"resourceVersion":{"type":"string"}},"additionalProperties":true}`)},
		{Name: "host_app_action", Description: "安装、启停、更新或卸载已注册应用", Schema: json.RawMessage(`{"type":"object","required":["appId","action"],"properties":{"appId":{"type":"string"},"action":{"enum":["install","start","stop","restart","check_update","update","uninstall","direct_access"]},"hostPort":{"type":"integer"},"accessMode":{"type":"string"},"resourceVersion":{"type":"string"}},"additionalProperties":false}`)},
		{Name: "host_docker_container_action", Description: "启停、重启或删除容器", Schema: json.RawMessage(`{"type":"object","required":["containerId","action","resourceVersion"],"properties":{"containerId":{"type":"string"},"action":{"enum":["start","stop","restart","remove"]},"resourceVersion":{"type":"string"}},"additionalProperties":false}`)},
		{Name: "host_diagnostic_start", Description: "启动固定诊断检查并返回任务 ID", Schema: json.RawMessage(`{"type":"object","required":["checkId"],"properties":{"checkId":{"type":"string"}},"additionalProperties":false}`)},
		{Name: "host_docker_task", Description: "启动 KPanel 已支持的结构化 Docker 任务", Schema: json.RawMessage(`{"type":"object","required":["type"],"properties":{"type":{"type":"string"}},"additionalProperties":true}`)},
		{Name: "host_site_change", Description: "创建、更新或删除网站配置", Schema: json.RawMessage(`{"type":"object","required":["operation"],"properties":{"operation":{"enum":["create","update","delete"]},"siteId":{"type":"string"},"payload":{"type":"object"}},"additionalProperties":false}`)},
		{Name: "host_job_input", Description: "向已存在的 KPanel 交互任务提交输入", Schema: json.RawMessage(`{"type":"object","required":["kind","jobId","data"],"properties":{"kind":{"enum":["site","app","diagnostic"]},"jobId":{"type":"string"},"data":{"type":"string","maxLength":16384}},"additionalProperties":false}`)},
		{Name: "host_docker_exec", Description: "在指定 Docker 容器内执行受 Agent 校验的命令", Schema: json.RawMessage(`{"type":"object","required":["containerId","resourceVersion","command"],"properties":{"containerId":{"type":"string"},"resourceVersion":{"type":"string"},"command":{"type":"array","items":{"type":"string"},"maxItems":32}},"additionalProperties":false}`)},
	}
}

func (t *panelAITools) Execute(ctx context.Context, execution ai.ToolExecutionContext, name string, arguments json.RawMessage) (string, error) {
	if t == nil || t.server == nil {
		return "", errors.New("panel tool service is unavailable")
	}
	readPaths := map[string]string{
		"host_system_summary": "/v1/system/summary", "host_public_network": "/v1/system/public-network", "host_sites_list": "/v1/sites",
		"host_apps_list": "/v1/apps", "host_diagnostics_list": "/v1/diagnostics", "host_docker_summary": "/v1/docker/summary",
		"host_docker_containers": "/v1/docker/containers", "host_docker_images": "/v1/docker/images", "host_docker_networks": "/v1/docker/networks",
		"host_docker_volumes": "/v1/docker/volumes", "host_docker_jobs": "/v1/docker/jobs",
	}
	requestID := newRequestID()
	if path := readPaths[name]; path != "" {
		response, err := t.server.hostOps.Get(ctx, path, "", requestID)
		return t.finish(execution, name, path, requestID, nil, response, err)
	}
	method, path, target, body, err := t.prepareWrite(name, arguments)
	if err != nil {
		return "", err
	}
	change := map[string]any{"tool": name, "target": target, "sessionId": execution.SessionID, "runId": execution.RunID, "toolCallId": execution.ToolCallID, "arguments": safeArgumentSummary(arguments)}
	if err := t.server.store.AppendAudit(store.AuditEvent{ID: newRequestID(), OccurredAt: time.Now().UTC(), ActorType: "user", ActorID: execution.UserID, Action: "ai.tool." + name, TargetKind: "host_operation", TargetID: target, Result: "intent", RequestID: requestID, Change: change}, 10_000); err != nil {
		return "", errors.New("AI tool audit unavailable")
	}
	response, callErr := t.server.hostOps.Do(ctx, method, path, "", requestID, body)
	return t.finish(execution, name, target, requestID, change, response, callErr)
}

func (t *panelAITools) prepareWrite(name string, raw json.RawMessage) (method, path, target string, body []byte, err error) {
	switch name {
	case "host_system_action":
		var input contract.SystemActionRequest
		if err = json.Unmarshal(raw, &input); err != nil {
			return
		}
		if field, detail := validateSystemAction(&input); field != "" {
			err = fmt.Errorf("%s: %s", field, detail)
			return
		}
		method, path, target = http.MethodPost, "/v1/system/actions", input.Action
		body, err = json.Marshal(input)
	case "host_app_action":
		var input struct {
			AppID, Action               string
			HostPort                    *int `json:"hostPort"`
			AccessMode, ResourceVersion *string
		}
		if err = json.Unmarshal(raw, &input); err != nil {
			return
		}
		if !appIDPattern.MatchString(input.AppID) {
			err = errors.New("invalid appId")
			return
		}
		public := "/api/v1/apps/" + input.AppID + "/" + input.Action
		var ok bool
		path, _, _, ok = allowedAppActionPath(public)
		if !ok {
			err = errors.New("unsupported app action")
			return
		}
		payload := map[string]any{}
		if input.HostPort != nil {
			payload["hostPort"] = *input.HostPort
		}
		if input.AccessMode != nil {
			payload["accessMode"] = *input.AccessMode
		}
		if input.ResourceVersion != nil {
			payload["resourceVersion"] = *input.ResourceVersion
		}
		method, target = http.MethodPost, input.AppID
		body, err = json.Marshal(payload)
	case "host_docker_container_action":
		var input struct{ ContainerID, Action, ResourceVersion string }
		if err = json.Unmarshal(raw, &input); err != nil {
			return
		}
		public := "/api/v1/docker/containers/" + input.ContainerID + "/" + input.Action
		var ok bool
		path, target, _, ok = allowedDockerActionPath(public)
		if !ok || !resourceVersionPattern.MatchString(input.ResourceVersion) {
			err = errors.New("invalid container action or resourceVersion")
			return
		}
		method = http.MethodPost
		body, err = json.Marshal(map[string]string{"resourceVersion": input.ResourceVersion})
	case "host_diagnostic_start":
		var input struct {
			CheckID string `json:"checkId"`
		}
		if err = json.Unmarshal(raw, &input); err != nil {
			return
		}
		if !diagnosticCheckIDPattern.MatchString(input.CheckID) {
			err = errors.New("invalid checkId")
			return
		}
		method, path, target, body = http.MethodPost, "/v1/diagnostic-jobs", input.CheckID, raw
	case "host_docker_task":
		method, path, target, body = http.MethodPost, "/v1/docker/tasks", "docker-task", raw
	case "host_site_change":
		var input struct {
			Operation, SiteID string
			Payload           json.RawMessage
		}
		if err = json.Unmarshal(raw, &input); err != nil {
			return
		}
		switch input.Operation {
		case "create":
			method, path, target = http.MethodPost, "/v1/sites", "new-site"
		case "update":
			if !siteIDPattern.MatchString(input.SiteID) {
				err = errors.New("invalid siteId")
				return
			}
			method, path, target = http.MethodPatch, "/v1/sites/"+input.SiteID, input.SiteID
		case "delete":
			if !siteIDPattern.MatchString(input.SiteID) {
				err = errors.New("invalid siteId")
				return
			}
			method, path, target = http.MethodDelete, "/v1/sites/"+input.SiteID, input.SiteID
		default:
			err = errors.New("unsupported site operation")
			return
		}
		body = input.Payload
	case "host_job_input":
		var input struct{ Kind, JobID, Data string }
		if err = json.Unmarshal(raw, &input); err != nil {
			return
		}
		if !siteIDPattern.MatchString(input.JobID) || input.Data == "" || len(input.Data) > 16<<10 || strings.IndexByte(input.Data, 0) >= 0 {
			err = errors.New("invalid job input")
			return
		}
		prefixes := map[string]string{"site": "/v1/site-installations/", "app": "/v1/app-jobs/", "diagnostic": "/v1/diagnostic-jobs/"}
		prefix := prefixes[input.Kind]
		if prefix == "" {
			err = errors.New("invalid job kind")
			return
		}
		method, path, target = http.MethodPost, prefix+input.JobID+"/input", input.JobID
		body, err = json.Marshal(map[string]string{"data": input.Data})
	case "host_docker_exec":
		var input struct {
			ContainerID     string   `json:"containerId"`
			ResourceVersion string   `json:"resourceVersion"`
			Command         []string `json:"command"`
		}
		if err = json.Unmarshal(raw, &input); err != nil {
			return
		}
		if !containerIDPattern.MatchString(input.ContainerID) || !resourceVersionPattern.MatchString(input.ResourceVersion) || len(input.Command) == 0 || len(input.Command) > 32 {
			err = errors.New("invalid Docker exec request")
			return
		}
		method, path, target = http.MethodPost, "/v1/docker/containers/"+input.ContainerID+"/exec", input.ContainerID
		body, err = json.Marshal(map[string]any{"resourceVersion": input.ResourceVersion, "command": input.Command})
	default:
		err = errors.New("unknown KPanel tool")
	}
	return
}

func (t *panelAITools) finish(execution ai.ToolExecutionContext, name, target, requestID string, change map[string]any, response AgentResponse, callErr error) (string, error) {
	result := "success"
	if callErr != nil || response.StatusCode < 200 || response.StatusCode >= 300 {
		result = "failure"
	}
	if change == nil {
		change = map[string]any{"tool": name, "target": target, "sessionId": execution.SessionID, "runId": execution.RunID, "toolCallId": execution.ToolCallID}
	}
	_ = t.server.store.AppendAudit(store.AuditEvent{ID: newRequestID(), OccurredAt: time.Now().UTC(), ActorType: "user", ActorID: execution.UserID, Action: "ai.tool." + name, TargetKind: "host_operation", TargetID: target, Result: result, RequestID: requestID, Change: change}, 10_000)
	if callErr != nil {
		return "", errors.New("Agent unavailable")
	}
	if result == "failure" {
		if response.StatusCode == http.StatusConflict || response.StatusCode == http.StatusPreconditionFailed {
			return "", fmt.Errorf("%w: Agent returned HTTP %d", ai.ErrToolConflict, response.StatusCode)
		}
		return "", fmt.Errorf("Agent returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(response.Body)))
	}
	return string(response.Body), nil
}

func safeArgumentSummary(raw json.RawMessage) map[string]any {
	var value map[string]any
	if json.Unmarshal(raw, &value) != nil {
		return map[string]any{"valid": false}
	}
	for _, key := range []string{"data", "password", "token", "apiKey", "key", "command"} {
		if _, ok := value[key]; ok {
			value[key] = "[REDACTED]"
		}
	}
	return value
}
