# Changelog

本项目遵循语义化版本。所有日期均使用 `YYYY-MM-DD`。

## [0.11.0] - 2026-07-26

### Added

- 系统监控与管理新增服务器重启入口；要求输入大写 `REBOOT` 并勾选二次确认。
- Agent 使用固定参数的 systemd transient timer 延迟 15 秒执行重启，为 Panel 写入成功审计和
  浏览器接收结果预留时间。

### Security

- 重启接口只接受 `action=reboot` 和固定确认值，不接受命令、执行时间或 Shell 参数。
- 系统更新或清理任务运行期间拒绝重启；宿主机缺少 `systemctl` 或 `systemd-run` 时能力保持禁用。

## [0.10.0] - 2026-07-26

### Added

- 新增应用市场，完整收录 `app.kejilion.sh` 当前 146 个应用、7 个分类和本地图标，并与
  `kejilion.sh` 的 115 个内置应用编号、主容器、镜像和默认端口建立审计映射。
- 应用详情整合运行状态、启动/停止/重启、镜像更新检查、域名绑定与解绑、IP+端口访问策略、
  更新和卸载；首批为 LibreSpeed、IT-Tools 和 DOS 游戏开放声明式安装器。
- 新增 Panel 管理的反向代理安全删除事务；仅允许删除未漂移的 Web 来源反向代理，删除失败时
  自动恢复原配置并重新校验、加载 Nginx。

### Changed

- KPanel 安装的应用同步写入 `/home/docker/appno.txt` 和对应 `*_port.conf`，脚本端可继续识别；
  Docker 仍是运行状态真相，孤立脚本标记显示为待核对。
- “阻止 IP+端口”改为把经过适配的容器监听地址切换到 `127.0.0.1`，不写入全局 iptables，
  域名反向代理继续通过宿主机回环地址访问。

### Security

- Agent 不执行 `k app`、远程 Shell、第三方 Compose 或脚本菜单；未审计应用完整展示但写操作
  保持禁用，并明确给出原因。
- 应用更新先拉取镜像，再使用固定容器规格重建；创建或启动失败时恢复原镜像、端口和运行状态。
  卸载仅接受资源版本一致、无挂载、无特权、配置完全匹配的声明式容器。

## [0.9.0] - 2026-07-26

### Added

- 新建网站扩展为静态站、PHP 动态站、IP/端口反代、域名反代、负载均衡和
  域名重定向六类服务，并提供不使用下拉框的卡片式选择向导。
- PHP 支持脚本现有 `php` 与 `php74` Socket；重定向支持
  301/302/307/308；负载均衡支持 2–8 个 HTTP 后端。
- 发现器可区分域名反代、负载均衡和 WordPress，并忽略 ACME Webroot 对
  实际站点根目录识别的干扰。

### Changed

- Panel 管理模板升级到 v2，沿用 `/home/web/conf.d`、`/home/web/html` 与
  Nginx 容器路径；旧 v1 固定模板在安全更新时原地迁移，不修改业务目录。
- 域名反代固定上游 Host 与 SNI；所有写入继续使用资源版本、原子替换、
  `nginx -t`、reload 和失败回滚。

### Security

- IP/端口反代仅允许本机、私网地址或 Docker DNS 名；域名反代和跳转目标
  必须是 ASCII FQDN；负载均衡目标数量、协议、端口和重复项均受限制。
- 应用安装器、Nginx Stream 和 TLS 签发不混入普通网站事务，避免执行远程
  `main` 脚本、处理数据库凭据、修改全局 Nginx 或停止现有服务。

## [0.8.0] - 2026-07-26

### Added

- 系统首页补齐 `kejilion.sh` 的 `k info` 主机信息：CPU 型号与频率、
  1/5/15 分钟负载、TCP/UDP 连接数、宿主机时间、公网 IPv4/IPv6、
  ISP 和地理位置。
- 公网信息查询使用固定 IPv4/IPv6 HTTPS 端点、严格地址族校验和 30 分钟缓存；
  查询失败不影响本地主机监控，并可通过 Agent 环境变量关闭。

### Changed

