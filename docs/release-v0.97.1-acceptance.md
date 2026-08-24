# KPanel v0.97.1 发布验收记录

日期：2026-08-24

发布级别：L3

候选提交 / 标签：`b18a8aa0da3ebcf6ae52cec7659d4e3396a847fb` / `v0.97.1`

上一稳定版本 / 回滚点：`v0.97.0` / `sha256:ae60f409410af74153f20d0e47265ff59e80f4a6938ca1018d074609136d6410`

## 发布画像

- 业务域：系统日志管理、服务器体检报告布局。
- 变更面：服务日志固定读取全部 `*.service`，移除服务选择器，改为按服务名、PID 和消息本地搜索；体检报告按宽屏、普通桌面和 390px 手机重新组织信息密度；真实大 journal 读取超时由 6 秒有界调整为 15 秒。
- 不变边界：不修改权限模型、安装契约、应用市场配置、受管 `kejilion.sh`、数据 schema、端口或 Agent 写操作。
- 风险等级：中；服务日志查询范围扩大并涉及真实宿主 journal，必须验证运行时耗时、Agent 超时、桌面/手机布局和生产回滚。

## 精确提交范围

- `6ad4d1d`：简化服务日志搜索，patch-id 与来源提交 `5b30434` 一致。
- `2601db0`：压缩并重排体检报告，patch-id 与来源提交 `cefda5e` 一致。
- `2b8a3a0`：准备 `0.97.1` 版本与 CHANGELOG。
- `b18a8aa`：将服务日志读取超时有界调整为 15 秒，修复 arena-154 大 journal 下的 504。
- 未纳入其他工作树、未提交草稿、旧候选证据或 108 环境。

## 多维质量结论

| 维度 | 状态 | 证据 | 边界 |
| --- | --- | --- | --- |
| 业务正确性 | 已验证 | 真实 Panel→Agent→`journalctl`；系统/服务/安全/登录来源；服务日志搜索和 50/100/200 行 | 日志搜索仅过滤当前读取结果 |
| 网络与供应链安全 | 已验证 | govulncheck 可达漏洞 0、npm audit 0、Trivy source/image 0、OCI revision/version 固定 | 无新增匿名入口或出站链路 |
| 稳定性与兼容 | 已验证 | canonical L3、候选/main/Tag 门禁、公开 OCI E2E、停写备份与标准更新全绿 | 旧 6 秒候选结论已因 SHA 变化作废 |
| 性能与资源 | 已验证 | 生产 5 次采样 CPU 0.02–0.03%、内存 74 MiB/256 MiB、7 PIDs，restart=0、OOM=false | 15 秒是 Agent 有界读取上限，不是常驻任务 |
| 用户体验与可访问性 | 已验证 | Chrome 151 桌面/390×844；日志弹窗、关键词输入、体检两段报告；无横向溢出、page error 0 | 未执行破坏性系统操作 |
| 数据与配置 | 已验证 | `.env` 与 `panel-state.json` 摘要一致；原 11 个持久文件逐项不变，仅新增运行中的 SQLite WAL/SHM | 无数据迁移 |

## 自动门禁

- canonical L3：`v0.97.1-b18a8aa-l3-r2`，exit 0；Go 全包、核心 race、vet、Web 121 文件/1016 项、i18n 2581/21、typecheck、production build、双架构构建、安装安全、应用生命周期、govulncheck、npm audit、Trivy source/image 全绿。
- bundle SHA-256=`4da99da14a17c2319ab5d99447277089ee9f220d1e8d4d29a5133b707afda048`；L3 日志 SHA-256=`c9b7f3af5599ec45b00da3dfe46795d451c31f89783a7dde8235d33cd38f2326`；manifest SHA-256=`f192951f47f6e3401518819e06c8053de284aa43b24c45d6a8495fb003e53ce0`。
- 固定 Runner：`kpanel-release-gate:go1.26.6-node24`，image ID=`sha256:b593c0ffe32e80a6a6ae8fcac38ed916587dbf698a6b6a55fa33887298737148`。
- 候选 CI `32722395074`、Dependency freshness `32722395084`；main CI `32722638115`、Dependency freshness `32722637988`；Release workflow `32723089507`、Tag Dependency freshness `32723089419`，均 completed/success 且绑定精确 SHA。
- 首轮 `2b8a3a0` L3 虽通过，但真机服务日志触发 6 秒超时；该证据因产品 SHA 变化作废，最终发布仅使用 `b18a8aa` 的 r2 证据。

