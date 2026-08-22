# KPanel v0.90.3 发布验收记录

日期：2026-08-22

发布级别：L3

候选提交 / 标签：`c8a58677c9633270fac31dbdab2315a07d8b3f6c` / `v0.90.3`

上一稳定版本 / 回滚点：`v0.90.2` / `sha256:1e2b387b0d06450c74086fb88c905f43fdeca600cff1e55bd1c915cff695c4e3`

## 发布范围

- 终端主机列表滚动条与深色终端主题统一。
- 刷新主机时保留旧列表，移除大块“正在读取主机”占位，避免列表闪烁。
- 主机列表和收起窄栏按最新遥测显示真实发行版图标，未知系统回退通用 Linux 图标。
- 调整连接元数据与状态字号，兼顾可读性、列表密度和窄窗口布局。
- 清理废弃中英繁文案并补充布局回归测试。
- 本版不改变数据库、API、Compose、Agent 权限、受管脚本固定版本或应用市场配置；未连接、测试或部署 108。

## 多维质量结论

| 维度 | 状态 | 证据 | 未验证风险或不适用依据 |
| --- | --- | --- | --- |
| 业务正确性与双端互通 | 已验证 | Go 全包、Web 107 文件/839 测试、arena-154 真实 Agent 遥测、终端主机刷新与折叠栏验收 | 生产未使用管理员凭据重复登录；业务旅程已在同主机隔离候选完成 |
| 网络入侵与供应链安全 | 已验证 | govulncheck 可达漏洞 0、npm audit 0、Trivy source/image 0、OCI revision 与受管脚本摘要固定 | 无新增依赖 |
| 稳定性、失败恢复与兼容 | 已验证 | race、双架构、应用生命周期、公开 OCI E2E、停写备份与旧版本恢复核验 | 无协议或数据迁移 |
| 性能与资源预算 | 已验证 | 生产 3 次采样 CPU 0.02%～0.03%、内存 72.69～72.70 MiB/256 MiB、7 PIDs | 纯 UI 补丁，不改变后台资源模型 |
| 用户体验与可访问性 | 已验证 | 桌面/390px、深浅主题、键盘打开、刷新不闪烁、横向溢出 0、发行版图标实际渲染 | 生产公网登录页只读复核，无横向溢出 |
| 数据、配置与迁移 | 已验证 | SQLite integrity=ok、20 个 JSON 可解析、Compose config 通过 | 无迁移 |

## 自动门禁与隔离验收

- 定向测试 3 文件/43 项通过；最终 Web 107 文件/839 项通过；typecheck、i18n 2414 条、生产构建通过。
- Git bundle：`kpanel-v0.90.3-c8a5867.bundle`，SHA-256=`f868fc65edfa9d04f35aee6b6b034f6ea380bd2d883d72af56d3e7e5660123f5`。
- arena-154 完整 `make verify-release` exit 0；日志 `/root/kpanel-release-evidence/v0.90.3-c8a5867/l3-verify-release.log`，SHA-256=`226269a0f989d1bee283ea6031d785cca2b3e67dea2e5ac81be936e2f2df8dd8`。
- 候选 CI `32541454653`、候选依赖新鲜度 `32541454652`、main CI `32541617899`、main 依赖新鲜度 `32541617896` 均 success，head SHA 精确匹配 `c8a5867...`。
- arena-154 隔离候选镜像 ID=`sha256:4a6bd595ef02b31bd9dfdc250c8d5c96595495684bd5f9f4c5a5820f8fb97ca6`；真实 Agent 遥测下 Debian 图标、刷新保留旧列表、13px 元数据、12px 状态、主题滚动条、折叠窄栏和无横向溢出均通过。
- 隔离候选截图：`C:\GitHub\_release-artifacts\v0.90.3\arena154-candidate-terminal.png`；本地深浅主题证据位于 `C:\GitHub\_release-artifacts\v0.90.3\local-preview-r2`。

## 发布产物与公开复核

