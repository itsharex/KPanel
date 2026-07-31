# KPanel 性能、稳定性、资源与网络入侵安全开发规范

- 版本：2026-07-31
- 状态：长期强制规范

本规范服从仓库根目录的 [`PROJECT_RULES.md`](../PROJECT_RULES.md)，适用于 KPanel 的设计、
开发、代码评审、测试和发布。历史实现或文档与本规范冲突时，新改动必须按本规范收敛。

## 1. 范围与解释

本规范中的“安全性”专指抵御来自网络、恶意请求、被篡改依赖和被攻陷上游服务的攻击，包括：

- 未授权访问、撞库、Session 劫持和跨站请求；
- 命令注入、路径穿越、SSRF、Host Header/代理头伪造和请求走私；
- 凭据泄露、供应链篡改、容器逃逸和宿主机提权；
- 利用超大输入、无限并发、慢连接、日志或任务耗尽 CPU、内存、FD、PID、磁盘和连接。

以下内容不属于安全控制，不能作为安全改进：

- 因资源不是 KPanel 创建而禁止管理员管理；
- 因操作危险、会卸载或会改变系统而隐藏按钮、禁用 API；
- 用固定确认词、所有权标签或 KPanel 自保护代替真实鉴权；
- 以“为了安全”为理由不实现 `kejilion.sh` 已支持的功能。

已经通过身份验证的管理员仍应获得底层业务实际支持的完整能力。防入侵措施必须约束攻击入口、
协议、输入、权限、供应链和资源消耗，不能缩减合法业务范围。

## 2. 质量目标

每项改动同时满足四个目标：

1. **性能可预测**：常用读取链路无无界工作量，关键延迟和吞吐无不可解释退化。
2. **稳定可恢复**：页面关闭、网络抖动、Agent 重启或脚本失败后，任务状态和真实资源可恢复。
3. **资源有上限**：请求、响应、缓存、并发、日志、归档、FD、PID、CPU 和内存均有明确边界。
4. **抵御网络入侵**：公网请求不能越过 Panel、Agent、Docker 和宿主机之间的信任边界。

“测试通过”必须有可重复命令或实机证据；不得仅凭代码阅读、进程退出码、systemd
`Result=success` 或页面提示判定。

## 3. 固定信任边界

```text
Internet
   |
   v
k fd / HTTPS reverse proxy
   |
   v
paneld (unprivileged container)
   |
   v
Unix Socket + independent token
   |
   v
kejilion-agent (host service)
   |
   +--> kejilion.sh / systemd / filesystem
   +--> local Docker Unix Socket
```

| 边界 | 强制要求 |
| --- | --- |
| Internet → 反向代理 | HTTPS；Host 和客户端地址只由明确配置的代理产生；公网来源不得伪造转发头 |
| 反向代理 → Panel | 只信任最小 CIDR；`X-Forwarded-Proto` 必须是单一 `https`；`k fd` 域名反代必须持续兼容 |
| 浏览器 → Panel API | 身份、Session、CSRF/Origin、方法、Content-Type、请求体和速率均验证 |
| Panel → Agent | 仅 Unix Socket；独立 Bearer Token；Panel 不挂载 Docker Socket、不获得宿主机特权 |
| Agent → 宿主机 | 固定动作枚举、结构化参数、固定业务根；不接受由 Web 字段拼出的任意 Shell |
| 交互终端 → 脚本 | 只能连接已登记的脚本任务和 PTY；输入有长度上限，任务仍执行脚本原生交互语义 |
| 供应链 → 运行时 | 镜像、脚本、Action 和工具固定版本或摘要；下载有大小、超时和完整性校验 |

新增网络入口、监听端口、代理 CIDR、外联地址、systemd capability、Docker Socket 或宿主机目录
挂载，默认按 L2 以上变更处理。

## 4. 网络入侵安全基线

### 4.1 HTTP 与反向代理

