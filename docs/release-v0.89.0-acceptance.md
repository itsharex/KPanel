# KPanel v0.89.0 发布验收记录

日期：2026-08-20

发布级别：L3（兼容功能版本）

候选提交 / 标签：`c7e59e9105e254bc4ea9909d778b997af5449bb5` / `v0.89.0`

上一稳定版本 / 回滚点：`v0.88.1` / `sha256:0710dce2c8272f657bc171c4abf576abe4ec968d267f209c93ccce5d6c645660`

## 发布画像

- 业务域：文件下载兼容与 AI 助手 Provider 历史重建。
- 变更面：新增单文件、文件夹及多选 ZIP 的可靠下载后备链路和短期票据；修复多工具批次与 Provider 原生 reasoning 历史重建；没有数据库 schema、权限模型、Agent 协议或第三方依赖变化。
- 受影响用户旅程：Windows 原生拖拽下载失败后的显式下载、文件夹/多选打包下载，以及 DeepSeek/OpenAI/Anthropic/Gemini 多轮工具调用。
- 未变化契约：原生 `DownloadURL` 仍是渐进增强；浏览器企业 DLP/下载策略不被绕过；reasoning 不作为普通正文泄露；一键跑分页面已在上一稳定版本等价上线，本版不重复引入旧分支。
- 风险等级及理由：中；文件下载增加短期授权链路，AI 修复跨多个 Provider 适配器，但安全边界、确定性合同和真实文件链路均有自动化或隔离真机证据。

## 发布范围与未纳入内容

- `8072b29`：可靠 Windows 下载后备、5 分钟单文件/归档票据、归档预检与失败提示。
- `a03d46d`：完整保留同一 assistant 响应中的多工具调用与结果批次。
- `ec5cc61`：保留 Provider 原生 reasoning turn，并同步 DeepSeek reasoning/model 语义。
- `c7e59e9`：版本 `0.89.0`、CHANGELOG 与回滚说明。
- 未纳入：陈旧工作树、Windows 企业策略绕过方案、旧一键跑分分支以及其它未审查草稿；`kejilion/apps` 配置与候选归一化一致，没有空提交；未连接或操作 108。

## 多维质量结论

| 维度 | 状态 | 证据 | 未验证风险或不适用依据 |
| --- | --- | --- | --- |
| 业务正确性与双端互通 | 已验证 | Go/Web 全量、AI Provider 合同竞态、154 精确候选 Panel→Agent 文件票据与 ZIP 真链路 | Windows Explorer/Chrome 企业策略环境由用户上线后亲测，不能宣称原生拖拽已完全解决 |
| 网络入侵与供应链安全 | 已验证 | govulncheck 可达漏洞 0、npm audit 0、Trivy 源码/镜像 0、短期 cookieless ticket、路径穿越/无鉴权/无 CSRF 拒绝 | 不尝试绕过组织 DLP 或 Chrome 企业策略 |
| 稳定性、失败恢复与兼容 | 已验证 | Go 全量/核心竞态/vet、Web 106 文件/807 测试、公开镜像 E2E、生产健康与备份恢复核验 | 真实第三方 AI Provider 将来改变私有合同仍需新增夹具 |
| 性能与资源预算 | 已验证 | 生产三次采样 CPU 0.03%～0.05%、内存 73.09 MiB/256 MiB，0 restart/OOM | 无常驻进程或资源模型变化，不执行长时间 soak |
| 用户体验与可访问性 | 部分验证 | Web 全量、typecheck、i18n 2423/20、production build；可靠下载失败态与回退测试通过 | 本轮浏览器控制工具不可用，未形成新的真实视觉证据；Windows 拖拽由用户亲测 |
| 数据、配置与迁移 | 已验证 | SQLite `integrity_check=ok`、Compose parse、停写备份独立解包/比对/旧镜像恢复核验 | 无 schema 或配置迁移 |

## 自动门禁

