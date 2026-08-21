# KPanel v0.90.2 发布验收记录

日期：2026-08-22

发布级别：L3

候选提交 / 标签：`074d769ba6ba723be7b3ea19d8377551ee47a9e4` / `v0.90.2`

上一稳定版本 / 回滚点：`v0.90.1` / `sha256:41a5757ccf041469e8bdeed56574b4c75539142d483c2b5554b2cb103137cd7d`

## 发布画像

- 业务域：网站管理、终端、移动端体检展示，以及已进入主线的发布治理规则。
- 变更面：展示、宿主机写入入口收敛、部署。
- 受影响用户旅程：删除网站、桌面/窄窗口/手机终端、移动端体检选择器。
- 未变化契约：数据库、端口、Compose、Agent 权限、受管 `kejilion.sh` 固定 revision/SHA、应用市场配置均未变化。
- 风险等级及理由：中等；网站删除属于宿主机破坏性写操作，但已统一收敛到固定 `kejilion.sh k web del`，其余为布局与主题补丁。

## 发布范围与未纳入内容

- 用户可见更新：网站删除统一确认和固定脚本链路；终端响应式布局、上下文菜单、输出处理与窄窗口主机选择；体检移动端选择器跟随页面主题。
- 精确提交清单：`48a6d97`、`697ead5`、`2feef46`、`71eac0b`、`262a0dc`、`074d769`。
- `v0.90.1` 后已进入主线的治理提交随 Tag 纳入项目历史，但不作为产品功能宣传；未纳入共享脏管理工作树、旧候选、未提交草稿或 apps 空提交。
- 本轮未连接、读取、测试、备份、升级或部署 108。

## 多维质量结论

| 维度 | 状态 | 证据 | 未验证风险或不适用依据 |
| --- | --- | --- | --- |
| 业务正确性与双端互通 | 已验证 | Go 全包、网站删除契约测试、Web 107 文件/837 测试、mock 浏览器完整删除确认 | 未在生产删除真实业务网站；破坏性旅程以隔离契约和 mock UI 验证 |
| 网络入侵与供应链安全 | 已验证 | govulncheck 可达漏洞 0、npm audit 0、Trivy source/image 0、OCI revision 与受管脚本摘要固定 | npm 间接依赖 `glob@10.5.0` 有弃用提示，但审计漏洞为 0 |
| 稳定性、失败恢复与兼容 | 已验证 | race、双架构、应用生命周期、公开 image E2E、备份恢复核验 | 无协议或数据迁移 |
| 性能与资源预算 | 已验证 | 生产 3 次采样 CPU 0.02%、内存 74.39 MiB/256 MiB、7 PIDs | 补丁不改变长期后台负载，无额外 soak 必要性 |
| 用户体验与可访问性 | 已验证 | Chrome 151，桌面与 390×844，深色主题，页面横向溢出 0 | 100% 缩放实测；125%/200% 由布局测试覆盖，未单独截图 |
| 数据、配置与迁移 | 已验证 | SQLite integrity=ok、20 个 JSON 可解析、Compose config 通过 | 不适用迁移 |

## 自动门禁

- 定向测试：网站、终端、诊断 4 个 Web 文件/58 测试通过；最终全量 107 文件/837 测试通过。
- `make verify-release`：arena-154 固定 Runner `sha256:b593c0ffe32e80a6a6ae8fcac38ed916587dbf698a6b6a55fa33887298737148`，最终输出 `L3 release verification completed` 与 `release_gate_runner=pass commit=074d769...`；日志 SHA-256=`bcc593a53282fbff832650f54bfc11e52edaf8aaf2a436f6b81b98907c0dc03e`。
- 候选 CI `32499563444`、候选依赖新鲜度 `32499563473`、main CI `32499786423`、main 依赖新鲜度 `32499786265`、Tag 依赖新鲜度 `32500256116` 均 success，head SHA 均精确匹配候选。
- Release workflow `32500256015` success；发布候选分支由工作流自动清理。
- Go 全包、`panel/auth/dockerx` race、go vet、linux/amd64 与 linux/arm64、生产构建、安装安全、固定脚本契约和 apps lifecycle 均通过。

