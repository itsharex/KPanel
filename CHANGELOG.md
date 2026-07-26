# Changelog

## [Unreleased]

## [0.16.2] - 2026-07-27

### Fixed

- 系统更新、系统清理和 Swap 后台任务改为使用当前运行中的 Agent 绝对路径，兼容
  `/usr/local/libexec/kejilion-agent` 标准安装与
  `/home/docker/kpanel/bin/kejilion-agent` 应用市场安装，不再错误回退到不存在的
  FHS 路径。
- 软件源位于自定义目录或缺少 `journalctl` 时不再误判整个维护功能不可用；原生包
  管理器负责校验其仓库，标准清理会安全跳过不可用的可选日志步骤。

### Changed

- 更新与清理覆盖 APT/dpkg、DNF/DNF5/YUM、APK、Pacman 和 Zypper；无法识别的
  Linux 发行版可按本机唯一受支持的原生包管理器安全降级。
- Pacman 标准清理对 `pacman -Qdtq` 返回的孤立包名进行数量、格式和重复校验后，
  以固定参数直接调用 `pacman -Rns`，不经过 Shell。

### Security

- 保持 KPanel 安全清理边界：不删除软件包锁、完整日志目录、`/tmp`、网站数据或
  Docker 资源；journal 默认保留 7 天并限制为 500 MiB。

## [0.16.1] - 2026-07-27

### Added

- 安装、更新、卸载和访问策略变更统一为可恢复查看的后台任务，并在应用市场显示实时阶段、进度与日志。
- 识别 `docker_app_service` 主服务容器，并持久化脚本/Web 共用的访问策略状态。

### Changed

- `kejilion.sh` 应用市场安装 KPanel 成功后直接显示首次初始化 Token，并提示完成管理员账户初始化后 Token 自动失效，不再要求用户手动读取凭证文件。
- 对齐 `kejilion.sh` 原生应用管理：脚本安装的内置与第三方应用在安装标记、配置和主容器精确匹配后，可继续更新、卸载和切换 IP + 端口访问。

### Security

- 第三方应用仅只读解析 `/root/apps/<token>.conf` 的字面量容器字段，不执行配置；拒绝软链接、非 root 所有或可写配置。

## [0.16.0] - 2026-07-26

### Added

- 应用安装改为可持久化后台任务：提交后立即返回任务 ID，应用市场和任务中心持续显示
  阶段、百分比、结果与最近日志，离开页面或刷新浏览器不会中断安装。
- `kejilion.sh` 新增受限的 KPanel 非交互安装协议。当前审计快照中 99 个标准
  `docker_app`/`docker_app_plus` 内置应用及官方第三方标准应用可直接后台安装，
  产物继续由脚本原生函数创建。
- Agent 重启时会核对 systemd 临时安装服务；已中断任务转为“需要处理”，避免永久
  卡在队列并阻塞后续安装。

### Changed

- 应用市场默认进入“已安装”，并将该筛选放在首位；列表下方新增“浏览全部应用”入口。
- 未安装且具备安装能力的应用卡片直接显示“安装”，其余专属交互应用明确显示需要
  配置向导。
- 安装端口可留空沿用脚本默认值；标准应用复用 `kejilion.sh` 的镜像、Compose、
  数据目录、账号提示、`appno.txt` 与端口文件业务。

### Security

- 后台脚本任务只接受固定 `install` 动作、受限应用编号/token 和
  `1-65535` 端口并检查占用；由 root-only Agent 创建独立 systemd 服务，Web 容器仍不
  挂载 Docker Socket 或宿主机脚本。
- 任务状态与有界日志使用 `0600` 持久化，只向已认证面板会话提供；并发应用安装
  首版串行化，避免端口、包管理器和脚本状态文件争用。
- KPanel 不自动接受 `kejilion.sh` 首次许可；仅当用户已在终端明确接受且脚本支持
  非交互协议时开放直接安装。

## [0.15.1] - 2026-07-26

### 修复

- 应用市场安装、更新、卸载不再调用 `kejilion.sh` 覆盖的
  `systemctl()` 包装函数，改为校验并调用绝对路径的 systemd 客户端，
  修复创建 unit 链接后报 `Too many arguments.` 并中断的问题。
- 应用市场安装现在从 `/dev/urandom` 原子生成独立的 256-bit Agent
  Token，并以 `root:kejilion-panel 0640` 保存；更新保留既有 Token，
  解决真实 Agent 因令牌文件缺失而拒绝启动的问题。
- Agent systemd 写入白名单补齐 `/home/web/certs` 与
  `/home/web/letsencrypt`，确保 WordPress 一键搭建能够发布证书，并保持
  其余 `/home` 路径只读。
- 应用配置生命周期测试现在在与 `kejilion.sh` 相同的 `systemctl()`
  函数覆盖环境中运行，避免同类集成缺陷再次遗漏。

## [0.15.0] - 2026-07-26

### Added

- “新建网站”把 WordPress 与 IP + 端口反代置于最前，并增加“热门 / 一键成品”
  标签；其余基础站点类型保持直观卡片选择。
- WordPress 一键搭建对齐 `kejilion.sh` 的源码包、`<domain>/wordpress` 目录、
  同名数据库、现有 MySQL 账号、Redis 配置、证书文件名和 Nginx 行为。