- 公网访问优先经 HTTPS 反向代理；直连端口不得改变 Agent 和宿主机信任边界。
- `k fd` 必须支持可信代理传递的动态域名，但只有立即来源位于可信 CIDR 且
  `X-Forwarded-Proto: https` 时才信任该 Host/Origin。
- 未可信来源的 `Forwarded`、`X-Forwarded-*`、Host 和客户端 IP 不得参与授权、安全 Cookie
  或审计身份判断。
- HTTP Server 必须设置 `ReadHeaderTimeout`、`ReadTimeout`、`WriteTimeout`、`IdleTimeout`、
  `MaxHeaderBytes` 和请求体上限；禁止使用无边界默认值。
- API 只接受声明的方法和 Content-Type；未知字段、重复冲突参数和超限请求必须明确拒绝。
- 保留 CSP、HSTS（仅真实 HTTPS/可信 HTTPS 代理）、`nosniff`、Frame 限制、
  Referrer Policy、COOP 和 CORP。
- 健康检查不得返回版本以外的敏感配置、路径、凭据、错误堆栈或业务数据。

### 4.2 身份、Session 与抗撞库

- 密码使用当前项目 Argon2id 强度，不得仅为降低内存而下调算法参数。
- 昂贵密码哈希必须有小型并发信号量和快速 `429`；不能让并发登录把容器推到 OOM 边缘。
- 登录失败按用户与来源进行有界限速；错误信息不得泄露用户名是否存在。
- Bootstrap Token 一次使用并及时失效；Session ID 登录后重新生成，Cookie 保持
  `Secure`、`HttpOnly`、严格 SameSite 和限定 Path。
- 所有改变状态的浏览器请求必须验证 Session 与 CSRF/Origin；反向代理不能绕过该检查。

### 4.3 输入、命令和 SSRF

- HTTP 输入先解析为严格结构体，再做长度、范围、枚举和语法验证。
- Web 字段不得通过字符串拼接进入 `sh -c`、`bash -c`、SQL、Nginx 模板或 Docker URL。
- 系统命令使用固定可执行文件和独立参数；需要完整交互时调用登记的
  `kejilion.sh` 入口并连接 PTY。
- Panel/Agent 的出站 HTTP 只能访问代码或配置登记的可信来源；API 不接受任意 URL、
  Socket 地址、Docker endpoint 或代理配置。
- 所有出站请求必须设置超时、禁止意外重定向、限制响应体大小，并验证预期内容结构。
- 对 Host、路径、查询串、归档名、容器 ID、任务 ID、域名、端口和资源版本分别验证；
  不得用一个宽松“通用字符串”校验器覆盖所有场景。

### 4.4 文件、归档与竞态

- 文件动作必须限定在明确业务根目录，使用清理后的绝对路径并复核最终目标仍在根内。
- 不跟随不受信任符号链接；高权限读写在 `Lstat` 后打开文件，并用打开后的
  `Stat`/inode 或 root-scoped API 复核。
- 写配置使用同目录临时文件、权限设置、`fsync`、语法检查和原子替换。
- 解压前拒绝绝对路径、`..`、越界链接、硬链接、设备文件、超大单文件、超大总量和条目炸弹。
- 备份/还原必须校验摘要和结构；还原先进入暂存目录，再原子交换，失败恢复原目录。

### 4.5 凭据与日志

- Token、密码、私钥和云平台凭据使用 `0600` 文件或不回显输入通道，不进入命令行参数。
- 敏感值不得写入任务 JSON、审计、终端历史、错误详情、URL、镜像标签或 Git。
- 日志输出采用 allowlist 式字段；转发第三方输出前执行脱敏并限制长度。
- 测试凭据必须是临时随机值；测试完成后清理，不复用线上 Token 或数据目录。

### 4.6 Panel、容器和 Agent