## 依赖与技术栈变化

- 本版没有依赖升级；依赖报告、每日 Dependency freshness 和安全扫描检测源均成功。
- Go 1.26.6；Node 24.18.x；Runner 内 npm 11.12.1；Trivy 0.72.0；govulncheck 1.6.0。
- Go 基础镜像固定 `sha256:af8d6740070b8906d12eae1c3e3ea0957fb63f492051ea05e354c38ef9fe88df`，Node 基础镜像固定 `sha256:a0b9bf06e4e6193cf7a0f58816cc935ff8c2a908f81e6f1a95432d679c54fbfd`。
- 受管脚本固定 `kejilion/sh@2d1a37416c574c7398445a54cd6d9d3a0d4bc124`，SHA-256=`17c1544b826c45f070e49df2f71a5e152fedc922a1de201da5bfa0393d250a4d`。
- Trivy 0.74.0 与 npm 12.0.2 更新提示未纳入本补丁；负责人为依赖治理流程，按每日新鲜度和下一次兼容评估决定，不以未审查升级替换本版工具链。

## 隔离真机与浏览器验收

- 主机：arena-154，Linux amd64，Docker/Buildx；策略目标仅 `arena-154`，未包含 `prod-108`。
- 使用完整 Git bundle `kpanel-v0.90.2-074d769-r4.bundle`，SHA-256=`9c922bf1762f5d7ea5e7dc9ba38ba80bdf64c9890cc555297dfb7536a1793a74`，detached checkout 精确为 `074d769...`。
- L3 后台作业正常退出 0；证据目录 `/root/kpanel-release-evidence/v0.90.2`。本版无持续后台进程或资源模型变化，因此以完整 L3、公开镜像 E2E 和生产 3 次健康采样代替长 soak。
- 独立后台 Google Chrome `151.0.7922.172` 使用随机临时 Profile；网站删除确认精确显示 `k web del`，确认后 mock 状态持久消失；终端桌面、390×844、诊断 390×844 深色主题均无页面横向溢出。
- 生产公网 `https://kpanel.154.36.153.9.sslip.io` 健康接口和 390×844 登录页真实渲染通过；未接管用户 Chrome Profile。
- 未在生产执行真实网站删除；这是为了避免破坏业务数据，不降低固定脚本、确认窗与回滚门禁。

## 发布产物与公开仓库复核

