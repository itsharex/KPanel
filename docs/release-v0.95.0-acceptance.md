# KPanel v0.95.0 发布验收记录

日期：2026-08-24

发布级别：L3

候选提交 / 标签：`e84651f7e4b530fdbef96d6f9fc94fc887efb45a` / `v0.95.0`

上一稳定版本 / 回滚点：`v0.94.2` / `sha256:548ed3c20a685298d71da62d4db8a63cf07bdf8e4f0b090d7448e280722d2149`

## 发布画像

- 业务域：桌面工作区、窗口标题栏和任务栏的 KPanel 原生图标体系。
- 变更面：Web 静态资源、展示映射和降级逻辑；不改 API、数据、端口、Compose、Agent 权限、`kejilion.sh` 或应用市场契约。
- 受影响用户旅程：经典桌面打开系统应用、切换窗口、查看任务栏，以及文件/目录快捷方式的桌面识别。
- 未变化契约：后端协议、持久数据、安装更新、镜像运行参数和宿主机权限均未变化。
- 风险等级及理由：中低；新增 12 个受版本控制的 512×512 WebP 资产并替换系统应用的图标呈现，保留加载失败时的既有图标回退。

## 发布范围与未纳入内容

- 用户可见更新：概览、AI、网站、应用、Docker、文件、终端、体检、集群、系统、活动记录和设置使用统一原生图标；窗口标题栏与任务栏复用同一资产；文件/目录桌面入口继续使用各自类型图标。
- 精确提交清单：`1c6a58e`（从聚焦源提交 `2cc8ba7` 安全重放的原生图标套装）和 `e84651f`（0.95.0 版本、CHANGELOG 与升级/回滚说明）。
- 明确未纳入：设计工作树中约 50 个未引用的旧 WebP 生成稿、已上线的 Monitoring/视频修复、重复旧候选和其他未提交草稿均未进入候选；源设计工作树未删除。

## 多维质量结论

| 维度 | 状态 | 证据 | 未验证风险或不适用依据 |
| --- | --- | --- | --- |
| 业务正确性与双端互通 | 已验证 | 12 个映射资产全部存在并以 512×512 加载；窗口、任务栏与桌面复用一致；生产 12 个静态 URL 均 HTTP 200 | 不涉及后端或跨主机协议 |
| 网络入侵与供应链安全 | 已验证 | govulncheck 可达漏洞 0、npm audit 0、Trivy source/image 0、正式 OCI revision/version 固定 | 新增内容仅本地静态 WebP，无新网络入口 |
| 稳定性、失败恢复与兼容 | 已验证 | 资产失败回退、完整 L3、候选/main/Tag CI、公开 OCI E2E、停写备份和标准更新事务通过 | 浏览器缓存按内容哈希更新，不改工作区 schema |
| 性能与资源预算 | 已验证 | 图片按构建资产加载；生产 3 次采样 CPU 0.01%～0.02%、内存 73.92 MiB/256 MiB、7 PIDs，restart=0、OOM=false | 未做长 soak；本版没有后台任务或常驻进程变化 |
| 用户体验与可访问性 | 已验证 | 独立浏览器桌面 1440×900、手机 390×844、明暗主题、窗口/任务栏验收；无横向溢出，console error/warn=0 | 125%/200% 未单独截图；CSS 尺寸与既有桌面缩放机制未改变 |
| 数据、配置与迁移 | 已验证 | `.env`、`panel-state.json`、apps 配置升级前后逐字节不变；21 个 JSON、2 个 SQLite 备份恢复检查通过 | 本版无 schema 或配置迁移 |

## 自动门禁

- 定向测试及结果：图标目录/文件快捷方式相关 2 文件/31 项通过；最终候选完整 Web 115 文件/923 项、i18n 2468/21、typecheck 和 production build 通过。
- `make verify-release` 环境和结果：固定 Linux Runner 内完整 L3 exit 0；Go 全包、核心 race、vet、Linux amd64/arm64、Web、安装安全、应用生命周期、安全扫描和镜像契约全部通过。
- L3 外层入口：`v0.95.0-e84651f-l3-r2`；bundle `kpanel-v0.95.0-e84651f-l3-r2.bundle` SHA-256=`88843a9ad661d7c1bb836900bc74773791c8182878ef158bcb48f6cac4adce7e`；Runner `kpanel-release-gate:go1.26.6-node24` / `sha256:b593c0ffe32e80a6a6ae8fcac38ed916587dbf698a6b6a55fa33887298737148`；L3 日志 SHA-256=`89ad27160ea3c4b57b593723fa3fecf6d7d2793574f4c2a3fc0fc63c9cc7bd07`；远端证据 `/root/kpanel-release-evidence/v0.95.0-e84651f-l3-r2`。
- 候选 CI：CI `32653771005`、Dependency freshness `32653770951`，均 completed/success，精确绑定 `e84651f...`。
- 主线 CI：CI `32654032846`、Dependency freshness `32654032839`，均 completed/success，精确绑定 `e84651f...`。
- Release workflow：`32654291860` completed/success；Tag Dependency freshness `32654291915` completed/success。
- 安全扫描、镜像契约、SBOM/provenance：Trivy source/image、govulncheck、npm audit 全绿；正式 amd64/arm64 OCI 包含 SBOM/provenance attestations，revision=`e84651f...`、version=`0.95.0`。