- Panel 最终镜像保持非 root、scratch/最小运行时、只读根文件系统、`cap_drop: ALL`、
  `no-new-privileges`、受限 PID/CPU/内存和小型 tmpfs。
- Panel 容器不得挂载 `/var/run/docker.sock`、宿主机根目录或 Agent 可写状态目录。
- Agent 权限按实际宿主机动作最小化；新增 capability、可写目录、AddressFamily 或命令
  必须给出必要性、攻击影响和撤销方案。
- Agent 只监听权限受限的 Unix Socket；禁止为方便调试新增 TCP 监听。
- Docker、containerd、runc 和宿主内核属于运行时攻击面，发布验收必须记录实际版本和已知高危状态。

### 4.7 供应链

- GitHub Actions 固定完整提交 SHA；构建镜像、测试工具和远程脚本固定不可变摘要。
- 生产镜像发布版本标签和 `latest` 必须指向同一已验证 digest；安装端按 digest 拉取。
- Go、npm、最终镜像和脚本分别执行漏洞、Secret、配置和完整性扫描。
- 发布产物保留 SHA-256、SBOM、provenance、源码提交和脚本提交/摘要。
- 扫描器本身也必须固定安全版本与摘要；不得临时改用未经复核的 `latest` 扫描器。

### 4.8 持久化与数据库

- 持久化选型、迁移和回滚必须遵守
  [`storage-strategy.md`](storage-strategy.md)，不得默认把所有业务数据库化。
- Docker、Nginx、systemd、系统文件和 `kejilion.sh` 产物是真实状态；JSON 或数据库只保存
  面板自身数据和可丢弃缓存。
- SQLite 仅用于 Panel 本机数据，使用本地文件系统、最小权限、固定 Schema、参数化 SQL、
  版本化迁移和一致性备份；禁止任意 SQL、动态扩展和网络文件系统数据库。
- 数据库、WAL、SHM、备份、导出和驱动供应链均视为敏感攻击面；Token、Cookie 和密码不得
  以明文进入数据库、错误信息或审计。
- 每种存储都必须有容量、保留期和损坏恢复上限，禁止无界 JSON/JSONL、无界查询和长期双写。

## 5. 性能预算

预算基于 [`runtime-performance-baseline.md`](runtime-performance-baseline.md) 和
[`security-performance-hardening-2026-07-28.md`](security-performance-hardening-2026-07-28.md)。
同一验收机、相同数据量和相同并发下，以下门槛默认生效：

| 指标 | 发布预算 |
| --- | ---: |
| Panel 预热后空闲 RSS | `≤ 32 MiB` |
| Agent 空闲 RSS | `≤ 32 MiB` |
| 256 MiB cgroup 中的登录突发峰值 | `< 192 MiB`，且 `memory.events.max/oom/oom_kill` 不增加 |
| Agent 常规只读采集峰值 RSS | `< 128 MiB` |
| Panel 冷启动到健康，P95 | `≤ 2 s` |
| `/health`、Session API，10 并发同机 P95 | `≤ 10 ms`，错误率 0 |
| `/system/summary` 同机 P95 | `≤ 250 ms` |
| 主入口 JS gzip | `≤ 70 KiB` |
| 单个懒加载路由 JS gzip | `≤ 120 KiB` |
| 交互终端本机输入到 PTY，P95 | `≤ 250 ms` |
| 同场景 P95、CPU 或峰值 RSS 回退 | 不得超过上一个稳定基线 `20%` |

确有业务必要超过预算时，不能静默放宽阈值。变更必须包含：

1. 数据量和测试环境；
2. 超预算原因；
3. 已比较的低成本方案；
4. 新预算和对低配主机的影响；
5. 回滚点。

### 5.1 实现规则

