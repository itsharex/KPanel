# KPanel v0.90.4 发布验收记录

日期：2026-08-22

发布级别：L3

候选提交 / 标签：`3b97f01de7482ff772590352254ae7bd9d1c6a68` / `v0.90.4`

上一稳定版本 / 回滚点：`v0.90.3` / `sha256:621ca9fab03e8c08a427d1b7afcc0ff86d4d49d0e636259f69f7fbf388055b6e`

## 发布范围

- 修复终端主机栏收起后，嵌套系统图标 tooltip 覆盖完整“主机名 · 状态”提示的问题。
- `OperatingSystemIcon` 新增默认开启的可选 tooltip 开关，仅终端收起栏关闭内部图标提示；其他调用方行为不变。
- 补充组件与终端布局回归测试。
- 本版不改变数据库、API、Agent、Compose、受管脚本或应用市场配置；未连接、读取、测试或部署 108。

## 多维质量结论

| 维度 | 状态 | 证据 | 未验证风险或不适用依据 |
| --- | --- | --- | --- |
| 业务正确性与双端互通 | 已验证 | 定向 2 文件/34 项、Web 107 文件/840 项、完整 L3、折叠主机栏真实 DOM 验收 | 纯前端 tooltip 归属修复，无跨端协议变化 |
| 网络入侵与供应链安全 | 已验证 | govulncheck 可达漏洞 0、npm audit 0、Trivy source/image 0、OCI revision 与受管脚本摘要固定 | 无新增依赖 |
| 稳定性、失败恢复与兼容 | 已验证 | race、双架构、应用生命周期、公开 OCI E2E、停写备份与旧版恢复材料校验 | 无数据迁移 |
| 性能与资源预算 | 已验证 | 生产 3 次采样 CPU 0.02%～0.03%、内存 72.34 MiB/256 MiB、7 PIDs | 仅移除一个嵌套 title，不改变资源模型 |
| 用户体验与可访问性 | 已验证 | 按钮 title/aria-label 完整，内部图标 title 为空，原生 button 键盘语义保留，横向溢出 0 | Mock 终端缺省会话产生既有 Vue warning，与本补丁无关 |
| 数据、配置与迁移 | 已验证 | SQLite integrity=ok、20 个 JSON 可解析、Compose config 通过 | 无迁移或配置契约变化 |

## 自动门禁与隔离验收

- 定向测试 2 文件/34 项通过；最终 Web 107 文件/840 项、typecheck、i18n 2414 条、生产构建通过。
- Git bundle：`kpanel-v0.90.4-3b97f01.bundle`，SHA-256=`2e4c3416a2248a6b316517e4a1b2044b0d5043df7b4a162a1424e0a5798043db`。
- arena-154 完整 `make verify-release` exit 0；日志 `/root/kpanel-release-evidence/v0.90.4-3b97f01/l3-verify-release.log`，SHA-256=`938359323524bb74fc1b81903e59c5b20baa96f63660b8b37b87409c7a3e8a1c`。
- 候选 CI `32551499535`、候选依赖新鲜度 `32551499607`、main CI `32551693814`、main 依赖新鲜度 `32551693773`、Tag 依赖新鲜度 `32551904814` 均 success，head SHA 精确匹配 `3b97f01...`。
- 本地 acceptance mock/draft 验收通过：两个收起主机按钮保留完整 title/aria-label，内部 `.os-identity__mark` 无 title，页面无横向溢出；证据 `C:\GitHub\_release-artifacts\v0.90.4\browser-acceptance.md`，SHA-256=`d14042514619bb5ec294f740a3255d3a2c73c199acda46570f2961df3966de60`，预览已停止。

## 发布产物与公开复核