- L3：arena-154 Debian 13、Docker 29.6.2、固定 Runner `sha256:b593c0ffe32e80a6a6ae8fcac38ed916587dbf698a6b6a55fa33887298737148`；最终标记 `L3 release verification completed`，日志 SHA-256=`c422e88dbb7a87453787a95e84074ddc95682f97b4e817c87c79dc8712b31b8a`。
- L3 覆盖：Go 全量/核心竞态/vet、Web 106 文件/807 测试、i18n 2423/20、typecheck/build、双架构、安装安全、应用生命周期、govulncheck、npm audit、Trivy 源码/镜像。
- AI Provider 合同竞态：`go test -count=1 -race ./internal/ai` 通过；日志 SHA-256=`28fb4fc269ef12e70c497efc54b92c55cd7cfc234b765da90d08cbb212ef0c7d`。
- 候选 CI：run `32361114836` success；Dependency freshness run `32361114839` success。
- 主线 CI：run `32361517208` success；Dependency freshness run `32361517078` success。
- Release workflow：run `32361948656` success；Tag Dependency freshness run `32361948615` success。

## 隔离真机与浏览器验收

- 环境：`arena-154`，仅用于 candidate-validation、performance-validation、production-safety-check 和 production-deploy；未连接 108。
- 精确候选：源码 `c7e59e9105e254bc4ea9909d778b997af5449bb5`；验证镜像 `sha256:34cc3bc765e031093e44873b623cd2ac56c1703497d89cc789b33d410aad04b5`；公开 OCI `sha256:d04966d68fc7d1a7be5b68e12b32135455b54e52926ec6cec537da79e9690d19`。
- 文件真链路：无鉴权 401、缺失 CSRF 403；单文件票据无 Cookie 下载和 HEAD 通过；两份同名文件归档名称稳定且内容、0640 模式正确；非法 ticket 404；临时容器、网络和目录已清理。结果 SHA-256=`593a6625a947e957a26ad48b33abf056cf6b453a829f06f290a59138cecd1e6c`。
- 浏览器证据：内置浏览器控制工具无法访问本地/临时公开候选入口，但 curl 与服务健康正常；该工具故障不替代产品旅程，也不作为“通过”证据。Windows Explorer/Chrome/Edge 原生拖拽由用户在正式版本亲测；可靠后备下载链路已由自动化和 154 真链路覆盖。
- 未执行：没有调用生产 AI Provider 或读取 AI 密钥；没有尝试规避 Chrome 企业下载策略。

## 发布产物与公开仓库复核