## 依赖与技术栈变化

- `make dependency-report` 检测源完整性：候选、main 和 Tag 的 Dependency freshness 均成功；本版没有依赖差异。
- 最近每日安全通告审计、EOL 复核状态及证据：上述同 SHA GitHub 门禁与 L3 安全扫描通过。
- 本版采用的依赖、工具链、基础镜像、Action、扫描器或受管脚本候选：全部沿用 v0.94.2 固定版本。
- 版本/锁文件/Action SHA/镜像 digest/脚本提交与摘要：仅产品版本更新到 0.95.0；锁文件无依赖变化，受管脚本契约不变。
- 暂缓或拒绝候选、证据、负责人、复核日期和退出条件：未引用的旧图标生成稿留在源设计工作树，不进入发布；无依赖暂缓项。
- 升级后的兼容、安全、构建、性能资源和回滚结论：均通过；旧版成套备份和 OCI 已验证可恢复。

## 隔离真机与浏览器验收

- 主机/发行版/架构/运行时版本：`arena-154`、Linux amd64、Docker；Codex 独立应用内浏览器，不接管用户 Chrome。
- 环境策略 ID 与允许用途：`arena-154` / candidate-validation、production-deploy、production-safety-check；`prod-108` 不在允许目标内。
- 使用的精确候选或公开产物：源码 `e84651f...`；浏览器预览由同一 clean 候选构建，正式 OCI revision/version 与该提交一致。
- 后台作业终态：标准 `local-feature-preview` 预览已停止；浏览器证据位于 `C:\GitHub\_release-artifacts\v0.95.0-e84651f-l3-r2\browser-acceptance`，桌面暗色截图 SHA-256=`60a9c1b019df5dff92712ee075148c2340c78611ab1946276e9d2b91551683a0`，手机截图 SHA-256=`a1cfb3b55c51292540a70d3219f360ffd9f99c31cd9197a15dc9330c9b56ebcc`。
- 测试窗口/循环数及风险依据：确定性静态资产和 UI 映射采用桌面/手机、明暗主题及窗口/任务栏单次完整旅程；无后台逻辑，长 soak 不适用。
- 受影响用户旅程、视口、100%/125%/200% 缩放、最小计算字号、主题、键盘/焦点、语言和失败态：100% 下 1440×900 与 390×844 通过；12 个原生图标全部自然尺寸 512×512、failed=0；桌面和手机 `clientWidth=scrollWidth`；窗口/任务栏图标一致；加载失败回退有自动化覆盖。125%/200% 未单独截图，交互、文字和焦点结构未修改。
- 宿主机写入、失败注入、重启恢复和回滚结果：候选只写隔离临时目录，预览已清理；生产回滚材料单独验证。
- 未执行场景及原因：不逐个打开全部 12 个业务页面；本版只改变统一图标资产映射，窗口/任务栏复用与失败回退已覆盖。

## 发布产物与公开仓库复核

