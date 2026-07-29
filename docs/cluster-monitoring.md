# KPanel 集群监控与联邦只读协议

- 状态：已实现
- 协议：新配对默认 `v2`，兼容既有 `v1`
- 范围：主机概要监控、独立面板跳转、接入授权与撤销

## 1. 产品边界

每台 KPanel 都是对等节点，可同时作为：

- **中心端**：保存远端节点授权，后台轮询远端概要并向当前浏览器提供缓存列表。
- **被控端**：签发一次性授权码，向已授权中心端返回本机只读概要。

当前不提供跨主机 Shell、批量写操作、共享 Session 或免登录跳转。点击“打开面板”只打开
配对时保存的根地址，目标面板仍需独立登录。集群采集链路支持 HTTPS，或在没有域名时使用
端到端加密的 `http://公网IP:非80端口`；后者只保护 KPanel 间的集群数据，浏览器登录目标
面板仍是普通 HTTP，页面会在跳转前明确警告。

主机列表自动包含当前 KPanel，标记为“本机”并排在第一位。本机摘要直接经现有 Unix
Socket 读取本地 Agent，不配对、不生成密钥、不写入远端主机存储，也不占 100 台远端配额。
本机不能重复添加或移除；本机 Agent 暂时不可用时只把本机卡片标记为异常，不隐藏远端数据。

## 2. 数据与页面

左侧“概览”后增加“集群”，路由为 `/cluster`。页面一次读取中心端缓存并展示：

- CPU、内存、磁盘使用率；
- 网络累计量换算的收发速率；
- 系统、内核、架构、运行时间；
- 公网 IP、国家/地区、城市和 ISP；
- Panel/Agent 版本、轮询延迟、最后成功时间和错误状态。

浏览器每 15 秒读取一次中心端列表，请求不重叠；页面隐藏时暂停，恢复可见时立即刷新。
刷新失败保留上一次成功数据。最多 100 台远端主机，桌面三列、平板两列、手机一列。

联邦摘要使用独立的 `HostTelemetry`，不返回网站、应用、Docker、SSH 端口、DNS、系统配置、
凭据或宿主机管理能力。

## 3. 配对、身份与兼容

### 3.1 v2（新配对默认）

被控端在“集群 → 接入授权”生成：

```text
kp2.<base64url-json>
```

规则：

- 授权码包含目标节点 ID、X25519 公钥、短期 code ID、32 字节随机 secret 和到期时间；
- 5 分钟过期，只能成功消费一次；
- 连续 5 次错误后失效；
- secret 只用于本地派生配对 PSK，不出现在联邦 HTTP 请求、状态、审计或日志中；
- 权限固定为 `cluster.summary.read`。

中心端为每台远端主机生成独立 X25519 身份。初次配对使用
`Noise_IKpsk0_25519_ChaChaPoly_SHA256`，日常采集和撤销使用
`Noise_IK_25519_ChaChaPoly_SHA256`。请求方法、固定路径、控制端 ID、目标节点 ID、
code ID、时间戳和 request ID 都进入认证 prologue；业务正文始终位于 Noise 加密消息内。

配对采用 `Pair → Commit` 两阶段落盘。一次性授权码仍在 5 分钟后失效；目标端已认证并绑定的
事务可在 24 小时内完成 Commit，使中心端重启或短时断网后能够继续收敛。未 Commit 的控制端
不会出现在有效授权列表中。

节点私钥和每主机凭据只存放在 `cluster-secrets-v2/`，目录权限 `0700`、文件权限 `0600`；
`cluster-state-v2.json` 只保存公钥、指纹、状态和引用名。密钥写入与状态引用使用同一临界区，
状态采用同步、原子替换和恢复副本，启动及 checkpoint 会清理无引用凭据。

被控端验证目标身份、固定路径、协议版本、±2 分钟时间窗和有界 request ID 重放缓存。节点
ID 或静态公钥变化时停止信任，不自动接受新身份。

### 3.2 v1 兼容

