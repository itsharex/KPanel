# KPanel v0.94.0 发布验收记录

日期：2026-08-23

发布级别：L3

候选提交 / 标签：`105f4366dea10aa386767b6eebdd41845e820fb6` / `v0.94.0`

上一稳定版本 / 回滚点：`v0.93.0` / `66199b7106b8be7b3b532f36454a5628e0732eb9`

## 发布画像

- 业务域：文件管理状态反馈、系统中心磁盘与分区管理、发布治理事实新鲜度。
- 变更面：展示、只读、宿主机写入、Panel/Agent 协议、`kejilion.sh` 固定协议、部署。
- 受影响用户旅程：远程下载/文件传输进度；窄桌面文件工具栏；磁盘拓扑查看、格式化、检查/修复、挂载/卸载及后台任务恢复。
- 未变化契约：端口、Compose、数据库和应用市场安装契约未变化；不创建、删除或调整分区表，不编排 LVM/RAID/LUKS。
- 风险等级及理由：高；新增 root Agent 的磁盘写能力，但以保护拓扑、opaque ID、资源版本、可信脚本、共享锁、持久任务和执行后回读限制范围。

## 发布范围与未纳入内容

- 用户可见更新：文件传输状态更紧凑、空闲不显示空占位、持久下载说明更清楚、窄桌面工具栏纵排；系统中心新增安全磁盘与分区管理。
- 精确提交清单：`813b4d9`、`6f9914e`、`a826c72`、`3d6efa1`、`e94840c`、`a3bb87c`、`f91474a`、`1c83c73`、`105f436`。
- `1c83c73` 只为非 root CI 提供未导出的磁盘脚本所有权测试缝；默认生产校验仍复用 root 所有权真源，没有放宽脚本可信边界。
- `105f436` 更新现有业务上下文基线到 v0.93.0，使固定 50 提交新鲜度门禁恢复通过，不改变阈值。
- 治理候选 `e8b6404ade4003f97ec0902d7edd9e448fc35857` 未纳入；本次仍使用候选冻结时的 `release-kpanel v2.8`，待发布完全结束后另行独立复核。
- 未纳入分区创建/删除/扩缩、LVM/RAID/LUKS 写操作；未在生产执行任何磁盘写验收。

## 多维质量结论

| 维度 | 状态 | 证据 | 未验证风险或不适用依据 |
| --- | --- | --- | --- |
| 业务正确性与双端互通 | 已验证 | Go/Web 全量、真实 `kejilion.sh`→Agent→Panel、真实 loop 设备格式化/检查/修复/挂载/卸载、Agent 重启恢复 | 物理盘写操作未在生产执行 |
| 网络入侵与供应链安全 | 已验证 | govulncheck 可达漏洞 0、npm audit 0、Trivy source/image 0、脚本固定 revision/hash、正式 OCI provenance | 无新增公网匿名入口 |
| 稳定性、失败恢复与兼容 | 已验证 | stale/未知设备/并发冲突 fail-closed、持久终态、恢复快照、应用更新事务、v0.93.0 成套回滚备份 | 不替代所有发行版真实物理存储矩阵 |
| 性能与资源预算 | 已验证 | 3 次生产采样 CPU 0.02%～0.04%、内存 11.09～74.51 MiB/256 MiB、7 PIDs、restart=0、OOM=false | 无长时间高并发磁盘写 soak，因功能为管理员低频操作 |
| 用户体验与可访问性 | 已验证 | 正式 Chrome 151，中/英/繁、明/暗、1280/768/390 视口，无横向溢出，确认弹窗和操作可达 | 未做 200% 缩放物理显示器人工验收 |
| 数据、配置与迁移 | 已验证 | 停写备份独立解包；Compose、`.env`、Agent 配置一致；21 个 JSON、1 个非空 SQLite 完整 | 回滚不会撤销升级后已可信完成的宿主机磁盘操作 |

