# KPanel v0.88.0 发布验收记录

日期：2026-08-20

发布级别：L3（兼容功能）

候选提交 / 标签：`14962aeff7186e846f52db3c989d5220a854869f` / `v0.88.0`

上一稳定版本 / 回滚点：`v0.87.0` / `sha256:9ff54a33fb9f325bc205c06c5416678394e7e1a5d63a452c22f8ffcac6100fb2`

## 发布画像

- 业务域：系统体检与一键跑分、桌面服务状态卡片、AI 助手历史消息兼容、一条龙系统调优脚本运行时。
- 变更面：展示、只读诊断、受控宿主机写入脚本、受管脚本供应链固定；没有数据库迁移或部署拓扑变化。
- 受影响用户旅程：体检页执行并阅读五维结果、桌面读取服务状态卡片、打开包含旧推理消息的 AI 会话、从 KPanel 执行一条龙系统调优。
- 未变化契约：外部端口、Compose、Agent 权限、应用市场安装/更新协议和现有数据目录保持不变；诊断继续通过受管脚本与既有后台任务边界执行。
- 风险等级及理由：中；诊断与脚本涉及宿主机信息读取和可选管理员写操作，但执行边界、固定 revision/hash、失败停止和回滚入口均保留。

## 发布范围与未纳入内容

- 用户可见更新：体检页原生五维跑分与更紧凑的结果摘要；服务状态卡片正文提高到 13px、指标提高到 15px 并保留省略提示；AI 旧推理历史不再破坏渲染；一条龙调优下载和外部脚本调用不再继承宿主代理。
- 精确提交清单：`b7a6449`、`3cfd1d5`、`fe73050`、`3f7365f`、`4ec802f`、`46e2a99`、`370f2de`、`1d72d19`、`df766d1`、`7c083a9`、`14962ae`；受管脚本提交 `kejilion/sh@6fa7bcc7c2d15fe09d829cb9664941ff40bf4aaf`。
- 明确未纳入的分支、文件或后续事项：未合入其它旧工作树、未提交草稿或无关实验；没有应用市场空提交；没有连接或操作 108。

## 多维质量结论

| 维度 | 状态 | 证据 | 未验证风险或不适用依据 |
| --- | --- | --- | --- |
| 业务正确性与双端互通 | 已验证 | L3 全量、arena-154 隔离 Panel+Agent 原生五维诊断、公开镜像 E2E | 生产未执行会改变 SSH、防火墙、DNS 的十二项调优 |
| 网络入侵与供应链安全 | 已验证 | govulncheck 可达漏洞 0、npm audit 0、Trivy 源码/镜像 0、脚本 revision/hash 与镜像内文件一致 | 不适用：未放宽网络或权限边界 |
| 稳定性、失败恢复与兼容 | 已验证 | Go/Web/竞态/vet、双架构构建、旧 AI 历史回归、生产三次健康和三次资源采样 | 首次单测在 5 秒边界超时，原测试 5 次定向及完整 L3 重跑均通过 |
| 性能与资源预算 | 已验证 | 生产 Panel 三次采样 CPU 0.02%～0.03%、内存 73.65～73.75 MiB/256 MiB，0 restart/OOM | 未做长时间压力测试；本版没有高负载常驻服务变化 |
| 用户体验与可访问性 | 已验证 | 105 文件/798 测试；标准本地预览的深浅主题截图；服务卡片最小正文 13px，省略内容提供 title | 125%/200% 缩放和生产登录后截图未新增实测 |
| 数据、配置与迁移 | 已验证 | SQLite `integrity_check=ok`、Compose config、apps 配置 hash、停写备份独立解包/比对/旧镜像恢复核验 | 不适用：无 schema 迁移 |

## 自动门禁