- GitHub Release：[v0.95.0](https://github.com/kejilion/KPanel/releases/tag/v0.95.0)，非 draft、非 prerelease。
- Docker 版本 `0.95.0` 与 `latest` OCI index：`sha256:c9f9ec7588e6760065614f9aa436273f4b2764dae1a1e465e2acb3ebcaacf0ad`，两者一致。
- `linux/amd64`=`sha256:5bc84eb6dc73f73d4c69c984506eeefdf107dfcd87d2ffc45b737368a5e11cdc`；`linux/arm64`=`sha256:fdcf0425cdd5a7d6c2854f242e77e9afad0d94b174e320500dbd709e51634103`；额外 unknown/unknown 为 attestations。
- 附件及 `SHA256SUMS`：8 个公开附件均可下载；5 个二进制/部署包逐项校验通过，`SHA256SUMS` SHA-256=`9a22f17f9428164171723ab56e0e7de4822ddb4d3cff0b7391d108b0ee431119`。
- 公开镜像 `image_e2e=pass`：arena-154 从正式标签独立回拉并通过项目标准 E2E，日志 SHA-256=`d6918030f72af42d7c376ceb7a1f902705a8473b11ff4ccacb39706e81322eb9`。
- `kejilion/apps` / `kejilion.sh` 契约结论：`packaging/kejilion-app/kpanel.conf` 相对 v0.94.2 无变化，154 apps 仓库 clean、配置 SHA-256=`82f06ca32ce827ef8d0c9c72e65eed9180841a23cbc507237072b58a0807ef04`，无需制造 apps 空提交；受管脚本契约由 L3 验证通过。

## 生产部署安全核对

- 生产目标和部署授权范围：仅 `arena-154`，执行停写备份、标准应用市场更新和最小健康/数据/资源核对。
- 验证/灰度环境：`arena-154`，来源为 `environment-policy.json` 的 candidate-validation 和 production-safety-check。
- 正式部署环境：`arena-154`，来源为 production-deploy。
- `prod-108`：禁用全部 KPanel 操作；确认本次未连接、未备份、未部署、未升级、未核对。
- 部署前版本、健康、备份位置及摘要：v0.94.2，Panel healthy、Agent active、restart=0、OOM=false；备份 `/root/kpanel-backups/pre-v0.95.0-20260823T172850Z`，`SHA256SUMS` SHA-256=`522487f0113d9668d89b819757a0deaddcc58914d7fb581c8da6f478e88090b2`。
- 部署命令/入口：`KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update bash /home/docker/kpanel/bin/kejilion.sh app kpanel`，标准入口一次成功；更新日志 SHA-256=`78918bd858b428727f5037a3adca4e18d8b786bbc730325a603e94104c8b1ec9`。
- 部署后版本、Panel/Agent 状态、重启、日志、数据完整性和公网入口：Panel/Agent 0.95.0，healthy/active、restart=0、OOM=false；日志无 panic/fatal/OOM；`.env`、`panel-state.json` 和 apps 配置逐字节不变；21 个 JSON、1 个非空 SQLite 生产检查通过；公网 health 为 0.95.0，12 个图标资产均 HTTP 200。
- 生产已执行写操作：停写一致性备份、旧版原位恢复核验、标准 KPanel 更新和服务重建；没有修改用户文件或业务设置。
- 仅在隔离真机执行、未在生产执行的场景：桌面图标交互、主题、手机视口和窗口/任务栏视觉验收在精确候选执行；生产只做静态资产、版本、健康、数据、日志和资源核对。

## 回滚

- 源码/tag：`v0.94.2` / `00ffb9fb50d055c75bc7dfc5df7f72ccf3bddde0`。
- 镜像 digest：`sha256:548ed3c20a685298d71da62d4db8a63cf07bdf8e4f0b090d7448e280722d2149`。
- 数据/配置备份：`/root/kpanel-backups/pre-v0.95.0-20260823T172850Z`，含完整 `/home/docker/kpanel`、Compose、`.env`、Agent unit/二进制/脚本、数据、apps 配置和旧 OCI；独立解包、SQLite/JSON/Compose、关键文件 `cmp` 和 `docker load` 均通过。
- 回滚步骤和回滚后复核：停写；校验 `SHA256SUMS`；加载旧 OCI；成套恢复完整 KPanel 目录、apps 配置和 Agent unit；`systemctl daemon-reload`；启动 Agent/Panel；复核 v0.94.2、digest、health、restart/OOM、Compose、SQLite/JSON 和公网入口。禁止只换镜像。
- 回滚后生产实际版本与健康状态：未执行回滚；当前 v0.95.0 healthy/active。
- GitHub Latest、Docker `latest` 与标准更新入口实际指向：v0.95.0 / `sha256:c9f9ec7588e6760065614f9aa436273f4b2764dae1a1e465e2acb3ebcaacf0ad`。
- 公共默认更新通道决策：不适用；本版验收通过并保持 v0.95.0。

## 交付节奏数据

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-08-24T00:50:44+08:00
- 候选冻结时间：2026-08-24T00:51:24+08:00
- 生产完成时间：2026-08-24T01:29:49+08:00
- 提交到生产用时：0.65 小时
- 是否回滚、紧急热修复或重复发布：否
- 若发生失败，发现时间、恢复时间和逃逸门禁：不适用
<!-- kpanel-release-metrics:end -->

<!-- kpanel-release-process-metrics:start -->
- 已记录发布流程异常或无效证据拦截次数：5
- 其中生产写操作开始后异常次数：0
<!-- kpanel-release-process-metrics:end -->

### 流程异常明细

<!-- kpanel-release-process-incidents:start -->
[
  {
    "fingerprint": "candidate-integration/local-worktree/wrong-working-directory",
    "position": "before-production-write",
    "count": 1,
    "impact": "首次 cherry-pick 在共享本地 main 形成仅本机重复提交；远端、候选和生产未触碰。",
    "recoveryEvidence": "从最新 origin/main 建立独立候选并精确重放为 1c6a58e；发布 SHA e84651f 与远端 main、Tag、OCI 一致。",
    "permanentAction": "后续集成命令先同时断言 pwd、git rev-parse --show-toplevel 和目标分支，再允许 cherry-pick。",
    "historicalReleases": []
  },
  {
    "fingerprint": "candidate-local-gate/git-bash/missing-go-runtime",
    "position": "before-production-write",
    "count": 1,
    "impact": "本机 Git Bash verify-change 因没有 go 命令退出，未形成有效完整门禁；候选和远端未改变。",
    "recoveryEvidence": "同一冻结 SHA 在固定 Runner 完整 L3 exit 0，包含 Go 全量、race、vet 和双架构构建。",
    "permanentAction": "Windows 本机仅做前端定向检查；权威 Go/L3 固定使用 kpanel-release-gate Runner。",
    "historicalReleases": []
  },
  {
    "fingerprint": "candidate-l3/prepare-entry/wrong-full-sha",
    "position": "before-production-write",
    "count": 1,
    "impact": "第一次 L3 prepare 输入了猜测的完整 SHA，在打包和远端执行前由候选身份门禁拒绝。",
    "recoveryEvidence": "重新读取 git rev-parse HEAD 后以 e84651f... 启动 r2；status=passed、exit_code=0。",
    "permanentAction": "L3 参数只允许直接读取 git rev-parse HEAD，不手工补全短 SHA。",
    "historicalReleases": []
  },
  {
    "fingerprint": "candidate-tag-fetch/local-repository/stale-tag-ref",
    "position": "before-production-write",
    "count": 1,
    "impact": "全量 Tag fetch 再次发现本地旧 v0.86.2 与远端同名 Tag 不一致并拒绝覆盖；v0.95.0 Tag 创建与推送未受影响。",
    "recoveryEvidence": "远端 v0.95.0 annotated Tag peel 精确为 e84651f，Release workflow completed/success。",
    "permanentAction": "该指纹已在滚动 5 个版本内重复；下一次 L3 前必须把 Tag 预检固定为远端目标 Tag 的 ls-remote/peel 核对，禁止全量 fetch --tags 进入发布主路径。",
    "historicalReleases": ["v0.94.2"]
  },
  {
    "fingerprint": "public-image-identity/powershell-ssh/cross-shell-command-substitution",
    "position": "before-production-write",
    "count": 1,
    "impact": "首次公开镜像身份核对命令被 PowerShell 提前解释远端命令替换，未执行有效远端检查；生产未触碰。",
    "recoveryEvidence": "改为 literal 单引号远端脚本后，正式 OCI version/revision/digest 检查与 image_e2e 均通过。",
    "permanentAction": "跨 PowerShell/SSH 的多行验证改用版本化脚本或 literal 远端脚本，不在双引号中传递命令替换。",
    "historicalReleases": []
  }
]
<!-- kpanel-release-process-incidents:end -->

## 遗留风险与后续准入

- 未验证风险：未单独截图 125%/200% 缩放，未逐个打开全部 12 个业务页面；资产加载、桌面/手机布局、窗口/任务栏复用和生产静态 URL 已覆盖。
- 已实现待实机准入：不适用；本版受影响的静态资产和呈现链路已在精确候选与生产核对。
- 不阻断本版的理由：不改 API、数据或权限；完整 L3、独立浏览器、正式 OCI、备份和生产健康均绑定同一发布 revision 并通过。
- 后续应进入的自动门禁或专项工作流：下一次 L3 前修复重复出现的全量 Tag fetch 流程指纹；图标资产完整性与失败回退继续由现有 Web 测试覆盖，不创建第二发布真源。
