# KPanel v0.42.0 发布验收

发布日期：2026-08-04

## 发布范围

- 新增 AI 三栏工作台、多会话、Provider/模型切换、REST/SSE 流式对话、工具进度和逐次审批。
- 支持 OpenAI-compatible 的 Chat Completions 与 Responses、Anthropic Messages、Gemini `generateContent`。
- 新增原生 Go Agent Runtime、受控 KPanel 工具注册表、记忆、流程和进化提案审核。
- AI 数据独立使用 CGO-free SQLite；API Key 使用 XChaCha20-Poly1305 和独立 `0600` 主密钥保护。
- Panel Compose 从旧单网络迁移为内部网络加受控出口网络，不引入 Hermes、Sidecar、通用宿主机 Shell、Web 工具或本地模型托管。

## 自动化与安全验收

- 本地和 154 Linux 均通过 Go 全量测试与 vet；154 L3 通过核心特权包 race、`govulncheck`、`npm audit`、源码/Secret/配置扫描、amd64/arm64 CGO-free 构建、最终镜像 HIGH/CRITICAL 扫描和运行契约。
- 前端 38 个测试文件、243 项测试通过；1597 条延迟本地化短语校验、TypeScript 检查和生产构建通过。
- `kpanel.conf` 安装、成功更新、失败回滚和卸载生命周期通过；从旧单网络迁移时 `.env` 可信代理网段自动刷新并可回滚。
- 公网 HTTP Provider 被拒绝；内网 HTTP 必须显式确认。Provider API 仅返回 `apiKeySet` 和末四位，未返回密钥或密文。
- 候选分支 CI、主线 CI 和 Release 工作流均成功。

## 154 真机 AI 验收

候选提交 `7c5f99379579ab4be3cc71eb8c26c65c475c263a` 使用独立 Panel 数据目录，复用生产 Agent 的只读 Socket/Token，并保持生产 Panel 全程 healthy。

- Chat Completions 与 Responses 均完成真实 SSE 对话、宿主机 `host_system_summary` 只读工具调用和最终消息持久化。
- SSE 实际观察到 `run.snapshot`、`message.delta`、`message.completed`、`tool.started`、`tool.completed`、`approval.required` 和 `run.completed`。
- 写工具进入 `pending_approval`；拒绝后工具状态持久化为 rejected，宿主机未发生变化。
- 两个 Provider、两个会话、会话模型切换和两个并行 Run 通过；并行期间为 `139.8 MiB / 256 MiB`，`OOMKilled=false`。
- `ai.db` 已持久化；`ai-secrets.key` 为 32 字节、权限 `0600`。
- 验收只使用本地 Mock Provider，不调用真实付费 API，也未复用历史对话中出现过的密钥。

## 轻量指标

同一 154、同一 Go 1.26.5 和 stripped 构建参数下，以发布前主线 `de02e95`（0.40.3）为基线：

| 指标 | 0.40.3 | 0.42.0 | 增量 | 目标 |
| --- | ---: | ---: | ---: | ---: |
| `paneld` | 8,704,162 B | 12,955,810 B | 4,251,648 B（4.06 MiB） | ≤ 30 MiB |
| 空闲 RSS（第一次） | 82,252 KiB | 85,868 KiB | 3,616 KiB（3.53 MiB） | ≤ 25 MiB |
| 空闲 RSS（第二次） | 82,384 KiB | 86,064 KiB | 3,680 KiB（3.59 MiB） | ≤ 25 MiB |

AI 路由保持懒加载，生产构建 chunk 为 184.76 kB（gzip 72.02 kB）。上线后生产 Panel 空闲采样为 `72.6 MiB / 256 MiB`，进程 RSS 83,808 KiB。

## 发布产物

- 功能提交：`52b49d4fa7f17de9d34f6495ebe60994eefdfa2d`。
- 版本准备提交：`56c9921ca32f5227f54c683ef4fd154d8958b5e0`。
- 发布 Tag：`v0.42.0`，指向 `7c5f99379579ab4be3cc71eb8c26c65c475c263a`。
- 候选分支 CI：[30871510543](https://github.com/kejilion/KPanel/actions/runs/30871510543)。
- 主线 CI：[30871802978](https://github.com/kejilion/KPanel/actions/runs/30871802978)。
- Release 工作流：[30872028864](https://github.com/kejilion/KPanel/actions/runs/30872028864)。
- GitHub Release：[v0.42.0](https://github.com/kejilion/KPanel/releases/tag/v0.42.0)，非草稿、非预发布，附件包含双架构 Agent、轻量节点、部署包、校验和与许可声明。
- `docker.io/kjlion/kejilion-panel:0.42.0` 与 `latest` 均指向 OCI 摘要 `sha256:ef44987f6698fa8916e2567207786be38c03f284a13ff5143a080bc2f56873ff`。
- amd64 清单摘要为 `sha256:cae8a5930c82e4f274ca9e68313299e3a12bc254c8f39e17dc62a6861302921d`；arm64 为 `sha256:129abf5bde0238b9ad6b24063d14c83e46484c9e56b4dcdbb0891e59e26e0171`。
- 公开镜像重新拉取后运行 `packaging/tests/image-e2e.sh`，输出 `image_e2e=pass`。
- 应用市场仓库同步提交为 `fc40b49ffe0fc98858bb63e0fe7238c117c5d492`；GitHub Raw 配置通过 Bash 语法检查，Blob 与本仓库发布配置一致。

## 154 上线与回滚

- 生产实例从 `0.40.1 v1alpha1` 更新到 `0.42.0 v1alpha1`，应用市场输出 `KPanel 更新完成 / Update Complete`。
- 上线后 Panel healthy，镜像 revision 与发布提交一致；`/ai` 返回 200，未登录 AI API 返回 401，Agent 实际 Socket/Token 健康检查通过。
- 容器仍为只读根文件系统、`cap_drop: ALL`、`no-new-privileges`、256 MiB；网络为 `kejilion-panel-internal` 与 `kejilion-panel-egress`，旧 `kejilion-panel-network` 已删除。
- 可信代理网段已更新为新内部网络真实子网；AI 数据库与密钥已创建，更新目录无 `.rollback` 残留。
- 升级前备份位于 `/root/kpanel-backups/v0.42.0-preupgrade-20260804T023300Z/kpanel-home.tar.gz`，权限 `0600`、大小 20,532,667 B、SHA256 为 `7e442790d41ec1ddd279c4cfb11f296281c38a9afb5f94afd668de510daa07cd`。
- 公共源码与镜像回滚点为 `v0.40.3`；154 现场另保留升级前 0.40.1 配置、数据、Agent 与镜像标识。回滚时必须成对保留 `ai.db*` 与 `ai-secrets.key`。
