# KPanel v0.94.2 发布验收记录

日期：2026-08-23

发布级别：L3

候选提交 / 标签：`00ffb9fb50d055c75bc7dfc5df7f72ccf3bddde0` / `v0.94.2`

上一稳定版本 / 回滚点：`v0.94.1` / `sha256:e666a8e67680e9a838e752ac1828286fa504a770afd224d439aa6452f38eb62a`

## 发布画像

- 业务域：Files 文件管理器的视频预览兼容性。
- 变更面：仅 Web 展示、播放状态判断、错误提示和文档；不改后端、API、依赖、端口、Compose、数据格式、Agent 权限、`kejilion.sh` 或应用市场契约。
- 受影响用户旅程：在文件管理器中预览浏览器不能解码视频轨、但仍能解码音轨的视频。
- 未变化契约：文件流、Range、认证、数据、Compose、Agent 和受管脚本均未变化。
- 风险等级及理由：低；修复只把 `videoWidth/videoHeight=0` 的音频假成功识别为视频轨不兼容并停止播放，不引入转码或新网络边界。

## 发布范围与未纳入内容

- 用户可见更新：视频元数据/数据/canplay 已触发但没有可渲染画面时，不再显示黑屏并继续播音轨；改为暂停并显示明确的浏览器视频轨不兼容提示，建议转换为 H.264 + AAC 或使用系统播放器。
- 精确提交清单：`28f2e1cec20e2ea12cfcc97ea9136daf4cc98fa5`（视频轨兼容修复）和 `00ffb9fb50d055c75bc7dfc5df7f72ccf3bddde0`（0.94.2 版本、CHANGELOG 与回滚说明）。
- 明确未纳入：不做服务端转码，不修改文件流后端，不夹带其他工作树或正在开发的功能。

## 多维质量结论

| 维度 | 状态 | 证据 | 未验证风险或不适用依据 |
| --- | --- | --- | --- |
| 业务正确性与双端互通 | 已验证 | Web 115 文件/920 项；arena-154 精确候选真实 H.264+AAC 正常出画，MPEG-4 Visual+AAC 音轨可解码但视频尺寸 0×0 时准确停止并提示 | 不为浏览器新增编解码器 |
| 网络入侵与供应链安全 | 已验证 | govulncheck 可达漏洞 0、npm audit 0、Trivy source/image 0、正式 OCI revision/version 固定 | 无新增网络/API/权限边界 |
| 稳定性、失败恢复与兼容 | 已验证 | 完整 L3、候选/main/Tag CI、公开 OCI E2E、停写备份和标准更新事务通过 | 不支持的视频仍需转换或外部打开 |
| 性能与资源预算 | 已验证 | 生产 5 次采样 CPU 0.02%～0.04%、内存 73.35 MiB/256 MiB、7 PIDs，restart=0、OOM=false | 无后台任务或转码成本 |
| 用户体验与可访问性 | 已验证 | Chrome 151；桌面和 390×844；错误标题、恢复建议、系统播放器入口清晰，无错误重试按钮；390px 无横向溢出 | 未覆盖每一种厂商私有视频编码 |
| 数据、配置与迁移 | 已验证 | `.env`、`panel-state.json`、apps 配置升级前后不变；21 个 JSON、2 个 SQLite 备份恢复检查通过 | 本版无 schema 或配置迁移 |

## 自动门禁

