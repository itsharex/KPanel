# 架构说明

## 技术选型

- 后端与宿主机 Agent：Go。编译为无 CGO 的单文件 Linux 二进制，内存占用低，
  类型化接口便于限制高权限操作，也减少生产运行时依赖。
- 前端：Vue 3 + TypeScript + Vite。构建后为静态文件，由 `paneld` 直接提供，
  不在生产机运行 Node.js。
- 面板状态：使用带进程锁和原子落盘的 JSON Store，仅保存认证、任务与审计；
  宿主机资源始终从实际文件系统和 Docker Engine 读取。
- 部署：无特权 `paneld` Docker 容器 + 受 systemd 限制的宿主机 Agent。
  这种拆分让 Web 进程不接触 Docker Socket、宿主机根目录或任意 Shell。

该组合优先满足“轻量、快速、稳定、安全、可打包为多架构 Docker
镜像”的目标。资源量或审计查询规模增长后，可以在不改变 Agent 安全边界的
前提下把面板 Store 迁移到 SQLite/PostgreSQL。

## 进程边界

```text
Browser
   │ HTTPS / REST / SSE
   ▼
paneld（Docker，非 root）
   │ HTTP+JSON over Unix Socket
   ▼
kejilion-agent（宿主机 systemd 服务）
   ├─ /proc 与系统状态
   ├─ Docker Engine API
   └─ /home/web 实际产物
```

`paneld` 负责认证、公开 API 和前端静态资源。`kejilion-agent` 是唯一宿主机管理入口，只监听 `/run/kejilion-panel/agent.sock`，不开放 TCP。

## 集群监控边界

每台 KPanel 都可以同时作为中心端和被控端。中心端通过 HTTPS，或在无域名时通过 Noise
端到端加密的公网 `IP + 端口`，与远端 `paneld` 的固定联邦只读接口通信；远端 `paneld`
再通过本机 Unix Socket 读取 Agent 的窄化主机摘要。Agent Token、管理员 Session 和
宿主机写入能力不会跨主机传递。

当前面板自动作为“本机”节点显示，直接读取本地 Agent；远端节点使用一次性授权码和每主机
独立 X25519 身份配对。v2 状态和密钥与既有 Ed25519 v1 文件分离，因此旧节点可继续运行和
回滚。浏览器只读取当前中心端缓存，不直接请求远端，也不共享远端登录态。HTTP 加密直连只
保护联邦数据，不保护浏览器登录目标面板。协议、SSRF/TLS 控制、资源上限和状态语义见
[集群监控与联邦只读协议](cluster-monitoring.md)。

## 与 kejilion.sh 的关系

Agent 直接发现和管理 `kejilion.sh`、KPanel、Compose、Docker CLI 与人工维护生成的实际产物：

- 网站：`/home/web/conf.d`、`/home/web/html`、`/home/web/certs`。
- LDNMP：Docker 中的 Nginx、MySQL、PHP 等容器。
- 应用：Docker/Compose 实际状态与 `/home/docker` 兼容目录。

Agent 不 `source` Shell 函数。需要写入脚本已定义的外联配置时，Agent 必须调用本机脚本的
非交互协议、消费脚本同一权威模板，或调用双方共享生成器；直接写入共享目录不代表可以另写
一套配置。每次查询和写操作前重新读取真实状态，不依赖缓存的影子资源。

网站 favicon 是唯一的展示派生数据，不属于业务状态。Agent 仅通过本机 Nginx 读取已发现
站点，并将经过限制和校验的位图写入自身状态目录；缓存不写回站点目录、不改变网站资源版本。
详细边界见 [网站图标派生缓存](site-icon-cache.md)。

脚本没有机器接口时先改造 `kejilion.sh`，再开放对应 Web 写入。未来可让脚本菜单通过
`kpctl` 调用同一套 Agent Action，使 CLI 和 Web 同时共享配置来源、事务锁和失败恢复。

## 状态规则

- Docker Engine、Nginx 配置、证书文件与系统状态是真实状态。
- 原子 JSON 存储只保存用户、Session、登录限速、任务和审计；它由单个
  `paneld` writer 独占。宿主机资源不写入该文件。
- 每个资源包含基于实际产物计算的 `resourceVersion`。
- 更新必须携带预期版本；版本变化时返回冲突，不静默覆盖。
- 对脚本、SSH 或其他工具的修改重新解析并继续管理；资源版本用于避免并发静默覆盖。

## 写操作事务

站点创建或更新：

1. 校验域名、端口、上游和目录。
2. 确认目标位于配置的共享 Web 业务根目录，防止路径穿越。
3. 获取 Agent 进程内资源锁并重新校验版本。
4. 在同一文件系统写临时文件并同步。
5. 创建时使用不覆盖语义；已存在域名返回冲突。
6. 在 Nginx 容器内执行固定的 `nginx -t`。
7. 原子替换并执行固定的 reload。
8. reload 失败时恢复原文件；检测到外部并发改写时停止并告警。

请求中不得出现任意命令、Shell、绝对目标路径或 Nginx 配置片段。

## 后台交互任务

应用安装、体检和 `kejilion.sh` 建站等长任务遵循同一执行契约：

1. Agent 先校验固定动作和参数，持久化任务 ID，再以 `systemd-run --no-block`
   启动独立 worker；浏览器窗口和 Agent 主进程不持有任务生命周期。
2. worker 通过 PTY 执行受信任入口，把原始 ANSI 输出写入有界日志，并只从该任务的
   `0600` FIFO 接收受长度与 NUL 校验的输入。
3. Agent 重启后先查询原 systemd 单元；仍在运行时保留任务，单元明确结束且没有结果时
   才标记为 `interrupted`。任务列表每次从原子状态文件刷新，不能依赖进程内旧副本。
4. Web 遇到 502、503、504 或网络错误时保持原业务状态并自动重连；只有任务记录明确返回
   `succeeded` 或 `failed`，才能结束进度、关闭输入或展示最终失败。

## Docker 策略

所有容器均可查看和按 Docker 实时状态管理，不检查来源、label、Compose 工作目录、
特权配置、宿主机挂载或是否为 KPanel 自身。删除使用 Docker 强制删除语义；镜像、网络、
卷和 Prune 直接使用固定 Docker API。

运行中的任意容器都可使用有界单次无 TTY 控制台。当前尚未实现交互式 TTY、宿主机终端和
Compose 通用编辑器；这些是待实现适配器，不是外部资源权限限制。资源归属字段只用于展示
生态来源和排查问题，不参与后端授权。
