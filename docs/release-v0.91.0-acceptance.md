# KPanel v0.91.0 发布验收记录

日期：2026-08-22

发布级别：L3

候选提交 / 标签：`9321afc3ce51731ffb7b442c127d6441ce83d885` / `v0.91.0`

上一稳定版本 / 回滚点：`v0.90.5` / `sha256:29683a371983e30e5d2ea695f60debef4aa5602f21abf41fdf48cbb63bcbc286`

## 发布画像

- 业务域：文件管理与匿名单文件分享。
- 变更面：前端展示、Panel/Agent 协议、匿名只读网络入口、持久化状态与部署镜像。
- 受影响用户旅程：创建 7 天、30 天或永久分享；复制公开页或原始文件链接；查看、轮换和撤销分享；图片跨站嵌入。
- 未变化契约：端口、Compose、Agent 权限、受管 `kejilion.sh` 和应用市场安装/更新契约均未变化。
- 风险等级及理由：中高；新增匿名文件入口和流式读取，但已采用内容绑定、类型收口、完整预算、限流和 fail-closed 策略。

## 发布范围与未纳入内容

- 新增普通单文件分享、公开页 `/share/file/{token}`、直链 `/f/{token}`、分享管理、轮换和撤销；不支持目录或批量分享。
- `shareVersion v2` 绑定资源版本、Linux 文件身份和完整 SHA-256；同长度原地改写、恢复 mtime、同内容 inode 替换后旧链接均失效。
- 仅 JPEG、PNG、GIF、WebP、AVIF 可 inline；HTML、SVG、JavaScript、CSS、PDF、文本及其他内容统一附件下载，并由 Panel 最终覆盖 sandbox CSP、`nosniff` 和无 CORS 读取边界。
- 单文件上限 512 MiB，强校验并发 2；每路径及全局均为 1 GiB/分钟，GET、HEAD、Range、304、416 共用强校验和预算。
- 精确功能提交为 `b5e5483`、`2eccf73`、`5ab552c`，版本准备为 `9321afc`；未纳入旧失败候选 `dc20a34`、旧 bundle、任何 CSS 猜测或 108 环境操作。

## 多维质量结论

| 维度 | 状态 | 证据 | 未验证风险或不适用依据 |
| --- | --- | --- | --- |
| 业务正确性与双端互通 | 已验证 | Go/Web 全量、真实 Panel+Agent、公开元数据、GET/HEAD/Range/304/416、轮换和撤销 | 生产不创建测试分享，完整写旅程在同一 arena-154 隔离实例完成 |
| 网络入侵与供应链安全 | 已验证 | 内容 SHA-256 绑定、强 If-Match、主动内容附件化、sandbox CSP、无 CORS、govulncheck、npm audit、Trivy source/image 0 | bearer token 的保密仍依赖用户安全保存和 HTTPS |
| 稳定性、失败恢复与兼容 | 已验证 | 同元数据改写/替换 fail-closed、慢流撤销释放、并发闸门恢复、v0.90.5 往返及旧版写状态语义 | 回滚到旧版后再次写状态会按文档丢弃 `fileShares` |
| 性能与资源预算 | 已验证 | 512 MiB 边界、强校验并发 2、1 GiB/分钟预算、生产三次资源采样 | 完整内容哈希是安全要求，超大文件被限制为 512 MiB |
| 用户体验与可访问性 | 已验证 | 桌面/390px、明暗主题、三语、键盘；同一 CDP target 的 390×844 指标与截图无横向溢出 | 旧 `share-mobile-r2.png` 来自不一致上下文，已作废并由同 target 证据替换 |
| 数据、配置与迁移 | 已验证 | 旧状态加载、回滚语义、停写备份独立恢复、20 个 JSON、SQLite integrity、Compose 和配置哈希 | 目录及批量分享明确不在本版范围 |

## 自动门禁

