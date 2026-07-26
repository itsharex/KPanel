# 系统监控与管理

KPanel 的系统管理业务以 `kejilion.sh` 为功能基准，但不把整个 Shell
脚本暴露给 Web。Agent 只提供固定输入、固定输出的类型化接口；Panel
负责登录、CSRF、Origin 校验和审计，不能传入任意命令。

## 状态一致性

- 首页状态直接读取宿主机当前配置，不以 KPanel 数据库代替系统事实。
- `kejilion.sh` 或人工修改产生的 SSH、DNS、时区、Swap、镜像源、
  `gai.conf`、内核优化和 BBR 状态，刷新后应被 Agent 重新识别。
- Web 后续执行变更后必须回读同一状态接口；回读不一致时任务不得标记成功。
- 新字段保持向后兼容。旧 Agent 未返回时，前端显示“待 Agent 升级”，
  不构造虚假的默认状态。

## 功能与安全边界

| 业务 | 脚本产物识别 | Web 写入开放条件 |
| --- | --- | --- |
| 主机名 | kernel hostname | hostname 规则校验、原子更新、回读 |
| SSH 端口 | `sshd_config` 与片段 | 首版只安全新增端口；`sshd -t`、防火墙放行、reload、监听探测并保留旧端口 |
| DNS | `resolv.conf` 与管理器 | 首版只接管 systemd-resolved drop-in，不替换或锁定 `resolv.conf` |
| 时区 | `/etc/timezone` 或 `localtime` | IANA 名称白名单、回读 |
| 虚拟内存 | `/proc/meminfo`、`/proc/swaps`、`/swapfile` | 与 `kejilion.sh` 共用 `/swapfile`；合并旧版 KPanel Swap，不清除现有分区或第三方 swapfile |
| 系统镜像源 | APT/RPM/APK/Pacman/Zypper 源地址 | 首版支持 Debian/Ubuntu 官方源与阿里云源；第三方源不修改，`apt-get update` 失败回滚 |
| V4/V6 优先 | `gai.conf` | 维护 `kejilion.sh` 同一 precedence 规则并保留其他用户配置 |
| 内核优化 | Kejilion sysctl 产物 | 五种固定预设、内存自适应、逐项应用和版本化回滚；合法脚本产物可接管 |
| BBR | 当前/可用拥塞算法与 qdisc | 内核能力检查、独立 sysctl 文件、回读 |
| 系统更新 | APT/DNF/YUM/Pacman/Zypper 源与后台任务状态 | 按发行版白名单选择固定命令序列；不杀死软件包进程、不删除锁、不自动重启 |
| 系统清理 | 软件包管理器与 journal 后台任务状态 | 缓存模式或标准安全模式；不清空日志目录、临时目录、Docker、网站和备份 |
| 重装系统 | 不适用 | 带外控制台、备份证明、一次性恢复凭证、二次确认 |

## v0.3 写入范围

Agent 根据宿主机命令、配置管理器和 root/sandbox 条件动态返回 capability。
满足条件时，登录管理员可在页面填写固定字段并二次确认；不满足条件时只展示
真实状态和明确原因。

已开放：主机名、安全新增 SSH 端口、systemd-resolved DNS、时区、脚本兼容
`/swapfile`、Debian/Ubuntu APT 镜像预设、地址优先级、KPanel 内核调优预设和
BBR。重装系统继续锁定；SSH 旧端口删除、静态 `resolv.conf` 接管、任意软件源
URL、任意 sysctl 和任意 Shell 均不开放。

## v0.5 后台系统维护

“系统更新”和“系统清理”参考当前 `kejilion.sh` 的业务顺序，但由固定参数的
systemd transient service 执行。Web 请求只能选择 `update/full`、
`cleanup/cache` 或 `cleanup/standard`，不能传入命令、包名或文件路径。

- 更新：APT 执行 dpkg 恢复、刷新索引和 `full-upgrade`；RHEL 系执行
  DNF/DNF5/YUM 缓存刷新与升级；Arch/Manjaro 执行 `pacman -Syu`；
  openSUSE/SLES 执行 Zypper 刷新与升级。
- 缓存清理：只调用对应软件包管理器的固定缓存清理参数。
- 标准清理：APT、DNF/YUM 在自身支持时额外执行 `autoremove`；Pacman 和
  Zypper 不根据动态包名删除软件包。所有系统轮转 journal，保留最近 7 天并
  限制到 500 MiB。
- 后台状态持久化在
  `/var/lib/kejilion-panel/system/maintenance-state.json`；同一时间只允许
  一个维护任务。
