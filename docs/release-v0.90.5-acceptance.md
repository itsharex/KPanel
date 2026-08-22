# KPanel v0.90.5 发布验收记录

日期：2026-08-22

发布级别：L3

候选提交 / 标签：`957d30be1a3cfb3c1f66cc8dcbf32c7b4ab6adf1` / `v0.90.5`

上一稳定版本 / 回滚点：`v0.90.4` / `sha256:d2199e61c3fdcd7e5ab847911bffbd8fa849b9571e055f062b00d51def6df8a4`

## 发布范围

- 修复 IPING 风险率只读取 API 元数据、与官网详情页风险值不一致的问题；风险分数改为从官方固定详情页提取，运营商等元数据继续使用 API。
- API 与官网请求并发、各自 5 秒超时和 512 KiB 响应上限；官网失败回退 API，两端都失败才进入原生探测，保持既有失败恢复语义。
- 仅允许已解析的公网 IPv4 拼接固定 IPING URL，不接受用户提供的任意 URL；没有数据库、前端、Agent 协议、依赖、Compose、受管脚本或应用市场契约变化。
- 生产目标仅 arena-154；108 未连接、未读取、未测试、未备份、未部署。

## 多维质量结论

| 维度 | 状态 | 证据 | 未验证风险或不适用依据 |
| --- | --- | --- | --- |
| 业务正确性与双端互通 | 已验证 | diagnostics 单元回归、Go 全包、Web 107 文件/840 项、arena-154 真实 Agent→IPING 链路 | 生产未重复创建诊断任务；同一冻结源码已在 arena-154 隔离 systemd Agent 验证 |
| 网络入侵与供应链安全 | 已验证 | 固定 HTTPS Origin、公网 IPv4 约束、5 秒/512 KiB 上限、govulncheck、npm audit、Trivy source/image 0 | 无新增依赖或用户可控请求目标 |
| 稳定性、失败恢复与兼容 | 已验证 | API/页面独立回退测试、race、双架构、应用生命周期、公开 OCI E2E、停写备份和旧版恢复材料 | IPING 页面结构未来变化时会安全回退 API，可能暂时降低精度 |
| 性能与资源预算 | 已验证 | 两个上游请求并发；生产三次采样 CPU 0.02%～0.03%、内存 74.36 MiB/256 MiB、7 PIDs | 单次诊断最多增加两次有界外部请求 |
| 用户体验与可访问性 | 已验证 | 返回字段和现有 UI 契约不变；真实 IPING 风险值与官网一致 | 本版无前端布局或交互变化 |
| 数据、配置与迁移 | 已验证 | SQLite integrity=ok、20 个 JSON 可解析、Compose config 通过 | 无迁移或配置变更 |

## 自动门禁与隔离验收

- Git bundle：`kpanel-v0.90.5-957d30b.bundle`，SHA-256=`1b06db5fd284b5e5e848e4ea8b815cdefc3ea5899e96a30980386ccff3919b88`。
- arena-154 完整 `make verify-release` exit 0；Go 全包、race、vet、Linux 双架构、Web 107 文件/840 项、typecheck、i18n 2414 条、生产构建、govulncheck、npm audit、Trivy source/image、受管脚本和应用生命周期全部通过。日志 `/root/kpanel-release-evidence/v0.90.5-957d30b/l3-verify-release.log`，SHA-256=`75e18a68900a2e90002584be8708f5c9d95f07b3571cf7cebadb71e89af3e856`。
- arena-154 真实 transient systemd Agent 执行 `native-ip-quality`：公网 IP `154.36.153.9` 返回风险值 9，IPING 官网同值；日志 SHA-256=`edce0e038eceec309c291e7ce61c7da8681f1dded9fd361cd1af807ccbf58bb1`，结构化结果 SHA-256=`dc2edb20d78f78a0c5a634623175fc58c0a38e16dee96e21d9706dfc80192637`。临时 unit、二进制和任务目录已清理。
- 候选 CI `32554031267`、候选依赖新鲜度 `32554031255`、main CI `32554223963`、main 依赖新鲜度 `32554223965`、Tag 依赖新鲜度 `32554440012` 均 success，head SHA 精确匹配 `957d30b...`。