- Git bundle：`kpanel-v0.91.0-9321afc-r2.bundle`，SHA-256=`75ab191c0f8f9482fbcc1008bc3f9ce3122a22d328379ffa9028e488fa2a4b96`。
- arena-154 固定 Runner `kpanel-release-gate:go1.26.6-node24` / `sha256:b593c0ffe32e80a6a6ae8fcac38ed916587dbf698a6b6a55fa33887298737148` 完整 L3 exit 0；Go 全包、race、vet、双架构、Web 110 文件/870 项、i18n 2443/21、typecheck、生产构建、govulncheck、npm audit、Trivy source/image、受管脚本、安装安全和应用生命周期均通过。
- L3 日志 `/root/kpanel-release-evidence/v0.91.0-9321afc-r2/l3-verify-release.log`，SHA-256=`91a7833bb5504fab965ff5988a4121689159fca8811e6ef65558dad68652dc00`。
- 候选 CI `32575093364`、候选依赖新鲜度 `32575093354`、main CI `32575326602`、main 依赖新鲜度 `32575326591`、Tag 依赖新鲜度 `32575580903` 均 success，head SHA 精确匹配 `9321afc...`。
- Release workflow `32575580908` success；源码、运行时、SBOM/provenance、双架构推送、`latest` 提升和 Release 发布步骤全部通过。

## 隔离真机与浏览器验收

- 环境：`arena-154`，Linux amd64、Docker 29.6.2；策略允许 candidate-validation、browser-validation、production-safety-check 和 production-deploy。
- 真链路覆盖同元数据 rewrite/replacement、符号链接/FIFO、主动 HTML/SVG、PNG 外链、轮换、慢流撤销、512 MiB 边界和并发预算；`share_e2e=pass`，证据清单 SHA-256=`670fe385d900ca4abefb916cd94ea335b76b29c6cbed2abd6aff31a583aba96e`。
- 慢流撤销后输出字节立即停止、旧 token 404；两个新流 200、第三个 429，证明 stream permit 立即恢复。该诊断证据清单 SHA-256=`e4fa63d31551181ef1287f6b7ab1b9366cbe8e8ba6a2fecf413e41fc3bebcb3b`。
- v0.91.0→v0.90.5→v0.91.0 往返、旧 route 不可用、候选状态保留及旧版写状态丢弃语义均通过；回滚证据清单 SHA-256=`843b223f28d6671dfbfe3ea512e235db0fd9654a07d77b5be07b7d2db6b99f99`。
- 正式 Chrome 151.0.7922.172 使用独立临时 Profile；同一 CDP target 记录 DPR=1、visual viewport=390×844、document/body clientWidth=scrollWidth=390、card right=378，并由 `Page.captureScreenshot` 生成 `share-mobile-cdp.png`，SHA-256=`abb2c5ad80ed99ba02a5b5e3a2a297984bacc45a8f55aecaec7c2ae56444778e`。
- 未在生产创建分享、修改业务文件、跑 512 MiB 预算或慢流撤销；这些写场景均在隔离实例完成。

## 发布产物与公开仓库复核