## 自动门禁

- 最终 bundle：`kpanel-v0.94.0-105f436.bundle`，SHA-256=`e9a00123ca23a9bf61fbf66de8aaaab54ecfc219211ee7ea871aaa6f4c2d858c`。
- 固定 Runner `kpanel-release-gate:go1.26.6-node24` / `sha256:b593c0ffe32e80a6a6ae8fcac38ed916587dbf698a6b6a55fa33887298737148` 完整 L3 exit 0；日志 SHA-256=`027de5eae19741ab9291f20c0f415988779e913933e26a4e94f1e92dab25caf3`，摘要 SHA-256=`294bd4c47504998aab6ae448cab59185f7a11a0fb87075c393d58cb455069435`。
- Go 全包、核心 race、vet、Linux amd64/arm64 二进制通过；Web 113 文件/908 项、i18n 2465/21、typecheck、production build 通过。
- 候选 CI：`32627959541` completed/success，精确绑定 `105f4366...`。
- 主线 CI：`32628098003` completed/success，精确绑定 `105f4366...`。
- Release workflow：`32628217485` completed/success，精确绑定 `v0.94.0` / `105f4366...`。
- 安全扫描、受管脚本契约、安装安全和应用生命周期均通过；正式镜像含双架构 provenance attestation。

## 依赖与技术栈变化

- 本版未修改 Go/npm 依赖、Action SHA 或基础镜像。
- 最终 L3 使用仓库固定检测源；govulncheck 可达漏洞 0、npm audit 0、Trivy 0.72 source/image 的 HIGH/CRITICAL/secret/misconfiguration 均为 0。
- Go 基础镜像仍为 `golang:1.26.6-alpine@sha256:af8d6740070b8906d12eae1c3e3ea0957fb63f492051ea05e354c38ef9fe88df`；Node 仍为 `24.18.0-alpine@sha256:a0b9bf06e4e6193cf7a0f58816cc935ff8c2a908f81e6f1a95432d679c54fbfd`。
- 受管脚本固定到 `9fec61b50cc6ef798dfac1edf11c2ec60ca6b0d1`，LF SHA-256=`54ceb0e72c4c342382500fc35da636fa436c484a12c4766fb9c7f806a23ae8fa`；脚本小功能没有修改脚本版本号。
- 没有新增期限例外；Trivy 新版本提示不改变本次固定 Runner 结论，依赖升级需独立候选。

## 隔离真机与浏览器验收

- 环境：`arena-154`，Linux amd64、Docker/loop device、正式 Google Chrome 151.0.7922.172；策略允许 `candidate-validation`、`browser-validation` 和本次已授权生产部署。
- 精确候选镜像：`kejilion-panel:verify-105f436`，image ID=`sha256:cb835cfcca94294ff441c59aaac2445aece832f8a63304d553e092a92781524e`，version/revision/script labels 全部精确匹配。
- 磁盘 L2：真实 loop 设备、ext4 格式化、只读检查、修复、挂载/卸载、共享锁并发、stale/未知设备、Agent 重启恢复和审计脱敏均通过；证据清单 SHA-256=`d9d85a2a1c63e22ff69d3f71ba5d71f6430ba6131b02f9a7ca1819e1f7f4d72b`。
- Chrome 四组矩阵：zh-CN/light 1280、zh-TW/dark 768、en-US/light 390、zh-CN/dark 390；报告 SHA-256=`56a6213b2dc28b3c08cf827d2c62dbc4a83ca2c07705b6e3fda93c07662d6745`。
- 无 soak：磁盘写为低频管理员操作，完整状态转换、失败注入、重启恢复和生产短采样提供更直接风险证据。
- 夹具 loop、容器、隧道、测试 Agent 和临时目录均已清理；生产在隔离验收期间保持 v0.93.0 healthy。

## 发布产物与公开仓库复核