- 禁止 N+1 Docker inspect、逐项远程请求和无界 fan-out；并发必须有小型固定上限。
- 相同的并发只读请求可 singleflight 合并，但完成后不保留陈旧状态；写后必须读取真实状态。
- 缓存必须有容量、TTL、失效和最大对象大小；缓存不得成为第二套业务事实。
- 大文本静态资源构建期预压缩；指纹资源长期 immutable，`index.html` 保持 `no-cache`。
- 大 JSON 可快速 gzip，但下载、Range、HEAD 和不可压缩内容保持协议正确。
- 前端按路由和大型终端组件懒加载；不得把完整应用目录重复嵌入多个首屏接口。
- 采样、排序、摘要和格式化尽量在一次采集内完成；不得为每个并发页面重复读取宿主机。

### 5.2 存储预算

- 单个可变 JSON 文件达到 `8 MiB`、有效记录达到 `5,000`、整体写入 P95 达到 `50 ms`，
  或持续写入超过每秒 1 次时，必须评审分片、JSONL 或 SQLite，不能直接放宽文件上限。
- SQLite 方案必须与当前实现比较二进制体积、空闲/峰值 RSS、读写 P95、磁盘放大和恢复时间；
  任一指标回退超过 `20%` 时须说明产品收益和低配主机影响。
- 数据库连接、事务时长、WAL 大小、分页、查询返回量和历史保留期必须有固定上限。
- 大日志、终端输出、归档和二进制文件保存在有界文件中，不作为数据库 BLOB 持续增长。

## 6. 资源占用规范

### 6.1 默认上限

以下现有边界不得无证据放宽：

| 资源 | 当前边界 |
| --- | --- |
| Panel 容器 | `256 MiB`、`1 CPU`、`128 PIDs`、`16 MiB tmpfs` |
| Panel 请求体 | 默认 `1 MiB` |
| Agent 请求体 | `64 KiB` |
| Panel 读取 Agent 响应 | 默认 `8 MiB` |
| 交互终端单次输入/输出块 | `16 KiB / 64 KiB` |
| 应用/建站/体检终端日志 | 按任务类型 `1–32 MiB` 有界 |
| Docker API JSON/日志 | `16 MiB / 1 MiB` |
| Docker 备份还原总量 | `50 GiB`，单条目另有限制 |
| Panel Docker 日志 | `10 MiB × 3` |

新增列表、终端、备份、下载、目录扫描或第三方响应时，必须同时定义：

- 单对象、单请求、单任务和总量上限；
- 达到上限时的截断或错误语义；
- 临时文件权限、清理时机和磁盘不足行为；
- 并发数、超时、取消和进程回收；
- 关闭页面、Agent/Panel 重启后的恢复方式。

### 6.2 生命周期

- 每个 goroutine、Timer、Ticker、连接、文件、PTY 和子进程必须有明确所有者及退出路径。
- 请求取消应停止纯读取；已进入安全后台阶段的写任务由独立任务生命周期接管。
- 连接关闭后 FD、goroutine 和内存必须回落；压测结束 30 秒后不得持续增长。
- 终端只传输增量块，保留 ANSI；清屏控制序列是输出内容，不能被误判为任务结束。
- 日志和进度采用追加/有界尾部读取，禁止每次轮询读取整个不断增长的文件。

## 7. 稳定性规范

### 7.1 状态与成功判定

- Docker、系统文件、Nginx、systemd 和 `kejilion.sh` 产物是真实状态；数据库缓存不是成功依据。
- 后台任务至少有：排队、运行、等待输入、成功、失败、需要人工处理。
- systemd 或子进程退出码为 0 只是证据之一；成功必须同时具备完成凭据和产物复核。
- 页面刷新、弹窗关闭、浏览器退出不终止后台任务；重新进入可恢复进度和终端偏移。
- Agent 重启后从原子状态文件恢复任务结论；损坏或不完整状态不得被解释为成功。

### 7.2 写入、重试与回滚

