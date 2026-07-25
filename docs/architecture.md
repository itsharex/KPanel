# 架构说明

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

首版 Agent 只发现和管理 `kejilion.sh` 已生成的实际产物：

- 网站：`/home/web/conf.d`、`/home/web/html`、`/home/web/certs`。
- LDNMP：Docker 中的 Nginx、MySQL、PHP 等容器。
- 应用：Docker/Compose 实际状态与 `/home/docker` 兼容目录。

Agent 不调用 `/usr/local/bin/k`，不载入 Shell 函数，也不改写脚本。v0.1
在每次资源查询和写操作前重新读取 Docker 与文件系统事实，不依赖缓存的影子资源。
Docker Events、文件事件与后台周期对账属于后续优化。

当未来开始改造脚本时，`kejilion.sh` 菜单将通过 `kpctl` 调用同一套 Agent Action。届时 CLI 和 Web 才能共享同一资源锁并获得严格串行保证。

## 状态规则

- Docker Engine、Nginx 配置、证书文件与系统状态是真实状态。
- v0.1 的原子 JSON 存储只保存用户、Session、登录限速和审计；它由单个
  `paneld` writer 独占。宿主机资源不写入该文件。
- 每个资源包含基于实际产物计算的 `resourceVersion`。
- 更新必须携带预期版本；版本变化时返回冲突，不静默覆盖。
- 对脚本外部修改只记录漂移，不自动恢复或覆盖。

## 写操作事务

站点创建或更新：

1. 校验域名、端口、上游和目录。
2. 确认目标位于发现出的 Kejilion Web 根目录。
3. 获取 Agent 进程内资源锁并重新校验版本。
4. 在同一文件系统写临时文件并同步。
5. 创建时使用不覆盖语义；已存在域名返回冲突。
6. 在 Nginx 容器内执行固定的 `nginx -t`。
7. 原子替换并执行固定的 reload。
8. reload 失败时恢复原文件；检测到外部并发改写时停止并告警。

请求中不得出现任意命令、Shell、绝对目标路径或 Nginx 配置片段。

## Docker 策略

所有容器均可查看。首版只有以下容器可操作：

- 带 `io.kejilion.panel.managed=true` 标签的面板资源。
- Compose 工作目录为 `/home/web` 的 LDNMP 容器。
- Compose 工作目录位于解析后的 `/home/docker` 下且身份唯一的应用。

外部或归属不明确的容器保持只读。首版不提供创建、删除、Exec、Prune、Pull 或 Compose 编辑。
