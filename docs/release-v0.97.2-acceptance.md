# KPanel v0.97.2 发布验收记录

日期：2026-08-24

发布级别：L3

候选提交 / 标签：`687c3f061d9a17f0b4ae1f37966a47a497c2e9de` / `v0.97.2`

上一稳定版本 / 回滚点：`v0.97.1` / `sha256:536db80fd6c5699e6c7bc98a2f4b4c8ad206fd21b522d2be5cf37f54dd9141e5`

## 发布画像

- 业务域：KPanel 应用市场安装生命周期、系统日志管理界面。
- 变更面：安装、更新、卸载和失败清理在比较 systemd unit 与应用目录时同时规范化真实路径，兼容 `/home/docker` 为 NAS 挂载或符号链接的拓扑；系统日志增加稳定严重级别、消息层级、字面量搜索高亮和更清晰的元数据呈现。
- 不变边界：不修改 API、Agent 协议、权限模型、端口、数据 schema、受管 `kejilion.sh` 或 Compose 网络；系统日志查询与清理语义不变。
- 风险等级：中；安装契约改变必须验证普通与符号链接拓扑、应用市场同步、失败回滚和生产配置保持，日志改动必须验证特殊字符搜索及桌面/手机几何。

## 精确提交范围

- `fe7fa40`：重放来源提交 `068d2f23dd11cbd48b3053aca5e18b602e8627d6`，规范化受管 unit 与应用目录两侧真实路径。
- `cc28d6c`：重放来源提交 `729f35b2430f5af245c5f4c8df156aa5051f9eef`，改善系统日志严重级别、层级与字面量高亮。
- `687c3f0`：准备 `0.97.2` 版本与 CHANGELOG。
- 未纳入旧工作树、未提交草稿、其他会话内容或 108 环境。

## 多维质量结论

| 维度 | 状态 | 证据 | 边界 |
| --- | --- | --- | --- |
| 业务正确性 | 已验证 | 应用配置 lifecycle 覆盖普通及 `/home/docker` 符号链接拓扑；生产 Agent 系统日志 summary/entries 均返回 200 | 生产宿主当前为普通目录，未把 NAS 文件系统性能写成已验证 |
| 网络与供应链安全 | 已验证 | govulncheck 可达漏洞 0、npm audit 0、Trivy source/image/config 0、OCI revision/version 固定 | 无新增匿名入口、出站链路或权限 |
| 稳定性与兼容 | 已验证 | canonical L3、候选/main/Tag 门禁、公开 OCI E2E、apps 生命周期、停写备份与标准更新全绿 | NAS 兼容结论限于真实路径等价与 lifecycle，不替代所有 NAS 厂商矩阵 |
| 性能与资源 | 已验证 | 生产 5 次采样 CPU 0.01–0.03%、内存 74.37 MiB/256 MiB、7 PIDs，restart=0、OOM=false | 本版无性能算法变化 |
| 用户体验与可访问性 | 已验证 | Codex 后台内置浏览器；桌面与 390×844；严重级别区分、点号字面匹配 6 处高亮、移动端无横向溢出 | UI 证据为精确候选 mock 数据，不冒充生产日志内容 |
| 数据与配置 | 已验证 | `.env`、Compose、Agent 配置、密钥、Panel 状态与集群身份逐项保持；SQLite `quick_check=ok` | 无数据迁移 |

## 自动门禁

- canonical L3：`v0972-687c3f0-20260824`，exit 0；Go 全包、核心 race、vet、Web 121 文件/1017 项、i18n 2581/21、typecheck、production build、双架构构建、安装安全、普通/符号链接应用生命周期、govulncheck、npm audit、Trivy source/image/config 全绿。
- bundle SHA-256=`7e875cf56707cde46269a5fbfd168695b4870e826b619ee9522a8af2806b4396`；L3 日志 SHA-256=`0bc5f76c62ff2abd3db6d8ad26bc7567818ebcc1240352f163f759f2780da539`；manifest SHA-256=`4015c69b7035a16b31b602e593189fe0679676e06bacc405fca2ff65b3ddd050`。
- 固定 Runner：`kpanel-release-gate:go1.26.6-node24`，image ID=`sha256:b593c0ffe32e80a6a6ae8fcac38ed916587dbf698a6b6a55fa33887298737148`；preflight 的 Go、gcc、Buildx、Node 与 npm 全部通过且无宿主 PATH 覆盖。
- 候选 CI `32732215691`、Dependency freshness `32732215795`；main CI `32732714692`、Dependency freshness `32732714783`；Release workflow `32733190548`、Tag Dependency freshness `32733190550`，均 completed/success 且绑定精确 SHA。
- Windows 本地 change-aware 在治理、版本和 Web 阶段通过后按设计因缺少 Go/gofmt 停止；未将该环境边界写成完整本地门禁通过，Linux 固定 Runner 提供权威 L3。

## arena-154 隔离真机与浏览器验收