- 定向测试及结果：服务卡片 2 文件/21 测试通过；首次 L3 中既有 DesktopView 位置测试触及 5 秒上限，随后同一冻结 SHA 定向连续 5 次及完整 L3 重跑通过。
- `make verify-release` 环境和结果：arena-154 Debian 13、Docker 29.6.2、固定 Runner `sha256:b593c0ffe32e80a6a6ae8fcac38ed916587dbf698a6b6a55fa33887298737148`；Go 全量/竞态/vet、Web 105 文件/798 测试、i18n 2422/20、typecheck/build、双架构、安装安全、应用生命周期全部通过，日志 SHA-256=`b7ddf928cf4dfbe4a5e8c73def81d023b7f4487dd45f3bfc2518320166c2438e`。
- 候选 CI：run `32339019688` success；Dependency freshness run `32339019668` success。
- 主线 CI：run `32339258825` success；Dependency freshness run `32339258862` success。
- Release workflow：run `32339617576` success；Tag Dependency freshness run `32339617881` success。
- 安全扫描、镜像契约、SBOM/provenance：govulncheck、npm audit、Trivy 源码/镜像和 OCI 标签契约通过；双架构 provenance/SBOM attestation 保留。

## 依赖与技术栈变化

- `make dependency-report` 生成时间及检测源完整性：2026-08-20 Release L3；依赖策略、治理一致性和每日新鲜度门禁通过。
- 最近每日安全通告审计、EOL 复核状态及证据：候选、main、Tag 的 Dependency freshness 均 success；govulncheck 可达漏洞 0，npm audit 0，Trivy 0。
- 本版采用的依赖、工具链、基础镜像、Action、扫描器或受管脚本候选：没有新增第三方依赖；Go 1.26.6、Node 24.18.0、现有固定 Actions/扫描器；受管脚本更新为 `6fa7bcc`。
- 版本/锁文件/Action SHA/镜像 digest/脚本提交与摘要：Node 基础镜像 `sha256:a0b9bf06e4e6193cf7a0f58816cc935ff8c2a908f81e6f1a95432d679c54fbfd`；Go 基础镜像 `sha256:af8d6740070b8906d12eae1c3e3ea0957fb63f492051ea05e354c38ef9fe88df`；脚本 SHA-256=`534a7a1866a7ab4f36229571b74492cb333c183a4375968141d3d40a9802b8d9`。
- 暂缓或拒绝候选、证据、负责人、复核日期和退出条件：不适用；没有待升级依赖候选。
- 升级后的兼容、安全、构建、性能资源和回滚结论：均通过；旧 OCI 和完整状态备份可执行成套回滚。

## 隔离真机与浏览器验收

- 主机/发行版/架构/运行时版本：arena-154 Debian 13、x86_64、Docker 29.6.2；发布构建覆盖 linux/amd64 与 linux/arm64。
- 环境策略 ID 与允许用途：`arena-154`，candidate-validation、performance-validation、production-safety-check 和 production-deploy。
- 使用的精确候选或公开产物：源码 `14962aeff7186e846f52db3c989d5220a854869f`；候选镜像 `sha256:1601fe7ce5593892411e84f7ddc382ed833b7fe94edeb74dbab399c9f8204f57`；公开 OCI `sha256:81b76fe59b15ceaed985421e79f81afd744eb9776fbdb2566b1464814cecfb99`。
- 后台作业 ID、终态、退出码、超时、证据目录、命令规格路径及 SHA-256：原生诊断 job `1d1df0e1a19cccebbeb0fe0e86e7c45a` succeeded/exit 0；证据 `/root/kpanel-release-evidence/v0.88.0/isolated-native/`，日志 SHA-256=`3d7f55fed1d1e0cc7b1f1dbaa0bfb0d58db27cfcb11190e9b5d09f38a5808b4b`。
- 测试窗口/循环数及风险依据（无 soak 时写不适用依据）：隔离原生五维任务完整执行一次；生产 3 次健康/资源采样。本版没有新增常驻高负载进程，不执行长时间 soak。
- 受影响用户旅程、视口、100%/125%/200% 缩放、最小计算字号、主题、键盘/焦点、语言和失败态：100% 桌面深浅主题、简中卡片协调度截图通过；服务卡片正文 13px、主指标 15px，与监控卡片 12/14px 层级协调；自动化覆盖失败态和省略提示。125%/200% 未新增人工证据。
- 宿主机写入、失败注入、重启恢复和回滚结果：诊断在隔离 Panel+Agent 链路完成，输出 `kpanel-native-v1` 与五维结果；停写备份已完成独立恢复核验。未在生产执行破坏性十二项调优。
- 未执行场景及原因：生产浏览器登录态截图、生产 SSH/防火墙/DNS 写操作不执行；发布前本地标准 mock 截图只作为视觉证据，不冒充生产真数据。

