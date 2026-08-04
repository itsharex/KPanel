# KPanel AI 工作台

## 范围与架构

AI 工作台位于 `/ai` 与 `/ai/s/{sessionId}`，由 `paneld` 内的原生 Go Runtime 驱动。它不安装 Hermes、Eino、Sidecar、本地模型或向量数据库，也不提供宿主机通用 Shell、通用 HTTP/Web、附件、语音和多 Agent。

数据流为：

```text
Vue 三栏工作台 ── REST/SSE ── paneld AgentRuntime
                                  ├─ OpenAI-compatible / Anthropic / Gemini
                                  ├─ /var/lib/kejilion-panel/ai.db
                                  └─ 固定 KPanel Tool Registry ── Unix Socket ── kejilion-agent
```

身份、Origin、CSRF 与现有 KPanel API 完全一致。AI 数据独立保存到 `ai.db`；宿主机资源仍由 Agent 实时读取，不写入 AI 数据库。

## Provider 与网络

- 固定协议：`openai_compatible`、`anthropic`、`gemini`。
- `openai_compatible` 必须明确选择 `apiMode`：`responses` 使用 `POST /v1/responses` 语义事件，`chat_completions` 使用 `POST /v1/chat/completions` 数据分片。历史 Provider 自动迁移为 `chat_completions`，避免升级后改变行为；OpenAI 官方预设默认使用 `responses`，其他兼容预设默认保守使用 `chat_completions`。
- Responses 模式使用 KPanel 持久化的完整会话上下文，并设置 `store=false`。工具调用使用 `function_call` / `function_call_output`；推理模型返回的加密 reasoning item 只保存在内部工具上下文、仅向同一 Provider 回放，不通过 REST、SSE、日志或审计暴露。
- Responses 的 HTTP SSE 与 Realtime WebSocket 是不同接口；v1 不接入 `/v1/realtime`、音频或语音会话。
- 公网 Provider 必须使用 HTTPS；拒绝 URL userinfo、query、fragment，以及每次 DNS 解析得到的回环、私网、链路本地、组播、未指定和保留地址。
- 内网/本地 Provider 必须显式选择 `private`；只有此模式允许 HTTP。
- 重定向最多三次；跨源重定向移除 `Authorization`、`X-Api-Key` 与 `X-Goog-Api-Key`。
- Compose 同时连接固定 `panel-internal` 与普通 `panel-egress`。容器仍为 UID 65532、只读根文件系统、`cap_drop: ALL`、`no-new-privileges`、256 MiB。
- Provider 只能访问模型 API；模型自身只能调用固定 KPanel 工具，没有通用联网工具。
- Ollama/LM Studio 预设通过 `host.docker.internal` 访问宿主机；本地服务需监听 Docker 网关可达地址，且保存前仍需显式确认内网模式。

## 用户交互

- 首次进入先展示三步引导：选择 API、验证连接、启用模型。新增 API 默认执行“保存 → 测试 → 同步模型”；后两步失败时保留已加密保存的连接，允许修正后单独重试。
- 点击“新建会话”会直接使用管理员设置的默认模型；没有显式默认值时使用第一个可用模型。会话标题由第一条用户消息自动生成，仍可手工重命名。
- 模型选择器位于输入框右下角，实时包含所有已启用 Provider 的已启用模型。运行中允许切换，但当前 Run 使用启动时快照，新选择明确标记为“下一轮”并只影响后续 Run。
- 会话列表提供搜索、置顶、重命名、归档、恢复和删除；当前与已归档会话有独立视图，移动端通过抽屉管理。
- 对话在首个模型分片到达前显示规划/重连状态；输入框随内容自动增高，运行中仍可追加要求。桌面端使用容器内滚动，避免固定输入框与全局侧栏宽度耦合。

## 密钥与数据

