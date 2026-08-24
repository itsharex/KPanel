# KPanel v0.96.0 发布验收记录

日期：2026-08-24

发布级别：L3

候选提交 / 标签：`9fd8492d71739434a85a745b284e3566f1c508ed` / `v0.96.0`

上一稳定版本 / 回滚点：`v0.95.2` / `sha256:cfe375d0390d73ddac7f0bab82ba7da48dbf919a9c886c2609cbae1f3fc3b89f`

## 发布画像

- 业务域：系统中心、日常维护、宿主机系统日志。
- 变更面：Panel/Agent typed API、持久化维护任务、Web 弹窗与共享日志脱敏器；不修改应用市场安装契约、端口或受管 `kejilion.sh` 固定版本。
- 受影响用户旅程：按来源、服务、级别和条数读取日志；本地搜索；显式 3 秒刷新；查看日志占用；执行三种固定 journal 清理策略。
- 风险等级及理由：中；新增宿主机只读日志访问和不可恢复的固定 journal vacuum 动作，但不接受任意路径、命令、日志编辑或删除参数，写操作通过持久化任务、固定策略和审计边界执行。

## 发布范围与未纳入内容

- 功能提交：`9e9a1e2dab8739dfa8b8add5a19265d4cb42feeb`；版本准备：`833f978...`；最终门禁修复：`9fd8492d71739434a85a745b284e3566f1c508ed`。
- 系统日志以 Linux journal、真实 `.service`、认证/安全日志和 `last` 为真源；Web 仅允许 50/100/200 条、有界优先级和本地搜索。
- 3 秒刷新默认关闭，弹窗关闭、页面隐藏或离开时停止；不创建数据库副本、常驻采集器或无限日志流。
- 清理策略固定为保留 7 天、保留 3 天、归档 journal 最大 500 MiB；不修改 `journald.conf` 或 `logrotate`。
- 最终修复保持日志清理确认字段为空且拒绝任何非空确认值，以满足生态策略并避免形成第二套确认协议。
- 未纳入其他工作树、未提交草稿、无关功能或 108 环境。

## 多维质量结论

| 维度 | 状态 | 证据 | 未验证风险或不适用依据 |
| --- | --- | --- | --- |
| 业务正确性 | 已验证 | 真实 host summary/read、固定服务读取、来源/级别/条数/搜索/刷新；隔离 systemd 环境三种 vacuum 与完整 Panel→Agent 任务成功 | 未在生产执行破坏性 vacuum |
| 网络与供应链安全 | 已验证 | govulncheck 可达漏洞 0、npm audit 0、Trivy source/image 0；公开 OCI revision/version/脚本摘要固定 | 无新增公网匿名入口 |
| 稳定性与兼容 | 已验证 | canonical L3、候选/main/Tag CI、公开 OCI E2E、停写备份、标准更新和生产核验全绿 | 目标依赖 systemd/journal；非 systemd 主机按能力边界返回不可用 |
| 性能与资源 | 已验证 | 生产 3 次采样 CPU 0.02%、内存 72.61 MiB/256 MiB、7 PIDs，restart=0、OOM=false | 未做长 soak；功能按需读取且刷新显式开启 |
| 用户体验与可访问性 | 已验证 | 真实浏览器桌面及 390×844，切换来源/级别、搜索和刷新均正常；手机 document clientWidth=scrollWidth=390，console error=0 | 125%/200% 缩放未单独截图 |
| 数据与配置 | 已验证 | 升级前后配置摘要一致；21 个 JSON、1 个 SQLite 完整；无 schema 或应用市场契约变化 | KPanel 备份不包含宿主机完整日志归档 |

## 自动门禁

- canonical L3：`v0.96.0-9fd8492-l3-r2`，exit 0；Go 全包、核心 race、vet、Web 116 文件/945 项、系统日志 UI 16 项、i18n 2550/21、typecheck、production build、双架构构建、安装安全、应用生命周期、govulncheck、npm audit、Trivy source/image 全绿。
- bundle SHA-256=`6d82e5bc0e0afed7ce9c50f637c5845ae97733af15d1de96faf7311bb2071eb9`；L3 日志 SHA-256=`ff04b71e1447c1e0ba64a66e0e70dcf6ff93d96d4447682095f85e0758c35c10`；状态 SHA-256=`46bfd638af7abe011f60e553f7dfab380b29bd2580c75ede0d0b39c8044bb436`；manifest SHA-256=`634efb98d3a4f989c79318a1b4e819179d7761b76d4f88b7f273ab7a079db122`。
- 固定 Runner：`kpanel-release-gate:go1.26.6-node24`，image ID=`sha256:b593c0ffe32e80a6a6ae8fcac38ed916587dbf698a6b6a55fa33887298737148`。
- 候选 CI `32692284768`、Dependency freshness `32692284754`；main CI `32692430841`、Dependency freshness `32692430803`；Release workflow `32692746277`、Tag Dependency freshness `32692746330`，均 completed/success 且绑定精确 SHA。
- 第一次 L3 `v0.96.0-833f978-l3-r1` 被生态策略在产品发布前 fail-closed；修复产品 SHA 后旧证据作废，r2 从零执行。