## 发布产物与公开仓库复核

- GitHub Release：[v0.88.0](https://github.com/kejilion/KPanel/releases/tag/v0.88.0)，非 draft、非 prerelease。
- Docker 版本与 `latest` OCI index：两者均为 `sha256:81b76fe59b15ceaed985421e79f81afd744eb9776fbdb2566b1464814cecfb99`。
- `linux/amd64`、`linux/arm64` digest：amd64 `sha256:db8db386719fde5049525cc091d7e52eefbb7299c8872a67c57f850a730fe956`；arm64 `sha256:33ff9f5a82faec1c23da9e38589a064ddebc225c55fad98ce09ea816aca03262`。
- 附件及 `SHA256SUMS`：Agent/Node 双架构、部署包、LICENSE、THIRD_PARTY_NOTICES 与 SHA256SUMS 齐全；SHA256SUMS asset digest=`sha256:09e2b46510f9f5310f904ab30ffa9e16d795a1209cc547e5017bff60e6d62fed`。
- 公开镜像 `image_e2e=pass`：arena-154 隔离端口 `18097` 从公开仓库回拉通过，日志 SHA-256=`919d8894c476fdd4244dfd74359e15438dc7ff43c831c35c6b06ee415ab523f8`。
- `kejilion/apps` / `kejilion.sh` 契约结论：`packaging/kejilion-app/kpanel.conf` 与 apps main 归一化内容一致，hash=`82f06ca32ce827ef8d0c9c72e65eed9180841a23cbc507237072b58a0807ef04`，无需空提交；`kejilion/sh main@6fa7bcc` 和公开 raw SHA-256 已核对。

## 生产部署安全核对

- 生产目标和部署授权范围：仅 `arena-154`；用户已授权 v0.88.0 正式升级。
- 验证/灰度环境（必须来自 `environment-policy.json`，不得包含 `prod-108`）：`arena-154` 隔离容器和回环端口。
- 正式部署环境（默认 `arena-154`；不得包含 `prod-108`）：`arena-154`。
- `prod-108`：禁用全部 KPanel 操作；确认本次未连接、未备份、未部署、未升级、未核对：已确认。
- 部署前版本、健康、备份位置及摘要：v0.87.0，Panel healthy、Agent active、restart=0、OOM=false；备份 `/root/kpanel-backups/v0.88.0-preupgrade-arena154-20260820T063912Z`；状态包 SHA-256=`2ded29d5da3724d98e6b36874fea4becbe8ce7fd2dc0613e17c1ba021621c606`，旧镜像 tar SHA-256=`3a83b148e92b2ecbd4a48bc10a8dad9ba8210773599542526f6aec99e32fb977`。
- 部署命令/入口：标准应用市场更新入口 `KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update bash /home/docker/kpanel/bin/kejilion.sh app kpanel`，进度到 `KPANEL_PROGRESS 100`。
- 部署后版本、Panel/Agent 状态、重启、日志、数据完整性和公网入口：Panel/Agent 0.88.0，Panel healthy、Agent active、双方 restart=0、OOM=false；三次健康与资源采样通过，SQLite integrity=ok，近 10 分钟 panic/fatal/error/OOM 为空，HTTPS 健康入口 200。
- 生产已执行写操作：停写一致性备份、旧服务原位恢复核验、标准应用市场升级；运行时脚本只按既有安装契约设置 `permission_granted=true`，归一化后与固定脚本逐字节一致。
- 仅在隔离真机执行、未在生产执行的场景：完整原生五维诊断作业及任何会修改 SSH、防火墙、DNS、内核或软件包的一条龙操作。

## 回滚

- 源码/tag：`v0.87.0` / `15261a3ee1fa7af45d62134719f21f28f6554a9a`。
- 镜像 digest：`sha256:9ff54a33fb9f325bc205c06c5416678394e7e1a5d63a452c22f8ffcac6100fb2`。
- 数据/配置备份：`/root/kpanel-backups/v0.88.0-preupgrade-arena154-20260820T063912Z`，含 Compose、`.env`、Agent/脚本、Panel/Agent 数据、apps 配置、systemd unit 和旧镜像 tar；独立解包、关键文件 cmp、Compose parse、SQLite integrity 和 `docker load` 均通过。
- 回滚步骤和回滚后复核：停写；从备份加载旧镜像并成套恢复 `kpanel` 目录、apps 配置和 systemd unit；`systemctl daemon-reload` 后启动 Panel/Agent；核对 v0.87.0、health、restart/OOM、SQLite、Compose 和公网入口。
- 回滚后生产实际版本与健康状态：未执行回滚；当前 v0.88.0 healthy。
- GitHub Latest、Docker `latest` 与标准更新入口实际指向：均为 v0.88.0 / `sha256:81b76fe59b15ceaed985421e79f81afd744eb9776fbdb2566b1464814cecfb99`。
- 公共默认更新通道决策：不适用；所有生产门禁通过，无需恢复上一稳定默认版本。

## 交付节奏数据

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-08-20T10:04:14+08:00
- 候选冻结时间：2026-08-20T13:46:51+08:00
- 生产完成时间：2026-08-20T14:42:46+08:00
- 提交到生产用时：4.64 小时
- 是否回滚、紧急热修复或重复发布：否
- 若发生失败，发现时间、恢复时间和逃逸门禁：不适用
<!-- kpanel-release-metrics:end -->

<!-- kpanel-release-process-metrics:start -->
- 已记录发布流程异常或无效证据拦截次数：10
- 其中生产写操作开始后异常次数：4
<!-- kpanel-release-process-metrics:end -->

流程异常均与产品载荷无关：生产写入前包括一次测试超时、两次隔离凭据夹具错误、一次错误 SSH 身份及两次 PowerShell 远端命令转义；均在门禁内更换等价验证通道后通过。生产写入后首次校验错误地要求已授权宿主脚本保持发行默认 `permission_granted=false`，差异审计确认仅该安装态字段变化，归一化逐字节一致后重新执行完整生产校验通过；验收文档提交后的 Windows Git Bash 无法识别 linked-worktree gitdir，独立 clone 的首次启动路径与稳定 Tag 集合又各需一次修正，随后在同一验收内容的独立普通 Git clone 中重跑文档变更门禁并通过。没有产品失败、回滚或门禁逃逸。

## 遗留风险与后续准入

- 未验证风险：125%/200% 人工缩放、生产登录态截图以及真实生产宿主的一条龙十二项写操作未执行。
- 已实现待实机准入：用户可在实际桌面确认服务卡片主观观感；系统调优写操作仍应先在专用测试机执行。
- 不阻断本版的理由：视觉自动化与标准 mock 深浅主题证据通过；破坏性写操作不属于生产发布健康验收，且脚本协议、固定摘要和隔离链路已验证。
- 后续应进入的自动门禁或专项工作流：将 DesktopView 位置测试的 5 秒边界调整为稳定预算；继续保留服务卡片 13px 最小正文和受管脚本归一化契约回归。