- 精确候选受限容器连接真实生产 Agent socket；重新创建临时数据并完成 bootstrap，`/api/v1/system/logs/summary` 与 `/api/v1/system/logs` 均为 200，Panel/Agent 链路通过；临时容器、数据和凭据已清理。
- 应用配置 lifecycle 在一次性 root 容器中完成 install/update/uninstall/negative path，并额外把 `/home/docker` 设为符号链接，输出 `app_conf_lifecycle=pass`。
- Codex 后台内置浏览器打开精确 clean 候选 mock：桌面截图、390×844 截图与同目标几何均已保存；`innerWidth=clientWidth=scrollWidth=390`，日志弹窗 right=390，无页面横向溢出。
- 点号搜索按字面量匹配 3/3 条并产生 6 个 `.` 高亮；INFO 与 WARNING 使用不同语义色，消息、来源、时间层级清楚。
- 浏览器证据：`system-logs-desktop.png` SHA-256=`fc767238e4262acfe9a45af8bcd48316ee8464339383a07f6d0f7e5a4775497b`；`system-logs-mobile-390x844.png` SHA-256=`61aee6aacc1d0ee1595bf48c1caec7beaae8cebe92e5f32478f5e0d568c0d704`；布局指标 SHA-256=`da8f3d5792312bc70ff8ec1b2fc7edc7d0f54c22ab4804a6d0cd89273a0e9ff2`。

## 发布产物