既有 v1 主机、`cluster-state.json`、`cluster-secrets/*.ed25519` 和
`POST /api/v1/cluster/pairing-codes` 保持可读可用。v1 仍要求公网 HTTPS，并使用原有
Ed25519 签名。新前端调用 `/api/v1/cluster/pairing-codes/v2`，不会把旧授权码接口改成
另一种格式。v1 与 v2 主机可同时存在，撤销和凭据互不影响。

## 4. 网络与 SSRF

联邦只访问用户登记的规范化 origin：

- v1 只允许 `https://host[:port]`；
- v2 允许 `https://host[:port]`，或 `http://字面量IP:非80端口`；
- 两种形式都禁止 userinfo、路径、查询、fragment 和重定向；
- TLS 最低 1.2，验证系统 CA、主机名和证书有效期；
- 不继承 `HTTP_PROXY`/`HTTPS_PROXY`；
- 每次拨号重新解析全部地址，先校验再直接拨校验后的 IP，TLS SNI 保留原主机名；
- 默认拒绝 loopback、link-local、multicast、unspecified、RFC1918、ULA、CGNAT、
  文档保留地址、NAT64/6to4/Teredo 转换前缀和云元数据链路；
- 私网只能通过部署端 `KEJILION_PANEL_CLUSTER_PRIVATE_CIDRS` 精确放行；
- 混合返回公网与受限地址时整体拒绝，防止 DNS rebinding。

HTTP v2 的集群正文使用 Noise 端到端加密且绑定目标静态身份，但 HTTP 本身不能保护浏览器
访问目标管理页。Agent 仍只监听本机权限受限的 Unix Socket。联邦入口位于 Panel，不开放
Agent TCP，也不复用 Agent Token、Bootstrap Token、管理员密码或 Session Cookie。

## 5. API

浏览器接口需要 Panel Session；所有写入同时验证 Origin、CSRF 并写审计：

```text
GET    /api/v1/cluster/hosts
POST   /api/v1/cluster/hosts
GET    /api/v1/cluster/hosts/{id}
PATCH  /api/v1/cluster/hosts/{id}
DELETE /api/v1/cluster/hosts/{id}
POST   /api/v1/cluster/hosts/{id}/refresh
POST   /api/v1/cluster/pairing-codes
POST   /api/v1/cluster/pairing-codes/v2
GET    /api/v1/cluster/controllers
DELETE /api/v1/cluster/controllers/{id}
```

Panel 间固定接口：

```text
POST   /api/v1/federation/pair
GET    /api/v1/federation/summary
DELETE /api/v1/federation/revoke

POST   /api/v2/federation/pair
POST   /api/v2/federation/commit
POST   /api/v2/federation/summary
POST   /api/v2/federation/revoke
```

v2 只接受以上四个精确 POST 路径，不接受查询参数或 `RawPath` 变体；Noise 外层请求上限
96 KiB，解密后业务负载上限 64 KiB。所有接口使用严格 JSON，拒绝未知字段和多值 JSON。

## 6. 轮询与状态

中心端默认每 30 秒轮询，加入 ±20% 抖动；全局并发上限 8、单主机连接上限 2、总超时
6 秒、响应头超时 3 秒。失败按指数退避，最大 5 分钟，不自动重复配对或写操作。

状态规则：

- `pairing`：两阶段安全配对尚在后台收敛；
- `revoking`：进程中断后恢复到待撤销状态；
- `online`：最近成功且没有连续失败；
- `degraded`：有最近快照，但当前出现少量失败；
- `stale`：最近成功超过 90 秒；
- `offline`：连续 3 次失败；
- `auth_failed`：签名或授权失效；
- `tls_error`：证书或 TLS 校验失败；
- `incompatible`：协议或远端节点身份变化。

只保存最新快照，不保存高频历史。v1 的 `cluster-state.json` 和 v2 的
`cluster-state-v2.json` 均采用同目录临时文件、`0600`、同步和原子替换；每 5 分钟及正常
退出时 checkpoint。轮询进程重启后从最新快照和未完成事务恢复，真实状态仍由下一次远端
摘要刷新。

