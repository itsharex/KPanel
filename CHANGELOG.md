# Changelog

本项目遵循语义化版本。所有日期均使用 `YYYY-MM-DD`。

## [0.5.0] - 2026-07-26

### Added

- 系统监控与管理页增加“系统更新”和“系统清理”，对应 154 当前
  `kejilion.sh` 的 `linux_update` 与 `linux_clean` 业务。
- 系统维护通过固定参数的 systemd 后台任务运行，浏览器断开或 Panel 重启
  不会中断 APT/dpkg，并持续返回阶段、进度、结果和重启提示。
- 清理提供“仅缓存”和“标准安全清理”；标准模式移除无用依赖、清理 APT
  缓存，并保留最近 7 天且限制最大 500 MiB 的 journal。

### Security

- 不复制脚本中的 `pkill -9 apt/dpkg`、删除 dpkg 锁、清空 `/var/log`、
  清空 `/tmp` 或 Docker prune；Web API 仍不接受命令、路径或包名。

## [0.4.0] - 2026-07-26

### Added

- DNS 管理增加 Cloudflare、Google、阿里云、腾讯 DNSPod 和 Quad9 便捷预设，
  同时保留自定义地址。
- 时区管理增加常用城市与 IANA 时区下拉选择，同时保留其他合法时区的
  自定义入口。
- 打开管理弹窗时自动识别当前 DNS 和时区，不会默认改写宿主机配置。

## [0.3.0] - 2026-07-26

### Added

- 系统管理页开放主机名、安全新增 SSH 端口、systemd-resolved DNS、时区、
  KPanel 专属 Swap、Debian/Ubuntu APT 镜像源、V4/V6 优先级、内核优化
  预设和 BBR 的类型化写入。
- 每次系统变更均记录意图与结果审计，并在宿主机保存变更前配置快照；执行失败
  自动回滚，成功后重新读取真实状态。
- SSH 新端口在 `sshd -t`、防火墙放行、reload 和监听探测全部成功前不会完成，
  且首版保留所有旧端口，避免远程失联。

### Security

- Web 与 Agent 均只接受固定 action 和字段，不接受命令名、Shell、脚本内容或
  任意目标路径；所有写请求继续强制登录、同源、CSRF 和审计。
- Swap 只管理 `/var/lib/kejilion-panel/system/swapfile`，不会停用现有 Swap；
  APT 切换不修改 Docker、NodeSource 等第三方仓库。
- 检测到外部 `kejilion.sh` 内核调优配置时拒绝覆盖；系统重装仍保持锁定。

## [0.2.0] - 2026-07-26

### Added

- “概览”升级为“系统监控与管理”，整合主机名、SSH 端口、DNS、时区、
  Swap、系统镜像源、V4/V6 优先级、内核优化、BBR 与重装系统入口。
- Agent 直接读取宿主机配置，识别 `kejilion.sh`、人工或其他工具产生的
  SSH、DNS、时区、Swap、软件源、`gai.conf`、sysctl 与 BBR 实际状态。
- 系统写能力以独立 capability 呈现，页面显示当前状态、安全要求和禁用原因。

### Security

- v0.2.0 不执行系统配置变更；SSH、DNS、Swap、内核和重装等操作在具备
  白名单输入、前置校验、回读验证与回滚前保持锁定。
- 不复用脚本中清理防火墙、清除全部 Swap、锁定 `resolv.conf` 或下载后
  直接执行远程脚本等高风险组合命令。

## [0.1.3] - 2026-07-25

### Added

- 设置页支持验证当前密码后修改管理员密码；密码状态原子持久化，并立即注销该管理员的全部现有会话。

### Security

- 密码修改接口强制登录、同源和 CSRF 校验，复用 Argon2id 及密码强度策略，审计记录不包含密码内容。

## [0.1.2] - 2026-07-25

### Fixed

- 保留 Docker `internal` 网络隔离，同时为 Panel 分配确定的私网地址并取消无效的宿主端口发布；宿主健康检查和 Kejilion Nginx 直接访问该私网端点。
- 发布构建通过 BuildKit secret 注入可选代理，避免把构建基础设施地址写入镜像 Provenance；镜像记录源提交 revision，并在验收版本 digest 后才更新 `latest`。

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
