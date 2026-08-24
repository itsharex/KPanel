# KPanel v0.95.1 发布验收记录

日期：2026-08-24

发布级别：L3

候选提交 / 标签：`4d0d8721e1d5c2c69b609f11213326ed40d4ad5f` / `v0.95.1`

上一稳定版本 / 回滚点：`v0.95.0` / `sha256:c9f9ec7588e6760065614f9aa436273f4b2764dae1a1e465e2acb3ebcaacf0ad`

## 发布画像

- 业务域：桌面工作区原生应用图标、窗口标题栏、任务栏及文件/目录快捷方式。
- 变更面：Web 静态资源与展示组件；不涉及后端写入、协议、数据或部署契约变化。
- 受影响用户旅程：经典桌面识别并打开系统应用、窗口和任务栏切换、文件/目录桌面入口识别。
- 未变化契约：API、持久数据、端口、Compose、Agent 权限、`kejilion.sh` 和应用市场均未变化。
- 风险等级及理由：中低；统一替换 12 个受版本控制的 WebP 图标，并新增 1 个目录快捷方式图标和 Link2 角标组件，不改变操作语义。

## 发布范围与未纳入内容

- 用户可见更新：12 个 KPanel 原生应用改用 `flat-v1` 套装；桌面、窗口标题栏、任务栏复用同一资产；文件与目录快捷方式采用统一家族图形，目录保留独立链接角标。
- 精确提交清单：`2fbb22d43ae1aa24735ed0d24a3b45da34db99a5`（从聚焦源提交 `353fcaf...` 安全重放）和 `4d0d8721e1d5c2c69b609f11213326ed40d4ad5f`（0.95.1 版本、CHANGELOG 与升级说明）。
- 明确未纳入：其他工作树、未提交草稿、旧图标候选和与本轮无关的业务功能均未进入候选；源工作树未删除。

## 多维质量结论

| 维度 | 状态 | 证据 | 未验证风险或不适用依据 |
| --- | --- | --- | --- |
| 业务正确性与双端互通 | 已验证 | 12 个应用图标和目录快捷方式资源存在；桌面、窗口和任务栏复用一致；生产 13 个静态 URL 均 HTTP 200 | 不涉及后端或跨主机协议 |
| 网络入侵与供应链安全 | 已验证 | govulncheck 可达漏洞 0、npm audit 0、Trivy source/image 0、正式 OCI revision/version 固定 | 新内容仅本地静态 WebP 和展示组件，无新网络入口 |
| 稳定性、失败恢复与兼容 | 已验证 | 完整 L3、候选/main/Tag CI、公开 OCI E2E、停写备份和标准更新事务均通过 | 旧浏览器缓存由构建资源更新，不改 workspace schema |
| 性能与资源预算 | 已验证 | 生产三次采样 CPU 0.02%～0.03%、内存 73.91～73.92 MiB/256 MiB、7 PIDs，restart=0、OOM=false | 本版无后台任务或常驻进程变化，长 soak 不适用 |
| 用户体验与可访问性 | 已验证 | 1440×900 暗/亮主题、窗口/任务栏、390×844 手机验收通过；手机 clientWidth=scrollWidth=390；console warn/error=0 | 125%/200% 未单独截图；既有缩放与键盘结构未修改 |
| 数据、配置与迁移 | 已验证 | `.env`、`panel-state.json`、apps 配置升级前后摘要一致；21 个 JSON、2 个 SQLite 备份恢复检查通过 | 本版无 schema 或配置迁移 |

## 自动门禁

- 定向测试及结果：图标相关 4 文件/52 项通过；最终候选完整 Web 115 文件/925 项、i18n 2468/21、typecheck 和 production build 通过。
- `make verify-release` 环境和结果：固定 Linux Runner 内完整 L3 exit 0；Go 全包、核心 race、vet、Linux amd64/arm64、Web、安装安全、应用生命周期、安全扫描和镜像契约全部通过。
- L3 外层入口：`v0.95.1-4d0d872-l3-r1`；bundle SHA-256=`82cf675c8f5554d40ce91af24bb453be8cf77574c01ddd094ea6983d0ba381f3`；Runner `kpanel-release-gate:go1.26.6-node24` / `sha256:b593c0ffe32e80a6a6ae8fcac38ed916587dbf698a6b6a55fa33887298737148`；L3 日志 SHA-256=`fa0b2ac2c946a1a5f52e655eaca231af10422f32e1a0f092395f672ba0d1af96`；status=passed、exit_code=0；远端证据 `/root/kpanel-release-evidence/v0.95.1-4d0d872-l3-r1`。
- 候选 CI：CI `32684760884`、Dependency freshness `32684760880`，均 completed/success，精确绑定 `4d0d872...`。
- 主线 CI：CI `32685034698`、Dependency freshness `32685034721`，均 completed/success，精确绑定 `4d0d872...`。
- Release workflow：`32685325571` completed/success；Tag Dependency freshness `32685325572` completed/success。
- 安全扫描、镜像契约、SBOM/provenance：Trivy source/image、govulncheck、npm audit 全绿；正式 amd64/arm64 OCI 包含 attestations，revision=`4d0d872...`、version=`0.95.1`。

