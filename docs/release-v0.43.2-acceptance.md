# KPanel v0.43.2 发布验收

## 发布范围

v0.43.2 将模型工具调用兼容从 Nginx 专项补丁收敛为通用行为层：所有注册工具统一声明可选 `reason` 元数据，并在业务校验、审批分类、审计和 Agent 调用前剥离。其他未知字段继续严格拒绝；可纠正的本地参数错误在审批和宿主机执行前拦截，并交还模型在当前轮按 Schema 重新规划。

AI 工作台把工具卡片明确作为执行过程，按调用顺序稳定显示在同一 Run 的最终文字结论之前；卡片状态更新不再改变位置，重新打开会话时恢复最近一轮过程。

## 版本与制品

- 发布提交：`5219a2156f2a9d698afab98fe132ec96ec9175ee`
- 标签：`v0.43.2`
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.43.2>
- main CI：<https://github.com/kejilion/KPanel/actions/runs/30899232084>
- Release 工作流：<https://github.com/kejilion/KPanel/actions/runs/30899456374>
- Docker 多架构摘要：`sha256:c5dea64c64a435ca32d09d46a0f74ba076c3b11f31c87cf65b72c64a643834c2`
- linux/amd64：`sha256:1bb608de1e84fa14c250593cb3a4f9cf76cff2947f1ccebd4dce613cdd3cebab`
- linux/arm64：`sha256:13ff29098e985440dc3e9a3b5eed5169e32585967b16bc16221da4a4999bc528`

## 行为与安全验收

- 工具注册表契约测试逐项确认所有 Schema 均保留 `additionalProperties: false`，同时声明最多 500 Unicode 字符的可选 `reason`。
- 读工具、写工具和 `host_nginx_reload` 集成测试确认 `reason` 在统一入口被剥离；Agent 收到的 Nginx reload 请求仍为固定 `{}`，审计不含说明内容。
- 无参只读工具与原先使用宽松反序列化的写工具统一改为严格业务字段校验；`reason` 之外的未知字段不能到达 Agent。
- Runtime 在审批前执行 dry-run。参数错误只生成失败过程卡片和模型纠错上下文，不请求用户批准、不执行宿主机操作；网络失败、资源版本冲突和写入结果不确定仍不自动重放。
- 工具结果继续以不可信数据进入模型，原有受保护路径、resourceVersion、审批和固定 Host Operation 边界不变。

## 自动化验收

- 前端 typecheck、i18n 和生产构建通过；Vitest `38` 个文件、`248` 项测试通过。
- Linux `go test ./...` 与 `go vet ./...` 通过；`internal/ai`、`internal/panel`、`internal/auth`、`internal/dockerx` 的 race 测试通过。
- linux/amd64 与 linux/arm64 的 Panel、Agent、Node、kpctl 均完成 CGO-free 构建。
- Trivy 源码、依赖、Secret、Dockerfile 和发布镜像 HIGH/CRITICAL 扫描通过；govulncheck 可达漏洞为 0，npm audit 漏洞为 0。
- 版本一致性、生态策略和部署安全检查通过；本次没有数据库迁移、依赖、Compose、端口或应用市场安装契约变化。

## 154 生产上线

- 升级前 Panel 为 `0.43.1/ok`，容器 healthy、重启数 0，Agent active。
- 一致性备份：`/root/kpanel-backups/v0.43.2-preupgrade-20260804T101512Z/kpanel-home.tar.gz`
- 备份权限 `0600`，大小 `20,624,081` B，SHA-256：`a5fc9f632f02b4f69954b3e58e78bcba366813d90b1334a7be656879a4106e89`。
- 使用 `KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update k app kpanel` 完成原生应用市场更新；Panel 与 Agent 均为 `0.43.2 v1alpha1`。
- AI SQLite `integrity_check=ok`，已有 6 个迁移；`ai.db*` 与 `ai-secrets.key` 权限均为 `0600`。
- 连续 10 分钟、每 30 秒一次共 20 次采样均为 `0.43.2/ok`、Panel healthy、Agent active，二者重启数均为 0；最终 RSS 为 `12.95 MiB / 256 MiB`。
- 本机与公共 IP 的 `/`、`/ai` 均返回 200；升级后 15 分钟 Panel/Agent 日志未发现 panic、fatal 或 error。
- 运行边界保持 `65532:65532`、只读根文件系统、`privileged=false`、256 MiB 内存和 128 PIDs。

## 回滚与边界

- 公共镜像回滚点为 v0.43.1：`sha256:854d7f542e54b62cd7cc1aeb998b364e1da14d6743a3f2d6fdc86f3a05903ab2`。
- 现场回滚使用上述升级前备份，恢复时必须成对保留 `ai.db*` 与 `ai-secrets.key`，并恢复旧镜像、Agent、Compose 和 systemd 单元。
- 本轮未使用用户曾在聊天中发送的 API Key，也未调用真实付费模型。协议和通用行为由单元、契约、Stub Agent 与发布镜像测试覆盖，不把它表述为某个真实模型质量验收。
- `https://kpanel.kejilion.eu.org` 黑盒复核仍为 `0.43.1/ok`，`/` 与 `/ai` 返回 200；当前没有该主机的登录或 SSH 权限，因此没有替它执行升级。
- 清理了 154 上一条已失去测试容器、持续重试约 4.9 小时的旧验收监控进程；未删除业务容器或业务数据。