- Agent 新增 `sites.wordpress.install` 能力探测；LDNMP、MySQL、目录或证书条件
  不完整时入口明确禁用并给出原因。
- WordPress 安装改为可持久化后台任务，Web 每 2 秒读取进度；避免证书、镜像或
  源码下载超过反代超时后被误报为失败。
- systemd 沙箱仅新增独立可写目录
  `/var/lib/kejilion-panel/wordpress-jobs`，面板登录与审计数据目录继续只读隔离。

### Changed

- WordPress 使用 ACME Webroot 在现有 Nginx 在线时签发证书，不再照搬脚本停止
  整个 Nginx 容器的实现；最终业务产物与脚本保持一致。
- 安装过程中先发布仅服务 ACME 的临时配置，最终配置通过 `nginx -t` 后原子切换。
- 已存在脚本标准 `/home/web/docker-compose.yml`、但 MySQL/PHP/Redis 当前停机
  的主机，只启动这四个固定服务且使用 `--no-recreate`，不重建 Nginx。
- Kejilion 应用市场安装配置、宿主 Agent 版本校验与 Docker 镜像引用同步更新
  到 `0.15.0`。

### Security

- 固定 WordPress 源码包 SHA-256 与 Certbot 多架构镜像 digest；远端内容变化时
  拒绝安装，避免未审计代码静默进入宿主机。
- 同名域名、目录、配置、证书或数据库一律冲突拒绝，不执行脚本的“同名先删除”逻辑。
- 数据库与 Certbot 只使用固定 Docker API 操作；失败时核对内容后回滚本次新建的
  Nginx 配置、目录、数据库和复制证书，不删除任何既有业务产物。

## [0.14.0] - 2026-07-26

### Added

- 官方容器镜像同时携带同架构 `kejilion-agent`、systemd unit、Compose 与环境变量模板，
  可由 `kejilion/apps` 配置完成一站式安装。
- 支持在 IP + 端口部署后直接使用 `k fd <domain> 127.0.0.1 <port>` 添加 HTTPS 域名。

### Changed

- 直连端口覆盖改为单一 bridge 网络，确保 `kejilion.sh` 的容器端口访问策略能取得
  唯一、有效的容器 IP。

### Security

- 仅可信代理 CIDR 可提供动态 HTTPS Host/Origin；代理 HTTPS 自动启用 Secure Cookie。
- 当 `k fd` 模板未保留 `X-Real-IP` 时，从右向左验证 `X-Forwarded-For`，避免客户端
  伪造地址影响登录限速和审计。

## [0.13.1] - 2026-07-26

### Fixed

- IP + 端口覆盖文件为 Panel 增加独立发布网络，修复 Docker `internal` 网络不会
  建立宿主端口转发的问题；原内部 Agent 网络与固定私网地址保持不变。

## [0.13.0] - 2026-07-26

### Added

- 应用市场由 Agent 从固定的 `https://app.kejilion.sh/` 动态同步第三方入驻目录，
  五分钟缓存并在上游不可用时回退最近一次安全目录或内置快照。
- 新增可选 `direct-port.yml`，支持测试服务器以 IP + 端口访问而不依赖 Nginx。

### Fixed

- Docker 镜像构建现在会复制 `web/public`，应用市场图标不再在生产镜像中缺失。

### Security

- 远程应用目录仅作为展示元数据：固定 HTTPS 来源、限制响应大小和超时、严格校验
  分类、数量、ID、URL 与重复项；远程数据不能获得安装或 Shell 执行权限。
- 新入驻但尚未随版本发布图标的应用使用本地通用图标，不加载第三方远程图标。

本项目遵循语义化版本。所有日期均使用 `YYYY-MM-DD`。

## [0.12.0] - 2026-07-26

### Added

- 系统更新源切换对齐 `kejilion.sh` 的四个业务入口：中国大陆默认、中国大陆教育网、
  海外地区和智能切换；页面改为直观的线路卡片，不再使用二选一下拉框。
- 智能模式沿用脚本判定：公网国家为 `CN` 时使用华为云，其他地区使用发行版官方源；
  地区查询失败时明确回退官方源。
- Agent 可识别 LinuxMirrors 写入的 one-line 与 DEB822 APT 地址，包括带镜像路径前缀的
  海外源，确保脚本换源后 Web 能继续呈现和安全切换。

### Changed

- 中国大陆默认、教育网和海外入口分别固化为 LinuxMirrors 对应列表的首选阿里云、
  北京大学和 xTom 香港线路；换源本身不升级软件、不清缓存。
- 当前写入范围仍严格限定为 Debian/Ubuntu 发行版仓库；RPM、Pacman、Zypper 与第三方
  仓库继续只读，避免未经充分适配就改写系统源。

### Security

- Web 只接受四个固定枚举，不接受 URL 或 Shell；Agent 不下载、执行 LinuxMirrors 在线脚本。
- 修改前创建版本化备份，使用隔离 APT lists/cache 执行短超时 `apt-get update`；验证失败
  自动恢复全部已修改文件，Docker、NodeSource 等第三方源保持不变。

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