- GitHub Release：[v0.97.2](https://github.com/kejilion/KPanel/releases/tag/v0.97.2)，非 draft、非 prerelease，8 个附件；annotated Tag object=`19e353e3eed5050382f28d46e539c5246609bd14`，peel=`687c3f061d9a17f0b4ae1f37966a47a497c2e9de`。
- Docker `0.97.2` 与 `latest` OCI index：`sha256:d94605a00bdf2d4a3a73acc23549ca7851f1876b2ba581b55e5a369e7d7fd7c1`。
- `linux/amd64`=`sha256:f7e6d18342c192997413f375831d7a6bc9ac382a87dc150eaa83837464ca5895`；`linux/arm64`=`sha256:97715ab80e78ffefa6c3f9f4328db26e72ed89f32a94e5a57f0bf884f459e930`。
- OCI labels：version=`0.97.2`、revision=`687c3f0...`；managed script revision=`9fec61b50cc6ef798dfac1edf11c2ec60ca6b0d1`、SHA-256=`54ceb0e72c4c342382500fc35da636fa436c484a12c4766fb9c7f806a23ae8fa`。
- arena-154 从公开仓库重新拉取并执行正式 `packaging/tests/image-e2e.sh`，输出 `image_e2e=pass`；额外受限容器 health、revision、version、restart=0、OOM=false 全部通过并已清理。
- `kejilion/apps` 主线提交=`f07bb5112b776cc9c2384dc1d417e0ba2b825892`；`kpanel.conf` Git blob 与 KPanel 打包配置均为 `abf0efd22876f34aa3731f5b6d8ba04e373b965e`，公开文件 SHA-256=`7b5b52af0ff20cff4bebf114e747ddf1c82996500f2767ba8d3733217e83121c`。

## 生产部署与回滚

- 目标仅 `arena-154`；108 禁用全部 KPanel 操作，本次未连接、未测试、未备份、未部署、未核对。
- 部署前：Panel `0.97.1` healthy、Agent active、restart=0、OOM=false；旧 OCI=`sha256:536db80fd6c5699e6c7bc98a2f4b4c8ad206fd21b522d2be5cf37f54dd9141e5`。
- 停写一致性备份：`/root/kpanel-backups/pre-v0.97.2-20260824T134520Z`；`SHA256SUMS` SHA-256=`b6c321c0a54107b6c336b382ed4f879192aeadcdd3468b47488420ab41062f38`。状态目录、旧 OCI、Compose、`.env`、Agent 文件和 apps 配置均完成压缩校验、tar 逐项比较、旧镜像加载与旧版重启核验。
- 标准更新入口一次成功：`KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update bash /home/docker/kpanel/bin/kejilion.sh app kpanel`；日志 SHA-256=`8b63f3d9de1be0b0f1bd42b7ed52548f4b06d1da255c39e0beb6354dc190e74f`。
- 部署后：Panel/Agent 均为 `0.97.2`，revision/OCI 精确匹配、Agent active、Panel healthy、restart=0、OOM=false；公网 health 返回 `0.97.2`，近 10 分钟 Panel/Agent 日志无 panic/fatal/OOM。
- 宿主 `kejilion-agent` 与正式镜像逐字节一致；宿主 `kejilion.sh` 仅保留既有 `permission_granted` 本地状态，规范化后与镜像固定脚本一致；用户 `.env`、Compose、Panel 状态、密钥和集群身份未被更新覆盖。
- 回滚：停写并校验备份 `SHA256SUMS`；加载 `old-image.tar.zst`；成套恢复 `kpanel.tar.zst`、apps `kpanel.conf`、Compose、`.env`、密钥与 Agent 文件；`systemctl daemon-reload` 后启动 Agent/Panel；复核 `0.97.1`、旧 digest、health、restart/OOM 和数据。禁止只换浮动 `latest`。

## 交付节奏数据

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-08-24T20:57:59+08:00
- 候选冻结时间：2026-08-24T20:58:59+08:00
- 生产完成时间：2026-08-24T21:48:45+08:00
- 提交到生产用时：0.85 小时
- 是否回滚、紧急热修复或重复发布：否
- 若发生失败，发现时间、恢复时间和逃逸门禁：不适用
<!-- kpanel-release-metrics:end -->

<!-- kpanel-release-process-metrics:start -->
- 已记录发布流程异常或无效证据拦截次数：13
- 其中生产写操作开始后异常次数：1
<!-- kpanel-release-process-metrics:end -->

### 流程异常明细

<!-- kpanel-release-process-incidents:start -->
[
  {
    "fingerprint": "release-operator/manual-command/entrypoint-path-argument-drift",
    "position": "before-production-write",
    "count": 7,
    "impact": "若干手工预检使用错误测试路径、旧脚本名、不支持参数、错误工作目录、错误用途名或带 v 的 OCI 标签，均在形成有效证据前失败，没有修改产品或生产。",
    "recoveryEvidence": "逐项改用仓库现存入口与 release-kpanel v2.10 的当前参数；精确 SHA 的 L3、浏览器、CI、Release、公开 OCI 和生产门禁随后全部通过。",
    "permanentAction": "后续发布命令只从当前已渲染工作流复制；不再凭历史命令或目录名称拼接预检。",
    "historicalReleases": ["v0.97.1"]
  },
  {
    "fingerprint": "release-operator/remote-shell/quoting-or-health-probe-drift",
    "position": "before-production-write",
    "count": 3,
    "impact": "一次远端 inbox 变量被本机 PowerShell 提前展开，两次公开镜像夹具使用了不正确的引号或健康路径；均未进入生产更新。",
    "recoveryEvidence": "改为远端单引号脚本、精确 `/api/v1/health` 和正式 `image-e2e.sh` 后通过，临时容器与目录已清理。",
    "permanentAction": "复杂远端操作使用经 bash 语法检查的临时脚本或项目固定入口，禁止在多层 shell 中内联 JSON/grep 引号。",
    "historicalReleases": []
  },
  {
    "fingerprint": "candidate-e2e/auth-evidence/stale-session-marked-pass",
    "position": "before-production-write",
    "count": 1,
    "impact": "首次候选真链路复用已失效 cookie 得到 401，但错误地先写入 pass 标记；人工读取响应键时立即识别并作废该证据。",
    "recoveryEvidence": "删除凭据证据，重新创建全新候选数据、bootstrap 和 cookie；summary/entries 精确返回 200，凭据及临时容器随后清理。",
    "permanentAction": "E2E 只有在 HTTP 状态和 JSON 契约断言全部完成后才能原子写入成功标记。",
    "historicalReleases": []
  },
  {
    "fingerprint": "release-monitor/github-api/upstream-504",
    "position": "before-production-write",
    "count": 1,
    "impact": "GitHub 公共 API 在 Release 监控期间连续返回 504，只影响状态读取，没有重触发或修改发布。",
    "recoveryEvidence": "使用公开 Actions 运行页核对进行中状态，API 恢复后再次确认 Release 与依赖门禁 completed/success。",
    "permanentAction": "保留 API 与公开运行页双只读核对；上游恢复前禁止重跑发布。",
    "historicalReleases": []
  },
  {
    "fingerprint": "release-final-check/git-remote/wrong-repository-url",
    "position": "after-production-write",
    "count": 1,
    "impact": "最终远端只读核对首次在无仓库上下文中直接使用 GitHub SSH URL，因本机默认 SSH key 不匹配而失败；没有远端写入，也未影响已健康运行的生产。",
    "recoveryEvidence": "立即改为 KPanel 与 apps 各自已配置的 origin，确认 KPanel main、Tag peel、apps 线性包含本轮提交及生产版本均正确。",
    "permanentAction": "所有最终 Git 核对必须在目标仓库上下文内使用该仓库 origin，禁止脱离仓库手写远端 URL。",
    "historicalReleases": []
  }
]
<!-- kpanel-release-process-incidents:end -->

## 遗留风险

- NAS 兼容通过符号链接 `/home/docker` 的完整应用生命周期验证；arena-154 当前为普通目录，没有声称覆盖 Synology、QNAP、TrueNAS 等全部文件系统、ACL 和挂载选项矩阵。
- 系统日志搜索只在本轮有界读取结果中筛选；点号等特殊字符按字面量高亮，不提供正则表达式查询。