- 写操作使用 `expectedResourceVersion` 或等价机制防止旧页面覆盖新状态。
- 配置变更流程为：读取真实状态 → 备份 → 暂存修改 → 语法/健康检查 → 原子切换 → 复核。
- 自动重试只用于幂等读取和明确幂等的提交，次数有限并带退避；不得自动重复安装、删除和数据库写入。
- 错误必须保留真实底层原因并给出已完成步骤；不得返回“完成”掩盖部分失败。
- 自动回滚只处理可证明安全的步骤；数据库格式升级等不可安全逆转场景标记“需要人工处理”。
- 停止任务只能在脚本安全阶段执行，不直接强杀正在替换目录、写数据库或更新包管理器的进程。

### 7.3 兼容稳定性

- `kejilion.sh` 与 KPanel 任一端修改后，另一端刷新即可读取真实结果。
- `k fd`、IP+端口、IPv4/IPv6、Debian/Ubuntu/Rocky/AlmaLinux、amd64/arm64 按改动范围回归。
- 新协议必须向后兼容当前稳定 Agent/Panel 的明确版本窗口；不兼容时在安装前失败，不在任务中途失败。
- 网络断开、Registry 超时、磁盘不足、只读文件系统、端口冲突和第三方脚本退出都必须有确定状态。

### 7.4 Schema、迁移与恢复

- Schema 版本单调递增，迁移可重复检测；迁移完成前不得删除旧数据或宣告升级成功。
- JSON → SQLite 先备份，在单个事务中导入，再核对记录数、关键字段、约束、摘要和
  `integrity_check`，最后原子切换存储标记。
- 禁止无期限双写。灰度双读只用于短期比对，并必须标注截止版本和删除计划。
- SQLite 备份使用 Backup API 或 `VACUUM INTO` 等一致性方式；WAL 模式下禁止只复制主文件。
- 升级中断、进程重启、磁盘写满、只读文件系统、损坏数据库和回滚到旧版本都必须有测试结论。

## 8. 开发流程

### 8.1 编码前质量记录

涉及 API、Agent、Docker、系统、脚本、终端或大数据列表的改动，开发前至少回答：

```text
流量路径：
不可信输入：
权限与可写范围：
最坏输入/输出字节数：
最大并发、CPU 和内存：
超时、取消与重试：
真实状态来源与缓存失效：
失败、回滚和重启恢复：
性能预算影响：
网络入侵风险：
```

没有明确答案时先补设计，不先写实现。

### 8.2 实现要求

- 先复用已有类型、限额、任务框架和 `kejilion.sh` 协议，不平行实现另一套。
- 新边界先写失败测试：超长、越界、重复、并发、取消、断网和损坏输入，再写成功路径。
- 性能优化必须保持结果一致，并包含防陈旧缓存、资源上限和并发回归测试。
- 安全修复必须描述可利用入口、信任边界和复核方式；不得只写“加强校验”。
- 任何扫描告警忽略都要记录规则、代码位置和不可利用证据，不能只加 suppress 注释。

## 9. 测试与发布门槛

| 改动类型 | 必测内容 |
| --- | --- |
| HTTP/代理 | Host、Origin、CSRF、Cookie、可信/不可信转发头、方法、Content-Type、超限和慢请求 |
| Agent/系统写入 | Token、Unix Socket、动作枚举、命令参数、路径/链接、资源版本、回滚和重启恢复 |
| 交互终端 | ANSI/颜色、清屏、连续输入、延迟、断线重连、窗口关闭、日志上限和任务后台继续 |
| Docker/归档 | 受限 Socket、容器并发、路径穿越、链接逃逸、损坏归档、体积炸弹和磁盘不足 |
| 前端与 API 性能 | 生产构建、gzip、缓存、路由包大小、同请求合并、无陈旧状态 |
| Store/迁移 | Schema、事务、并发、限额、敏感字段、损坏输入、备份恢复、迁移中断、回滚和性能基线 |
| 依赖/镜像 | `govulncheck`、`npm audit`、Trivy 源码/镜像、Secret、Misconfiguration、SBOM/provenance |

