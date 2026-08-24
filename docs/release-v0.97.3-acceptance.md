# KPanel v0.97.3 发布验收记录

日期：2026-08-24

发布级别：L3

候选提交 / 标签：`4aa1fed823e68648c5c819a0a1df947dcd0a25db` / `v0.97.3`

上一稳定版本 / 回滚点：`v0.97.2` / `sha256:d94605a00bdf2d4a3a73acc23549ca7851f1876b2ba581b55e5a369e7d7fd7c1`

## 发布画像

- 业务域：历史监控布局、Docker 容器分组显示偏好。
- 变更面：仅 Web 展示与当前浏览器本地偏好；不涉及宿主机写入、协议、数据或部署契约变化。
- 受影响用户旅程：打开历史监控时加载态保持紧凑；收起 Compose/独立容器分组后刷新或关闭重开仍保持。
- 未变化契约：API、数据库、端口、Compose、Agent 权限、受管 `kejilion.sh` 和应用市场安装契约均不变。
- 风险等级：低；代码范围小，但必须验证经典/桌面布局、损坏/禁用存储回退以及生产静态资产确实来自精确 OCI。

## 发布范围与未纳入内容

- `cb896c4447e9d9534df02ef396e57822686442ad`：从陈旧共享工作树的聚焦提交 `d4851ed` 提取两文件净差异，避免历史监控加载态被满高 Grid 拉伸。
- `092956d516670b09a8db3e4520fdd40bd0895670`：重放 `739e9849a7d496fa2ffc12ff0b5c754abe687359`，将 Docker 分组收起状态有界保存到当前浏览器。
- `4aa1fed823e68648c5c819a0a1df947dcd0a25db`：准备 `0.97.3` 版本与 CHANGELOG。
- 未纳入共享旧 `main` 的其他本地提交、其他工作树、未提交草稿或 108 环境。

## 多维质量结论

| 维度 | 状态 | 证据 | 未验证风险或不适用依据 |
| --- | --- | --- | --- |
| 业务正确性与双端互通 | 已验证 | Docker 持久化 4 条回归、Monitoring 布局回归、全量 Web 1022 项；生产资产包含固定 storage key 与 `align-content:start` | 本版不改变 Panel/Agent 协议 |
| 网络入侵与供应链安全 | 已验证 | govulncheck 可达漏洞 0、npm audit 0、Trivy source/image/config 0、OCI revision/version 固定 | 无新增网络入口、请求或权限 |
| 稳定性、失败恢复与兼容 | 已验证 | canonical L3、候选/main/Tag 门禁、公开 OCI E2E、停写备份、标准更新和旧版恢复材料均通过 | 浏览器禁用存储时按设计退化为当前页面内状态 |
| 性能与资源预算 | 已验证 | 生产 5 次采样 CPU 0.03–0.18%、内存无单调增长、7 PIDs、restart=0、OOM=false | 确定性 Web 补丁，不执行长时间 soak |
| 用户体验与可访问性 | 已验证 | 精确候选真实浏览器经典模式工具栏 54px、页面自然高度 262px、无横向溢出；分组交互与存储失败回退有自动回归 | 生产浏览器控制通道被客户端策略阻断，未把该通道写成生产交互已验证 |
| 数据、配置与迁移 | 已验证 | Compose、`.env`、Agent env、token 与更新前备份逐字节一致；SQLite quick_check=ok；宿主 Agent 与镜像逐字节一致 | 无数据迁移 |

## 自动门禁

- canonical L3：`v0973-4aa1fed-l3-r2`，exit 0；Go 全包、核心 race、vet、Web 121 文件/1022 项、i18n 2581/21、typecheck、production build、双架构构建、安装安全、应用生命周期、govulncheck、npm audit、Trivy source/image/config 全绿。
- bundle SHA-256=`ebb3f4d5ed6a77f9cb4da2b2afa821e5ce8e39ce349deec7f764ba34da6ce0ac`；L3 日志 SHA-256=`8a9a8993a18bd9dce95d400822d67186bbd58a7aec0ed74b32fbff4faecec283`；manifest SHA-256=`4a1d933e39a43e1a84108bd4f5de1ffe613d2c04c626418db03e0ad61f2b080f`。
- 固定 Runner：`kpanel-release-gate:go1.26.6-node24`，image ID=`sha256:b593c0ffe32e80a6a6ae8fcac38ed916587dbf698a6b6a55fa33887298737148`；远端证据目录=`/root/kpanel-release-evidence/v0973-4aa1fed-l3-r2`。
- 候选 CI `32738472744`、Dependency freshness `32738473134`；main CI `32739007120`、Dependency freshness `32739007118`；Release workflow `32739515509`、Tag Dependency freshness `32739515445`，均 completed/success 且绑定精确 SHA。
- Windows 本地 change-aware 在版本、治理和脚本契约后按设计因缺少 Go/gofmt 停止；Linux 固定 Runner 提供权威 L3，未把 Windows 结果冒充完整门禁。

