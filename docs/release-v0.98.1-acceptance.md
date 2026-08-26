# KPanel v0.98.1 发布验收记录

日期：2026-08-26

发布级别：L3

产品候选提交 / 标签：`a4848d5a468f258a1e7f2193cadfe0ce8e98043b` / `v0.98.1`

上一稳定版本 / 回滚点：`v0.98.0` / `sha256:23fcaf791c574b774a8eba8c008a03d7f16925950566458c60c70f2d26c71815`

## 发布画像

- 业务域：Docker 管理页桌面窗口布局。
- 变更面：展示层布局；版本元数据与验收记录；无后端、宿主机或协议变更。
- 受影响用户旅程：Docker 编组收起/展开后，桌面窗口内容继续从顶部开始布局；重新展开恢复明细；容器操作区和空态保持可用。
- 未变化契约：API / 数据 / 端口 / Compose / Agent 权限 / `kejilion.sh` / 应用市场。
- 风险等级及理由：低；仅覆盖桌面窗口内容容器的 CSS 对齐规则，并有源代码回归断言。

## 发布范围与未纳入内容

- `b669e0089b4c2122a0d7d26d7bec8ea04cb047de`：桌面窗口内 Docker 页面在编组收起后保持顶部锚定，避免内容区域被拉伸出大块空白。
- `a4848d5a468f258a1e7f2193cadfe0ce8e98043b`：版本 0.98.1 准备、CHANGELOG 与发布元数据。
- 旧分支中的 `739e984` 已由主线等价提交 `092956d` 吸收，本候选没有重复迁移。
- 未纳入其他工作树、未提交草稿、旧候选、应用市场配置变更或 108 环境。

## 多维质量结论

| 维度 | 状态 | 证据 | 未验证风险或不适用依据 |
| --- | --- | --- | --- |
| 业务正确性与双端互通 | 已验证 | 固定 Runner 全量 Go/Web 门禁；DockerView 定向 19 项测试；公开镜像 `image_e2e=pass` | 本版没有新的后端协议或 Agent 业务路径 |
| 网络入侵与供应链安全 | 已验证 | govulncheck 可达漏洞 0、npm audit 0、Trivy source/image/config 0、双架构 OCI revision/version 校验 | 未增加网络入口、权限或依赖 |
| 稳定性、失败恢复与兼容 | 已验证 | L3、候选/main CI、Release、停写备份恢复、标准更新、postdeploy 全通过 | 不适用新增迁移 |
| 性能与资源预算 | 已验证 | 生产快照 CPU 0.04%、内存 74.39 MiB/256 MiB、7 PIDs、restart=0、OOM=false | CSS 对齐规则无长任务或常驻状态 |
| 用户体验与可访问性 | 已验证 | 候选 Chrome 1920×900、1265×700、390×844 截图；布局测试和 `align-content:start` 断言通过 | 390px Docker 宽内容保持既有窄屏行为，本版不改响应式策略 |
| 数据、配置与迁移 | 已验证 | 受保护配置哈希无差异、SQLite quick_check=ok、停写备份逐项校验 | 无数据迁移 |

## 自动门禁

- 候选 L3：`v0.98.1-a4848d5-l3-r1`，exit 0；本地 bundle SHA-256=`8ca873e6c28ea1667b6a6ac1e7c5484d7b54ff249b239b2679420c993535d898`；L3 日志 SHA-256=`ed802814bfdd3acc584852efc624328443fb120cf571c9af376fd3efb9493a`；Runner=`sha256:b593c0ffe32e80a6a6ae8fcac38ed916587dbf698a6b6a55fa33887298737148`；证据目录 `C:\GitHub\_release-artifacts\v0.98.1-a4848d5-l3-r1`。
- 候选 CI `32930003449`、Dependency freshness `32930003426`；main CI `32931369964`、Dependency freshness `32931370046`；均绑定 `a4848d5a468f258a1e7f2193cadfe0ce8e98043b` 且 completed/success。
- Release workflow `32931668081`、Tag Dependency freshness `32931668161`：completed/success；Release 为非 draft、非 prerelease。
- Windows 上缺少 Go/gofmt 的 change-aware 入口按设计停止；Linux 固定 Runner 提供本版权威 L3，未把平台缺口写成通过。

## 依赖与技术栈变化