## 依赖与技术栈变化

- `make dependency-report` 检测源完整性：候选、main 和 Tag 的 Dependency freshness 均成功；本版无依赖差异。
- 最近每日安全通告审计、EOL 复核状态及证据：上述同 SHA GitHub 门禁和 L3 安全扫描通过。
- 本版采用的依赖、工具链、基础镜像、Action、扫描器或受管脚本候选：全部沿用 v0.95.0 固定版本。
- 版本/锁文件/Action SHA/镜像 digest/脚本提交与摘要：仅产品版本更新到 0.95.1；锁文件无依赖变化；受管脚本契约不变。
- 暂缓或拒绝候选、证据、负责人、复核日期和退出条件：旧图标候选与无关工作树不纳入；无依赖暂缓项。
- 升级后的兼容、安全、构建、性能资源和回滚结论：均通过；旧版成套备份与 OCI 已验证可恢复。

## 隔离真机与浏览器验收

- 主机/发行版/架构/运行时版本：`arena-154`、Linux amd64、Docker；Codex 独立应用内浏览器，不接管用户 Chrome。
- 环境策略 ID 与允许用途：`arena-154` / candidate-validation、production-deploy、production-safety-check；`prod-108` 不在允许目标内。
- 使用的精确候选或公开产物：源码 `4d0d872...`；浏览器预览来自同一 clean 候选；正式 OCI revision/version 精确一致。
- 后台作业终态：标准 `local-feature-preview` 暗/亮主题预览均已停止；证据目录 `C:\GitHub\_release-artifacts\v0.95.1-icon-browser` 与 `v0.95.1-icon-browser-light`；桌面暗色截图 SHA-256=`a4effe8c4c580d3ff9ab3f1298a81d7afceddd8138334cc5c20505dca207cf2d`，窗口/任务栏=`3df9fc62a1493bcd665d46558fa0deb09f1a4c3a865c62bb4d5cb01d9bdd0620`，手机=`a20ec95dffb3b462d8a15fd3680c575cd30762db5e4c457972cc206b95a0fff6`，浅色桌面=`796b943043a7c40d9421b21942289ef163bbff7937405f4b1afc2e4b2feb1056`。
- 测试窗口/循环数及风险依据：确定性静态资产和 UI 映射采用桌面/手机、明暗主题及窗口/任务栏单次完整旅程；无后台逻辑，长 soak 不适用。
- 受影响用户旅程、视口、缩放、主题和失败态：100% 下 1440×900 与 390×844 通过；暗/亮主题、窗口标题栏和任务栏复用正常；手机无横向溢出；加载失败回退由自动化覆盖。125%/200% 未单独截图。
- 宿主机写入、失败注入、重启恢复和回滚结果：候选只写隔离临时目录，预览已清理；生产回滚材料单独验证。
- 未执行场景及原因：不逐个打开全部 12 个业务页面；本版只改变图标资产与呈现映射，窗口/任务栏复用和失败回退已覆盖。

## 发布产物与公开仓库复核