## 依赖与技术栈变化

- 本版没有依赖、Action、基础镜像、工具链或受管脚本变化；Dependency freshness 三阶段均成功。
- 继续使用 Go 1.26.6、Node 24、固定 Runner 与 `kejilion.sh@9fec61b50cc6ef798dfac1edf11c2ec60ca6b0d1`，脚本 SHA-256=`54ceb0e72c4c342382500fc35da636fa436c484a12c4766fb9c7f806a23ae8fa`。

## 隔离真机与浏览器验收

- 环境：`arena-154`，用途 `candidate-validation`、`production-safety-check` 与 `production-deploy` 均由 `environment-policy.json` 放行；108 未连接。
- 精确候选由 canonical L3 在固定 Linux Runner 完整验证；公开 OCI 从 Docker Hub 重新拉取后执行 `packaging/tests/image-e2e.sh`，输出 `image_e2e=pass`。
- 精确候选本地真实浏览器经典模式：`align-content=start`、工具栏 54px、页面自然高度 262px、`scrollWidth<=clientWidth`；桌面壳和 Docker 页面正常加载。
- 本地浏览器运行时不暴露 `localStorage` 且 Mock Docker 列表为空，因此刷新持久化由冻结 SHA 的回归测试验证；生产部署后通过静态资产、真实 OCI/revision 和服务健康复核，不声称完成了生产登录态点击旅程。
- 本版不涉及生命周期、流式、重连或资源增长算法，无 soak 必要；生产执行 5 次有界资源采样。

## 发布产物与公开仓库复核