## arena-154 真机与浏览器验收

- 隔离正式 Agent/Panel 使用真实宿主 journal；系统摘要、服务 50 行、warning 200 行、系统 100 行均返回 200，旧 `unit` 参数返回 422，证明服务选择器契约已移除。
- 大 journal 的 `journalctl --unit=*.service` 实测超过旧 6 秒边界，修复后 Panel API 正常完成；隔离容器与 systemd unit 已清理，受限原始证据保存在 `/root/kpanel-release-evidence/v0.97.1-candidate-e2e-final`。
- 独立正式 Google Chrome `151.0.7922.172`、后台 headless、临时 Context；未读取或接管用户日常浏览器。
- 日志弹窗桌面与 390×844 均无横向溢出；服务输出包含 `.service`，输入 `ssh` 后结果非空，目标 API 全部 200，page error 0。
- 体检报告实际渲染性能/网络两段；桌面 `clientWidth=1365`、`scrollWidth=1355`，手机 `clientWidth=scrollWidth=390`，卡片与信息层级协调。
- 浏览器聚合证据 SHA-256=`7b7bf7d726078a96737d2547e24f8f11b4a1ba9d09b14edaf5ca2da46587bffb`；截图位于 `C:\GitHub\_release-artifacts\v0.97.1-b18a8aa-l3-r2`。

## 发布产物

