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
| SSH 端口 | `sshd_config` 与片段 | `sshd -t`、新端口探测、保留旧会话、可回滚 |
| DNS | `resolv.conf` 与管理器 | 按 resolved/resolvconf/静态模式处理、连通性回读 |
| 时区 | `/etc/timezone` 或 `localtime` | IANA 名称白名单、回读 |
| 虚拟内存 | `/proc/meminfo`、`/proc/swaps` | 仅管理 KPanel 专属 swapfile，不清除现有分区 |
| 系统镜像源 | APT/DNF/APK 源地址 | 备份、语法检查、连通性测试、失败回滚 |
| V4/V6 优先 | `gai.conf` | 仅维护带 KPanel 标识的规则，不删除用户配置 |
| 内核优化 | Kejilion sysctl 产物 | 参数白名单、独立文件、应用校验、版本化回滚 |
| BBR | 当前/可用拥塞算法与 qdisc | 内核能力检查、独立 sysctl 文件、回读 |
| 重装系统 | 不适用 | 带外控制台、备份证明、一次性恢复凭证、二次确认 |

## 当前阶段

当前实现开放真实状态读取和能力门控。所有写能力由 Agent 明确返回
`enabled: false` 及原因，页面可查看状态和安全要求，但不会发送修改命令。
这保证页面设计可以先稳定上线，同时不改变现有 `kejilion.sh` 业务和宿主机配置。

后续每开放一个写能力，都必须同时交付：

1. 类型化请求和输入验证；
2. 变更前快照及回滚；
3. 变更后回读验证；
4. Panel CSRF/Origin 防护和审计记录；
5. 脚本修改、Web 修改和外部修改三种一致性测试。