- GitHub Release：[v0.97.3](https://github.com/kejilion/KPanel/releases/tag/v0.97.3)，非 draft、非 prerelease，8 个附件；annotated Tag object=`5d8fce8ff071bacff2d59adaf389dafb1838e2a6`，peel=`4aa1fed823e68648c5c819a0a1df947dcd0a25db`。
- Docker `0.97.3` 与 `latest` OCI index：`sha256:1d1e5fe884c9580ac7ce357397d78323ca7905b8e1d13f8367c25091d2f6ad49`。
- `linux/amd64`=`sha256:c348807621eaff42a68ce288c8446f0ca0d49cdb2cbc8c7b7f3e92e74e5d4b82`；`linux/arm64`=`sha256:fcbacfc7e28e0210f0216453df6c27bbd2709ddf24a982fb7c38ea48ca7262b7`。
- Release 附件含双架构 Agent/Node、部署包、LICENSE、THIRD_PARTY_NOTICES 和 SHA256SUMS；公开镜像 `image_e2e=pass`。
- `packaging/kejilion-app/kpanel.conf` 相对 `v0.97.2` 无差异，无需 apps 提交；生产 `/root/apps` 因独立 AIStudioToAPI 更新前移到 `2d8044adec98e3eb16f47cdbb297f6be9632a66f`，KPanel 配置 SHA-256 仍为 `7b5b52af0ff20cff4bebf114e747ddf1c82996500f2767ba8d3733217e83121c`。

## 生产部署安全核对

- 唯一目标为 `arena-154`；108 禁用全部 KPanel 操作，本次未连接、未备份、未部署、未升级、未核对。
- 部署前 Panel `0.97.2` healthy、Agent active、restart=0、OOM=false，旧 OCI=`sha256:d94605a00bdf2d4a3a73acc23549ca7851f1876b2ba581b55e5a369e7d7fd7c1`。
- 停写一致性备份：`/root/kpanel-backups/pre-v0.97.3-20260824T144503Z`；`SHA256SUMS` SHA-256=`49727dc04abd61c1b7177c1108b79cfe6c6f7555718724b10f049ab882af0b15`。归档校验、解包逐项比较、旧镜像加载及旧版重启均通过。
- 标准入口：`KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update bash /home/docker/kpanel/bin/kejilion.sh app kpanel`。应用输出到达 Update Complete 和 100%；旁路 `tee` 证据写入失败，未重复更新，改由权威状态与固定脚本收口。
- 部署后 API health=`0.97.3`；Panel healthy、Agent active/enabled、NeedDaemonReload=no、restart=0、OOM=false；OCI/revision 精确匹配，近 10 分钟日志无 panic/fatal/OOM。
- 生产静态资产核对通过：`DockerView-hZzwKpZP.js` 含有界存储 key，`MonitoringView-gIFocTKK.css` 含 `align-content:start`；受保护配置逐字节不变、SQLite quick_check=ok、Agent 二进制与镜像一致。

## 回滚

- 源码/tag：`v0.97.2`；镜像 digest：`sha256:d94605a00bdf2d4a3a73acc23549ca7851f1876b2ba581b55e5a369e7d7fd7c1`。
- 数据/配置备份：`/root/kpanel-backups/pre-v0.97.3-20260824T144503Z`。
- 回滚步骤：停写并校验 SHA256SUMS；加载 `old-image.tar.zst`；成套恢复 `kpanel.tar.zst`、apps `kpanel.conf`、systemd unit、Compose、`.env`、密钥、数据和 Agent 文件；daemon-reload 后以 `--pull never` 启动旧 OCI并复核 `0.97.2`、health、restart/OOM 与 SQLite。
- 未执行生产回滚；部署前已验证备份可解包、旧镜像可加载且旧版可健康重启。GitHub Latest、Docker `latest` 与标准更新入口当前保持 `v0.97.3`。

## 交付节奏数据

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-08-24T21:43:51+08:00
- 候选冻结时间：2026-08-24T22:08:32+08:00
- 生产完成时间：2026-08-24T22:49:46+08:00
- 提交到生产用时：1.10 小时
- 是否回滚、紧急热修复或重复发布：否
- 若发生失败，发现时间、恢复时间和逃逸门禁：不适用
<!-- kpanel-release-metrics:end -->

<!-- kpanel-release-process-metrics:start -->
- 已记录发布流程异常或无效证据拦截次数：9
- 其中生产写操作开始后异常次数：3
<!-- kpanel-release-process-metrics:end -->

### 流程异常明细

<!-- kpanel-release-process-incidents:start -->
[
  {
    "fingerprint": "release-operator/local-preflight/entrypoint-and-sha-drift",
    "position": "before-production-write",
    "count": 2,
    "impact": "一次调用不存在的版本脚本，一次把短 SHA 错误展开为另一完整值；均在候选执行前 fail-closed，没有修改远端或生产。",
    "recoveryEvidence": "改用仓库真实版本入口与 git rev-parse 的完整 SHA；新 run-id 的 canonical L3 从零通过。",
    "permanentAction": "候选完整 SHA 与入口只从冻结 manifest 读取，不再手工扩展短 SHA或猜测脚本名。",
    "historicalReleases": []
  },
  {
    "fingerprint": "release-operator/local-preview/policy-blocked-launch",
    "position": "before-production-write",
    "count": 1,
    "impact": "首次后台 Start-Process 预览命令被执行策略拒绝，未启动进程或改候选。",
    "recoveryEvidence": "改用统一 exec session 启动并在验收后精确终止，候选浏览器几何完成。",
    "permanentAction": "本地预览只使用项目登记入口或可追踪 exec session，不再拼接后台启动命令。",
    "historicalReleases": []
  },
  {
    "fingerprint": "release-browser/control-channel/unavailable-production-target",
    "position": "before-production-write",
    "count": 2,
    "impact": "生产公网 IP 被浏览器客户端策略阻断，HTTPS sslip.io 控制会话随后超时；未修改生产，生产登录态点击证据未形成。",
    "recoveryEvidence": "保留精确候选浏览器几何与全量自动回归；生产改用 OCI/revision、静态资产、API health 和真实 Docker/Agent 状态核对，并明确证据边界。",
    "permanentAction": "需要登录态生产 UI 的版本须在冻结前登记可控 HTTPS 浏览器入口；不可用时不得伪造交互证据。",
    "historicalReleases": []
  },
  {
    "fingerprint": "release-operator/agent-probe/unsupported-flag",
    "position": "before-production-write",
    "count": 1,
    "impact": "只读预检对 Agent 使用了不存在的 --version 参数，产生一条错误日志但未影响 systemd 服务。",
    "recoveryEvidence": "版本改由标准更新校验、Panel health、OCI label、revision 和宿主/镜像二进制逐字节比较确认。",
    "permanentAction": "Agent 版本只能使用仓库已登记的校验方式，不把 Panel CLI 参数类推给 Agent。",
    "historicalReleases": []
  },
  {
    "fingerprint": "release-operator/remote-shell/quoting-or-health-probe-drift",
    "position": "after-production-write",
    "count": 3,
    "impact": "更新日志 tee、首次健康采样和首次资产核对的内联远端命令被 PowerShell 提前展开；产品更新已成功且服务未退化，但三份证据命令无效。",
    "recoveryEvidence": "没有重复更新；改用经 Git Bash 与远端 bash -n 双重检查的 production-verify、production-sample、production-assets、production-data-check 脚本，全部通过并保留摘要。",
    "permanentAction": "该根因在 v0.97.2 后再次出现；下一次生产写前必须把生产证据脚本固化进仓库唯一发布入口并补跨 PowerShell 回归，否则阻断部署。",
    "historicalReleases": ["v0.97.2"]
  }
]
<!-- kpanel-release-process-incidents:end -->

## 遗留风险与后续准入

- 当前没有产品 P0/P1/P2 遗留；Docker 分组偏好只在当前浏览器保存，不跨设备同步。
- 生产登录态真实浏览器点击未完成；本版以冻结候选浏览器几何、全量回归和生产精确资产核对放行，不把客户端控制通道故障归为产品成功证据。
- 下一次生产写前必须先消除重复的跨 PowerShell/SSH 证据拼接方式，将固定生产证据入口与回归纳入主线。
