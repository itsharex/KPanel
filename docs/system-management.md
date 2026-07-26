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
| 虚拟内存 | `/proc/meminfo`、`/proc/swaps` | 仅管理 KPanel 专属 swapfile，不清除现有分区 |
| 系统镜像源 | APT/DNF/APK 源地址 | 首版支持 Debian/Ubuntu 官方源与阿里云源；第三方源不修改，`apt-get update` 失败回滚 |
| V4/V6 优先 | `gai.conf` | 维护 `kejilion.sh` 同一 precedence 规则并保留其他用户配置 |
| 内核优化 | Kejilion sysctl 产物 | 固定参数白名单、应用校验、版本化回滚；外部脚本配置不覆盖 |
| BBR | 当前/可用拥塞算法与 qdisc | 内核能力检查、独立 sysctl 文件、回读 |
| 重装系统 | 不适用 | 带外控制台、备份证明、一次性恢复凭证、二次确认 |

## v0.3 写入范围

Agent 根据宿主机命令、配置管理器和 root/sandbox 条件动态返回 capability。
满足条件时，登录管理员可在页面填写固定字段并二次确认；不满足条件时只展示
真实状态和明确原因。

已开放：主机名、安全新增 SSH 端口、systemd-resolved DNS、时区、KPanel
专属 Swap、Debian/Ubuntu APT 镜像预设、地址优先级、KPanel 内核调优预设和
BBR。重装系统继续锁定；SSH 旧端口删除、静态 `resolv.conf` 接管、任意软件源
URL、任意 sysctl 和任意 Shell 均不开放。

系统备份保存在 `/var/lib/kejilion-panel/system/backups`。Panel 数据、
`kejilion.sh` 文件、现有网站、其他容器和其他 Swap 不进入系统操作事务。
