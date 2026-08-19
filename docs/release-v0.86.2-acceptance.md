# KPanel v0.86.2 发布验收记录

日期：2026-08-19

发布级别：L3（补丁）

候选提交 / 标签：`43e6cbf4ffaff1e4a70b0f98b00df4a54c89fee5` / `v0.86.2`

上一稳定版本 / 回滚点：`v0.86.1` / `sha256:232045ade043d571d3f51d2c07fcf82c3d4ebab324e9ec56c96ff357c0a1bf11`

## 发布画像

- 业务域：桌面文件/目录原生拖拽下载兼容性。
- 变更面：仅前端 `DataTransfer` 协议与拖拽效果提示；无后端、API、权限或数据模型变化。
- 受影响用户旅程：桌面文件、文件夹/多选 ZIP 拖到浏览器或 Windows 文件管理器；KPanel 内部桌面拖拽保持原状态机。
- 未变化契约：Agent、端口、Compose、apps、`kejilion.sh`、登录会话和同源 URL 安全校验均未改变。
- 风险等级及理由：中等；浏览器/组织 DLP 可能在 KPanel 之外拦截原生拖拽，产品不绕过该安全边界。

## 发布范围与未纳入内容

- 源修复提交：`7c05b2e0e5431e7e65dde3718b54d291558f3a00`。
- 最新主线重放提交：`7daf3caecd24ea5992e6f10aba8c04a0b2e6c73c`。
- 版本与发布元数据提交：`43e6cbf4ffaff1e4a70b0f98b00df4a54c89fee5`。
- 变更内容：Chromium promised-file 的 `DownloadURL` MIME 统一为 `application/octet-stream`；文件管理器原生拖拽使用 `effectAllowed=copyMove`、`dropEffect=copy`；保留 `text/uri-list`、同源 HTTP(S)/无凭据校验、单文件/ZIP 与内部拖拽行为。
- 明确未纳入：旧分支归档差异、未提交 MonitoringView 用户改动、后端/API、短期下载票据、108 环境及任何绕过浏览器企业策略的实现。

## 多维质量结论

| 维度 | 状态 | 证据 | 未验证风险或不适用依据 |
| --- | --- | --- | --- |
| 业务正确性与双端互通 | 已实现未实机验证 | 定向拖拽协议回归；前端 100 files / 772 tests；单文件/ZIP、URI、跨域/凭据拒绝覆盖 | Windows Explorer/Chrome/Edge 真机落地由用户发布后现场验证 |
| 网络入侵与供应链安全 | 已验证 | 候选/主线/Release workflow 的 govuln、Trivy、npm audit、镜像契约及 SBOM/provenance 均通过 | 无 |
| 稳定性、失败恢复与兼容 | 已验证 | 候选与主线 CI、双架构构建、公开镜像 E2E、154 升级后健康采样通过 | 未做长时 soak；本补丁无常驻后端改动 |
| 性能与资源预算 | 不适用 | 仅修改浏览器拖拽元数据，不增加服务端常驻任务 | 依赖浏览器/组织策略的行为仍需现场确认 |
| 用户体验与可访问性 | 已实现未实机验证 | MIME、拖拽效果和现有内部拖拽回归通过 | Chrome 企业下载策略可能显示组织安全拦截 |
| 数据、配置与迁移 | 已验证 | 停写备份 SHA256 全通过；升级前后 Compose、`.env`、apps 哈希一致 | 未执行用户业务数据写入 |

## 自动门禁

- 候选 CI：run `32212426494` success。
- 候选 dependency freshness：run `32212426507` success。
- 主线 CI：run `32212693053` success；主线 dependency freshness `32212693055` success。
- Tag dependency freshness：run `32212898409` success。
- Release workflow：run `32212898420` success，源码、安全、依赖、构建、双架构 OCI、latest promotion、Release 和候选清理均成功。
- 本地候选：`npm ci`、前端 100 files / 772 tests、typecheck、production build、i18n 2295/20、npm audit、governance/version consistency、`git diff --check` 均通过。

## 依赖与技术栈变化

- 本版未新增运行时依赖、基础镜像、受管脚本或 Action；版本元数据更新为 `0.86.2`。
- Release workflow 生成的正式 OCI 必须以不可变 digest 使用；公开标签 `0.86.2` 与 `latest` 均已回拉核验。
- 已知浏览器企业策略、Chrome/Edge 下载控制及接收端行为不由 KPanel 绕过；若现场仍拦截，按组织策略或受管客户端方向处理。

## 隔离真机与浏览器验收