- 更新和清理属于不可逆的软件包事务，KPanel 不宣称自动回滚；失败时保留
  阶段和错误摘要供人工检查。任务不会自动重启宿主机。

系统备份保存在 `/var/lib/kejilion-panel/system/backups`。Panel 数据、
`kejilion.sh` 文件、现有网站、其他容器和其他 Swap 不进入系统操作事务。

## v0.5.1 虚拟内存事务

- 状态读取同时区分 `/swapfile`、旧版 KPanel Swap 和其他活动 Swap，页面
  的总量仍以 `/proc/meminfo` 为准。
- 设置入口提供 `kejilion.sh` 相同的 1/2/4 GiB 常用值，并允许
  256–65536 MiB 自定义值；脚本默认值为 1 GiB。
- 创建或调整前先检查受管 Swap 已用空间能否安全回收到内存，再在同一文件
  系统分配临时 swapfile。内存安全门或空间分配不通过时不触碰现状。
- 事务只会 `swapoff` `/swapfile` 和旧版 KPanel 路径；不会执行 `wipefs`，
  不会停用 Swap 分区或第三方 swapfile。
- 新文件、`fstab` 和 `swapon` 任一步失败时，恢复原文件、原 `fstab` 和原
  活动状态。成功后启动项使用脚本同款
  `/swapfile swap swap defaults 0 0`，脚本与 Web 可双向识别。
- 文件系统写入由固定参数的 root systemd transient service 完成。常驻
  Agent 仍受原 systemd 沙箱限制，Web 不能传入路径或任意命令。浏览器请求
  中断不会杀死已经启动的事务，事务仍会完成或执行自身回滚。

## v0.6 内核优化

- Web 提供与当前 `kejilion.sh` 一致的高性能、均衡、网站、直播和游戏服五种
  本地预设，以及还原默认设置；API 只接受这些枚举值。
- 预设参数与脚本的 `_kernel_optimize_core` 保持一致，并按 `/proc/meminfo`
  的 `MemTotal` 使用 `<1 GiB`、`1–4 GiB`、`4–16 GiB`、`≥16 GiB` 四档
  自适应规则。
- 产物写入脚本相同的
  `/etc/sysctl.d/99-kejilion-optimize.conf`，保留 `# 模式:` 与 `# 场景:`
  标识。脚本执行后 Web 可读取；Web 切换后脚本菜单也可识别当前模式。
- 已识别的脚本手动预设允许由 Web 切换；`99-network-optimize.conf` 自动调优
  结果可读取并在用户选择其他模式或还原时清理。未知结构的同名文件仍返回冲突。
- 参数使用固定列表逐项 `sysctl -w`，兼容内核缺少可选参数的情况；若全部参数
  都无法应用，则恢复 sysctl、limits、modules 和 BBR 冲突文件并重新加载。
- BBR 可用时与脚本一样使用 `bbr + fq`，否则使用 `cubic + fq_codel`；同时
  管理透明大页、nofile 限制和 `tcp_bbr` 模块持久化。
- 脚本中的“自动调优”需要实时测速并在线获取 `network-optimize.sh`。首版 Web
  不执行远程脚本；已有自动调优产物仍会被状态接口正确识别。

## v0.7 多发行版维护

- 读取 `/etc/os-release` 的 `ID` 和 `ID_LIKE`，只允许 Debian/Ubuntu 系、
  RHEL/Fedora 系、Arch/Manjaro 和 openSUSE/SLES 的已知软件包管理器。
- 命令白名单为 APT/dpkg、DNF/DNF5/YUM、Pacman 和 Zypper；Web 仍只能提交
  `full`、`cache` 或 `standard` 枚举，不能指定命令、包名、仓库或参数。
- 启动后台任务前同时确认软件包管理器、发行版源文件和 `systemd-run`；
  缺少任一条件时直接返回只读原因，不创建失败任务。
- RHEL 系识别 `/etc/yum.repos.d/*.repo`，Arch 识别
  `/etc/pacman.d/mirrorlist`，openSUSE/SLES 识别
  `/etc/zypp/repos.d/*.repo`。
- 软件源切换仍只开放 Debian/Ubuntu 的官方源和阿里云预设。RPM、Pacman、
  Zypper 的源只读取不改写，避免破坏订阅、模块流、镜像排序或第三方仓库。
- Alpine/OpenRC 尚不能运行当前 systemd Agent 安装方式，因此不开放宿主机
  写入。未知发行版同样保持只读。