## 隔离真机与安全验收

- `arena-154` 真实宿主只读验收：`/var/log` 约 922,820,608 bytes、journal 约 920,859,443 bytes、131 个 units；系统、登录、安全和指定服务日志均返回结构化有界结果。
- 非法 service unit 在宿主访问前以 422 `invalid_system_log_query` 拒绝。
- 隔离 Ubuntu 24 systemd 容器中，三种清理策略均真实完成 `journalctl --rotate`/vacuum；完整 Panel→Agent 维护任务达到 succeeded/completed，审计仅记录固定 action/policy 的 intent 和 success。
- 共享脱敏器对合成敏感日志返回 `[REDACTED]`，原秘密未进入响应；Docker 日志展示复用相同保护。
- 候选 Panel、Agent 与隔离环境均 0 restart、0 OOM；测试容器、临时 unit、token、cookie 和隧道均已清理。
- L3 证据：`/root/kpanel-release-evidence/v0.96.0-9fd8492-l3-r2`；专项证据：`/root/kpanel-release-evidence/v0.96.0-system-logs-e2e`。

## 发布产物与公开仓库复核

- GitHub Release：[v0.96.0](https://github.com/kejilion/KPanel/releases/tag/v0.96.0)，非 draft、非 prerelease，含 8 个附件；annotated Tag object=`9dd80cd58189acd751221c96c2caf81e2de28e4e`，peel=`9fd8492d71739434a85a745b284e3566f1c508ed`。
- Docker `0.96.0` 与 `latest` OCI index：`sha256:1c5148551fa3bdf02a07c398705a1423b317aa74f23145d9fb0d4dd0e1cb632a`。
- `linux/amd64`=`sha256:636a37fdf956efb49e3007e6867482deba8504b11158b084d17a4986e15f1e9b`；`linux/arm64`=`sha256:a8aa2cfc8199f08195bab51e9de41fe17d570942d04896a9019418c90c92d6a6`。
- OCI labels：version=`0.96.0`、revision=`9fd8492d...`、managed script revision=`9fec61b50cc6ef798dfac1edf11c2ec60ca6b0d1`、managed script SHA-256=`54ceb0e72c4c342382500fc35da636fa436c484a12c4766fb9c7f806a23ae8fa`。
- arena-154 从公开标签独立回拉并通过正式 `packaging/tests/image-e2e.sh`；证据 `/root/kpanel-release-evidence/v0.96.0-public`。
- 应用市场安装契约相对 v0.95.2 无差异，未制造 apps 空提交。

## 生产部署安全核对

- 目标仅 `arena-154`；108 未连接、未测试、未备份、未部署。
- 部署前：v0.95.2、Panel healthy、Agent active、restart=0、OOM=false；旧 OCI=`sha256:cfe375d0390d73ddac7f0bab82ba7da48dbf919a9c886c2609cbae1f3fc3b89f`。
- 停写一致性备份：`/root/kpanel-backups/pre-v0.96.0-20260824T052328Z`；`SHA256SUMS` SHA-256=`e7ac14f6f488e9546d563259d7bd6a9ca6a04995dc70046709d119b709649fc5`。状态目录、旧 OCI、Compose、`.env`、Agent 文件、21 个 JSON 和 2 个 SQLite 均完成摘要、独立解包/比对、完整性与 `docker load` 校验。
- 标准更新入口一次成功：`KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update bash /home/docker/kpanel/bin/kejilion.sh app kpanel`；更新日志 SHA-256=`6d0f390289ddf5949b27d33b14b75a4cc6791c1cc296445cd38642c2c6592233`。
- 部署后：Panel 0.96.0、revision=`9fd8492d...`、OCI index 精确匹配，Agent active、Panel healthy、restart=0、OOM=false；公网 health 返回 0.96.0，系统日志路由未登录返回 401。
- `.env`、Compose 和 apps 配置升级前后摘要一致；21 个 JSON、1 个有效 SQLite 生产数据复核通过；生产日志无 fatal/panic/OOM/protocol/version mismatch。
- 生产资源证据 SHA-256=`7efefc8318d52070871df4a3eefe4c9105fc7bf74a08ab4c41e67152bb1fe9fe`；完整生产证据目录 `/root/kpanel-release-evidence/v0.96.0-production`。

## 回滚

- 源码/tag：`v0.95.2` / `19c59eaa4a140e2c538f2b86f92c99cfad64defb`。
- 旧镜像：`sha256:cfe375d0390d73ddac7f0bab82ba7da48dbf919a9c886c2609cbae1f3fc3b89f`。
- 备份：`/root/kpanel-backups/pre-v0.96.0-20260824T052328Z`，已完成摘要、独立恢复、JSON/SQLite/Compose、关键文件 `cmp` 与旧 OCI 加载验证。
- 回滚步骤：停写；校验 `SHA256SUMS`；加载旧 OCI；成套恢复 KPanel 目录、Compose、`.env`、apps 配置和 Agent 文件；daemon-reload 后启动 Agent/Panel；复核 v0.95.2、digest、health、restart/OOM、数据和公网入口。禁止只换镜像。
- 未执行生产实际回滚；发布前已恢复旧版并验证备份可执行，当前 v0.96.0 healthy/active。journal vacuum 不在生产验收中执行；历史日志若由管理员主动清理，不随 KPanel 回滚恢复。

## 交付节奏数据

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-08-24T12:23:35+08:00
- 候选冻结时间：2026-08-24T12:27:06+08:00
- 生产完成时间：2026-08-24T13:25:07+08:00
- 提交到生产用时：1.03 小时
- 是否回滚、紧急热修复或重复发布：否
- 若发生失败，发现时间、恢复时间和逃逸门禁：不适用
<!-- kpanel-release-metrics:end -->

<!-- kpanel-release-process-metrics:start -->
- 已记录发布流程异常或无效证据拦截次数：6
- 其中生产写操作开始后异常次数：0
<!-- kpanel-release-process-metrics:end -->

### 流程异常明细

<!-- kpanel-release-process-incidents:start -->
[
  {
    "fingerprint": "local-verification/windows-path/missing-gofmt",
    "position": "before-production-write",
    "count": 1,
    "impact": "最小策略修复后的本地 Windows 环境未在 PATH 提供 gofmt，未据此接受格式或 Go 测试结论。",
    "recoveryEvidence": "最终 SHA 在固定 Linux Runner 的完整 Go test、race、vet 和 canonical L3 全绿。",
    "permanentAction": "Go 发布证据固定由仓库 Linux Runner 生成，本地缺失工具只作预检提示。",
    "historicalReleases": []
  },
  {
    "fingerprint": "isolated-agent/token-file/ownership-mode-preflight",
    "position": "before-production-write",
    "count": 2,
    "impact": "隔离 Agent 两次因临时 token 文件 owner/mode 不符合正式契约而 fail-closed，未接触生产。",
    "recoveryEvidence": "按 root:kejilion-panel 0640 重建临时 token 后正式 Agent unit 和 Panel 链路通过。",
    "permanentAction": "隔离 Agent 启动前固定核对 token owner、group、mode 与 unit User/Group。",
    "historicalReleases": []
  },
  {
    "fingerprint": "isolated-evidence/minimal-container/tool-and-schema-assumption",
    "position": "before-production-write",
    "count": 2,
    "impact": "一次假设最小 systemd 容器含 jq，另一次按 .status 而非真实 .state 读取任务证据；两次输出均未被接受。",
    "recoveryEvidence": "改由宿主 jq 读取真实任务 schema，确认任务 succeeded/completed 且三种策略回读一致。",
    "permanentAction": "隔离证据解析使用 Runner/宿主工具，并先从真实 JSON 样本确认字段。",
    "historicalReleases": []
  },
  {
    "fingerprint": "isolated-audit/evidence-source/wrong-filename-assumption",
    "position": "before-production-write",
    "count": 1,
    "impact": "首次查找不存在的 audit.jsonl，未形成审计结论。",
    "recoveryEvidence": "按项目真实 panel-state.json 审计真源确认精确 intent 和 success 各一条且无敏感日志。",
    "permanentAction": "审计验收先读取仓库 Store 真源与现有测试，不猜测文件名。",
    "historicalReleases": []
  }
]
<!-- kpanel-release-process-incidents:end -->

## 遗留风险与后续准入

- 生产仅做安全只读核验，未执行任何 journal vacuum；固定清理策略已在隔离 systemd 环境验证。管理员实际清理前仍应按合规要求独立归档宿主机日志。
- 非 systemd、无 journal 或发行版认证日志位置不同的主机只能按能力边界返回可理解错误，不应回退到任意文件路径或 shell。
- 后续若新增来源、级别、条数或清理策略，必须继续使用 typed allowlist、共享脱敏器、持久化任务和真实宿主回读，并重新执行生产前隔离写验收。