删除主机时先尽力撤销远端授权，再让本地状态和凭据收敛。远端不可达不会把本地条目永久卡
在“撤销中”；API 返回 `remoteRevoked=false`，目标端残留授权可在其“接入授权”页面手动
撤销。若凭据清理暂时失败，API 会明确返回清理状态；下次服务初始化会删除无引用凭据。

标准 Compose 为 Panel 增加独立出站网络，仅用于联邦 HTTPS 或 Noise 加密 HTTP；容器仍
保持非 root、只读根、`cap_drop: ALL` 和 `no-new-privileges`。应用层每次拨号都执行上述
SSRF 与 TLS 校验。对
网络隔离要求更高的部署，仍建议在宿主机 `DOCKER-USER`/专用出口网关增加出站 ACL；首版
尚未把联邦轮询拆成独立 sidecar。

## 7. 编码前质量记录

| 项目 | 决策 |
| --- | --- |
| 流量路径 | 浏览器 → 当前 Panel；当前 Panel → 远端 Panel HTTPS 或 Noise 加密 HTTP；远端 Panel → 本机 Agent Unix Socket |
| 不可信输入 | 主机名称、origin、授权码、DNS 结果、远端证书和远端 JSON |
| 权限与可写范围 | Panel 只写自身 v1/v2 集群状态与凭据目录；不写宿主机业务目录 |
| 最坏输入/输出 | 主机 100；v1 配对 16 KiB；v2 外层 96 KiB、解密负载 64 KiB；摘要 64 KiB；Store 4 MiB；控制端 256；有效授权码 16 |
| 最大并发 | 轮询 8；单主机连接 2；请求与 nonce/rate-limit 缓存均有界 |
| 超时与重试 | 连接 2 秒、响应头 3 秒、总计 6 秒；读取失败退避到 5 分钟；无写入自动重试 |
| 真实状态与缓存 | 远端 Agent 实时摘要是事实；中心只缓存最近快照；本机摘要缓存 5 秒 |
| 失败与恢复 | 保留最近成功快照；认证/TLS/身份错误单独标识；先尽力撤销远端授权，再删除状态，最后清理凭据；孤立凭据启动时回收 |
| 性能预算 | 浏览器单请求，无 N+1；100 台按 30 秒轮询约 3.3 请求/秒，最多 8 并发 |
| 网络入侵风险 | SSRF、DNS rebinding、TLS 劫持、授权码猜测、签名重放、恶意大响应和轮询 DoS |

## 8. 验收

自动测试至少覆盖：

- HTTPS 与字面量 IP origin 规范化、loopback/私网/元数据/IPv4-mapped IPv6、混合 DNS 与 rebinding；
- 授权码过期、错误次数、并发单次消费及明文不落盘；
- v1 签名及 v2 Noise 篡改、错误 PSK/身份/路径、过期/未来时间和 request ID 重放；
- 两阶段配对重启恢复、密钥/状态原子性、慢节点不阻塞其他轮询与管理操作；
- 本机始终只出现一次、不落盘、不占远端配额、不可删除；
- Browser Session、Origin、CSRF、未知字段、请求体和响应体上限；
- 100 台并发上限、退避、取消、重启恢复和 `go test -race`；
- 前端无 N+1、轮询不重叠、失败保留旧数据、外链安全与移动端布局；
- 标准 Compose 和应用市场部署都能出站验证 HTTPS，Panel 仍无 Docker Socket 和宿主权限。

发布前执行 L2 验证；正式版本与镜像发布仍按 L3 流程执行。

## 9. 回滚

该功能不迁移网站、Docker 或系统业务状态。回滚 Panel 到上一稳定镜像后：

- v1 的 `cluster-state.json`/`cluster-secrets` 与 v2 的
  `cluster-state-v2.json`/`cluster-secrets-v2` 保留，不影响其他面板功能；
- 旧版本继续读取 v1 文件并忽略 v2 文件，因此原有 v1 主机仍可回滚使用；
- 如需彻底撤销，先在各节点“接入授权”中撤销控制端，再在停机维护窗口备份并删除集群文件；
- 不得通过回滚删除 `/home/web`、Docker 容器或 Agent Token。