- GitHub Release：[v0.89.0](https://github.com/kejilion/KPanel/releases/tag/v0.89.0)，非 draft、非 prerelease；Tag 解引用到 `c7e59e9105e254bc4ea9909d778b997af5449bb5`。
- Docker `0.89.0` 与 `latest` OCI index：均为 `sha256:d04966d68fc7d1a7be5b68e12b32135455b54e52926ec6cec537da79e9690d19`。
- `linux/amd64`、`linux/arm64` digest：amd64 `sha256:6d6998a6965df0c51c2907179ca21adfefad513e1ce5f820e905907a8ce7f034`；arm64 `sha256:3469113d358b6fa4ce3954f3e88fe3929b21661b0b16619c6520576bb2acb4bc`。
- 公开镜像标签：version=`0.89.0`、revision=`c7e59e9...`、script revision=`6fa7bcc...`、script SHA-256=`534a7a18...`；公开 `image_e2e=pass`。
- Release 附件齐全；`SHA256SUMS` asset digest=`sha256:2fa2c31b1242cc5aae0cd1f3eabab69fcc42829691801e05d6102079ec234bd8`。
- `kejilion/apps` main=`6d86eee24a477320f4d8ffb32d9e85b785cf3c2c`，`kpanel.conf` 与候选归一化内容一致，无需提交。

## 生产部署安全核对

- 生产目标：仅 `arena-154`；`prod-108` 未连接、未测试、未部署、未核对。
- 部署前：v0.88.1，Panel healthy、Agent active、restart=0、OOM=false。
- 停写一致性备份：`/root/kpanel-backups/v0.89.0-preupgrade-arena154-20260820T111328Z`；状态包 SHA-256=`11c50601345e09cfbb10f9b1b2c8c28af607323ddaa52fea9a9917e8a3d60f74`，旧镜像 tar SHA-256=`646aca1b09d89654de920c37d3c0be99722defe5290fa5569b7dee7ca8b74c7e`。
- 备份验证：独立解包、关键文件 `cmp`、Compose parse、SQLite integrity、旧镜像 `docker load` 和旧 v0.88.1 原位恢复均通过。
- 部署入口：`KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update bash /home/docker/kpanel/bin/kejilion.sh app kpanel`，进度到 `KPANEL_PROGRESS 100`。
- 部署后：Panel v0.89.0 healthy、Agent active，正式镜像 digest/revision 精确匹配，restart=0、OOM=false；公网健康 v0.89.0，SQLite integrity=ok，近 15 分钟 panic/fatal/OOM/协议错误标记 0。
- Agent 二进制与镜像 `/release/kejilion-agent` 逐字节一致；运行时脚本仅按既有契约把 `permission_granted` 设为 true，归一化后与镜像固定脚本一致。

## 回滚

- 源码/tag：`v0.88.1` / `46981a51f1836d77b94be0826971a1cbde749b0c`。
- 镜像 digest：`sha256:0710dce2c8272f657bc171c4abf576abe4ec968d267f209c93ccce5d6c645660`。
- 备份：`/root/kpanel-backups/v0.89.0-preupgrade-arena154-20260820T111328Z`，包含 Compose、`.env`、数据、Agent/脚本、apps 配置、systemd unit 和旧 OCI。
- 回滚步骤：停写；加载旧 OCI；成套恢复 `kpanel` 目录、apps 配置和 systemd unit；`systemctl daemon-reload`；启动 Panel/Agent；核对 v0.88.1、健康、restart/OOM、SQLite、Compose 和公网入口。
- 未执行生产回滚；部署前已用该备份实际恢复旧服务并确认 healthy/active。

## 交付节奏数据

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-08-20T18:12:27+08:00
- 候选冻结时间：2026-08-20T18:20:45+08:00
- 生产完成时间：2026-08-20T19:15:17+08:00
- 提交到生产用时：1.05 小时
- 是否回滚、紧急热修复或重复发布：否
- 若发生失败，发现时间、恢复时间和逃逸门禁：不适用
<!-- kpanel-release-metrics:end -->

<!-- kpanel-release-process-metrics:start -->
- 已记录发布流程异常或无效证据拦截次数：13
- 其中生产写操作开始后异常次数：1
<!-- kpanel-release-process-metrics:end -->

流程异常均与产品载荷无关，并在 fail-closed 边界内处理：包括候选身份断言、Windows/PowerShell 转义、Runner/Trivy 上下文、一次文件夹具启动等待、浏览器控制工具不可用、本地 Tag 解引用命令转义，以及生产后一次不受支持的 Agent `--version` 只读探测。没有数据损坏、门禁逃逸、生产回滚或 108 操作。

## 遗留风险与后续准入

- Windows Explorer/Chrome/Edge 的原生拖拽会受浏览器版本、企业 DLP 与组织下载策略影响；用户将在正式版本亲测。若仍出现组织策略阻断，应使用本版显式可靠下载后备或调整浏览器策略，不能在 KPanel 中绕过安全策略。
- 没有使用真实 DeepSeek/OpenAI/Anthropic/Gemini 凭据执行在线多工具 E2E；确定性 Provider 合同、Go 全量/竞态、L3 和生产健康均通过。
- 后续门禁继续保留：无鉴权/过期票据拒绝、归档路径与大小预算、多工具合批、reasoning 不外泄、Provider 原生序列回归。
