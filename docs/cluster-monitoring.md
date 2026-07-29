# KPanel 集群监控与联邦只读协议

- 状态：首版实现
- 协议：`v1`
- 范围：主机概要监控、独立面板跳转、接入授权与撤销

## 1. 产品边界

每台 KPanel 都是对等节点，可同时作为：

- **中心端**：保存远端节点授权，后台轮询远端概要并向当前浏览器提供缓存列表。
- **被控端**：签发一次性授权码，向已授权中心端返回本机只读概要。

首版不提供跨主机 Shell、批量写操作、共享 Session 或免登录跳转。点击“打开面板”只打开
配对时保存的 HTTPS 根地址，目标面板仍需独立登录。

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

## 3. 配对与身份

被控端在“集群 → 接入授权”生成：

```text
<16 hex id>.<64 hex secret>
```

规则：

- secret 由 32 字节安全随机数生成；
- 5 分钟过期，只能成功消费一次；
- 连续 5 次错误后失效；
- 仅保存 SHA-256，不把明文写入状态、审计、日志或 URL；
- 权限固定为 `cluster.summary.read`。

中心端为每台远端主机生成独立 Ed25519 密钥。被控端只保存公钥、指纹和 scope；中心端私钥
单独存放在 `cluster-secrets/*.ed25519`，目录权限 `0700`、文件权限 `0600`，不进入
`cluster-state.json`。

签名覆盖：

```text
method
escaped path
controller ID
target node ID
Unix timestamp
nonce
SHA-256(body)
```

被控端验证目标身份、固定路径、协议版本、±60 秒时间窗和 5 分钟有界 nonce 缓存。节点 ID
或协议变化时停止信任，不自动接受新身份。

`v1` 的签名摘要与撤销接口均严格无正文，因此最后一项固定为空正文摘要；chunked 或任意
非空正文都会在验签前被拒绝。未来如增加带正文的签名动作，必须先实现真实正文摘要并升级
协议及篡改测试，不能沿用 `v1` 的空正文常量。

## 4. 网络与 SSRF

联邦只访问用户登记的规范化 HTTPS origin：

- 只允许 `https://host[:port]`，禁止 userinfo、路径、查询、fragment 和重定向；
- TLS 最低 1.2，验证系统 CA、主机名和证书有效期；
- 不继承 `HTTP_PROXY`/`HTTPS_PROXY`；
- 每次拨号重新解析全部地址，先校验再直接拨校验后的 IP，TLS SNI 保留原主机名；
- 默认拒绝 loopback、link-local、multicast、unspecified、RFC1918、ULA、CGNAT、
  文档保留地址、NAT64/6to4/Teredo 转换前缀和云元数据链路；
- 私网只能通过部署端 `KEJILION_PANEL_CLUSTER_PRIVATE_CIDRS` 精确放行；
- 混合返回公网与受限地址时整体拒绝，防止 DNS rebinding。

