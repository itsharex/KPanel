# KPanel v0.44.0 发布验收

## 发布范围

v0.44.0 为 AI 工作台增加图片与 UTF-8 文本附件、模型视觉/推理能力、低中高思考强度，
并轻量优化流式输出、滚动跟随、复制与时间信息。助手可见说明、紧凑工具执行行和最终结论
按真实发生时间穿插在同一助手回合中；隐藏思维链不通过前端、REST、SSE、日志或审计暴露。

## 版本与制品

- 发布提交：`74dd5d2984b546c78712140569dadc5bab02dbe9`
- 标签：`v0.44.0`
- 候选 CI：<https://github.com/kejilion/KPanel/actions/runs/30910851524>
- main CI：<https://github.com/kejilion/KPanel/actions/runs/30911075069>
- Release 工作流：<https://github.com/kejilion/KPanel/actions/runs/30911321200>
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.44.0>
- Docker 多架构摘要：`sha256:4cee5407443de5452e1887b6682a4df8186bb26711f7486dc49a7b64b4a3e098`
- linux/amd64：`sha256:d2eee57111b9ffff63f6a49e1aedf16dff2c16ee390fc8e04bc318f9103a5a8d`
- linux/arm64：`sha256:17ae28b73613865437c1f84da5f43b5df61d0077c54f5f18c30affd4d3bd344f`

版本镜像与 `latest` 指向同一多架构摘要；额外 `unknown/unknown` 清单为每个架构的
SBOM/Provenance attestation。Release 已公开且不是 prerelease，Agent、Node、部署归档、
`SHA256SUMS` 和许可证附件完整。应用市场安装契约自 v0.43.2 后没有变化，
`kejilion/apps` 远端 `main` 的 `kpanel.conf` Git Blob 与本仓库权威文件同为
`0a603abfe77beb045c4e7648dd60f5e4a1876e4d`，无需额外提交。

## 自动化验收

- 前端 typecheck、国际化和生产构建通过；Vitest `38` 个文件、`253` 项测试通过。
- Linux `go test ./...`、`go vet ./...` 通过；`internal/panel`、`internal/auth`、
  `internal/dockerx` race 测试通过。
- `govulncheck` 可达漏洞为 0，npm audit 漏洞为 0；固定摘要 Trivy 的源码、依赖、Secret、
  Dockerfile 和最终镜像 HIGH/CRITICAL 扫描在候选、main 与 Release 工作流中通过。
- linux/amd64 与 linux/arm64 的 Panel、Agent、Node、kpctl 均完成 CGO-free 构建。
- 安装安全、应用市场安装/更新/失败回滚/卸载生命周期、最终镜像权限与资源契约通过；
  从公开 Docker Hub 拉取 `0.44.0` 后执行 `image-e2e` 返回 `image_e2e=pass`。
- 本机 WSL 的 Trivy 数据库下载分别访问 `mirror.gcr.io` 和 `ghcr.io` 超时；没有跳过门禁，
  对应扫描由候选、main 和 Release 三条独立 GitHub Linux 流水线完成并成功。

## 行为、安全与资源边界

- 附件最多 4 个：单图 4 MiB、单文本 512 KiB、总计 8 MiB；服务端根据真实内容重新检测
  PNG/JPEG/WebP/GIF 或无 NUL 的 UTF-8 文本，不信任前端文件名与 MIME。
- 图片只允许交给明确启用视觉能力的模型；附件按上下文预算计量。模型同步保留人工能力设置，
  Run 固定启动时的模型、审批模式和思考强度快照。
- 三种 Provider 协议继续使用既有受控出站；没有新增依赖、端口、Sidecar、通用 Shell、
  通用 HTTP 工具或宿主机权限。
- 工具参数与结果继续限长、脱敏并作为不可信数据；写操作审批、受保护路径、严格 Schema、
  resourceVersion、审计与 Agent 类型化边界不变。

## 154 生产上线

- 升级前 Panel 为 `0.43.2/ok`，容器 healthy、重启数 0、Agent active，空闲 RSS 为
  `13.02 MiB / 256 MiB`。
- 一致性备份：`/root/kpanel-backups/v0.44.0-preupgrade-20260804T130708Z`；目录权限 `0700`，
  文件权限 `0600`。SQLite 在线备份 `ai.db` 为 `122,880` B，SHA-256
  `83454832e4708b151449ed05418ae28a411e9fa512001ef10ee29b7ce4f82ed4`；主密钥 SHA-256
  `73ab2b02f437e3b44918855d4cb5f875837f94d3678775e658cc91f37e7bea51`；整机应用归档为
  `20,646,811` B，SHA-256
  `390735f9e77b2f5150633d0796cfad4e49d7e7dd7306f5cf952681b7cc7b2b86`。
- 使用 `KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update k app kpanel` 完成原生应用市场更新，
  拉取摘要为 `sha256:4cee5407443de5452e1887b6682a4df8186bb26711f7486dc49a7b64b4a3e098`。
- 升级后本机与公网 `/api/v1/health` 均为 `0.44.0/ok`，公网 `/` 与 `/ai` 返回 200；
  Panel healthy、Agent active，二者重启数均为 0，OOM=false。
- AI SQLite `integrity_check=ok`，迁移由 6 增至 7；`ai.db*` 与 `ai-secrets.key` 权限保持
  `0600`。
- 连续约 10 分钟、每 30 秒一次共 20 次采样全部为 `0.44.0/ok`、healthy、Agent active、
  重启 0、OOM=false。启动首样 RSS 为 72.9 MiB，30 秒后回落至 12.99 MiB，随后稳定在
  约 13.02–13.13 MiB，最终 13.13 MiB。
- 升级与观察窗口内 Panel/Agent 的 panic、fatal 和 error 计数均为 0；运行边界保持
  `65532:65532`、只读根文件系统、`privileged=false`、`cap-drop ALL`、256 MiB 内存、
  128 PIDs 和入口/出口双网络。

## 回滚与未验证边界

- 公共镜像回滚点为 v0.43.2：
  `sha256:c5dea64c64a435ca32d09d46a0f74ba076c3b11f31c87cf65b72c64a643834c2`。
- 现场回滚使用上述升级前备份，必须成对恢复 `ai.db*` 与 `ai-secrets.key`，并恢复旧镜像、
  Agent、Compose 和 systemd 单元；不得覆盖网站、数据库、证书或其他业务容器。
- 154 当前数据库中 Provider 与模型数量均为 0，因此本轮没有调用真实付费 API，也没有在生产
  对话中实测图片理解或原生思考参数。三协议附件映射、边界校验、SSE/前端交互与思考强度由
  单元、集成、构建和公开镜像测试覆盖，不把它表述为真实模型质量验收。