- 依赖新鲜度：候选与 Tag 依赖检查均成功；`npm ci` 审计 0 vulnerabilities。
- 本版未新增依赖、工具链、基础镜像、Action 或受管脚本变更；镜像由 Tag workflow 从精确提交重建。
- 镜像 revision/version、受管脚本 revision/hash 与 v0.98.0 一致性均由 Release workflow 和公开回拉核对。
- 已知 `glob@10.5.0` deprecation warning 属既有依赖提示，不构成本版漏洞或发布阻断。

## 隔离真机与浏览器验收

- 候选 UI 预览：Chrome 151.0.7922.174，独立临时 profile；证据目录 `C:\GitHub\_release-artifacts\v0.98.1-docker-layout-browser`，预览已停止且进程/端口无残留。
- 取证视口：1920×900、1265×700、390×844。宽桌面下 Docker 页面内容从顶部对齐，空态/工具栏无异常；窄视口保留主线原有布局行为。模拟 API 仅提供空容器态，未将其冒充真实 Docker 数据。
- arena-154 公开 OCI：按 `docker.io/kjlion/kejilion-panel:0.98.1` 回拉并执行固定 `packaging/tests/image-e2e.sh`，输出 `image_e2e=pass`；临时脚本、容器、网络和数据已清理。
- 本版没有在生产执行容器写操作或布局交互；生产验证限于标准升级、健康、资源、日志和数据/配置安全核对。

## 发布产物与公开仓库复核

