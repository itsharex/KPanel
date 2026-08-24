# KPanel v0.95.2 发布验收记录

日期：2026-08-24

发布级别：L3

候选提交 / 标签：`19c59eaa4a140e2c538f2b86f92c99cfad64defb` / `v0.95.2`

上一稳定版本 / 回滚点：`v0.95.1` / `sha256:c0f969319d68f5860dcc053d87f452ebf403282da046268a42e447665ca98bcb`

## 发布画像

- 业务域：桌面文件与目录快捷方式图标。
- 变更面：Web 展示组件、样式、视觉契约测试和版本说明；不涉及 API、数据、权限、Agent、Compose、端口或 `kejilion.sh` 契约。
- 受影响用户旅程：桌面文件/目录快捷方式呈现、选择态、跨面板传输状态提示。
- 最终产品决策：彻底移除文件与目录快捷方式右下角冗余 Link2 角标；保留右上角具有业务含义的 `.desktop__icon-transfer-badge`。
- 风险等级及理由：低；删除冗余视觉节点和专属 CSS，不改变快捷方式或传输状态机。

## 发布范围与未纳入内容

- 精确产品提交：`aa9c98c5d5cd5e66ac27a5743f8f55725029c8f3`、`56f33cf2453240a11b9c1b6dd7ba54447a4ddeab`；版本冻结提交 `19c59eaa4a140e2c538f2b86f92c99cfad64defb`。
- 最终净结果：`DesktopShortcutArtwork` 不再导入或渲染 Link2，所有 `desktop__shortcut-link-badge` CSS 与对应视觉契约已删除；文件和目录 artwork、选中态及传输状态角标保持正常。
- `v0.95.1` 已部署且服务健康，但用户生产截图发现目录角标与文件角标不一致，随后又确认右下 Link2 与右上传输状态形成双角标冲突；其补充视觉验收结论判定失败并由本版取代。
- 旧候选 `4ff1a02...` 及其 L3 因产品决策和产品 SHA 改变而失效，未用于本次发布。
- 未纳入：其他工作树、未提交草稿、无关功能和 108 环境。

## 多维质量结论

| 维度 | 状态 | 证据 | 未验证风险或不适用依据 |
| --- | --- | --- | --- |
| 业务正确性 | 已验证 | 文件与目录 artwork 正常；冗余 Link2 数量 0；选中文件的传输角标数量 1、opacity=1 | 不改后端和传输协议 |
| 网络与供应链安全 | 已验证 | govulncheck 可达漏洞 0、npm audit 0、Trivy source/image 0、正式 OCI revision/version 固定 | 无新入口或依赖 |
| 稳定性与兼容 | 已验证 | 完整 L3、候选/main/Tag CI、公开 OCI E2E、停写备份和标准更新通过 | Web 展示补丁，无数据迁移 |
| 性能与资源 | 已验证 | 生产三次采样 CPU 0.02%～0.03%、内存约 73 MiB/256 MiB、8 PIDs，restart=0、OOM=false | 无常驻任务变化，长 soak 不适用 |
| 用户体验与可访问性 | 已验证 | 1440×900 与 390×844 同一候选真实浏览器验收，无冗余角标、无横向溢出、无 warn/error | 125%/200% 未单独截图 |
| 数据与配置 | 已验证 | `.env`、Compose、panel-state、apps 配置摘要升级前后一致；21 个 JSON、1 个 SQLite 完整 | 无 schema 或配置变化 |

## 自动门禁

- 源任务定向 4 文件/65 项通过；最终候选 Web 115 文件/926 项、typecheck、i18n 2468/21、production build 通过。
- canonical L3：`v0.95.2-19c59ea-l3-r1`，exit 0；Go 全包、核心 race、vet、Web、双架构构建、govulncheck、npm audit、Trivy source/image、安装安全、受管脚本和应用生命周期全绿。
- bundle SHA-256=`1bc5b517658e7fe516bb2d4b8d9849c0b4aad5ff20893d8920f864b80b24020d`；L3 日志 SHA-256=`c36c0e0e1fa4d3e3ce6e1c5d06c0472c1b70fd416193cba90edb509dabcd609f`；状态摘要 SHA-256=`3bb5cefb32d790acbb222ae6a1b17a2dd9e38042d9775a724c96ed57c17a7aae`；远端证据 `/root/kpanel-release-evidence/v0.95.2-19c59ea-l3-r1`。
- 候选 CI `32688190437`、Dependency freshness `32688190278`；main CI `32688340269`、Dependency freshness `32688340300`；均 completed/success 且绑定 `19c59eaa...`。
- Release workflow `32688637105`、Tag Dependency freshness `32688637104` 均 completed/success。

## 依赖与技术栈变化