- GitHub Release：[v0.97.1](https://github.com/kejilion/KPanel/releases/tag/v0.97.1)，非 draft、非 prerelease，8 个附件；annotated Tag object=`f189eac1a99154130f908ed05f772dd37cf1df80`，peel=`b18a8aa0da3ebcf6ae52cec7659d4e3396a847fb`。
- Docker `0.97.1` 与 `latest` OCI index：`sha256:536db80fd6c5699e6c7bc98a2f4b4c8ad206fd21b522d2be5cf37f54dd9141e5`。
- `linux/amd64`=`sha256:8c706567ea5b93699649ec80885365dd7eb95c54bbc41b93544ad775776f8be4`；`linux/arm64`=`sha256:b1495d246365daa827ed71299f222bd1d825cb25457080a40923c8dd41a4463a`。
- OCI labels：version=`0.97.1`、revision=`b18a8aa...`；managed script revision=`9fec61b50cc6ef798dfac1edf11c2ec60ca6b0d1`、SHA-256=`54ceb0e72c4c342382500fc35da636fa436c484a12c4766fb9c7f806a23ae8fa`。
- arena-154 从公开仓库重新拉取并运行 `image-e2e.sh`，输出 `image_e2e=pass`；安装契约无差异，不创建 apps 空提交。

## 生产部署与回滚

- 目标仅 `arena-154`；108 未连接、未测试、未备份、未部署。
- 部署前：Panel `0.97.0` healthy、Agent active、restart=0、OOM=false；旧 OCI=`sha256:ae60f409410af74153f20d0e47265ff59e80f4a6938ca1018d074609136d6410`。
- 停写一致性备份：`/root/kpanel-backups/pre-v0.97.1-20260824T115125Z`；`SHA256SUMS` SHA-256=`ea2a67474883665d38ae73df46f235215c51df6463b49950d7d55b33b77f9c2e`。状态目录、旧 OCI、Compose、`.env`、Agent 文件均完成解包逐文件比对、旧镜像加载与旧版重启核验。
- 标准更新入口一次成功：`KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update bash /home/docker/kpanel/bin/kejilion.sh app kpanel`；日志 SHA-256=`8ca7cbd77fc0b35d215f8ceb2d7e0b4e478f68a38aa0dfe9d5bda33f9f05644d`。
- 部署后：Panel `0.97.1`、revision/OCI 精确匹配、Agent active、Panel healthy、restart=0、OOM=false；公网 health 返回 `0.97.1`，日志无 panic/fatal/OOM。
- 宿主 `kejilion-agent` 与镜像逐字节一致；宿主 `kejilion.sh` 仅保留既有 `permission_granted` 本地状态，归一化后与镜像固定脚本逐字节一致，未覆盖用户配置。
- 回滚：停写并校验备份 `SHA256SUMS`；加载 `old-image.tar.zst`；成套恢复 `/home/docker/kpanel`、Compose、`.env`、密钥与 Agent 文件；`systemctl daemon-reload` 后启动 Agent/Panel；复核 `0.97.0`、旧 digest、health、restart/OOM 和数据。禁止只换镜像。

## 交付节奏数据

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-08-24T18:45:25+08:00
- 候选冻结时间：2026-08-24T19:05:19+08:00
- 生产完成时间：2026-08-24T19:55:58+08:00
- 提交到生产用时：1.18 小时
- 是否回滚、紧急热修复或重复发布：否
- 若发生失败，发现时间、恢复时间和逃逸门禁：不适用
<!-- kpanel-release-metrics:end -->

<!-- kpanel-release-process-metrics:start -->
- 已记录发布流程异常或无效证据拦截次数：4
- 其中生产写操作开始后异常次数：0
<!-- kpanel-release-process-metrics:end -->

### 流程异常明细

<!-- kpanel-release-process-incidents:start -->
[
  {
    "fingerprint": "production-policy/local-preflight/invalid-purpose-name",
    "position": "before-production-write",
    "count": 1,
    "impact": "首次使用不存在的 production-deployment 用途被策略入口拒绝，未连接生产写路径。",
    "recoveryEvidence": "改用规范固定的 production-deploy 与 production-safety-check，两项均通过。",
    "permanentAction": "生产用途参数只从 release-kpanel 当前工作流复制，不凭记忆输入。",
    "historicalReleases": []
  },
  {
    "fingerprint": "candidate-e2e/fixture/preflight-token-missing",
    "position": "before-production-write",
    "count": 1,
    "impact": "隔离 Agent 首次因临时 token 文件缺失在启动前失败，未触碰生产。",
    "recoveryEvidence": "夹具先创建 0640 root:kejilion-panel 随机 token，随后真实 Panel→Agent 门禁通过。",
    "permanentAction": "候选 E2E 夹具将 token 所有权与权限加入启动前 fail-closed 检查。",
    "historicalReleases": []
  },
  {
    "fingerprint": "visual-gate/in-app-browser/control-timeout",
    "position": "before-production-write",
    "count": 1,
    "impact": "内置浏览器控制 kernel 在导航等待时重置，未形成产品失败结论。",
    "recoveryEvidence": "切换到后台独立正式 Chrome 临时 Context，同 SHA 完成桌面与 390px 验收。",
    "permanentAction": "控制通道连续超时后直接使用隔离正式 Chrome，不接管用户 Profile。",
    "historicalReleases": ["v0.97.0"]
  },
  {
    "fingerprint": "tag-preflight/version-check/wrong-script-path",
    "position": "before-production-write",
    "count": 1,
    "impact": "Tag 命令批次误调用不存在的 .mjs 路径；PowerShell 未因该子命令失败而停止，但 Tag 仍精确指向已通过 L3 的发布提交。",
    "recoveryEvidence": "立即使用 scripts/check-version-consistency.sh 复核通过，并核对 Tag peel 与 HEAD 完全一致。",
    "permanentAction": "Tag 前置命令启用显式退出检查，并只调用仓库现有 shell 入口。",
    "historicalReleases": []
  }
]
<!-- kpanel-release-process-incidents:end -->

## 遗留风险

- 服务日志搜索只在本轮有界读取结果中筛选，不是 journal 全文索引；大 journal 读取由 15 秒 Agent 超时和单并发 gate 共同限制。
- 体检结果在不同出口、磁盘和 CPU 状态下会变化；本次验证的是报告信息架构、响应式几何和真实候选渲染，不将等待检测值描述为性能结论。