- 154 隔离生产目标使用正式公开 OCI，Panel/Agent 启动、健康与升级闭环通过；本轮未以自动化方式宣称 Windows Explorer 真机拖放已解决。
- 用户发布后负责现场验证：单文件、文件夹、多选 ZIP 分别从 KPanel 拖入 Windows/Chrome 及目标聊天网页；记录浏览器版本、响应状态和是否出现组织策略提示。
- 未执行场景：108 全部场景；Windows Explorer/Chrome/Edge 最终落地，原因是当前发布环境无法可靠控制用户桌面及组织策略。

## 发布产物与公开仓库复核

- GitHub Release：[v0.86.2](https://github.com/kejilion/KPanel/releases/tag/v0.86.2)。
- `origin/main` 与 `v0.86.2^{}`：`43e6cbf4ffaff1e4a70b0f98b00df4a54c89fee5`。
- 公开 OCI：`docker.io/kjlion/kejilion-panel:0.86.2` 与 `latest` 均为 `sha256:1ab9fdefbe7b7dbb46648ae33263ed13b855ad65a8683f820f99a6953c768b98`，含 linux/amd64 与 linux/arm64。
- OCI labels：`org.opencontainers.image.version=0.86.2`、`org.opencontainers.image.revision=43e6cbf4ffaff1e4a70b0f98b00df4a54c89fee5`；受管脚本 revision 沿用 `fdb0ac0e1f2b98d27339937e7f8eb0c9299c56a9`。
- `kejilion/apps` / `kejilion.sh` 契约：本版未修改，升级前后 apps 配置哈希一致。

## 154 生产部署安全核对

- 正式目标：仅 `arena-154`；`prod-108` 禁止全部 KPanel 操作，本轮未连接、未测试、未备份、未部署、未核对。
- 升级前 v0.86.1 healthy，Agent active，restart=0、OOM=false；Compose `.env` 与 apps 配置哈希已记录。
- 新停写一致性备份：`/root/kpanel-backups/pre-v0.86.2-20260819T034749Z`；包含旧镜像、Compose、`.env`、Agent 文件、数据和 apps 配置；`sha256sum -c SHA256SUMS`、归档读取与旧服务恢复校验通过。
- 标准更新入口：`KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update bash /home/docker/kpanel/bin/kejilion.sh app kpanel`。
- 升级后：Panel `0.86.2`，`/api/v1/health` 连续 3 次 `status=ok`；容器 healthy、restart=0、OOM=false；Agent systemd active 且日志为 `version=0.86.2`；近期日志无 panic/fatal/OOM。
- 升级前后哈希保持：Compose `4f47f9ffdd63b8a5082447dee80b7b574a086ab3f2daac3b15395bf9f2a4184d`、`.env` `0ba468d8031cd5f67c5a7ffb0333c13dabd1c72c5223f7004df0084712a096f4`、`/root/apps/kpanel.conf` `82f06ca32ce827ef8d0c9c72e65eed9180841a23cbc507237072b58a0807ef04`。
- 生产写入仅限标准应用更新产生的新镜像/容器、Agent 二进制和 systemd 运行文件；未执行 Docker 项目、SSH、防火墙、账户或业务数据写操作。

## 回滚

- 成套回滚点：`v0.86.1`、OCI `sha256:232045ade043d571d3f51d2c07fcf82c3d4ebab324e9ec56c96ff357c0a1bf11` 与 `/root/kpanel-backups/pre-v0.86.2-20260819T034749Z`。
- 回滚步骤：停止 Agent 与 Compose，恢复备份中的匹配镜像/Compose/`.env`/Agent 文件和数据，执行 `systemctl daemon-reload`，启动并核对 `/api/v1/health`、healthy、restart/OOM 及三项配置哈希；不得只换镜像或只改配置。
- 本轮未执行回滚；当前 154 保持 v0.86.2 healthy。公开 `latest` 与标准更新入口已指向 v0.86.2。

## 交付节奏数据

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-08-19T11:21:45+08:00
- 候选冻结时间：2026-08-19T11:29:32+08:00
- 生产完成时间：2026-08-19T11:48:41+08:00
- 提交到生产用时：0.45 小时
- 是否回滚、紧急热修复或重复发布：否
- 若发生失败，发现时间、恢复时间和逃逸门禁：不适用
<!-- kpanel-release-metrics:end -->

<!-- kpanel-release-process-metrics:start -->
- 已记录发布流程异常或无效证据拦截次数：0
- 其中生产写操作开始后异常次数：0
<!-- kpanel-release-process-metrics:end -->

## 遗留风险与后续准入

- 用户现场 Windows/Chrome/Telegram 拖拽验证尚待完成；若仍出现“贵组织屏蔽了此文件”，应归类为浏览器企业策略/DLP，而非通过 KPanel 代码绕过。
- 108 永久不纳入 KPanel 测试、灰度或部署。
- 若组织策略放行后仍出现产品错误，再基于最新 main 制作新的最小补丁；不得改写 `v0.86.2` Tag/Release。