- 定向测试及结果：Files 视频兼容回归通过；完整 Web 115 文件/920 项、i18n 2468/21、typecheck 和 production build 通过。
- `make verify-release` 环境和结果：固定 Linux Runner 内完整 L3 exit 0；Go 全包、核心 race、vet、Linux amd64/arm64、Web、安装安全、应用生命周期、安全扫描和镜像契约全部通过。
- L3 外层入口：`v0.94.2-00ffb9f-l3-r1`；bundle `kpanel-v0.94.2-00ffb9f-l3-r1.bundle` SHA-256=`b69addacfb6cb6d6bd217909dc8fd603c53224ef09298e5b52e2517719593615`；Runner `kpanel-release-gate:go1.26.6-node24` / `sha256:b593c0ffe32e80a6a6ae8fcac38ed916587dbf698a6b6a55fa33887298737148`；L3 日志 SHA-256=`4537bc5463480ee3cefe4c435188d8d781bc77e3c3ca23ac90cde46e684cff37`；远端证据 `/root/kpanel-release-evidence/v0.94.2-00ffb9f-l3-r1`。
- 候选 CI：CI `32646745062`、Dependency freshness `32646745061`，均 completed/success，精确绑定 `00ffb9f...`。
- 主线 CI：CI `32647019147`、Dependency freshness `32647019106`，均 completed/success，精确绑定 `00ffb9f...`。
- Release workflow：`32647281654` completed/success；Tag Dependency freshness `32647281612` completed/success。
- 安全扫描、镜像契约、SBOM/provenance：Trivy source/image、govulncheck、npm audit 全绿；正式 amd64/arm64 OCI 包含 SBOM/provenance attestations，revision=`00ffb9f...`、version=`0.94.2`。

## 依赖与技术栈变化

- `make dependency-report` 检测源完整性：候选、main 和 Tag 的 Dependency freshness 均成功；本版没有依赖差异。
- 最近每日安全通告审计、EOL 复核：由上述同 SHA GitHub 门禁覆盖并通过。
- 本版采用的依赖、工具链、基础镜像、Action、扫描器或受管脚本候选：全部沿用 v0.94.1 已固定版本。
- 版本/锁文件/Action SHA/镜像 digest/脚本提交与摘要：仅 VERSION 和 npm 包版本更新到 0.94.2；锁文件无依赖变化，受管脚本契约不变。
- 暂缓或拒绝候选：无依赖候选；服务端转码明确不在本补丁范围。
- 升级后的兼容、安全、构建、性能资源和回滚结论：均通过；旧版成套备份和 OCI 已验证可恢复。

## 隔离真机与浏览器验收

- 主机/发行版/架构/运行时版本：`arena-154`、Linux amd64、Docker；Google Chrome 151.0.0.0 独立临时 Profile。
- 环境策略 ID 与允许用途：`arena-154` / candidate-validation、production-deploy、production-safety-check；`prod-108` 不在允许目标内。
- 使用的精确候选：源码 `00ffb9f...`；隔离镜像 `sha256:48801f97225732ef824be362899d2d68ed527b70f81d06625d87114f87d23e83`，revision/version 精确匹配。
- 后台作业终态：隔离 Panel/Agent healthy/active，restart=0、OOM=false；浏览器报告 `passed=true`，SHA-256=`9261d64991f5961e45dcc8230c12c07a35d1f8b9230f2c7e700b32691e031c5e`；证据位于 `C:\GitHub\_release-artifacts\v0.94.2-00ffb9f-l3-r1\video-browser`。
- 测试窗口/循环数及风险依据：确定性 UI/媒体状态补丁采用单次 H.264 正常链路、单次不兼容链路和移动端复核；无后台任务，长 soak 不适用。
- 受影响用户旅程：H.264+AAC 为 640×360 且播放时间前进；MPEG-4 Visual+AAC 为 0×0、自动暂停并显示精确错误与 H.264+AAC 建议；390px `clientWidth=scrollWidth=390`；blocking console errors=0。
- 宿主机写入、失败注入、重启恢复和回滚结果：候选仅写隔离临时目录，结束后容器、网络、Agent、端口和凭据全部清理；生产回滚材料单独验证。
- 未执行场景及原因：未穷举所有私有编码；本补丁目标是可靠识别无法渲染的视频轨，不是增加浏览器解码能力。

## 发布产物与公开仓库复核