发布前至少执行：

```bash
go test ./...
go test -race ./internal/panel ./internal/auth ./internal/dockerx
npm --prefix web test
npm --prefix web run build
make security-audit
make verify-release
```

另外必须：

1. 用固定摘要的扫描器扫描源码和最终 scratch 镜像；
2. 在生产同限制下启动镜像并验证非 root、只读根、capability、PID/CPU/内存；
3. 复跑受影响性能基准并与最近稳定报告比较；
4. 实测 `k fd` 可信 HTTPS 反代和伪造代理头拒绝；
5. 对 L2/L3 系统动作执行失败注入和回滚验证；
6. 记录实际 Docker、containerd、runc、内核及 Agent/Panel 协议版本。

## 10. 漏洞响应

漏洞优先级由“是否可从网络到达、是否已被利用、权限影响和运行位置”共同决定，不能只看 CVSS：

| 级别 | 条件 | 要求 |
| --- | --- | --- |
| 紧急 | CISA KEV；或无需登录的 RCE、鉴权绕过、容器逃逸、宿主机/凭据读取 | 阻止发布；确认影响后 24 小时内修复或隔离 |
| 高 | 网络可达的 High/Critical；登录后可到达 Agent/宿主机的提权链 | 阻止发布；确认影响后 72 小时内修复或有效缓解 |
| 中 | 依赖存在但调用路径不可达、仅构建期存在、需要本机先决条件 | 记录调用证据和暴露窗口，14 天内升级或重新评估 |
| 低 | 不可达且影响低，无公开利用 | 纳入常规依赖升级，不得永久忽略 |

当上游尚无修复版本时，优先移除暴露入口、禁用受影响协议、固定安全旧版或隔离组件；这类临时缓解
只能限制攻击面，不能借机限制合法管理员业务动作。

持续关注来源：

- CISA Known Exploited Vulnerabilities；
- Go Vulnerability Database 与 `govulncheck`；
- npm advisory、Node.js security releases；
- Docker、containerd、runc、Linux 发行版安全公告；
- GitHub Actions、构建镜像和扫描工具自身供应链公告。

## 11. 代码评审核对

- [ ] 未把合法管理员能力削减包装成安全改进。
- [ ] 新网络入口、代理头、外联地址和权限变化已明确。
- [ ] 所有输入、响应、并发、缓存、日志和临时文件都有上限。
- [ ] 没有任意 Shell、任意 URL、任意 Docker endpoint 或任意文件路径。
- [ ] 页面关闭、超时、断网、重启和部分失败后状态可恢复。
- [ ] 性能预算通过，或已记录超预算证据和新预算。
- [ ] 安全扫描结果按可达性人工复核，没有无证据 suppress。
- [ ] `k fd`、`kejilion.sh`、Agent/Panel 和真实状态读取兼容性未退化。
- [ ] 存储选型符合产品查询与增长需求，权威来源、限额、迁移、备份和回滚均已明确。
- [ ] 发布产物有 commit、digest、SHA-256、SBOM、provenance 和回滚点。

## 12. 规范依据与维护

本规范参考：

- NIST SSDF 1.1：<https://csrc.nist.gov/pubs/sp/800/218/final>
- OWASP ASVS 5.0：<https://owasp.org/www-project-application-security-verification-standard/>
- OWASP Web Security Testing Guide：<https://owasp.org/www-project-web-security-testing-guide/latest/>
- CISA KEV：<https://www.cisa.gov/known-exploited-vulnerabilities-catalog>
- Go Vulnerability Management：<https://go.dev/doc/security/vuln/>
- Docker Engine Security：<https://docs.docker.com/engine/security/>

出现重大安全事件、架构边界改变、资源限制调整或稳定基线更新时立即复核本规范；无事件时至少每年
复核一次。预算更新必须同时更新基线报告和验收证据，不能只改本文数字。