- GitHub Release：[v0.91.0](https://github.com/kejilion/KPanel/releases/tag/v0.91.0)，非 draft、非 prerelease，附件包含 Agent/Node 双架构、部署包、`SHA256SUMS`、许可证和第三方声明。
- Docker `0.91.0` 与 `latest` OCI index 均为 `sha256:ada4847d90ed8430340abbafc4d15d4227025146a8c420b54d00b87e4f24b097`。
- `linux/amd64`=`sha256:5c25e2befc2899bf664bf9f5733f9a8da59f731727aaa40ae92684905ea2828d`；`linux/arm64`=`sha256:61deef09c5510950d6affb1060d8823cabc54d7f07e24fd380115b2ec2d9f25f`。
- arena-154 独立公开回拉验证 version=`0.91.0`、revision=`9321afc...`、双标签同 index、非 root、受限容器和健康检查通过；摘要证据 SHA-256=`11695887a8ffb5aff9a6900b41bd2c3a4b4ac206d4f24d34d403eaeaee218271`。
- `packaging/kejilion-app/kpanel.conf` 相对 v0.90.5 零差异；生产 `kejilion/apps@6d86eee24a477320f4d8ffb32d9e85b785cf3c2c` 工作树干净，无需制造应用市场空提交。

## 生产部署安全核对

- 唯一生产目标为 `arena-154`；用户已授权正式发布、备份、标准升级和验收。
- `prod-108` 禁用全部 KPanel 操作；本次未连接、未读取、未测试、未备份、未部署、未核对。
- 部署前 v0.90.5 healthy/active、restart=0、OOM=false；停写一致性备份为 `/root/kpanel-backups/v0.91.0-preupgrade-arena154-20260822T133158Z`，`SHA256SUMS` 摘要=`e05914ac32402d1686b60ad207d45ee8bc5aa25d0ff056e32d563d87dd021e02`。
- 备份已独立解包、比较 Compose/`.env`/Agent unit/token、校验 20 个 JSON、2 个 SQLite、旧 OCI 归档，并重新启动 v0.90.5 healthy 后才部署。
- 使用 `KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update bash /home/docker/kpanel/bin/kejilion.sh app kpanel` 标准入口升级；更新日志 SHA-256=`fd79da62fafea7a29d35e8b818fc431fd9e227ef060557ccc81876f254a33bea`。
- 部署后 Panel v0.91.0 healthy、Agent active、restart=0、OOM=false、systemd NRestarts=0/NeedDaemonReload=no；公网 HTTPS health 200，未知分享 token 404，Compose/JSON/SQLite/镜像 revision/index 和错误日志均通过。
- `.env`、Compose、Agent 配置和 service 文件的组合哈希在升级前后完全一致；生产证据清单 SHA-256=`ff9124524b9ccdbc7a815131727b55b3393a613c2d8dc1f445255ab8f33409bf`。
- 三次采样 CPU 0.02%～0.03%，内存 74.46～74.47 MiB/256 MiB、7 PIDs，无单调增长。

## 回滚

- 源码/tag：`v0.90.5` / `957d30be1a3cfb3c1f66cc8dcbf32c7b4ab6adf1`。
- 旧 OCI：`sha256:29683a371983e30e5d2ea695f60debef4aa5602f21abf41fdf48cbb63bcbc286`。
- 数据/配置备份：`/root/kpanel-backups/v0.91.0-preupgrade-arena154-20260822T133158Z`。
- 回滚必须停写并成套恢复旧镜像、完整 `/home/docker/kpanel`、Compose、`.env`、数据、Agent unit 和二进制；禁止只换镜像。恢复后核对 Panel/Agent、数据完整性、公开路由和标准更新入口。
- 隔离实例已实际完成候选→旧版→候选往返；生产无需回滚，当前保持 v0.91.0 healthy。
- GitHub Latest、Docker `latest` 与标准更新入口均指向已验收 v0.91.0；公共默认更新通道无需恢复。

## 交付节奏数据

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-08-22T18:32:06+08:00
- 候选冻结时间：2026-08-22T20:20:29+08:00
- 生产完成时间：2026-08-22T21:34:33+08:00
- 提交到生产用时：3.04 小时
- 是否回滚、紧急热修复或重复发布：否
- 若发生失败，发现时间、恢复时间和逃逸门禁：不适用
<!-- kpanel-release-metrics:end -->

<!-- kpanel-release-process-metrics:start -->
- 已记录发布流程异常或无效证据拦截次数：10
- 其中生产写操作开始后异常次数：1
<!-- kpanel-release-process-metrics:end -->

### 流程异常明细

<!-- kpanel-release-process-incidents:start -->
[
  {
    "fingerprint": "candidate-local/docker-preflight/stale-exit-state",
    "position": "before-production-write",
    "count": 1,
    "impact": "本机 Docker 不可用且 PowerShell 旧 LASTEXITCODE 使一次 OCI 占用预检结论无效。",
    "recoveryEvidence": "停止使用本机 Docker 证据，arena-154 固定 Runner 与公开 OCI E2E 均从精确源码通过。",
    "permanentAction": "Docker 门禁固定在登记的 Linux Runner，并在每条命令后直接检查本次退出码。",
    "historicalReleases": []
  },
  {
    "fingerprint": "remote-execution/powershell-ssh/command-substitution",
    "position": "before-production-write",
    "count": 3,
    "impact": "三次复杂内联 SSH 的状态或命令替换被本机 PowerShell 提前解释，组合输出不能作为权威证据。",
    "recoveryEvidence": "所有受影响步骤均改为 apply_patch 生成、Git Bash bash -n、SHA-256 记录后上传的独立脚本并通过。",
    "permanentAction": "复杂远程验收和生产步骤禁止使用内联 PowerShell SSH 字符串，仅执行固定摘要脚本。",
    "historicalReleases": ["v0.90.5"]
  },
  {
    "fingerprint": "l3-bundle/git-bundle/missing-baseline-tag",
    "position": "before-production-write",
    "count": 1,
    "impact": "第一份重建 bundle 未包含业务新鲜度所需的 v0.90.1 基线 Tag，L3 在业务上下文门禁提前停止。",
    "recoveryEvidence": "以同一源码重建自包含 bundle，校验 SHA 后从零运行完整 L3 成功。",
    "permanentAction": "bundle 预检显式枚举 release workflow 所需稳定基线 Tag，并在上传前离线 clone 运行新鲜度检查。",
    "historicalReleases": []
  },
  {
    "fingerprint": "tag-verification/git-fetch/stale-local-tag",
    "position": "before-production-write",
    "count": 1,
    "impact": "本地历史 v0.86.2 Tag 与远端不一致，禁止无边界 fetch tags；目标 v0.91.0 未受影响。",
    "recoveryEvidence": "仅使用精确远端 refs 核对 main 和新 Tag，v0.91.0 解引用精确指向 9321afc。",
    "permanentAction": "发布过程只 fetch 或查询目标 Tag，不用无边界 fetch tags 改写本地历史。",
    "historicalReleases": ["v0.90.5"]
  },
  {
    "fingerprint": "share-cancel/curl-client/buffered-liveness-false-positive",
    "position": "before-production-write",
    "count": 1,
    "impact": "限速 curl 在服务端撤销并停止输出后仍因本地缓冲存活，最初被误判为 stream permit 未释放。",
    "recoveryEvidence": "旧 token 404 且立即可建立两个新流、第三个 429，证明服务端 permit 已完整释放。",
    "permanentAction": "撤销门禁改为输出字节停止、旧 token 状态和新 permit 容量三项联合判定，不以客户端进程存活单独判定。",
    "historicalReleases": []
  },
  {
    "fingerprint": "browser-evidence/cdp-screenshot/context-mismatch",
    "position": "before-production-write",
    "count": 1,
    "impact": "旧移动截图来自不一致 viewport 或错误裁切，与同页布局指标互斥，不能作为溢出证据。",
    "recoveryEvidence": "同一 CDP target 同步采集 DPR、visualViewport、client/scrollWidth、关键 rect 与 Page.captureScreenshot，确认无溢出。",
    "permanentAction": "移动视觉门禁必须由同一浏览器 target 同步输出布局指标与截图并绑定摘要。",
    "historicalReleases": []
  },
  {
    "fingerprint": "script-preflight/bash-runtime/wsl-path-mismatch",
    "position": "before-production-write",
    "count": 1,
    "impact": "一次本地 bash -n 实际调用 WSL Bash，不能访问 C: 风格路径，该次本地语法证据无效。",
    "recoveryEvidence": "改用精确 Git for Windows bash 路径完成 bash -n，远端 Bash 执行也成功。",
    "permanentAction": "Windows 发布脚本预检固定调用 Git Bash 绝对路径并立即检查退出码。",
    "historicalReleases": []
  },
  {
    "fingerprint": "candidate-cleanup/powershell-git/boolean-exit-misread",
    "position": "after-production-write",
    "count": 1,
    "impact": "候选清理时把 git merge-base 成功且无标准输出误解释为布尔假，安全地停在远端删除前。",
    "recoveryEvidence": "改为显式检查 LASTEXITCODE 后确认候选已被 Tag 和最终 main 包含，并正常删除远端候选分支；生产未受影响。",
    "permanentAction": "PowerShell 调用只用进程退出码判定 Git 布尔命令，不以标准输出是否为空判定成功。",
    "historicalReleases": []
  }
]
<!-- kpanel-release-process-incidents:end -->

## 遗留风险与后续准入

- 匿名分享 token 泄露后在过期或撤销前可被持有者访问；产品不记录明文 token，用户仍需按敏感链接管理。
- 图片允许跨站嵌入但禁止 CORS 脚本读取；上游站点缓存和流量消耗仍需由管理员结合 7/30 天有效期和撤销功能控制。
- Agent 强校验对大文件执行完整哈希，已由 512 MiB、并发 2 和双层 1 GiB/分钟预算封顶；P3 的保守预算预留最多影响一分钟可用性，不削弱 fail-closed。
- 本版无 P0/P1/P2 遗留项，无需回滚；生产未执行危险或高负载分享写验收，隔离证据与生产只读核对分层记录。