- GitHub Release：[v0.94.0](https://github.com/kejilion/KPanel/releases/tag/v0.94.0)，已发布且非 draft/prerelease。
- Docker 版本与 `latest` OCI index 均为 `sha256:041d39cbd4fa042fad71d9a180177c7eb01cd017ac5cca36f228b204f0870d00`。
- `linux/amd64`=`sha256:5432e539d29a32b125f85e0ba7ae861cdf6b562c0a2b64a73830d79de502d9e1`；`linux/arm64`=`sha256:21801f2d1c960686e45eb7a258d5a16d927192c05027dec011e90f1499518182`。
- Release 附件：Agent/Node 双架构、部署包、LICENSE、THIRD_PARTY_NOTICES 和 `SHA256SUMS`。
- 公开镜像按不可变 index 独立回拉，以非 root、read-only、cap-drop、no-new-privileges 和资源限制运行，`/api/v1/health` 返回 v0.94.0；`image_e2e=pass`。
- `kejilion/apps@6d86eee24a477320f4d8ffb32d9e85b785cf3c2c` 的 `kpanel.conf` 与发布仓库 blob `34316059d4e42f527819bc7d56e0ff14ec434c96` 完全相同，生命周期在 L3 通过，因此没有空提交。
- `kejilion/sh main@9fec61b50cc6ef798dfac1edf11c2ec60ca6b0d1` 已先发布；公开 Raw、根/CN 同步、语法与磁盘协议 smoke 已通过。

## 生产部署安全核对

- 唯一生产目标和验证/灰度环境均为 `arena-154`，来自 `environment-policy.json`；本次有明确发布和部署授权。
- `prod-108` 固定禁用全部 KPanel 操作；本次未连接、未读取、未测试、未备份、未部署、未升级、未核对。
- 部署前 v0.93.0 healthy/active、restart=0、OOM=false；旧 OCI=`sha256:3ffdd29f78cba50d10c2efe2140af46dee2104bc6151d91c6f0031b1449bee2b`。
- 新停写一致性备份：`/root/kpanel-backups/v0.94.0-preupgrade-arena154-20260823T083657Z`；`SHA256SUMS` SHA-256=`7627c11e617d5871db773430035facb1ee13b8055a629c420ce336782be5ceea`。状态包和旧 OCI 均通过 zstd 校验并独立解包比较关键配置。
- 部署入口：`KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update bash /home/docker/kpanel/bin/kejilion.sh app kpanel`；日志 SHA-256=`25e75cf8aa188c50897b2ae1152093e4febed7062ec841a6c570b6dd5366576d`。
- 部署后 Panel v0.94.0 healthy、Agent active、container restart=0、OOM=false、Agent NRestarts=0、NeedDaemonReload=no；镜像、Agent 和归一化后的受管脚本与正式 OCI 一致。
- Compose、`.env`、Agent 配置/service 与备份一致；21 个 JSON、1 个非空 SQLite 完整。三次采样证据 SHA-256=`a07ece72de20bf4e618bdd348a6a1e1299551c3480e1622f01accf6865741105`。
- 宿主机自身通过公开地址访问 8080 health 为 HTTP 200；INPUT 策略明确 DROP 外部 8080，发布执行器不能直连该端口，这是既有生产访问边界，不把它写成公网可达验收。
- 生产写操作仅为标准升级事务；磁盘格式化、检查/修复和挂载写旅程只在隔离 loop 设备执行，未触碰生产物理磁盘或 `fstab`。

## 回滚

- 源码/tag：`v0.93.0` / `66199b7106b8be7b3b532f36454a5628e0732eb9`。
- 旧 OCI：`sha256:3ffdd29f78cba50d10c2efe2140af46dee2104bc6151d91c6f0031b1449bee2b`。
- 数据/配置备份：`/root/kpanel-backups/v0.94.0-preupgrade-arena154-20260823T083657Z`。
- 回滚必须停写并成套恢复旧 image archive、完整 `/home/docker/kpanel`、Compose、`.env`、数据、Agent unit 和二进制，随后 daemon-reload、启动 Agent/Panel 并复核 v0.93.0/health/digest/JSON/SQLite；禁止只换镜像。
- 备份恢复检查和旧 OCI 归档已验证；生产无需回滚，当前保持 v0.94.0 healthy。
- GitHub Latest、Docker `latest` 和标准更新入口均指向 v0.94.0；公共默认通道无需恢复。

## 交付节奏数据

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-08-23T13:21:14+08:00
- 候选冻结时间：2026-08-23T16:05:18+08:00
- 生产完成时间：2026-08-23T16:41:53+08:00
- 提交到生产用时：3.34 小时
- 是否回滚、紧急热修复或重复发布：否
- 若发生失败，发现时间、恢复时间和逃逸门禁：不适用
<!-- kpanel-release-metrics:end -->

<!-- kpanel-release-process-metrics:start -->
- 已记录发布流程异常或无效证据拦截次数：18
- 其中生产写操作开始后异常次数：4
<!-- kpanel-release-process-metrics:end -->

### 流程异常明细

<!-- kpanel-release-process-incidents:start -->
[
  {
    "fingerprint": "main-ci/ci/nonroot-owner-fixture",
    "position": "before-production-write",
    "count": 1,
    "impact": "f91474a 的 main CI 在非 root GitHub Runner 中因测试夹具假定临时脚本归 root 所有而失败，Tag 和生产被停止。",
    "recoveryEvidence": "1c83c73 增加未导出的所有权测试缝；root、UID 65534 和 non-root race 均通过，最终候选与 main CI 成功。",
    "permanentAction": "测试显式注入所有权判断，生产默认继续使用真实 root 所有权真源。",
    "historicalReleases": []
  },
  {
    "fingerprint": "l3/bundle/missing-required-tags",
    "position": "before-production-write",
    "count": 2,
    "impact": "前两份最终 L3 bundle 分别缺少 v0.93.0 和业务新鲜度所需 v0.90.1，均在产品测试前 fail-closed。",
    "recoveryEvidence": "最终完整历史与全部稳定 Tag bundle SHA-256=e9a00123...，L3 exit 0。",
    "permanentAction": "本次 bundle 生成固定包含完整 refs；后续由待独立复核的 release-kpanel v2.9 候选评估统一 Tag 获取入口。",
    "historicalReleases": []
  },
  {
    "fingerprint": "l3/governance/stale-business-context",
    "position": "before-production-write",
    "count": 1,
    "impact": "加入 CI 修复后正好达到 50 提交阈值，业务上下文新鲜度门禁拒绝继续。",
    "recoveryEvidence": "105f436 将既有当前业务评审基线更新到 v0.93.0，门禁显示 baseline=66199b7、commits=10。",
    "permanentAction": "不修改阈值；按规范更新现有唯一业务上下文真源。",
    "historicalReleases": []
  },
  {
    "fingerprint": "disk-l2/harness/fixture-assumptions",
    "position": "before-production-write",
    "count": 2,
    "impact": "f91474a 首轮磁盘夹具分别因容器路径可见性和锁冲突终态预期不准确而重跑，未触碰生产。",
    "recoveryEvidence": "修正夹具后完整 disk L2 通过，并在最终 105f436 镜像再次一次通过。",
    "permanentAction": "最终脚本固定共享路径、真实锁语义和 cleanup 前置校验；旧失败证据保留。",
    "historicalReleases": []
  },
  {
    "fingerprint": "browser/harness/automation-assumptions",
    "position": "before-production-write",
    "count": 3,
    "impact": "f91474a 浏览器夹具依次暴露 URL 对象、选择器和通用资源错误分类问题，产品磁盘页未失败。",
    "recoveryEvidence": "修正夹具后四组矩阵通过，最终 105f436 同一脚本再次 4/4 通过。",
    "permanentAction": "使用独立正式 Chrome、精确业务 API 错误分类和稳定语义选择器；报告绑定精确 revision。",
    "historicalReleases": []
  },
  {
    "fingerprint": "remote-shell/powershell/variable-expansion",
    "position": "before-production-write",
    "count": 2,
    "impact": "两条内联 PowerShell→SSH 命令的远端命令替换被本地解释，未执行目标构建或清理核验。",
    "recoveryEvidence": "改用 SHA 固定临时脚本或 PowerShell 单引号后，镜像构建和 cleanup 核验成功。",
    "permanentAction": "复杂远端命令固定上传脚本，简单只读命令使用不展开的引号；不复用内联命令替换。",
    "historicalReleases": []
  },
  {
    "fingerprint": "backup/preflight/windows-bash-path",
    "position": "before-production-write",
    "count": 2,
    "impact": "远端备份前的本地 bash -n 两次使用错误 Windows 挂载路径，均未发生生产写入。",
    "recoveryEvidence": "使用 /mnt/c/... 完成语法检查后，停写备份一次成功并恢复 healthy。",
    "permanentAction": "当前 Windows Bash 明确为 WSL 路径模型；后续预检固定 /mnt/c 解析。",
    "historicalReleases": []
  },
  {
    "fingerprint": "public-image-e2e/harness/wrong-health-endpoint",
    "position": "before-production-write",
    "count": 1,
    "impact": "公开 OCI 首次独立启动错误请求 /health，返回前端 HTML，未涉及产品失败。",
    "recoveryEvidence": "改用产品固定 /api/v1/health 后公开不可变 digest E2E 一次通过。",
    "permanentAction": "公开镜像 E2E 固定复用 paneld healthcheck 对应的 /api/v1/health。",
    "historicalReleases": []
  },
  {
    "fingerprint": "post-deploy/harness/production-contract-assumptions",
    "position": "after-production-write",
    "count": 3,
    "impact": "上线后只读核验先后误设未登录 API=403、宿主脚本与镜像逐字节相同、宿主脚本升级前后不变；生产始终 healthy。",
    "recoveryEvidence": "按实际契约确认 401；保留 permission_granted 安装标记并归一化后与镜像一致；新脚本固定摘要正确且旧脚本已在备份。",
    "permanentAction": "后续生产核验复用认证中间件状态码、受管脚本归一化规则和升级前后职责边界。",
    "historicalReleases": []
  },
  {
    "fingerprint": "post-deploy/verification/external-direct-route",
    "position": "after-production-write",
    "count": 1,
    "impact": "发布执行器经代理得到 503、绕过代理后直连 8080 超时；该端口被生产 INPUT 策略明确 DROP。",
    "recoveryEvidence": "arena-154 宿主自身公开地址返回 HTTP 200，0.0.0.0:8080 监听、Panel/Agent/容器健康和既有 DROP 规则均已取证。",
    "permanentAction": "不把被防火墙禁止的外部直连当生产可达入口；未来若登记正式域名，应以该登记入口单独验收。",
    "historicalReleases": []
  }
]
<!-- kpanel-release-process-incidents:end -->

## 遗留风险与后续准入

- 未验证风险：真实物理盘、复杂 LVM/RAID/LUKS/多路径和所有发行版文件系统工具矩阵；外部 8080 按既有防火墙策略不可直连。
- 已实现待实机准入：无；已开放能力在隔离真实 loop/Agent/Panel/Chrome 完成准入。
- 不阻断本版的理由：公开能力范围明确排除复杂存储编排；所有写操作前后均 fail-closed 并有可恢复终态，生产升级未执行磁盘写操作。
- 后续：治理候选 `e8b6404...` 只能在本发布完全收口后基于最新 main 独立复核，不回写本次候选或历史 Tag。