- GitHub Release：[v0.95.1](https://github.com/kejilion/KPanel/releases/tag/v0.95.1)，非 draft、非 prerelease。
- Docker `0.95.1` 与 `latest` OCI index：`sha256:c0f969319d68f5860dcc053d87f452ebf403282da046268a42e447665ca98bcb`，两者一致。
- `linux/amd64`=`sha256:ae92d55c878efbfb8c8d01e66fda2943e1581756104d0c5d6cadaf429be7ad6d`；`linux/arm64`=`sha256:584ab32030035003e4cff71c07540e0f37d0fae35e0efdfa865b4e9bc49507da`；额外 unknown/unknown 为 attestations。
- 附件及 `SHA256SUMS`：8 个公开附件；`SHA256SUMS` SHA-256=`a939497473fcdf152a855831e4b53f04f58376fc486c2b206a369bcf2e0d2947`。
- 公开镜像 `image_e2e=pass`：arena-154 从正式标签独立回拉并通过项目标准 E2E。
- `kejilion/apps` / `kejilion.sh` 契约结论：`packaging/kejilion-app/kpanel.conf` 相对 v0.95.0 无变化；154 apps 仓库 clean，配置 SHA-256=`82f06ca32ce827ef8d0c9c72e65eed9180841a23cbc507237072b58a0807ef04`，无需制造 apps 空提交；受管脚本契约由 L3 验证通过。

## 生产部署安全核对

- 生产目标和部署授权范围：仅 `arena-154`，执行停写备份、标准应用市场更新和最小健康/数据/资源核对。
- 验证/灰度环境：`arena-154`，来源为 `environment-policy.json` 的 candidate-validation 和 production-safety-check。
- 正式部署环境：`arena-154`，来源为 production-deploy。
- `prod-108`：禁用全部 KPanel 操作；本次未连接、未备份、未部署、未升级、未核对。
- 部署前版本、健康、备份位置及摘要：v0.95.0，Panel healthy、Agent active、restart=0、OOM=false；备份 `/root/kpanel-backups/pre-v0.95.1-20260824T031653Z`，`SHA256SUMS` SHA-256=`6f241eede5974172a2d1afa4ec7533d862789d69f7121dd4ea5fa1d0f264bdf1`。
- 部署命令/入口：`KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update bash /home/docker/kpanel/bin/kejilion.sh app kpanel`，标准入口一次成功；更新日志 SHA-256=`55ba8cb90cc8dac6b022a3de90da373981ba4db0c9f25f5e492e915f7413caaf`。
- 部署后版本、Panel/Agent 状态、重启、日志、数据完整性和公网入口：Panel 0.95.1、Agent active、healthy、restart=0、OOM=false；日志无 panic/fatal/OOM；配置摘要前后一致；21 个 JSON、1 个非空 SQLite 生产检查通过；公网 health 为 0.95.1，13 个新图标资源均 HTTP 200。
- 生产已执行写操作：停写一致性备份、旧版原位恢复核验、标准 KPanel 更新和服务重建；未修改用户文件或业务设置。
- 仅在隔离真机执行、未在生产执行的场景：桌面图标交互、明暗主题、手机视口和窗口/任务栏视觉验收在精确候选执行；生产只做资源、版本、健康、数据、日志和资源核对。

## 回滚

- 源码/tag：`v0.95.0` / `e84651f7e4b530fdbef96d6f9fc94fc887efb45a`。
- 镜像 digest：`sha256:c9f9ec7588e6760065614f9aa436273f4b2764dae1a1e465e2acb3ebcaacf0ad`。
- 数据/配置备份：`/root/kpanel-backups/pre-v0.95.1-20260824T031653Z`，含完整 `/home/docker/kpanel`、Compose、`.env`、Agent unit/二进制/脚本、数据、apps 配置和旧 OCI；独立解包、SQLite/JSON/Compose、关键文件 `cmp` 和 `docker load` 均通过。
- 回滚步骤和回滚后复核：停写；校验 `SHA256SUMS`；加载旧 OCI；成套恢复 KPanel 目录、apps 配置和 Agent unit；`systemctl daemon-reload`；启动 Agent/Panel；复核 v0.95.0、digest、health、restart/OOM、Compose、SQLite/JSON 和公网入口。禁止只换镜像。
- 回滚后生产实际版本与健康状态：未执行回滚；当前 v0.95.1 healthy/active。
- GitHub Latest、Docker `latest` 与标准更新入口实际指向：v0.95.1 / `sha256:c0f969319d68f5860dcc053d87f452ebf403282da046268a42e447665ca98bcb`。
- 公共默认更新通道决策：不适用；本版验收通过并保持 v0.95.1。

## 交付节奏数据

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-08-24T10:42:31+08:00
- 候选冻结时间：2026-08-24T10:43:47+08:00
- 生产完成时间：2026-08-24T11:17:46+08:00
- 提交到生产用时：0.59 小时
- 是否回滚、紧急热修复或重复发布：否
- 若发生失败，发现时间、恢复时间和逃逸门禁：不适用
<!-- kpanel-release-metrics:end -->

<!-- kpanel-release-process-metrics:start -->
- 已记录发布流程异常或无效证据拦截次数：11
- 其中生产写操作开始后异常次数：0
<!-- kpanel-release-process-metrics:end -->

### 流程异常明细

<!-- kpanel-release-process-incidents:start -->
[
  {
    "fingerprint": "candidate-tag-fetch/local-repository/stale-tag-ref",
    "position": "before-production-write",
    "count": 1,
    "impact": "源任务全量 fetch tags 再次发现本地 v0.86.2 冲突并拒绝覆盖；产品、远端和生产未改变。",
    "recoveryEvidence": "正式流程使用 --no-tags 和目标 Tag ls-remote；v0.95.1 peel 精确为 4d0d872，Release success。",
    "permanentAction": "release-kpanel v2.9 已固定 --no-tags；源任务预检也必须复用该唯一入口，不再直接全量 fetch tags。",
    "historicalReleases": ["v0.94.2", "v0.95.0"]
  },
  {
    "fingerprint": "candidate-integration/local-worktree/wrong-working-directory",
    "position": "before-production-write",
    "count": 1,
    "impact": "首次 cherry-pick 在共享本地 main 形成仅本机提交；候选、远端和生产未触碰。",
    "recoveryEvidence": "本地 main 非破坏性恢复到 origin/main；独立候选精确重放为 2fbb22d，check-collaboration-state 6 项测试和写入前检查通过。",
    "permanentAction": "每个远端写阶段强制执行 check-collaboration-state writer，并拒绝 primary worktree 作为候选写入点。",
    "historicalReleases": ["v0.95.0"]
  },
  {
    "fingerprint": "github-observability/local-client/unavailable-auth-or-route",
    "position": "before-production-write",
    "count": 2,
    "impact": "一次直接仓库 URL ls-remote 因 SSH 路由权限失败，gh run list 因本机未认证失败；均未改变远端。",
    "recoveryEvidence": "在仓库内使用 origin 完成精确 ref 核对，并以 GitHub 公共 API 读取候选、main、Tag 的精确运行结论。",
    "permanentAction": "发布观察优先使用仓库 origin 与无写权限的公共 Actions API，不依赖本机 gh 登录状态。",
    "historicalReleases": []
  },
  {
    "fingerprint": "release-command/local-invocation/guessed-interface",
    "position": "before-production-write",
    "count": 3,
    "impact": "版本检查器解释器、L3 help 参数和协作检查参数各有一次错误调用，均在状态写入前失败。",
    "recoveryEvidence": "改用 Git Bash 执行 shell 检查、按工作流参数启动唯一 L3、按脚本 Usage 使用 --role/--base-ref 后全部通过。",
    "permanentAction": "发布命令从工作流渲染和脚本 Usage 读取，不手工猜测解释器或参数。",
    "historicalReleases": []
  },
  {
    "fingerprint": "browser-acceptance/local-preview/missing-or-unsupported-precondition",
    "position": "before-production-write",
    "count": 2,
    "impact": "首次预览缺少候选 node_modules；一次浏览器等待使用不支持的 networkidle，未形成有效视觉证据。",
    "recoveryEvidence": "npm ci 0 vulnerabilities 后重启标准 acceptance 预览，并改用 domcontentloaded 加确定性等待；四张截图与 0 warn/error 绑定同一候选。",
    "permanentAction": "本地视觉验收先执行 Vite 依赖 preflight，并仅使用浏览器运行时文档声明的等待状态。",
    "historicalReleases": []
  },
  {
    "fingerprint": "public-image-identity/local-host/missing-docker-cli",
    "position": "before-production-write",
    "count": 1,
    "impact": "本机尝试检查 OCI 时无 docker 命令，未形成镜像身份结论。",
    "recoveryEvidence": "arena-154 使用 docker buildx 精确核对 0.95.1/latest index、双架构 digest、revision/version，并从公开仓库 image_e2e=pass。",
    "permanentAction": "公开 OCI 身份和 E2E 固定在登记的 arena-154 Docker 环境执行。",
    "historicalReleases": []
  },
  {
    "fingerprint": "production-preflight/powershell-ssh/cross-shell-command-substitution",
    "position": "before-production-write",
    "count": 1,
    "impact": "首次 apps 清洁度命令被 PowerShell 提前解释远端命令替换并产生错误的路径不存在结论；生产未触碰。",
    "recoveryEvidence": "改用 literal 单引号远端脚本后确认 /root/apps clean、HEAD=6d86eee、kpanel.conf 摘要与既有真源一致。",
    "permanentAction": "PowerShell 到 SSH 的多命令生产预检固定使用 literal 远端脚本，不在双引号中传递命令替换。",
    "historicalReleases": ["v0.95.0"]
  }
]
<!-- kpanel-release-process-incidents:end -->

## 遗留风险与后续准入

- 未验证风险：未单独截图 125%/200% 缩放，未逐个打开全部 12 个业务页面；资产加载、明暗主题、桌面/手机布局、窗口/任务栏复用和生产静态 URL 已覆盖。
- 已实现待实机准入：不适用；本版受影响的静态资产和呈现链路已在精确候选与生产核对。
- 不阻断本版的理由：不改 API、数据或权限；完整 L3、独立浏览器、正式 OCI、备份和生产健康均绑定同一发布 revision 并通过。
- 后续应进入的自动门禁或专项工作流：继续强制 `check-collaboration-state` 和工作流内 `--no-tags`；跨 shell 生产预检统一改用 literal/versioned script；图标资产完整性与失败回退由现有 Web 测试持续覆盖。
