# KPanel v0.92.0 发布验收记录

日期：2026-08-23

发布级别：L3

候选提交 / 标签：`8da419349ed195123a665c46e274f07aa295407a` / `v0.92.0`

上一稳定版本 / 回滚点：`v0.91.0` / `sha256:ada4847d90ed8430340abbafc4d15d4227025146a8c420b54d00b87e4f24b097`

## 发布画像

- 业务域：文件管理与安全远程下载。
- 变更面：Panel 受限出站客户端、Panel/Agent 文件流协议、文件管理前端、三语文案、安全策略与文档。
- 受影响用户旅程：在文件管理中输入公开 HTTP(S) URL，观察连接/传输/确认进度，将内容原子写入所选目录，取消或处理失败。
- 未变化契约：应用市场配置、端口、Compose、Agent root 权限、受管 `kejilion.sh` revision/hash 均未变化。
- 风险等级及理由：中高；新增出站网络和大文件流，但采用非 root Panel 拉取、完整 SSRF/DNS 重绑定阻断、严格预算和 Agent 原子落盘。

## 发布范围与未纳入内容

- 新增单文件远程下载；浏览器只向已登录 Panel 提交 URL，Agent 不接触 URL、DNS、Cookie、代理或外网。
- 仅允许公开 HTTP/HTTPS；拒绝 userinfo、私网/回环/link-local/保留地址、混合 DNS 和 HTTPS 降级；最多 5 次重定向、TLS 至少 1.2、响应头 64 KiB、单文件 512 MiB、Panel 并发 2、45 秒读取空闲和 2 小时硬上限。
- 文件名按用户指定、`Content-Disposition`、`download` 的顺序确定；不从 URL path 泄漏名称；同名目标不覆盖，Agent 继续使用同目录隐藏临时文件、`fsync` 和 no-replace 原子提交。
- 安全修复补齐 Azure WireServer `168.63.129.16`、IPv4-mapped 地址和 IANA 特殊用途 IPv4/IPv6 前缀；UI 补齐明文 HTTP 提示与校验错误的动态 `aria-describedby`。
- 精确功能提交为 `9c1e64f`、`2a79fa4`，版本准备为 `8da4193`；旧的 `9c1e64f` 单提交候选及其载体已作废，未纳入未审查工作树、108 或应用市场空提交。

## 多维质量结论

| 维度 | 状态 | 证据 | 未验证风险或不适用依据 |
| --- | --- | --- | --- |
| 业务正确性与双端互通 | 已验证 | Go/Web 全量、真实 Panel→Agent、显式名/响应名/default、重复名、取消、512 MiB 边界、并发闸门 | 生产不下载真实外部文件；写旅程在 arena-154 隔离实例完成 |
| 网络入侵与供应链安全 | 已验证 | direct/DNS/mixed/302、Azure/IANA/映射地址 socket 前阻断、TLS/重定向/头部/大小限制、govulncheck、npm audit、Trivy | 公网来源本身的内容可信度仍由管理员判断 |
| 稳定性、失败恢复与兼容 | 已验证 | partial/超限/取消临时文件清理、闸门恢复、原子 no-replace、旧版停写备份和完整恢复检查 | 请求不是持久后台任务，页面关闭会按设计取消 |
| 性能与资源预算 | 已验证 | 512 MiB 精确边界、512 MiB+1 拒绝、并发 2/第三个 429、生产三次资源采样 | 大文件仍占用 Panel 出站带宽，受并发和大小上限约束 |
| 用户体验与可访问性 | 已验证 | 正式 Chrome、成功/失败/明文提示、焦点回归、三语、明暗、390/768/1024/1280 与 100/125/200% | 390 物理像素配 125% 时项目既有 `min-width:320px` 形成 8px shell 基线溢出，功能弹窗自身未裁切 |
| 数据、配置与迁移 | 已验证 | 停写备份独立解包、20 个 JSON、2 个 SQLite、Compose/`.env`/Agent 文件与旧 OCI 归档 | 本版不新增持久 job registry 或应用市场配置 |

