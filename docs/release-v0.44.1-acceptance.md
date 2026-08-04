# KPanel v0.44.1 发布验收

## 发布范围

v0.44.1 修复 AI Provider、模型和会话生命周期兼容问题：新增与同步模型默认启用图像能力，
升级时一次性修正 0.44.0 的旧模型能力标记；遗留待审批 Run 不再永久阻止 Provider 同步或删除；
删除活跃会话时先取消所属 Run，不再返回 `AI run is already active`。

## 版本与制品

- 功能修复提交：`aee166545ccd8352c08ac034d4e7d51e2dd5368e`。
- 发布提交：`be17278ce82b78a4ac8d9db283227f2690229e4b`。
- 标签：`v0.44.1`。
- 候选 CI：<https://github.com/kejilion/KPanel/actions/runs/30917991065>。
- main CI：<https://github.com/kejilion/KPanel/actions/runs/30918274062>。
- Release 工作流：<https://github.com/kejilion/KPanel/actions/runs/30918551350>。
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.44.1>。
- Docker 多架构摘要：`sha256:41d74a39f7dab9bc61f319f3cffd7724cb115f2674b0b5d6bb45f7c88120b45a`。
- linux/amd64：`sha256:41359106d422c8e2ac6be338fee8289da08963d5cf28a94746541d32c6a31a4d`。
- linux/arm64：`sha256:c647e7e0874f32e89b27f7c4bc2b53916c3153936008557035f0f00d04129128`。

`0.44.1` 与 `latest` 指向同一 OCI index；额外 `unknown/unknown` 清单为 amd64/arm64 的
SBOM/Provenance attestation。Release 已公开且不是 prerelease，Agent、Node、部署归档、许可证、
第三方声明和 `SHA256SUMS` 共 8 个附件完整。

## 自动化与隔离真机验收

- 本地 Linux 变更门禁通过全量 `go test ./...` 与 `go vet ./...`。
- 154 隔离目录使用精确发布提交执行 `VERIFY_LEVEL=release`：全量 Go 测试、核心特权包 race、
  前端 typecheck/生产构建、38 个 Vitest 文件共 253 项测试全部通过。
- `govulncheck` 可达漏洞为 0，`npm audit` 漏洞为 0；固定摘要 Trivy 源码、依赖、Secret、配置和
  最终镜像 HIGH/CRITICAL 扫描通过。
- linux/amd64 与 linux/arm64 的 Panel、Agent、Node、kpctl 均完成 CGO-free 构建；候选镜像通过
  非 root、只读根、无额外 capability、256 MiB、1 CPU、128 PID 运行契约。
- 应用市场安装、更新、失败回滚和卸载生命周期输出 `app_conf_lifecycle=pass`；本地候选镜像输出
  `image_e2e=pass`。
- Release 公开后，154 从 Docker Hub 重新拉取 `0.44.1`，公开镜像再次输出 `image_e2e=pass`。
- `packaging/kejilion-app/kpanel.conf` 相对 v0.44.0 没有变化，应用市场继续动态使用 `latest`，
  无需修改或提交 `kejilion/apps`。

## 154 生产上线

- 升级前 Panel 为 `0.44.0/ok`，容器 healthy、重启 0，Agent active、重启 0；旧镜像摘要为
  `sha256:4cee5407443de5452e1887b6682a4df8186bb26711f7486dc49a7b64b4a3e098`。
- 一致性备份：`/root/kpanel-backups/v0.44.1-preupgrade-20260804T142852Z`；目录权限 `0700`，
  文件权限 `0600`。SQLite 在线备份为 122,880 B，SHA-256
  `49f0dce9ccb320b51425b08e7879ccbe230fb138ff0bc3c137b33095a7641f5a`；应用归档为
  20,656,385 B，SHA-256
  `64667297673689b7a5f6769cb5cbe44c2d65ddabbd06a11f6614d1fdc8d03612`。
- 使用 `KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update k app kpanel` 完成原生应用市场更新，
  拉取公开摘要 `sha256:41d74a39f7dab9bc61f319f3cffd7724cb115f2674b0b5d6bb45f7c88120b45a`。
- 升级后本机、公网 IP 与 `https://kpanel.kejilion.eu.org/api/v1/health` 均为 `0.44.1/ok`；
  公网 `/` 与 `/ai` 返回 200。Panel healthy、Agent active，二者重启均为 0，OOM=false。
- 运行边界保持 `65532:65532`、只读根文件系统、`privileged=false`、`cap-drop ALL`、256 MiB
  与 128 PID。
- AI SQLite `integrity_check=ok`，已应用迁移 8；`ai.db*` 与 `ai-secrets.key` 权限保持 `0600`。
  生产 Provider、模型和关闭图像能力模型数量均为 0。
- 连续约 10 分钟、每 30 秒一次共 20 次采样全部为 `0.44.1/ok`、healthy、Agent active、
  Panel/Agent 重启 0、OOM=false。启动阶段内存约 72.82 MiB，随后回落并稳定在
  12.88–13.00 MiB / 256 MiB；观察窗口内 Panel/Agent 的 panic、fatal、error 计数均为 0。

## 回滚与未验证边界

- 公共镜像回滚点为 v0.44.0：
  `sha256:4cee5407443de5452e1887b6682a4df8186bb26711f7486dc49a7b64b4a3e098`。
- 现场回滚使用上述升级前备份，必须成对恢复 `ai.db*` 与 `ai-secrets.key`，并恢复旧镜像、Agent、
  Compose 和 systemd 单元；不得覆盖其他业务容器、网站、证书或数据库。
- 154 生产当前没有 Provider 或模型，因此没有在生产调用真实付费 API，也没有用真实旧 Provider
  复现同步/删除和图像输入。数据库迁移、遗留审批、执行中保护、会话取消、SSE 取消通知和权限归属
  由单元、集成、race、发布镜像与公开 E2E 覆盖，不将其表述为真实第三方模型质量验收。