- GitHub Release：[v0.90.3](https://github.com/kejilion/KPanel/releases/tag/v0.90.3)，非 draft、非 prerelease；Release workflow `32541886444` success。
- Docker `0.90.3` 与 `latest` OCI index 均为 `sha256:621ca9fab03e8c08a427d1b7afcc0ff86d4d49d0e636259f69f7fbf388055b6e`。
- `linux/amd64`=`sha256:51128d5d04d879e09136b777dfc89565a21f907651d875beec88ea791abe1132`；`linux/arm64`=`sha256:95bf0258c8e950ee8c5b3be65269d11e1ef6d91629ef54edbaa6e7e5ec76788e`。
- arena-154 独立公开回拉验证 version=`0.90.3`、revision=`c8a5867...`、非 root 用户、healthcheck、受管脚本和受限容器均通过；日志 SHA-256=`39c5ba3571d07e9e196bb5ea292456e118d65a6c0097d4a0a6122d610096c85e`。
- `packaging/kejilion-app/kpanel.conf` 与 `kejilion/apps@6d86eee24a477320f4d8ffb32d9e85b785cf3c2c` 内容一致，无需 apps 空提交。

## 生产部署与回滚

- 生产目标仅 arena-154；108 未连接、未读取、未测试、未备份、未部署。
- 停写一致性备份：`/root/kpanel-backups/v0.90.3-preupgrade-arena154-20260822T010616Z`；`SHA256SUMS` 文件摘要=`dbac31c6b4d16208495941854154bc00091ce94958b3b0022be154f81896e71f`。
- 备份已独立校验归档摘要、解包、SQLite、JSON、Compose 和旧 OCI，并实际恢复 v0.90.2 healthy 后才升级；备份摘要=`98f409d5887ce70a4a1bdc90cb127771bd228ac4ea114700711fa41c55ff06fc`。
- 通过标准 `kejilion.sh app kpanel` 非交互更新入口升级，日志 SHA-256=`a2325acd3ff47d4c6a51be0cc895a24c97441bb348fe1a7cba36003431fba3d3`。
- 部署后 Panel v0.90.3 healthy、Agent active，restart=0、OOM=false、systemd NRestarts=0；公网 HTTPS health 返回 200/version 0.90.3；SQLite、JSON、Compose 和日志检查通过。
- 回滚必须停写后成套恢复备份中的镜像、Compose、`.env`、数据与 systemd unit；旧 OCI 为 `sha256:1e2b387b0d06450c74086fb88c905f43fdeca600cff1e55bd1c915cff695c4e3`，禁止只换镜像。

## 交付节奏数据

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-08-22T08:03:59+08:00
- 候选冻结时间：2026-08-22T08:09:37+08:00
- 生产完成时间：2026-08-22T09:16:22+08:00
- 提交到生产用时：1.21 小时
- 是否回滚、紧急热修复或重复发布：否
- 若发生失败，发现时间、恢复时间和逃逸门禁：不适用
<!-- kpanel-release-metrics:end -->

<!-- kpanel-release-process-metrics:start -->
- 已记录发布流程异常或无效证据拦截次数：9
- 其中生产写操作开始后异常次数：2
<!-- kpanel-release-process-metrics:end -->

### 流程异常明细

<!-- kpanel-release-process-incidents:start -->
[
  {
    "fingerprint": "local-web-tests/npm-working-directory/incorrect-entrypoint",
    "position": "before-production-write",
    "count": 1,
    "impact": "首次从仓库根目录执行 Web 测试，npm 未找到前端 package.json。",
    "recoveryEvidence": "切换到 web 目录后定向与全量测试均通过。",
    "permanentAction": "Web 门禁固定从 web 工作目录执行。",
    "historicalReleases": []
  },
  {
    "fingerprint": "local-web-preview/browser-control/session-timeout",
    "position": "before-production-write",
    "count": 1,
    "impact": "本地预览取证时浏览器控制会话超时，未形成有效证据。",
    "recoveryEvidence": "重连独立浏览器后完成深浅主题、桌面和窄屏验收。",
    "permanentAction": "浏览器会话超时只废弃该次证据，并从新会话复验受影响旅程。",
    "historicalReleases": []
  },
  {
    "fingerprint": "candidate-preflight/ssh/powershell-variable-expansion",
    "position": "before-production-write",
    "count": 1,
    "impact": "首次远端镜像元数据命令中的变量被本机 PowerShell 提前展开，证据无效。",
    "recoveryEvidence": "改用受限远端脚本和固定参数重新取证成功。",
    "permanentAction": "复杂 SSH 核验不再使用双层内联变量。",
    "historicalReleases": []
  },
  {
    "fingerprint": "candidate-runtime/docker-group-id/quoted-value",
    "position": "before-production-write",
    "count": 1,
    "impact": "候选首次启动把 Docker GID 作为带引号值传入，临时容器停在 Created。",
    "recoveryEvidence": "精确删除该候选容器并以数值 GID 重建，候选 healthy。",
    "permanentAction": "候选启动前校验 GID 为纯数字。",
    "historicalReleases": []
  },
  {
    "fingerprint": "candidate-auth/password-reset/noninteractive-rejected",
    "position": "before-production-write",
    "count": 1,
    "impact": "候选数据副本的非交互密码重置被安全策略拒绝。",
    "recoveryEvidence": "仅对候选副本使用交互 TTY 重置，生产账户未改动。",
    "permanentAction": "候选认证准备固定使用交互 TTY 和隔离数据副本。",
    "historicalReleases": []
  },
  {
    "fingerprint": "main-ci/github-api/powershell-query-interpolation",
    "position": "before-production-write",
    "count": 1,
    "impact": "查询 CI 时 URL 查询参数被 PowerShell 解释，得到一次无效 404。",
    "recoveryEvidence": "改用 gh 的结构化 run 查询，确认精确 head SHA 的 CI 全绿。",
    "permanentAction": "CI 状态统一使用 gh 结构化字段，不拼接查询 URL。",
    "historicalReleases": []
  },
  {
    "fingerprint": "public-oci/buildx-json/control-character-parse",
    "position": "before-production-write",
    "count": 1,
    "impact": "首次解析 buildx 格式化输出时控制字符导致 jq 失败。",
    "recoveryEvidence": "改用 buildx 原生 Digest 字段确认 index 与双架构摘要。",
    "permanentAction": "OCI 摘要取证固定读取结构化 Digest 字段。",
    "historicalReleases": []
  },
  {
    "fingerprint": "preupgrade-backup/verification-wrapper/post-archive-exit",
    "position": "after-production-write",
    "count": 1,
    "impact": "备份包装脚本在核心归档和数据校验后因后置验证包装问题退出 1，恢复 trap 已启动旧版。",
    "recoveryEvidence": "独立验证器重新校验全部摘要、解包数据、Compose 和旧 OCI，并恢复 v0.90.2 healthy。",
    "permanentAction": "备份产物生成与恢复验证拆分为两个受检阶段，后置证据失败继续 fail closed。",
    "historicalReleases": []
  },
  {
    "fingerprint": "production-browser/iab/navigation-timeout",
    "position": "after-production-write",
    "count": 1,
    "impact": "生产页面首次浏览器导航超时，控制会话重置，未影响服务。",
    "recoveryEvidence": "重连后登录页 DOM 完整、readyState=complete、1280px 横向溢出为 0；健康 API 与候选真实终端旅程均已通过。",
    "permanentAction": "生产浏览器复核先使用健康接口确认服务，再以新会话采集页面证据。",
    "historicalReleases": []
  }
]
<!-- kpanel-release-process-incidents:end -->

## 遗留风险与结论

- 生产未使用管理员凭据进入终端页面，避免修改或暴露生产认证信息；相同源码、相同主机真实 Agent 遥测的隔离候选已完成终端完整旅程。
- 生产公网登录页真实 DOM 渲染完整、页面无横向溢出；健康、数据、日志与资源门禁全绿。
- 本版没有阻断风险，无需回滚；保留 v0.90.2 不可变镜像和已验证停写备份作为回滚点。