- 系统首页将主机、网络与位置、资源一致性重新分组；内存、Swap、根磁盘、
  累计流量、DNS、时区、拥塞算法和 qdisc 继续以宿主机实时状态为准。
- 系统摘要 API 扩展为向后兼容的可选字段；旧 Agent 未提供公网信息时页面明确
  显示查询不可用，不伪造默认数据。

### Security

- Agent 不接受外部查询 URL 或任意命令；响应限制为 64 KiB，设置短超时、
  同主机重定向限制、控制字符过滤和字段长度上限。

## [0.7.0] - 2026-07-26

### Added

- 宿主机软件包维护新增 RHEL/Fedora 系 DNF/DNF5/YUM、Arch/Manjaro Pacman
  和 openSUSE/SLES Zypper 固定命令序列。
- 软件源状态发现新增 `/etc/pacman.d/mirrorlist` 与
  `/etc/zypp/repos.d/*.repo`，并继续识别 RPM `.repo` 文件。
- 增加宿主机系统兼容矩阵，区分已实机验证、已实现待准入和安全只读层级。

### Changed

- Agent 根据 `/etc/os-release` 的 `ID`、`ID_LIKE`、软件包工具和源文件动态
  开放系统更新与清理；启动任务前即拒绝不满足条件的主机。
- 系统维护页面改为显示宿主机实际软件包管理器，不再把说明固定为 APT。

### Security

- 多发行版维护仍只接受 `full`、`cache`、`standard` 枚举，不能由 Web 传入
  命令、包名、仓库、路径或 Shell。
- RPM、Pacman 和 Zypper 软件源首版只读取不改写；Alpine/OpenRC 与未知
  发行版不开放宿主机写入。

## [0.6.0] - 2026-07-26

### Added

- 内核优化补齐 `kejilion.sh` 的高性能、均衡、网站、直播和游戏服五种预设，
  包括 TCP 缓冲区、连接队列、端口、虚拟内存、调度、安全、文件描述符和
  连接跟踪参数。
- 与脚本一致按宿主机 `MemTotal` 自适应缓冲区、`min_free_kbytes`、
  `swappiness` 和小内存队列，并为直播、游戏场景增加对应的 UDP/低延迟参数。
- 调优时同步处理 BBR 能力、透明大页、nofile 限制和模块持久化；页面刷新可
  识别脚本或 Web 生成的同路径、同模式产物。

### Changed

- Web 可以接管结构和标识均合法的 `kejilion.sh` 内核预设，切换后仍写入
  `/etc/sysctl.d/99-kejilion-optimize.conf`，实现脚本与面板双向识别。
- “还原默认设置”同时清理已识别的手动/自动调优产物和对应 nofile/模块设置，
  但保留通过独立 BBR 功能管理的配置。

### Security

- API 只接受五个固定预设和 `off`，逐项调用 `sysctl -w`，不接受参数名、
  参数值、文件路径或 Shell；全项失败时恢复所有文件并重新加载原配置。
- Web 不下载或执行测速型 `network-optimize.sh`；脚本产生的自动调优结果可以
  被识别和安全还原，未知同名配置仍拒绝覆盖。

## [0.5.1] - 2026-07-26

### Changed

- 虚拟内存改为识别并管理 `kejilion.sh` 使用的 `/swapfile`，脚本端和
  Web 端不再生成两套互不相认的 Swap。
- 支持直接创建、扩容、缩容和停用 `/swapfile`；页面提供脚本一致的
  1/2/4 GiB 选项以及 256–65536 MiB 自定义大小。
- 调整时自动合并旧版 `/var/lib/kejilion-panel/system/swapfile`，但保留
  Swap 分区和第三方 swapfile。

### Security

- Swap 变更通过固定参数的 systemd 一次性事务执行；不向 Web 暴露命令或
  路径，并拒绝符号链接等异常产物。
- `swapoff` 前校验受管 Swap 已用空间与可用内存；创建、`fstab` 更新或
  激活失败时恢复原文件、原启动项和原活动状态。
- 不复用 `kejilion.sh` 中 `wipefs`、重建 Swap 分区的高风险行为。

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