Agent 仍只监听本机权限受限的 Unix Socket。联邦入口位于 Panel，不开放 Agent TCP，也不
复用 Agent Token、Bootstrap Token、管理员密码或 Session Cookie。

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
GET    /api/v1/cluster/controllers
DELETE /api/v1/cluster/controllers/{id}
```

Panel 间固定接口：

```text
POST   /api/v1/federation/pair
GET    /api/v1/federation/summary
DELETE /api/v1/federation/revoke
```

配对请求上限 16 KiB，摘要响应上限 64 KiB，严格 JSON、拒绝未知字段和多值 JSON。

## 6. 轮询与状态

中心端默认每 30 秒轮询，加入 ±20% 抖动；全局并发上限 8、单主机连接上限 2、总超时
6 秒、响应头超时 3 秒。失败按指数退避，最大 5 分钟，不自动重复配对或写操作。

状态规则：

- `online`：最近成功且没有连续失败；
- `degraded`：有最近快照，但当前出现少量失败；
- `stale`：最近成功超过 90 秒；
- `offline`：连续 3 次失败；
- `auth_failed`：签名或授权失效；
- `tls_error`：证书或 TLS 校验失败；
- `incompatible`：协议或远端节点身份变化。

只保存最新快照，不保存高频历史。`cluster-state.json` 最大 4 MiB，采用同目录临时文件、
`0600`、同步和原子替换；每 5 分钟及正常退出时 checkpoint。轮询进程重启后从最新快照
恢复，真实状态仍由下一次远端摘要刷新。

删除主机时先原子删除状态记录，再清理独立私钥文件，避免进程中断后出现“仍有主机记录、
但凭据已丢失”的不可恢复状态。若私钥清理暂时失败，API 会明确返回清理状态；下次服务
初始化会删除不再被任何主机记录引用的有效凭据文件。

标准 Compose 为 Panel 增加独立出站网络，仅用于联邦 HTTPS；容器仍保持非 root、只读根、
`cap_drop: ALL` 和 `no-new-privileges`。应用层每次拨号都执行上述 SSRF 与 TLS 校验。对
网络隔离要求更高的部署，仍建议在宿主机 `DOCKER-USER`/专用出口网关增加出站 ACL；首版
尚未把联邦轮询拆成独立 sidecar。

## 7. 编码前质量记录

| 项目 | 决策 |
| --- | --- |
| 流量路径 | 浏览器 → 当前 Panel；当前 Panel → 远端 Panel HTTPS；远端 Panel → 本机 Agent Unix Socket |
| 不可信输入 | 主机名称、origin、授权码、DNS 结果、远端证书和远端 JSON |
| 权限与可写范围 | Panel 只写自身 `cluster-state.json` 和 `cluster-secrets`；不写宿主机业务目录 |
| 最坏输入/输出 | 主机 100；配对 16 KiB；摘要 64 KiB；Store 4 MiB；控制端 256；有效授权码 16 |
| 最大并发 | 轮询 8；单主机连接 2；请求与 nonce/rate-limit 缓存均有界 |
| 超时与重试 | 连接 2 秒、响应头 3 秒、总计 6 秒；读取失败退避到 5 分钟；无写入自动重试 |
| 真实状态与缓存 | 远端 Agent 实时摘要是事实；中心只缓存最近快照；本机摘要缓存 5 秒 |
| 失败与恢复 | 保留最近成功快照；认证/TLS/身份错误单独标识；先尽力撤销远端授权，再删除状态，最后清理凭据；孤立凭据启动时回收 |
| 性能预算 | 浏览器单请求，无 N+1；100 台按 30 秒轮询约 3.3 请求/秒，最多 8 并发 |
| 网络入侵风险 | SSRF、DNS rebinding、TLS 劫持、授权码猜测、签名重放、恶意大响应和轮询 DoS |

## 8. 验收

自动测试至少覆盖：

- HTTPS origin 规范化、loopback/私网/元数据/IPv4-mapped IPv6、混合 DNS 与 rebinding；
- 授权码过期、错误次数、并发单次消费及明文不落盘；
- 签名篡改、过期/未来时间和 nonce 重放；
- 本机始终只出现一次、不落盘、不占远端配额、不可删除；
- Browser Session、Origin、CSRF、未知字段、请求体和响应体上限；
- 100 台并发上限、退避、取消、重启恢复和 `go test -race`；
- 前端无 N+1、轮询不重叠、失败保留旧数据、外链安全与移动端布局；
- 标准 Compose 和应用市场部署都能出站验证 HTTPS，Panel 仍无 Docker Socket 和宿主权限。

发布前执行 L2 验证；正式版本与镜像发布仍按 L3 流程执行。

## 9. 回滚

该功能不迁移网站、Docker 或系统业务状态。回滚 Panel 到上一稳定镜像后：

- `cluster-state.json` 与 `cluster-secrets` 保留，不影响其他面板功能；
- 旧版本不会读取集群文件；
- 如需彻底撤销，先在各节点“接入授权”中撤销控制端，再在停机维护窗口备份并删除集群文件；
- 不得通过回滚删除 `/home/web`、Docker 容器或 Agent Token。
