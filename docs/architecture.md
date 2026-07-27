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

## 与 kejilion.sh 的关系

Agent 直接发现和管理 `kejilion.sh`、KPanel、Compose、Docker CLI 与人工维护生成的实际产物：

- 网站：`/home/web/conf.d`、`/home/web/html`、`/home/web/certs`。
- LDNMP：Docker 中的 Nginx、MySQL、PHP 等容器。
- 应用：Docker/Compose 实际状态与 `/home/docker` 兼容目录。

Agent 不载入 Shell 函数，也不改写脚本。需要完整复用应用安装、一键建站等脚本业务时，
Agent 通过本机脚本已经实现的非交互协议调用固定业务动作；其他操作直接使用共享文件和
Docker Engine API。每次查询和写操作前重新读取真实状态，不依赖缓存的影子资源。

当未来开始改造脚本时，`kejilion.sh` 菜单将通过 `kpctl` 调用同一套 Agent Action。届时 CLI 和 Web 才能共享同一资源锁并获得严格串行保证。

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

## Docker 策略

所有容器均可查看和按 Docker 实时状态管理，不检查来源、label、Compose 工作目录、
特权配置、宿主机挂载或是否为 KPanel 自身。删除使用 Docker 强制删除语义；镜像、网络、
卷和 Prune 直接使用固定 Docker API。

运行中的任意容器都可使用有界单次无 TTY 控制台。当前尚未实现交互式 TTY、宿主机终端和
Compose 通用编辑器；这些是待实现适配器，不是外部资源权限限制。资源归属字段只用于展示
生态来源和排查问题，不参与后端授权。