- GitHub Release：[v0.98.1](https://github.com/kejilion/KPanel/releases/tag/v0.98.1)，workflow `32931668081` 成功；annotated Tag object=`e95de303a8a7cee225750178d61ebfa9441a5555`，peel=`a4848d5a468f258a1e7f2193cadfe0ce8e98043b`。
- Docker `0.98.1` 与 `latest` OCI index：`sha256:be6ffc64c97a37f733c703424858c6f04153e732fd2448a0b19e9ea79fa3e102`。
- `linux/amd64`=`sha256:be80a606171a988dc0a60689602cfb84b88716cdb482728c0aa2757a244e1a77`；`linux/arm64`=`sha256:d710db7badfb9446886b958c79c9032e443697b9d2c5861f45f3798ae1e05cb4`；额外 unknown/unknown 为 provenance/SBOM attestation。
- 公开镜像 labels：version=`0.98.1`、revision=`a4848d5a468f258a1e7f2193cadfe0ce8e98043b`、受管脚本 revision=`9fec61b50cc6ef798dfac1edf11c2ec60ca6b0d1`、脚本 SHA-256=`54ceb0e72c4c342382500fc35da636fa436c484a12c4766fb9c7f806a23ae8fa`。
- Release 附件含四个 amd64/arm64 Agent/Node 二进制、部署归档、`SHA256SUMS`、许可证和第三方声明。
- `packaging/kejilion-app/kpanel.conf` 相对 `v0.98.0` 无差异；按规范无需 apps 提交，`kejilion/sh` 也未变化。

## 生产部署安全核对

- 唯一目标：`arena-154`；108/`prod-108` 未连接、未检查、未测试、未备份、未部署、未升级。
- 当前基线 preflight：`v0981-baseline-r1`，生产版本 `0.98.0`，Panel health `ok`，Agent active/enabled，NeedDaemonReload=no。
- 停写一致性备份：`/root/kpanel-backups/pre-v0.98.1-20260826T050046Z`；备份 `SHA256SUMS` 文件 SHA-256=`90f047ed92de7375d2d5f455408597ac3781ae9cc1b25fd84cb5287d454226c`；归档、旧镜像加载、校验和和恢复健康均通过。
- 标准入口：`KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update bash /home/docker/kpanel/bin/kejilion.sh app kpanel`，更新输出到达 `KPanel 更新完成` 与 `KPANEL_PROGRESS 100`，exit 0。
- 部署后证据：`v0981-postdeploy-r1`，health=`0.98.1`；Panel healthy、Agent active/enabled、NeedDaemonReload=no、restart=0、OOM=false；资源约 74.39 MiB/256 MiB、CPU 0.04%、7 PIDs；近 10 分钟 Panel/Agent 日志无 panic/fatal/OOM；SQLite quick_check=ok，受保护配置逐字节摘要无差异。
- 生产写操作仅包括停写备份阶段的受控 stop/start 与标准应用市场更新；未执行 Docker 管理业务写操作。

## 回滚

- 源码/tag：`v0.98.0`（产品 commit `c4d1b5f4813397ae812b4d3704fad3264f7c5a1e`）。
- 镜像 digest：`sha256:23fcaf791c574b774a8eba8c008a03d7f16925950566458c60c70f2d26c71815`。
- 数据/配置备份：`/root/kpanel-backups/pre-v0.98.1-20260826T050046Z`，包含旧镜像、Compose、`.env`、Agent unit、apps 配置（若存在）和数据归档，并已恢复校验。
- 回滚步骤：停写后使用该备份成套恢复匹配的旧镜像、Compose、`.env`、Agent unit、apps 配置和数据，再用标准应用入口确认 health、Agent active、restart/OOM、SQLite 与保护摘要；不得只换镜像或只改版本字符串。
- 未执行生产回滚；当前正式版本保持 `0.98.1` 健康。
- GitHub Release、Docker `latest` 与应用市场仍指向当前 `0.98.1`；需要回滚时必须按成套恢复流程并重新核对公共默认更新通道。

## 交付节奏数据

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-08-26T12:20:42+08:00
- 候选冻结时间：2026-08-26T12:28:06+08:00
- 生产完成时间：2026-08-26T13:01:47+08:00
- 提交到生产用时：0.68 小时
- 是否回滚、紧急热修复或重复发布：是（发布流程参数纠正；产品未回滚、未紧急热修复、未重复发布）
- 若发生失败，发现时间、恢复时间和逃逸门禁：发现时间：2026-08-26T12:59:31+08:00; 恢复时间：2026-08-26T12:59:36+08:00; 逃逸门禁：未逃逸：固定生产入口在参数校验阶段拒绝了不适用于 preflight 的 revision 参数，未上传或写入生产，随后以正确 baseline preflight 重试并通过。
<!-- kpanel-release-metrics:end -->

<!-- kpanel-release-process-metrics:start -->
- 已记录发布流程异常或无效证据拦截次数：3
- 其中生产写操作开始后异常次数：0
<!-- kpanel-release-process-metrics:end -->

### 流程异常明细

<!-- kpanel-release-process-incidents:start -->
[
  {
    "fingerprint": "release-operator/production-entrypoint/unsupported-help-probe",
    "position": "before-production-write",
    "count": 1,
    "impact": "对固定生产证据入口执行了不支持的 --help 探测，按设计退出，未上传计划或修改服务器。",
    "recoveryEvidence": "随后严格按 release-kpanel v2.10 参数运行 prepare/preflight/backup/postdeploy。",
    "permanentAction": "固定入口参数只从工作流与脚本真源读取，不猜测可选参数。",
    "historicalReleases": []
  },
  {
    "fingerprint": "release-operator/production-entrypoint/preflight-parameter-mismatch",
    "position": "before-production-write",
    "count": 1,
    "impact": "首次 preflight 误带了仅 postdeploy 允许的 expected revision，远端入口在参数校验阶段 fail-closed，未上传或写入生产。",
    "recoveryEvidence": "v0981-baseline-r1 以当前 v0.98.0、无 revision 的正确 preflight 通过；随后 backup/postdeploy 均通过。",
    "permanentAction": "生产证据调用按 phase 逐项核对参数约束，并保留失败 run 作为不可复用证据。",
    "historicalReleases": []
  },
  {
    "fingerprint": "release-operator/oci-inspect/remote-quoting",
    "position": "before-production-write",
    "count": 1,
    "impact": "一次 SSH/PowerShell 内联 Docker inspect 模板转义失败，仅产生无效只读输出。",
    "recoveryEvidence": "改为回传完整 labels JSON 并在本地解析，确认公开镜像 version/revision、用户和摘要均精确匹配。",
    "permanentAction": "跨 PowerShell/SSH 的 OCI label 核对统一采用 JSON 回传，不再内联复杂模板。",
    "historicalReleases": []
  }
]
<!-- kpanel-release-process-incidents:end -->

## 遗留风险与后续准入

- 当前无已知产品 P0/P1/P2；Docker 编组收起状态和本次桌面窗口顶部锚定均由前端状态/布局测试覆盖。
- 390px 截图用于回归记录；Docker 操作区的窄屏横向布局属于既有策略，本版没有扩大其范围，也未将空 mock 数据当作真实 Docker 验收。
- `glob@10.5.0` deprecation warning 为既有依赖提示；不影响安全扫描和本版发布。
- 后续若继续调整 Docker 窄屏操作列或真实 Compose 数据呈现，应重新选择 `visual-composition` 旅程并在隔离真机完成真实数据浏览器验收。