## 发布产物与公开复核

- GitHub Release：[v0.90.5](https://github.com/kejilion/KPanel/releases/tag/v0.90.5)，非 draft、非 prerelease；Release workflow `32554440025` success。
- Docker `0.90.5` 与 `latest` OCI index 均为 `sha256:29683a371983e30e5d2ea695f60debef4aa5602f21abf41fdf48cbb63bcbc286`。
- `linux/amd64`=`sha256:5e350989dd99a13a025b3a6beb14aff26ed85d1aff58b4d3b20dd35ce6e8acbb`；`linux/arm64`=`sha256:b3c34d7102c40adb8eccb2092a342c793b874f3cb1fb606f963dc3dd2e210e3e`。
- arena-154 独立公开回拉验证 version=`0.90.5`、revision=`957d30b...`、双标签同一 index、非 root、受限容器、health 和受管脚本摘要均通过。
- `packaging/kejilion-app/kpanel.conf` 与 `kejilion/apps@6d86eee24a477320f4d8ffb32d9e85b785cf3c2c` 规范化内容一致，无需 apps 空提交。

## 生产部署与回滚

- 停写一致性备份：`/root/kpanel-backups/v0.90.5-preupgrade-arena154-20260822T054004Z`；`SHA256SUMS` 文件摘要=`98fa7b9470f432364d3ad2859f3395f304cc72c0b8dd81ed525376486d11551a`。
- 备份已校验归档摘要、独立解包、SQLite、20 个 JSON、Compose、配置文件和旧 OCI，并恢复 v0.90.4 healthy 后才升级。首次备份验证因宿主没有 sqlite3 CLI 被门禁拦截并自动恢复旧服务；该不完整目录仅保留为事故证据，不作为回滚点。
- 通过标准 `kejilion.sh app kpanel` 非交互更新入口升级，日志 SHA-256=`19e91db030c4576f8f6e55c0622a79ad6387a90c746cf9d36dc649aed4557b17`。
- 部署后 Panel v0.90.5 healthy、Agent active，restart=0、OOM=false、systemd NRestarts=0/NeedDaemonReload=no；公网 HTTPS health 返回 200/version 0.90.5；SQLite、JSON、Compose、镜像 revision/index 和错误日志检查通过。
- 回滚必须停写后成套恢复已验证备份中的镜像、Compose、`.env`、数据和 systemd unit；旧 OCI 为 `sha256:d2199e61c3fdcd7e5ab847911bffbd8fa849b9571e055f062b00d51def6df8a4`，禁止只换镜像。

## 交付节奏数据

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-08-22T11:59:37+08:00
- 候选冻结时间：2026-08-22T12:58:08+08:00
- 生产完成时间：2026-08-22T13:46:05+08:00
- 提交到生产用时：1.77 小时
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
    "fingerprint": "candidate-local/go-gate/runtime-missing",
    "position": "before-production-write",
    "count": 1,
    "impact": "Windows 本机没有 Go，首次本地 gofmt/Go 门禁不能执行。",
    "recoveryEvidence": "arena-154 固定 Linux Runner 完整 Go、race、vet、双架构和 L3 全绿。",
    "permanentAction": "本机无 Go 时不伪造通过，交由固定 Linux L3 Runner补齐。",
    "historicalReleases": []
  },
  {
    "fingerprint": "l3-bundle/verify/non-repository-context",
    "position": "before-production-write",
    "count": 1,
    "impact": "首次 git bundle verify 未在 Git 仓库上下文执行，该次证据无效。",
    "recoveryEvidence": "改在隔离 bare repository 中验证 bundle、clone 和冻结 SHA，随后完整 L3 通过。",
    "permanentAction": "bundle verify 固定在临时 bare repository 中执行。",
    "historicalReleases": []
  },
  {
    "fingerprint": "l3-runner/preflight/login-shell-path-reset",
    "position": "before-production-write",
    "count": 1,
    "impact": "Runner 首次以 sh -lc 启动导致固定 PATH 被登录 Shell 重置，Go preflight 返回 127。",
    "recoveryEvidence": "改用 sh -c 并核对不可变 Runner image ID 后，从冻结 SHA 完整重跑 L3 通过。",
    "permanentAction": "固定 Runner 预检不使用登录 Shell，并显式核对工具链和 image ID。",
    "historicalReleases": []
  },
  {
    "fingerprint": "production-preflight/ssh/structured-query-quoting",
    "position": "before-production-write",
    "count": 2,
    "impact": "两次只读 SSH 中 Docker Go template 或 jq 键引号被 PowerShell/远端 Shell 改写，结构化输出无效。",
    "recoveryEvidence": "部署与生产验证改用已审查、bash -n 通过的远端脚本，version/revision/digest 全部通过。",
    "permanentAction": "复杂生产结构化查询禁止内联到 PowerShell SSH 字符串。",
    "historicalReleases": []
  },
  {
    "fingerprint": "tag-verification/git-fetch/stale-local-tag",
    "position": "before-production-write",
    "count": 1,
    "impact": "本地旧 v0.86.2 tag 与远端历史不一致，fetch --tags 拒绝覆盖；v0.90.5 tag 未受影响。",
    "recoveryEvidence": "仅核对并推送全新的 v0.90.5 tag，远端 tag、main 和 Release SHA 精确一致。",
    "permanentAction": "发布核对按目标 tag 精确 fetch，不用无边界 fetch --tags 改写本地历史。",
    "historicalReleases": []
  },
  {
    "fingerprint": "public-oci/preflight/powershell-command-substitution",
    "position": "before-production-write",
    "count": 1,
    "impact": "首次公开 OCI 端口预检中的命令替换被本机 PowerShell提前解析；未启动容器。",
    "recoveryEvidence": "改为受检远端脚本后，公开 OCI 双标签、revision、健康和受限运行 E2E 全绿。",
    "permanentAction": "复杂 SSH E2E 只运行固定摘要的远端脚本。",
    "historicalReleases": []
  },
  {
    "fingerprint": "production-backup/sqlite-validation/cli-missing",
    "position": "after-production-write",
    "count": 1,
    "impact": "首次停写备份完成后，独立验证调用宿主未安装的 sqlite3 CLI 而失败。",
    "recoveryEvidence": "trap 自动恢复 v0.90.4 healthy；改用 Python sqlite3 3.46.1 后创建新的完整备份并通过恢复验证。",
    "permanentAction": "生产备份脚本预检并使用项目已依赖的 Python sqlite3 runtime。",
    "historicalReleases": []
  },
  {
    "fingerprint": "production-backup/log-pipeline/exit-masked",
    "position": "after-production-write",
    "count": 1,
    "impact": "首次备份命令外层 tee 掩盖远端脚本失败退出码；日志检查及时识别，未继续部署。",
    "recoveryEvidence": "新脚本内部 tee 且启用 pipefail，明确输出 backup_verified=pass 后才允许升级。",
    "permanentAction": "生产写脚本自行保留日志和退出码，SSH 外层不再用 tee 包装门禁。",
    "historicalReleases": []
  }
]
<!-- kpanel-release-process-incidents:end -->

## 遗留风险与结论

- IPING 是外部数据源；官网 DOM 或 API 字段将来变化时，本实现按固定顺序回退，不会把解析失败伪报为 0，但精度可能暂时回落到 API。
- 生产未创建额外诊断任务或使用管理员凭据；受影响链路已在同一 arena-154 的隔离真实 Agent 完成。
- 本版没有产品阻断风险，无需回滚；保留 v0.90.4 不可变镜像和已验证停写备份作为回滚点。