- 本版无依赖、锁文件、Action、基础镜像、扫描器或受管脚本差异。
- 正式 OCI 固定受管脚本 revision=`9fec61b50cc6ef798dfac1edf11c2ec60ca6b0d1`、SHA-256=`54ceb0e72c4c342382500fc35da636fa436c484a12c4766fb9c7f806a23ae8fa`，与镜像内文件一致。
- 版本仅从 0.95.1 更新到 0.95.2；应用市场安装契约不变，无需制造 apps 空提交。

## 隔离真机与浏览器验收

- 主机/运行时：`arena-154` Linux amd64 Docker 用于 L3、公开镜像 E2E 和生产安全核对；本地独立浏览器用于精确候选 UI 验收；108 未连接。
- 证据目录：`C:\GitHub\_release-artifacts\v0.95.2-shortcut-badge-final`。
- 1440×900：文件 `release-notes.md`、目录 `logs` 均渲染；选中文件后 Link2 数量 0，传输角标总数 2，选中项传输角标 1 且 opacity=1；页面 clientWidth=scrollWidth=1440。
- 390×844：Link2 数量 0，传输角标总数 2，选中项 1；document/body/client/scroll width 均为 390。
- 桌面截图 SHA-256=`c29c58c70ae17866bd5bb619f26e3d349316fb71b4e834a0394ac05c69ad264b`；预览和浏览器目标已停止。
- 本版新增的精确旅程弥补 v0.95.1 漏验：候选必须同时包含实际文件、实际目录、选中态及右上传输状态角标。

## 发布产物与公开仓库复核

