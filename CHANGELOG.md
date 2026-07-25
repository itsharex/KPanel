# Changelog

本项目遵循语义化版本。所有日期均使用 `YYYY-MM-DD`。

## [0.1.2] - 2026-07-25

### Fixed

- 保留 Docker `internal` 网络隔离，同时为 Panel 分配确定的私网地址并取消无效的宿主端口发布；宿主健康检查和 Kejilion Nginx 直接访问该私网端点。

## [0.1.1] - 2026-07-25

### Fixed

- 修复 systemd `ProtectProc=invisible` 隐藏 dockerd 进程，导致 Agent 的 Docker socket 防激活检查误判 `docker_unavailable`；改为显式 `ProtectProc=default`，其余沙箱限制保持不变。
- 防止 systemd 接管非 root Panel 数据目录的属主，并在 Agent、Panel 启动前后增加目录属主与权限门禁。

## [0.1.0] - 2026-07-25

首个可部署版本。

### Added

- 用户可见产品名称确定为 KPanel；为保持兼容，内部路径、服务、容器、网络及环境变量继续使用既有 `kejilion-panel` 标识。
- 一次性初始化、安全登录、Argon2id 密码哈希、服务端 Session、CSRF/Origin/Host 校验和登录限速。
- 通过受限 Unix Socket Agent 读取 Linux 主机状态、`/home/web` 网站产物和 Docker 资源。
- 现有静态站、反向代理站、证书及未知 Nginx 配置的只读发现与资源版本展示。
- 固定模板静态站和内网反向代理站的安全创建、更新、`nginx -t`、原子替换、reload 与失败回滚。
- 经归属验证的 Kejilion 容器启动、停止、重启、有界日志与一次性资源统计。
- 管理变更记录、结构化审计、敏感字段脱敏和 Agent 离线只读降级。
- 非 root 多架构 Panel 镜像、双架构 Agent、校验和、SBOM/Provenance 发布流程及隔离安装器。

### Compatibility

- 兼容基线为 `kejilion.sh` v4.5.2 与 Nginx 模板提交 `05f5a2eac269967706f30dc3ff7985339e1f3ce4`。
- Panel 不修改、执行或 `source` `kejilion.sh`；宿主机真实产物始终是事实来源。
- 脚本侧或人工修改会在下一次发现时呈现；Web 侧仅写入脚本既有路径和命名约定。

### Security

- 安装器仅允许全新安装，并固定使用本机 Docker Socket；dry-run 不连接 Docker daemon，也不启动服务。
- 安装前校验 Agent 版本、镜像 digest/用户/健康检查/版本标签、专用权限组、systemd/Compose 归属、监听端口、`/home/web` 和 Docker 子网冲突。
- 正式安装要求 Agent、Panel、Panel→Agent 及宿主回环端口全部健康；失败时尝试停止本次新进程并复核状态，无法确认则发出 `CRITICAL`，同时保留日志供恢复。
- 单 IP 与账户采用分级登录限速；成功登录按发生时间重置此前失败，避免少量匿名请求锁死管理员。
- Dockerfile frontend、构建镜像、BuildKit 与发布 Actions 均固定版本或摘要；发布前执行镜像运行时安全合约检查。

### Known limitations

- 首版不提供网站、证书、数据库、目录或 Docker 资源删除。
- 首版不提供任意 Shell、任意 Docker Exec、Compose 在线编辑、系统重装或应用市场远程脚本。
- 未改造的 `kejilion.sh` 与 Web 不共享事务锁；外部并发变更通过资源版本和再次校验拒绝覆盖。