## 自动门禁

- Git bundle：`kpanel-v0.92.0-8da4193.bundle`，SHA-256=`89f9c94db542906fee4d3a0ec25ddf969c03084a4d7f42ff7b8d31a023e610fd`。
- arena-154 固定 Runner `kpanel-release-gate:go1.26.6-node24` / `sha256:b593c0ffe32e80a6a6ae8fcac38ed916587dbf698a6b6a55fa33887298737148` 完整 L3 exit 0；Go 全包、core race、vet、双架构、Web 111 文件/883 项、i18n 2443/21、typecheck、生产构建、govulncheck、npm audit、Trivy source/image、受管脚本、安装安全和应用生命周期均通过。
- L3 日志 `/root/kpanel-release-evidence/v0.92.0-8da4193-r2/l3-verify-release.log`，SHA-256=`f6668c08f069fe8ac458f1c3a66372849c8ca8f3685fdf6e7beaa320f20933d8`；远程下载专项 Linux race 日志 SHA-256=`a94c74cb7e4db62867576b3694252a0d680293b0bd6e8a925c475176dd162810`。
- 候选 CI `32586028748`、候选依赖新鲜度 `32586028761`、main CI `32586290165`、main 依赖新鲜度 `32586290196`、Tag 依赖新鲜度 `32586581011` 均 success，head SHA 精确匹配 `8da4193...`。
- Release workflow `32586580926` success；Release、双架构 OCI、`latest` 提升、SBOM/provenance 和公开回拉步骤均绑定精确候选。

## 隔离真机与浏览器验收

- arena-154 真实 Panel→Agent 覆盖公开首跳、公开 302、显式/响应/default 名称、重复名不覆盖、partial/编码/头部/声明超限、精确 512 MiB、流式 512 MiB+1、并发 2/第三个 429、取消及闸门恢复；`remote_download_e2e=pass`，证据清单 SHA-256=`1eacdcc0c7b0e86848b8fd51c82f39bad7193f7b3b3ed5f75a13a844ebb5f9a8`。
- Azure WireServer、IPv4-mapped WireServer、IANA IPv4/IPv6、私网字面量和重定向均在第二个 socket 建立前 fail-closed；隔离 Panel/Agent restart=0、OOM=false，测试容器、网络和 r2 数据目录已清理。
- 正式 Google Chrome 151.0.7922.172 使用独立临时 Profile；成功/失败、HTTP warning、动态可访问性描述、焦点、三语、明暗主题及多视口/缩放通过，console error=0；报告 SHA-256=`61a5eda1f02a4a4f238b9066ed0ceb1f4e61f27b093c1cc7ec0b4663f5acb8ec`。
- 标准 mock/acceptance 预览绑定 clean `8da4193...`，manifest SHA-256=`5cde6c1208253d2ad77fc6fa082ea0121f262a387822971f3ace9df563d26b3e`；验收结束后已停止，URL 不再提供。
- 生产只执行 health、未登录接口 fail-closed、数据与配置完整性检查；未执行外部下载、512 MiB、慢流、取消或并发写验收。

## 发布产物与公开仓库复核

