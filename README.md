# KPanel

KPanel 是 `kejilion.sh` 的现代 Web 管理形态。首版聚焦安全登录、主机监控、现有网站发现、Docker 查看与受控生命周期管理。

## 核心原则

- 不修改、覆盖、执行或 `source` 现有 `kejilion.sh`。
- 宿主机实际状态是事实来源，面板数据库不是系统资源的唯一真相。
- Web/API 进程无特权运行，不挂载 Docker Socket 或宿主机根目录。
- 所有宿主机操作通过本地 Unix Socket 连接到白名单式 Agent。
- 无法证明安全的资源保持只读。
- 所有写操作具备计划、校验、资源版本、审计和失败回滚。

## 首版范围

- 首次初始化、安全登录、服务端 Session、CSRF、登录限速。
- CPU、内存、磁盘、负载、网络和服务状态。
- `/home/web` 现有站点、证书和 Nginx 状态发现。
- 固定模板的静态站及反向代理站点创建与安全更新。
- Docker 容器、镜像、网络、卷查看。
- 已识别 Kejilion 容器的启动、停止、重启和有界日志。
- 主机名、SSH 新端口、DNS、时区、专属 Swap、APT 镜像、IP 优先级、
  内核预设和 BBR 的类型化、可回滚管理。
- 管理变更记录、审计、资源版本冲突检测和只读降级。

任意 Shell、Docker Exec、系统重装、磁盘格式化、应用市场远程脚本、站点硬删除等高风险能力不进入首版。

## 文档

- [架构与事实来源](docs/architecture.md)
- [v0.1 范围与验收](docs/scope-v0.1.md)
- [安全边界](docs/security-model.md)
- [kejilion.sh 兼容基线](docs/compatibility.md)
- [kejilion.sh 网站业务分析](docs/legacy-site-contract.md)
- [构建、发布与部署](docs/deployment.md)
- [版本变更记录](CHANGELOG.md)

详细设计见 [架构说明](docs/architecture.md)、[兼容基线](docs/compatibility.md)、[安全模型](docs/security-model.md) 和 [v0.1 范围](docs/scope-v0.1.md)。