- GitHub Release：[v0.90.4](https://github.com/kejilion/KPanel/releases/tag/v0.90.4)，非 draft、非 prerelease；Release workflow `32551904802` success。
- Docker `0.90.4` 与 `latest` OCI index 均为 `sha256:d2199e61c3fdcd7e5ab847911bffbd8fa849b9571e055f062b00d51def6df8a4`。
- `linux/amd64`=`sha256:8a98d8bd4f22e1720520765a938358bd139d3bb7408c260cd472cd767035c511`；`linux/arm64`=`sha256:6bcb065ebada06833269ff3f4cb604f92b25b038325f589a9d05970ed16c9149`。
- arena-154 独立公开回拉 E2E 验证 version=`0.90.4`、revision=`3b97f01...`、非 root、受限容器、health、受管脚本摘要均通过；摘要证据 SHA-256=`57092ea15081c5f8eeda536762c23e09e6f1d61713bc71d327122e1bfcfe5690`。
- `packaging/kejilion-app/kpanel.conf` 与 `kejilion/apps@6d86eee24a477320f4d8ffb32d9e85b785cf3c2c` 内容一致，无需 apps 空提交。

## 生产部署与回滚

- 生产目标仅 arena-154；108 未连接、未读取、未测试、未备份、未部署。
- 停写一致性备份：`/root/kpanel-backups/v0.90.4-preupgrade-arena154-20260822T044140Z`；`SHA256SUMS` 文件摘要=`021d06b8b1ea0d337d4975b25024c22176cbef0946c2a88d2ad51089d4986cb9`。
- 备份已校验归档摘要、独立解包、SQLite、20 个 JSON、Compose、配置文件与旧 OCI，并恢复 v0.90.3 healthy 后才升级。
- 通过标准 `kejilion.sh app kpanel` 非交互更新入口升级，日志 SHA-256=`e3302454732022dca87fab966a3429646bb518e013b2ed51a3d8bc556df91abc`。
- 部署后 Panel v0.90.4 healthy、Agent active，restart=0、OOM=false、systemd NRestarts=0/NeedDaemonReload=no；公网 HTTPS health 返回 200/version 0.90.4；SQLite、JSON、Compose 和错误日志检查通过。
- 回滚必须停写后成套恢复备份中的镜像、Compose、`.env`、数据与 systemd unit；旧 OCI 为 `sha256:621ca9fab03e8c08a427d1b7afcc0ff86d4d49d0e636259f69f7fbf388055b6e`，禁止只换镜像。

## 交付节奏数据

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-08-22T12:00:22+08:00
- 候选冻结时间：2026-08-22T12:06:07+08:00
- 生产完成时间：2026-08-22T12:45:00+08:00
- 提交到生产用时：0.74 小时
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
    "fingerprint": "candidate-local-web/npm-dependencies/missing-node-modules",
    "position": "before-production-write",
    "count": 1,
    "impact": "新建隔离 worktree 首次运行 Web 测试时尚未安装 node_modules，测试入口未启动。",
    "recoveryEvidence": "npm ci 后 Web 107 文件/840 项、typecheck、i18n 和生产构建全部通过。",
    "permanentAction": "新 worktree 的 Web 门禁先执行 npm ci。",
    "historicalReleases": []
  },
  {
    "fingerprint": "candidate-local-gate/go-runtime/missing",
    "position": "before-production-write",
    "count": 1,
    "impact": "Windows 本机没有 Go，verify-change 在 Go 入口停止。",
    "recoveryEvidence": "arena-154 Linux L3 完整 Go、race、vet、双架构与安全门禁通过。",
    "permanentAction": "本机无 Go 时不伪造通过，交由固定 Linux L3 Runner 补齐。",
    "historicalReleases": []
  },
  {
    "fingerprint": "local-browser/iab/evaluate-focus-method-unsupported",
    "position": "before-production-write",
    "count": 1,
    "impact": "一次浏览器 evaluate 直接调用 focus 方法不受控制桥支持，该次交互证据无效。",
    "recoveryEvidence": "改用原生 button DOM、title/aria-label、tabIndex 与现有键盘回归测试完成等价核对。",
    "permanentAction": "IAB 中仅使用受支持的 DOM 读取和 locator 交互。",
    "historicalReleases": []
  },
  {
    "fingerprint": "tag-verification/powershell/tag-peel-expression",
    "position": "before-production-write",
    "count": 1,
    "impact": "首次 tag 解引用命令未引用 ^{}，被 PowerShell 解析为无效参数。",
    "recoveryEvidence": "使用单引号保护 tag peel 表达式后，Tag 与 main 均精确指向 3b97f01。",
    "permanentAction": "PowerShell 中 Git tag peel 表达式必须整体单引号保护。",
    "historicalReleases": []
  },
  {
    "fingerprint": "public-oci/ssh/powershell-command-substitution",
    "position": "before-production-write",
    "count": 1,
    "impact": "首次内联 SSH 中的命令替换被本机 PowerShell 提前执行，远端证据无效。",
    "recoveryEvidence": "改为固定 SHA、bash -n 通过的受检远端脚本后完成公开 OCI E2E。",
    "permanentAction": "复杂 SSH 核验只运行本地生成并校验摘要的远端脚本。",
    "historicalReleases": []
  },
  {
    "fingerprint": "public-oci-e2e/wrapper/runtime-assumptions",
    "position": "before-production-write",
    "count": 3,
    "impact": "E2E 包装器先假设最小镜像含 id 命令，又在 Docker health start-period 完成前读取状态；一次重跑用于定位失败行。",
    "recoveryEvidence": "改用容器 Config.User，并在循环中同时等待应用 health 与 Docker health，最终公开 OCI E2E 全绿。",
    "permanentAction": "最小镜像身份读取固定使用 inspect，健康门禁同时等待进程与容器状态。",
    "historicalReleases": []
  },
  {
    "fingerprint": "production-preflight/ssh/jq-key-quoting",
    "position": "before-production-write",
    "count": 1,
    "impact": "一次只读生产预检中的 jq 标签键引号被双层 Shell 吞掉，标签输出无效。",
    "recoveryEvidence": "受检备份、部署与生产验证脚本使用固定 jq 表达式，version/revision/脚本摘要均通过。",
    "permanentAction": "生产复杂 jq 表达式不再内联到 PowerShell SSH 字符串。",
    "historicalReleases": []
  },
  {
    "fingerprint": "acceptance-ci/github-api/anonymous-rate-limit",
    "position": "after-production-write",
    "count": 1,
    "impact": "轮询验收记录 CI 时 GitHub 匿名 REST API 达到速率限制；生产服务未受影响。",
    "recoveryEvidence": "改用已连接的 GitHub connector 核对 run 32552638529 completed/success，head SHA 精确为 046d901。",
    "permanentAction": "CI 轮询优先使用已认证连接，匿名 REST 仅作低频只读补充。",
    "historicalReleases": []
  }
]
<!-- kpanel-release-process-incidents:end -->

## 遗留风险与结论

- 生产未使用管理员凭据进入终端页面，避免修改或暴露生产认证信息；相同冻结源码的隔离预览已完成受影响 tooltip 旅程。
- 本版没有阻断风险，无需回滚；保留 v0.90.3 不可变镜像和已验证停写备份作为回滚点。