- GitHub Release：[v0.94.2](https://github.com/kejilion/KPanel/releases/tag/v0.94.2)。
- Docker 版本 `0.94.2` 与 `latest` OCI index：`sha256:548ed3c20a685298d71da62d4db8a63cf07bdf8e4f0b090d7448e280722d2149`，两者一致。
- `linux/amd64`=`sha256:aeff77514d77049c838046f7fe50a9f77cf1adc0b0d3ab06fcc75e18774075a9`；`linux/arm64`=`sha256:98d5cecefd31ec325a8d91b6ec90bb977c998378782287a1ee4bdd20d0c1294a`；额外 unknown/unknown 为 attestations。
- 附件及 `SHA256SUMS`：8 个公开附件均可下载；5 个二进制/部署包逐项校验通过，`SHA256SUMS` SHA-256=`8a712f5e0c0cca61e944d38c8bddeace1c39799b3abef7d91b7fa5a0962a3f32`。
- 公开镜像按正式标签回拉，version/revision/digest 和项目标准 `image_e2e=pass`；日志 SHA-256=`d6918030f72af42d7c376ceb7a1f902705a8473b11ff4ccacb39706e81322eb9`。
- `kejilion/apps` / `kejilion.sh` 契约结论：`packaging/kejilion-app/kpanel.conf` 未变化，154 apps 配置 SHA-256=`82f06ca32ce827ef8d0c9c72e65eed9180841a23cbc507237072b58a0807ef04`，无需 apps 空提交；受管脚本归一化后与 OCI 逐字节一致。

## 生产部署安全核对

- 生产目标和部署授权范围：仅 `arena-154`，执行停写备份、标准应用市场更新和最小健康/数据/资源核对。
- 验证/灰度环境：`arena-154`，来源为 `environment-policy.json` 的 candidate-validation 和 production-safety-check。
- 正式部署环境：`arena-154`，来源为 production-deploy。
- `prod-108`：禁用全部 KPanel 操作；本次未连接、未读取、未测试、未备份、未部署、未升级、未核对。
- 部署前版本、健康、备份位置及摘要：v0.94.1，Panel healthy、Agent active、restart=0、OOM=false；备份 `/root/kpanel-backups/pre-v0.94.2-20260823T151350Z`，`SHA256SUMS` SHA-256=`d9f51d772cda25a671d3f86df88a81c0b312e6c9da1af31a2bfea8f9db9ab94f`。
- 部署命令/入口：`KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update bash /home/docker/kpanel/bin/kejilion.sh app kpanel`，标准入口一次成功；更新日志 SHA-256=`f127db99c84d7c521acddee9fd8b8d5029e18fcba255b12811f4bdebf9b90dfa6`。
- 部署后版本、Panel/Agent 状态、重启、日志、数据完整性和公网入口：Panel/Agent 0.94.2，healthy/active、restart=0、OOM=false；日志无 panic/fatal/OOM；`.env` 和 `panel-state.json` 逐字节不变；Compose config 通过，无 `.new`/`.rollback` 残留。
- 生产已执行写操作：停写一致性备份、旧版原位恢复核验、标准 KPanel 更新和服务重建；没有修改用户文件或业务设置。
- 仅在隔离真机执行、未在生产执行的场景：真实视频样本上传、浏览器媒体播放和不兼容编码测试仅在隔离候选执行，生产没有注入测试数据。

## 回滚

- 源码/tag：`v0.94.1` / `cdaf6b82d9c2f1169f0411ee0d783fbff491bdc4`。
- 镜像 digest：`sha256:e666a8e67680e9a838e752ac1828286fa504a770afd224d439aa6452f38eb62a`。
- 数据/配置备份：`/root/kpanel-backups/pre-v0.94.2-20260823T151350Z`，含完整 `/home/docker/kpanel`、Compose、`.env`、Agent unit/二进制/脚本、数据、apps 配置和旧 OCI；独立解包、SQLite/JSON/Compose 和 `docker load` 均通过。
- 回滚步骤和回滚后复核：停写；加载旧 OCI；成套恢复完整 KPanel 目录和 Agent unit；`systemctl daemon-reload`；启动 Agent/Panel；复核 v0.94.1、digest、health、restart/OOM、Compose、SQLite/JSON 和公网入口。禁止只换镜像。
- 回滚后生产实际版本与健康状态：未执行回滚；当前 v0.94.2 healthy/active。
- GitHub Latest、Docker `latest` 与标准更新入口实际指向：v0.94.2 / `sha256:548ed3c...`。
- 公共默认更新通道决策：不适用；本版验收通过并保持 v0.94.2。

## 交付节奏数据

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-08-23T22:25:40+08:00
- 候选冻结时间：2026-08-23T22:26:13+08:00
- 生产完成时间：2026-08-23T23:16:11+08:00
- 提交到生产用时：0.84 小时
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
    "fingerprint": "candidate-browser-fixture/manual-ssh-command/cross-shell-argument-transport",
    "position": "before-production-write",
    "count": 3,
    "impact": "隔离视频夹具的 Windows 路径、PowerShell 命令替换和 CRLF 传输三次在候选启动前被拦截；生产未触碰。",
    "recoveryEvidence": "改用 Git Bash 正确挂载路径与远端 literal 脚本后，精确候选 fixture healthy，最终测试和清理均通过。",
    "permanentAction": "后续媒体门禁使用仓库化的固定远端命令规格，不再拼接 PowerShell/SSH 多层字符串。",
    "historicalReleases": []
  },
  {
    "fingerprint": "candidate-browser-gate/external-playwright-fixture/test-assertion-classification",
    "position": "before-production-write",
    "count": 4,
    "impact": "networkidle、播放采样时点、隔离 HTTP 静态资源噪声和未登记应用更新 409 使四次证据脚本返回失败；产品核心断言没有失败。",
    "recoveryEvidence": "最终同一候选采用 DOM/media 状态等待和精确 URL/status 分类后 passed=true，blocking console errors=0。",
    "permanentAction": "后续把确定性的媒体状态等待和隔离夹具噪声分类固化到受版本控制的浏览器门禁。",
    "historicalReleases": []
  },
  {
    "fingerprint": "candidate-tag-fetch/local-repository/stale-tag-ref",
    "position": "before-production-write",
    "count": 1,
    "impact": "Tag 前 fetch 发现本地旧 v0.86.2 与远端同名 Tag 不一致并拒绝覆盖；v0.94.2 创建与推送未受影响。",
    "recoveryEvidence": "远端 v0.94.2 tag object 和 peel commit 单独核对为精确 00ffb9f，Release workflow 成功。",
    "permanentAction": "发布 worktree 后续只 fetch 目标 base/tag，并在预检中报告无关本地旧 Tag，不把全量 tags fetch 与当前 Tag 创建串联。",
    "historicalReleases": []
  },
  {
    "fingerprint": "production-backup/preflight/assumed-state-file",
    "position": "before-production-write",
    "count": 1,
    "impact": "首个备份脚本假定 panel-state.json 位于 data 根目录，在停止服务前即失败；只留下不作为回滚点的审计目录。",
    "recoveryEvidence": "按实际 data/panel/panel-state.json 修正后，新备份一次通过，旧版恢复 healthy，再执行标准升级。",
    "permanentAction": "备份入口应从 Compose 数据根与实际状态目录枚举关键文件，不硬编码旧布局。",
    "historicalReleases": []
  },
  {
    "fingerprint": "public-release-assets/local-download/github-connectivity-timeout",
    "position": "after-production-write",
    "count": 1,
    "impact": "本机下载公开 SHA256SUMS 时 github.com:443 超时；正式 Release、OCI 与生产服务均正常。",
    "recoveryEvidence": "arena-154 从公开 Release 独立下载全部 8 个附件，5 个二进制/部署包 SHA256SUMS 全部通过。",
    "permanentAction": "公开资产验收保留本地和 arena-154 两条独立下载通道；2026-08-30 复核本机 GitHub 连通性后退出该临时例外。",
    "historicalReleases": []
  }
]
<!-- kpanel-release-process-incidents:end -->

## 遗留风险与后续准入

- 未验证风险：未穷举每种视频编码与浏览器组合；不兼容编码仍需转换或系统播放器。
- 已实现待实机准入：不适用；本补丁的正常与失败链路均已在 arena-154 的真实 Chrome 验证。
- 不阻断本版的理由：没有新增媒体解码或服务端转码能力；完整 L3、真实浏览器、正式 OCI、备份和生产健康均绑定精确发布 revision 并通过。
- 后续应进入的自动门禁或专项工作流：把本次媒体状态浏览器测试和稳定的隔离夹具噪声分类沉淀到项目门禁；本版沿用现有 release-kpanel v2.9，未创建第二发布真源。