- GitHub Release：[v0.92.0](https://github.com/kejilion/KPanel/releases/tag/v0.92.0)，非 draft、非 prerelease，Tag 解引用精确指向 `8da4193...`。
- Docker `0.92.0` 与 `latest` OCI index 均为 `sha256:3daedf937586c3ca5fde46c8c89b53feebf61dbd081dcc75e65f98117b3a708c`。
- `linux/amd64`=`sha256:d93735b79aed361a4ca9ceadb9cf6346a5356f67fda0ba7639a0f15fd1e28d8f`；`linux/arm64`=`sha256:a9e0446b5c253a30b75b950fc2799c442ebe51e18a0c52edc648429f271b187d`。
- arena-154 独立公开回拉验证 version=`0.92.0`、revision=`8da4193...`、内置 `kejilion.sh` SHA-256=`17c1544b826c45f070e49df2f71a5e152fedc922a1de201da5bfa0393d250a4d`、非 root、受限容器和健康检查通过。
- `packaging/kejilion-app/kpanel.conf` 相对 v0.91.0 零差异；生产 `kejilion/apps@6d86eee24a477320f4d8ffb32d9e85b785cf3c2c` 工作树干净，无应用市场提交。

## 生产部署安全核对

- 唯一生产目标为 arena-154；`prod-108` 本次未连接、未读取、未测试、未备份、未部署。
- 部署前 v0.91.0 healthy/active、restart=0、OOM=false；新停写一致性备份为 `/root/kpanel-backups/v0.92.0-preupgrade-arena154-20260822T171605Z`，`SHA256SUMS` SHA-256=`1dc0f6c83b64a8bda6e80a30e2218c7cc6cc79e8cd0c3e8bbe787c8215bcd345`。
- 备份已独立解包并比较 Compose、`.env`、Agent unit/token，验证 20 个 JSON、2 个 SQLite 与旧 OCI 归档，随后重新启动 v0.91.0 healthy 才进入升级。
- 使用 `KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update bash /home/docker/kpanel/bin/kejilion.sh app kpanel` 标准入口升级；更新日志 SHA-256=`78c9b9d27b8d19290c203f6ed51f9bcd2978bdaac3d31592482e303a84cff7d6`。
- 部署后 Panel v0.92.0 healthy、Agent active、restart=0、OOM=false、NRestarts=0、NeedDaemonReload=no；未登录远程下载请求返回 403，公网 health 为 200。
- `.env`、Compose、Agent 配置和 service 文件组合哈希升级前后相同；20 个 JSON、1 个非空 SQLite 完整，生产证据清单文件 SHA-256=`e1ab0ad2abefd0598875080a548d49bd23bc6200230f5d45387348d7f41b45f3`。
- 三次采样内存 74.43～74.49 MiB/256 MiB、7 PIDs，稳定样本 CPU 0.02%～0.04%；首个紧邻验收启动的样本为 4.35%，随后恢复稳定。

## 回滚

- 源码/tag：`v0.91.0` / `9321afc3ce51731ffb7b442c127d6441ce83d885`。
- 旧 OCI：`sha256:ada4847d90ed8430340abbafc4d15d4227025146a8c420b54d00b87e4f24b097`。
- 数据/配置备份：`/root/kpanel-backups/v0.92.0-preupgrade-arena154-20260822T171605Z`。
- 回滚必须停写并成套恢复旧镜像、完整 `/home/docker/kpanel`、Compose、`.env`、数据、Agent unit 和二进制；禁止只替换镜像。恢复后核对 Panel/Agent、数据完整性与标准更新入口。
- 备份的独立恢复检查和旧 OCI load 已通过；生产无需回滚，当前保持 v0.92.0 healthy。

## 交付节奏数据

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-08-22T23:55:11+08:00
- 候选冻结时间：2026-08-23T00:19:34+08:00
- 生产完成时间：2026-08-23T01:17:53+08:00
- 提交到生产用时：1.38 小时
- 是否回滚、紧急热修复或重复发布：否
- 若发生失败，发现时间、恢复时间和逃逸门禁：不适用
<!-- kpanel-release-metrics:end -->

<!-- kpanel-release-process-metrics:start -->
- 已记录发布流程异常或无效证据拦截次数：11
- 其中生产写操作开始后异常次数：1
<!-- kpanel-release-process-metrics:end -->

### 流程异常明细

<!-- kpanel-release-process-incidents:start -->
[
  {
    "fingerprint": "local-validation/repo-bash/execute-vs-syntax",
    "position": "before-production-write",
    "count": 1,
    "impact": "一次本地命令执行了验证脚本而非仅做语法检查；脚本在前置条件停止，未改变远端或生产。",
    "recoveryEvidence": "后续脚本固定使用 Git Bash bash -n、SHA-256 和远端 preflight，再执行权威门禁。",
    "permanentAction": "本地发布脚本语法预检固定为绝对 Git Bash 的 bash -n，不借用聚合执行器。",
    "historicalReleases": []
  },
  {
    "fingerprint": "l3-bundle/git-bundle/verify-outside-repository",
    "position": "before-production-write",
    "count": 1,
    "impact": "首轮远端脚本在仓库外执行 git bundle verify，产品门禁开始前即停止。",
    "recoveryEvidence": "r2 在临时 bare repo 中校验同一 bundle，并从零完成 L3 exit 0。",
    "permanentAction": "bundle verify 固定在显式初始化的临时 Git 仓库中执行。",
    "historicalReleases": []
  },
  {
    "fingerprint": "remote-preflight/powershell-ssh/quoting",
    "position": "before-production-write",
    "count": 2,
    "impact": "远端上传摘要和含空格路径的 Python compile 各有一次 PowerShell quoting 预检无效。",
    "recoveryEvidence": "改用本地摘要、独立文件上传、远端 sha256sum 与固定脚本执行，受影响门禁全部重新通过。",
    "permanentAction": "复杂远端命令不使用嵌套内联 quoting，改为 apply_patch 生成并核验独立脚本。",
    "historicalReleases": []
  },
  {
    "fingerprint": "browser-acceptance/playwright/harness-compatibility",
    "position": "before-production-write",
    "count": 5,
    "impact": "正式 Chrome 取证中依次修正预览依赖、旧 Playwright API、焦点检测、locale selector 和基线 min-width 判读；均为验收工具问题。",
    "recoveryEvidence": "最终 r7 在同一独立 Chrome target 完成多语言、主题、视口、缩放、成功/失败和 a11y 验收，console error=0。",
    "permanentAction": "浏览器门禁绑定工具版本，并把 shell 基线债务与功能组件 rect 分栏判定。",
    "historicalReleases": []
  },
  {
    "fingerprint": "remote-download-e2e/concurrency/agent-cleanup-overlap",
    "position": "before-production-write",
    "count": 1,
    "impact": "首轮并发断言与上一超限 Agent 上传清理重叠，一个慢流获 agent_write_busy，第三请求因此合法进入，最初被误判为并发闸门失败。",
    "recoveryEvidence": "r2 隔离前置清理并同步两个有效慢流，第三请求稳定 429，取消后闸门恢复。",
    "permanentAction": "并发门禁先确认两个流均进入有效传输态，再发第三请求。",
    "historicalReleases": []
  },
  {
    "fingerprint": "production-verification/remote-download/route-typo",
    "position": "after-production-write",
    "count": 1,
    "impact": "生产验收脚本将复数正式路由误写为单数，得到 404 后在早期断言停止；服务始终 healthy。",
    "recoveryEvidence": "只修正验收脚本为 /api/v1/files/remote-downloads，未登录返回 403，完整生产验证随后通过。",
    "permanentAction": "生产接口探针从前端 API 定义或路由注册表提取精确路径，避免手工拼写。",
    "historicalReleases": []
  }
]
<!-- kpanel-release-process-incidents:end -->

## 遗留风险与后续准入

- 管理员输入的公网 URL 可能包含敏感 query；产品对日志做脱敏，但管理员仍应避免分享含长期凭据的 URL。
- 下载不是后台持久任务；页面关闭、网络断开或刷新会取消，提交边界附近可能已经原子完成，前端会刷新目录并明确结果。
- 390 物理像素配 125% 缩放会触发项目既有 320px shell 最小宽度的 8px 基线溢出；本功能弹窗本身无裁切，后续应由独立布局治理处理。
- 本版无 P0/P1/P2 遗留项，无需回滚；生产未执行高流量或写入型下载验收，隔离真机与生产只读核对分层记录。