- SQLite 使用 WAL、外键、5 秒 `busy_timeout` 和事务迁移。
- API Key 由 XChaCha20-Poly1305 加密，主密钥为 `/var/lib/kejilion-panel/ai-secrets.key`，Linux 权限 `0600`。
- 数据库存在密文但主密钥丢失时，AI 模块失败关闭；KPanel 其他功能继续提供服务。不得自动生成新密钥覆盖。
- API 只返回 `apiKeySet` 和末四位；工具参数、结果、Provider 错误与审计进入统一限长和脱敏链路。
- 上下文估算超过模型窗口 70% 时，将旧消息压缩为最多 8 KiB 的持久摘要并保留最近消息。

## 运行与审批

- 单条消息 16 KiB、工具结果 64 KiB、12 个模型步骤、20 次工具调用。
- 单会话一个 Run，全局两个并行 Run；模型请求超时 180 秒，SSE 心跳 15 秒。
- 401 不重试；429/502/503/504 仅在尚未输出内容时重试两次。流式输出开始后不重放。
- 只读工具以及经过固定 Schema 约束的常规应用、网站和容器操作自动执行。应用安装、启停、更新，网站创建/更新，容器启停/重启和诊断任务无需逐次确认。
- 删除、系统核心设置、容器 exec、交互式任务输入、Docker 维护任务以及无法识别或无法解析的动作仍进入 `pending_approval`；分类失败时默认要求确认。
- 工具名与 Agent 路径是固定映射；输入继续使用现有 KPanel 校验器、`resourceVersion` 和 Agent 二次校验。不存在任意路径或任意宿主机命令入口。
- 工具结果以“不可信数据”标签返回模型，不能修改核心提示词、Schema、鉴权、审计或确认策略。
- Panel 重启会把模型请求中的 Run 标记为 `interrupted`；未执行审批保留，Agent 已提交的后台任务按原有任务恢复机制继续。

## 进化规则

用户明确说“记住/以后这样做”或成功完成多工具任务时，当前模型可以额外生成候选：

- `memory`：事实或偏好，不超过 500 字。
- `procedure`：适用条件和 1–10 个步骤，只能引用注册工具。

候选先脱敏、Schema 校验和工具 dry-run，再以 `pending` 保存。只有管理员在“进化提案”批准后才进入系统提示词。流程修订保存新版本；可停用、退休或回滚历史版本。任何已批准流程仍不能绕过上述受保护操作的逐次确认。

## 新增依赖与许可

| 依赖 | 用途 | 许可 |
| --- | --- | --- |
| `modernc.org/sqlite v1.55.0` | CGO-free SQLite | BSD-3-Clause |
| `markdown-it 15.0.0` | 禁用原始 HTML 的 Markdown 解析 | MIT |
| `DOMPurify 3.4.13` | 渲染后 DOM 二次净化 | MPL-2.0 OR Apache-2.0 |

构建仍使用 `CGO_ENABLED=0`。发布流水线生成的 SBOM 会从锁定后的 `go.mod/go.sum` 与 `package-lock.json` 收录这些依赖。

## 回滚

运行数据回滚前先停止 `paneld`，备份 `ai.db*` 与 `ai-secrets.key`；两者必须成对保留。应用市场更新会保留数据目录并自动恢复旧镜像、Agent、Compose 与 `.env`。移除 AI 功能不需要迁移或改写 `panel-state.json`，也不影响 Agent 宿主机状态。

## 轻量验收

0.42.0 在 154 Debian 13 真机完成发布级 L3、双架构 CGO-free 构建、公开镜像 E2E、隔离 AI 实聊和生产升级。相对发布前主线 0.40.3，同参数 stripped `paneld` 增加 4.06 MiB，两次空闲 RSS 增量为 3.53 MiB 和 3.59 MiB；均低于 30 MiB/25 MiB 目标。两个并行 Mock Run 使用 139.8 MiB/256 MiB，未发生 OOM。

Compose 的 `256M` 内存限制保持不变。完整数据、CI/Release 链接、镜像摘要和回滚证据见 [v0.42.0 发布验收](release-v0.42.0-acceptance.md)。