- GitHub Release：[v0.95.2](https://github.com/kejilion/KPanel/releases/tag/v0.95.2)，非 draft、非 prerelease；Tag peel=`19c59eaa4a140e2c538f2b86f92c99cfad64defb`。
- Docker `0.95.2` 与 `latest` OCI index：`sha256:cfe375d0390d73ddac7f0bab82ba7da48dbf919a9c886c2609cbae1f3fc3b89f`。
- `linux/amd64`=`sha256:6b25303f8e1949f00b6b1ce2c3b5d7adfba2951f5db6fad6da10c3cc188fddcd`；`linux/arm64`=`sha256:eed4471e269786eefb2059f9175d057953e4899d785aa99aacd5398805abcb37`；另含两项 attestations。
- OCI labels：version=`0.95.2`、revision=`19c59eaa...`；arena-154 从正式标签独立回拉，`image_e2e=pass`。
- 公开 Release 有 8 个附件；`SHA256SUMS` 文件 SHA-256=`031a29e5016fd7118e95828a565b5257e2f1f414271728f04500bd203c805ccb`。

## 生产部署安全核对

- 目标仅 `arena-154`；108 未连接、未测试、未备份、未部署。
- 部署前：v0.95.1、Panel healthy、Agent active、restart=0、OOM=false；旧 OCI=`sha256:c0f969319d68f5860dcc053d87f452ebf403282da046268a42e447665ca98bcb`。
- 停写一致性备份：`/root/kpanel-backups/pre-v0.95.2-20260824T041313Z`；`SHA256SUMS` SHA-256=`bdd4303249564e5690e206ac92b481643c9f63dcb595666946f1ff647e283f9d`。完整目录、旧镜像、Compose、Agent unit、21 个 JSON 和 1 个 SQLite 均完成独立恢复校验与 `docker load`。
- 标准更新入口一次成功：`KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update bash /home/docker/kpanel/bin/kejilion.sh app kpanel`；更新日志 SHA-256=`c648d69876b387fccf8427650bc9d20307126298c33d1738bc1318091d2d558d`。
- 部署后：Panel 0.95.2、revision=`19c59eaa...`、Agent active、healthy、restart=0、OOM=false、NRestarts=0；日志无 fatal/panic/OOM。
- 生产 HTTP CSS 中 `desktop__shortcut-link-badge` 已不存在，`desktop__icon-transfer-badge` 仍存在；公网 health 返回 0.95.2。
- `.env`、Compose、panel-state 与 apps 配置摘要前后一致；21 个 JSON 和 1 个非空 SQLite 复核通过；临时 v0952 资源已清理。

## 回滚

- 源码/tag：`v0.95.1` / `4d0d8721e1d5c2c69b609f11213326ed40d4ad5f`。
- 旧镜像：`sha256:c0f969319d68f5860dcc053d87f452ebf403282da046268a42e447665ca98bcb`。
- 备份：`/root/kpanel-backups/pre-v0.95.2-20260824T041313Z`，已完成摘要、独立解包、JSON/SQLite/Compose、关键文件 `cmp` 与旧 OCI 加载验证。
- 回滚步骤：停写；校验 `SHA256SUMS`；加载旧 OCI；成套恢复 KPanel 目录、Compose、`.env`、apps 配置和 Agent unit；daemon-reload 后启动 Agent/Panel；复核 v0.95.1、digest、health、restart/OOM、JSON/SQLite 和公网入口。禁止只换镜像。
- 未执行实际回滚；当前 v0.95.2 healthy/active。v0.95.0 和 v0.95.1 的历史备份与证据均保留。

## 交付节奏数据

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-08-24T11:26:45+08:00
- 候选冻结时间：2026-08-24T11:43:34+08:00
- 生产完成时间：2026-08-24T12:15:21+08:00
- 提交到生产用时：0.81 小时
- 是否回滚、紧急热修复或重复发布：是（v0.95.1 补充视觉验收失败后发布 v0.95.2 热修）
- 若发生失败，发现时间、恢复时间和逃逸门禁：发现时间: 2026-08-24T11:20:00+08:00; 恢复时间: 2026-08-24T12:15:21+08:00; 逃逸门禁: 已逃逸: v0.95.1 候选浏览器验收未同时放置实际目录快捷方式和选中态，双角标冲突由用户生产截图发现
<!-- kpanel-release-metrics:end -->

<!-- kpanel-release-process-metrics:start -->
- 已记录发布流程异常或无效证据拦截次数：6
- 其中生产写操作开始后异常次数：3
<!-- kpanel-release-process-metrics:end -->

### 流程异常明细

<!-- kpanel-release-process-incidents:start -->
[
  {
    "fingerprint": "release-command/local-invocation/guessed-version-checker",
    "position": "before-production-write",
    "count": 1,
    "impact": "误调用不存在的 JavaScript 版本检查器，命令立即失败，候选、远端和生产未改变。",
    "recoveryEvidence": "改用 canonical scripts/check-version-consistency.sh 并通过。",
    "permanentAction": "版本检查固定从 workflow 和仓库脚本入口读取，不猜测同名实现。",
    "historicalReleases": []
  },
  {
    "fingerprint": "github-observability/local-command/powershell-pipeline-parser",
    "position": "before-production-write",
    "count": 1,
    "impact": "Actions 轮询的一次空管道 PowerShell 语法错误，未形成错误 CI 结论。",
    "recoveryEvidence": "改用显式 rows 变量后核对候选、main 和 Tag CI 均 success 且 SHA 一致。",
    "permanentAction": "PowerShell 轮询统一先赋值再过滤，禁止空管道拼接。",
    "historicalReleases": []
  },
  {
    "fingerprint": "public-image-identity/remote-command/go-template-escaping",
    "position": "before-production-write",
    "count": 1,
    "impact": "一次 docker inspect format 引号错误，未接受该次镜像身份输出。",
    "recoveryEvidence": "使用 json labels 核对 version、revision、脚本摘要并完成 public image E2E。",
    "permanentAction": "远端 Docker metadata 统一输出 JSON 后解析，避免跨 shell 模板转义。",
    "historicalReleases": []
  },
  {
    "fingerprint": "production-backup/remote-host/missing-sqlite-cli",
    "position": "after-production-write",
    "count": 1,
    "impact": "停写备份已生成并恢复服务后，宿主缺少 sqlite3 CLI，首次恢复校验未完成；产品升级尚未开始。",
    "recoveryEvidence": "对同一不可变备份使用 Python sqlite3 完成 integrity_check，并通过 JSON、Compose、cmp 和 docker load 后才允许升级。",
    "permanentAction": "生产备份预检同时检测 sqlite3 CLI 和 Python sqlite3，固定可用实现。",
    "historicalReleases": []
  },
  {
    "fingerprint": "production-verification/distroless-image/missing-shell",
    "position": "after-production-write",
    "count": 1,
    "impact": "误尝试在 distroless Panel 容器内执行 sh；该次 CSS 结论被立即作废。",
    "recoveryEvidence": "改从生产 HTTP 获取已部署 CSS，确认 Link2 选择器不存在且 transfer badge 选择器存在。",
    "permanentAction": "镜像内验证先读取 OCI/runtime 契约；distroless 静态资源固定通过 HTTP 或外部 copy 核对。",
    "historicalReleases": []
  },
  {
    "fingerprint": "production-verification/ssh-command/cross-shell-quoting",
    "position": "after-production-write",
    "count": 1,
    "impact": "一次内联 SSH 生产核对发生跨 shell 引号错误，未接受其输出。",
    "recoveryEvidence": "改用版本化临时脚本完成版本、健康、配置、数据、日志和资源核对并通过。",
    "permanentAction": "多步骤生产核对只传输固定脚本，不使用嵌套内联引号。",
    "historicalReleases": []
  }
]
<!-- kpanel-release-process-incidents:end -->

## 遗留风险与后续准入

- 未单独截图 125%/200% 缩放；本版已覆盖实际文件、实际目录、选中态、传输状态角标、桌面/手机视口和生产 CSS。
- v0.95.1 历史验收文档不改写；本记录明确覆盖其关于快捷方式角标的补充视觉结论。
- 后续桌面快捷方式视觉变更必须在同一候选页面同时放置文件和目录，并覆盖未选中、选中及传输状态，避免单元素截图替代完整用户旅程。