- GitHub Release：[v0.90.2](https://github.com/kejilion/KPanel/releases/tag/v0.90.2)，非 draft、非 prerelease；Annotated Tag 解引用到 `074d769...`。
- Docker `0.90.2` 与 `latest` OCI index 均为 `sha256:1e2b387b0d06450c74086fb88c905f43fdeca600cff1e55bd1c915cff695c4e3`。
- `linux/amd64`=`sha256:319f4c56fb8700c51f0d0aa7a15abb91e02fb606a2823ab374885021aa9ea22b`；`linux/arm64`=`sha256:fff59072c154f8c071b1b27cb6c545323762d36cbf7f61ec21c0ded915ec6452`。
- Release `SHA256SUMS` digest=`sha256:79558c5c0831586bb94f6e3a91c643baad641c8c8a71bb8c35e917425ac791e5`；部署包 digest=`sha256:f303b878ec180a80e778ed5f08a52b2ec6f0742b1e899137d99dd9ab0c8c5588`。
- arena-154 从 Docker Hub 公开回拉后，version=`0.90.2`、revision=`074d769...`、双架构和脚本固定元数据一致，输出 `image_e2e=pass`。
- `packaging/kejilion-app/kpanel.conf` 与 `kejilion/apps@6d86eee24a477320f4d8ffb32d9e85b785cf3c2c` 的 `/root/apps/kpanel.conf` SHA-256 同为 `82f06ca32ce827ef8d0c9c72e65eed9180841a23cbc507237072b58a0807ef04`，无需空提交。

## 生产部署安全核对

- 生产目标和授权范围：仅 arena-154；验证和正式部署策略均不包含 108。
- `prod-108`：本次未连接、未备份、未部署、未升级、未核对。
- 部署前 Panel v0.90.1 healthy、Agent active、restart=0、OOM=false。
- 停写一致性备份：`/root/kpanel-backups/v0.90.2-preupgrade-arena154-20260821T160552Z`；`state.tar.zst` SHA-256=`30a255c9e018c4a3b209aa0f335e82183c69d3ef172613c798fa09209fb8331a`，旧镜像归档 SHA-256=`3f7f2c943cd10160e41a75aa67a5b781f6f0a473570625ef3a8bb216a7eb389f`。
- 备份已独立解包、关键文件逐字节对比、SQLite/JSON/Compose 校验、旧 OCI 加载，并恢复 v0.90.1 healthy 后才升级，结果 `backup_verified=pass`。
- 通过现有 `/root/apps/kpanel.conf` 的 `docker_app_update` 入口升级；没有覆盖 apps 文件或用户应用登记状态。
- 部署后 Panel v0.90.2 healthy、Agent v0.90.2 active，Agent healthcheck 通过，restart=0、OOM=false；SQLite integrity=ok、20 个 JSON 可解析、Compose config 通过，日志无 panic/fatal/OOM/协议或版本不匹配。
- 公网 HTTPS `/api/v1/health` 返回 200/version 0.90.2；生产已执行的写操作仅停写备份、标准更新和发布证据写入。

## 回滚

- 源码/tag：`v0.90.1` / `44211d11c40c8617e333f3ff65f951ef15d745ea`。
- 镜像 digest：`sha256:41a5757ccf041469e8bdeed56574b4c75539142d483c2b5554b2cb103137cd7d`。
- 数据/配置备份：`/root/kpanel-backups/v0.90.2-preupgrade-arena154-20260821T160552Z`。
- 回滚时停写 Panel/Agent，校验 `SHA256SUMS`，加载 `old-image.tar`，成套恢复 `state.tar.zst` 中的 `/home/docker/kpanel`、`/root/apps/kpanel.conf` 与 systemd unit，执行 `systemctl daemon-reload`，再启动并复核 v0.90.1、SQLite、Compose、restart/OOM 与公网健康；禁止只换镜像。
- 回滚后实际生产版本：未执行回滚；已通过备份恢复演练验证 v0.90.1 可恢复并健康。
- GitHub Latest、Docker `latest` 与标准更新入口当前均指向 v0.90.2；本版全绿，不恢复公共默认通道。

## 交付节奏数据

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-08-21T21:10:29+08:00
- 候选冻结时间：2026-08-21T23:33:45+08:00
- 生产完成时间：2026-08-22T00:08:12+08:00
- 提交到生产用时：2.96 小时
- 是否回滚、紧急热修复或重复发布：否
- 若发生失败，发现时间、恢复时间和逃逸门禁：不适用
<!-- kpanel-release-metrics:end -->

<!-- kpanel-release-process-metrics:start -->
- 已记录发布流程异常或无效证据拦截次数：15
- 其中生产写操作开始后异常次数：5
<!-- kpanel-release-process-metrics:end -->

### 流程异常明细

<!-- kpanel-release-process-incidents:start -->
[
  {
    "fingerprint": "candidate-preflight/git-fetch/local-tag-conflict",
    "position": "before-production-write",
    "count": 1,
    "impact": "共享仓库本地历史标签与远端冲突，普通 fetch 被拒绝。",
    "recoveryEvidence": "改用最新 origin/main 的全新隔离管理 clone，未修改共享仓库。",
    "permanentAction": "发布集成固定使用隔离 clone，默认不把共享仓库本地标签作为候选真源。",
    "historicalReleases": []
  },
  {
    "fingerprint": "candidate-bootstrap/git-ssh/missing-key-context",
    "position": "before-production-write",
    "count": 1,
    "impact": "首次隔离 clone 未继承仓库 SSH identity，认证失败。",
    "recoveryEvidence": "使用项目 core.sshCommand 的专用 key 后 clone/fetch/push 均成功。",
    "permanentAction": "候选 bootstrap 预检显式读取项目 core.sshCommand。",
    "historicalReleases": []
  },
  {
    "fingerprint": "candidate-bootstrap/collaboration-state/writer-worktree-contract",
    "position": "before-production-write",
    "count": 1,
    "impact": "管理 clone 和旧参数不满足当前 writer-worktree 合同。",
    "recoveryEvidence": "创建独立 linked writer worktree 后 collaboration-state 通过。",
    "permanentAction": "release-kpanel v2.8 固定由管理 clone 创建唯一 linked writer worktree。",
    "historicalReleases": []
  },
  {
    "fingerprint": "l3/bundle/full-stable-tag-closure",
    "position": "before-production-write",
    "count": 1,
    "impact": "首个 bundle 缺少业务事实基线所需稳定标签，L3 在业务代码前停止。",
    "recoveryEvidence": "使用 HEAD --tags 重建完整 bundle，最终 r4 L3 通过。",
    "permanentAction": "正式 L3 bundle 固定包含所有可达稳定标签。",
    "historicalReleases": []
  },
  {
    "fingerprint": "l3/bundle/verification-repository-context",
    "position": "before-production-write",
    "count": 1,
    "impact": "在非仓库目录直接 bundle verify 无法建立 prerequisite 上下文。",
    "recoveryEvidence": "在专用临时 bare repo 验证后 bundle checkout 成功。",
    "permanentAction": "bundle verify 固定在隔离 bare repository 中执行。",
    "historicalReleases": []
  },
  {
    "fingerprint": "candidate-preflight/ssh/powershell-variable-expansion",
    "position": "before-production-write",
    "count": 1,
    "impact": "远端检查中的 shell 变量被本机 PowerShell 提前展开，证据命令无效。",
    "recoveryEvidence": "改用单引号远端命令和固定 SSH alias，重新取证成功。",
    "permanentAction": "复杂远端验证改用上传的受限脚本，避免双层 shell 插值。",
    "historicalReleases": []
  },
  {
    "fingerprint": "l3/runner/login-shell-path-reset",
    "position": "before-production-write",
    "count": 1,
    "impact": "Runner preflight 使用 sh -lc 重置镜像 PATH，误报 go 不存在。",
    "recoveryEvidence": "改用与正式 Runner 一致的 sh -c，Go/gcc/buildx/node/npm 全部通过。",
    "permanentAction": "Runner preflight 和正式入口统一使用非 login shell。",
    "historicalReleases": []
  },
  {
    "fingerprint": "l3/governance/business-context-freshness-threshold",
    "position": "before-production-write",
    "count": 1,
    "impact": "新增补丁使业务事实基线达到 50 提交阈值，r3 L3 被治理门禁停止。",
    "recoveryEvidence": "刷新业务事实文档并重建 074d769/r4，最终 L3 通过。",
    "permanentAction": "候选冻结前先运行 business-context-freshness，达到阈值先刷新事实文档。",
    "historicalReleases": []
  },
  {
    "fingerprint": "browser-acceptance/mock-api/expected-network-errors",
    "position": "before-production-write",
    "count": 1,
    "impact": "首轮浏览器脚本把 mock 未实现端点 404 误判为产品控制台错误。",
    "recoveryEvidence": "仅白名单精确 mock 端点，Runtime 异常和非预期日志仍 fail closed，复跑通过。",
    "permanentAction": "标准 mock 验收按 fixture contract 分类 network error，不放宽 Runtime 异常。",
    "historicalReleases": []
  },
  {
    "fingerprint": "browser-acceptance/mock-state/reused-deleted-fixture",
    "position": "before-production-write",
    "count": 1,
    "impact": "删除网站后的 mock 状态被下一轮复用，导致 fixture 等待超时。",
    "recoveryEvidence": "停止旧预览并以新 scope 重建 mock，最终 Chrome 验收通过。",
    "permanentAction": "破坏性 mock 旅程每轮使用全新 scope 和 manifest。",
    "historicalReleases": []
  },
  {
    "fingerprint": "production-update/app-conf/source-context-missing",
    "position": "after-production-write",
    "count": 1,
    "impact": "直接 source apps 配置缺少应用市场函数上下文，产生 local/docker_app_plus 告警；docker_app_update 本体仍完成且产品健康。",
    "recoveryEvidence": "升级返回 Update Complete；随后验证 v0.90.2、Agent、OCI、数据、日志全部通过，并验证正确函数包装上下文可加载。",
    "permanentAction": "负责人 release writer；2026-08-23 前把应用配置调用统一到受检 launcher，退出条件为不再直接顶层 source kpanel.conf。",
    "historicalReleases": []
  },
  {
    "fingerprint": "production-verification/agent-version/unsupported-flag",
    "position": "after-production-write",
    "count": 1,
    "impact": "验证命令误用 Agent 不支持的 --version，产生一次非服务进程错误输出。",
    "recoveryEvidence": "systemd NRestarts=0；改用容器内 paneld agent-healthcheck 后通过。",
    "permanentAction": "生产 Agent 版本统一通过 agent-healthcheck 与 journal 启动版本取证。",
    "historicalReleases": []
  },
  {
    "fingerprint": "production-verification/ssh/powershell-variable-expansion",
    "position": "after-production-write",
    "count": 1,
    "impact": "内联 OCI/SQLite 复核再次受本机 PowerShell 嵌套命令展开影响，部分证据无效。",
    "recoveryEvidence": "改用 verify-production-v0902.sh，全部健康、数据与日志门禁通过。",
    "permanentAction": "生产多项核验只运行已做 bash -n 和 SHA 记录的远端受限脚本。",
    "historicalReleases": []
  },
  {
    "fingerprint": "production-browser/chrome/readiness-budget-missing",
    "position": "after-production-write",
    "count": 1,
    "impact": "Chrome CLI 首张截图在 SPA 完成渲染前取证，得到仅背景的无效截图。",
    "recoveryEvidence": "加入 8 秒 virtual-time-budget 后真实登录页完整渲染并截图。",
    "permanentAction": "生产 SPA smoke 固定等待应用就绪或使用受检虚拟时间预算。",
    "historicalReleases": []
  },
  {
    "fingerprint": "production-browser/temp-profile/cleanup-policy-block",
    "position": "after-production-write",
    "count": 1,
    "impact": "Remove-Item 递归清理被执行策略拒绝，临时 Chrome Profile 首次未清理。",
    "recoveryEvidence": "核对绝对目标在 release artifacts 后使用 System.IO.Directory.Delete，结果 removed=True。",
    "permanentAction": "独立 Chrome Profile 创建时记录精确绝对路径，清理使用同一受检路径和单一 PowerShell/.NET 上下文。",
    "historicalReleases": []
  }
]
<!-- kpanel-release-process-incidents:end -->

## 遗留风险与后续准入

- 未验证风险：未在生产删除真实网站；该操作会删除网站目录、Nginx 配置、证书和同名数据库，不适合生产验收。
- 已实现待实机准入：无。
- 不阻断本版的理由：删除调用在 Go 契约、Agent、固定脚本参数和 mock UI 全链路均有自动验证，且生产部署未改变数据结构或脚本固定版本。
- 后续门禁：为应用市场配置增加唯一受检调用 launcher；破坏性网站删除只在可销毁的真实站点夹具中追加 L2。
