# KPanel v0.93.0 发布验收记录

日期：2026-08-23

发布级别：L3

候选提交 / 标签：`66199b7106b8be7b3b532f36454a5628e0732eb9` / `v0.93.0`

上一稳定版本 / 回滚点：`v0.92.0` / `sha256:3daedf937586c3ca5fde46c8c89b53feebf61dbd081dcc75e65f98117b3a708c`

## 发布画像

- 业务域：文件管理与一条龙系统调优脚本。
- 变更面：远程下载后台任务、文件工具栏、目录拖放上传、三语文案、受管 `kejilion.sh` 固定版本。
- 受影响用户旅程：发起远程下载后离开或刷新页面、查看/取消历史任务、拖入本机目录并递归上传、在窄屏使用文件工具栏。
- 未变化契约：应用市场 `kpanel.conf`、端口、Compose、Agent root 权限和生产数据格式均未变化。
- 风险等级及理由：中高；新增持久后台任务与目录递归上传，但任务数、历史、状态文件、路径、并发、审计和敏感 URL 均有界且 fail-closed。

## 发布范围与未纳入内容

- 远程下载新增后台任务：页面刷新/关闭后继续，支持状态恢复、取消、中断恢复、10 个队列上限、100 条历史上限和 256 KiB/0600 状态文件。
- 文件页工具栏压缩并补齐可访问性；“新建文件夹”语义在简中、繁中、英文中统一。
- 桌面和文件管理器拖入目录时递归枚举并保留相对结构；继续执行 no-overwrite、路径校验、数量与大小预算。
- `kejilion/sh@ca91f4d757dab0f16bf3868a6e33e7e3f1d8ed2b` 将镜像源优化失败改为非阻断，真实系统更新失败仍阻断；脚本版本号未因小补丁变更。
- 未纳入共享脏工作树、旧候选、应用市场空提交、108 或与本轮无关的草稿。

## 多维质量结论

| 维度 | 状态 | 证据 | 未验证风险或不适用依据 |
| --- | --- | --- | --- |
| 业务正确性与双端互通 | 已验证 | Go/Web 全量、真实 Panel→Agent 后台下载、刷新恢复、取消、重启中断、队列/历史边界、递归目录拖放 | 生产不执行外部下载或目录写入；写旅程在 arena-154 隔离实例完成 |
| 网络入侵与供应链安全 | 已验证 | 沿用 v0.92 出站安全边界、URL 脱敏、状态文件无敏感路径/query、govulncheck、npm audit、Trivy source/image | 公网来源内容可信度仍由管理员判断 |
| 稳定性、失败恢复与兼容 | 已验证 | Panel 重启 active→interrupted、取消无目标/临时文件、旧任务裁剪、配置升级前后相同 | 后台任务不跨 Panel 进程续传，重启后按设计标记中断 |
| 性能与资源预算 | 已验证 | 队列 10、历史 100、状态 256 KiB、并发沿用受限客户端；生产 73.1 MiB/7 PIDs | 大文件仍占用 Panel 出站带宽，受既有上限约束 |
| 用户体验与可访问性 | 已验证 | 正式 Chrome，390/768/1280、DPR 1/1.25/2、三语、明暗、工具栏与后台任务恢复，无组件横向溢出 | 浏览器不展示系统级文件夹选择时仍受平台能力限制 |
| 数据、配置与迁移 | 已验证 | 停写备份独立解包、20 个 JSON、2 个 SQLite、Compose/`.env`/Agent 文件与旧 OCI 归档；生产升级后 21 个 JSON、1 个非空 SQLite 完整 | 本版新增的后台任务 JSON 为有界可重建状态，不影响旧版核心数据 |

## 自动门禁

- Git bundle：`kpanel-v0.93.0-66199b7.bundle`，SHA-256=`9ca24969d5b47d38a5e0306f4d311d9d481471c093c79b54c303018226b094de`。
- arena-154 固定 Runner `kpanel-release-gate:go1.26.6-node24` / `sha256:b593c0ffe32e80a6a6ae8fcac38ed916587dbf698a6b6a55fa33887298737148` 完整 L3 exit 0；Go 全包、core race、vet、双架构、Web 111 文件/896 项、i18n 2445/21、typecheck、生产构建、govulncheck、npm audit、Trivy source/image、受管脚本、安装安全和应用生命周期均通过。
- L3 日志 `/root/kpanel-release-evidence/v0.93.0-66199b7-r2/l3-verify-release.log`，SHA-256=`456bb10a96fde91612fbe28376e59b642eb5ec4479387826b8f0cc461cc4c464`；本地证据包 `kpanel-v0.93.0-66199b7-evidence-r2.tgz` SHA-256=`6b3e1489954c412fee63afb12acb965fa50f7735faa39276a8aa79917ff38a41`。
- 候选 CI `32617948833`、候选依赖新鲜度 `32617948844`、main CI `32618142512`、main 依赖新鲜度 `32618142508`、Tag 依赖新鲜度 `32618375314` 均 success，head SHA 精确匹配 `66199b7...`。
- Release workflow `32618375321` success；Release、双架构 OCI、`latest` 提升、SBOM/provenance 和公开回拉步骤均绑定精确候选。

## 隔离真机与浏览器验收

- arena-154 真实 Panel→Agent 覆盖后台完成、取消、Panel 重启中断、10/11 队列边界、105→100 历史裁剪、0600/256 KiB 状态文件和 URL/query 脱敏；证据清单 SHA-256=`ccf1bf78e14f319ab21d7fadeadbe964eec66871e1050c44f24b5eb2379c0338`。
- 正式 Google Chrome 151.0.7922.172 使用独立临时 Profile；390/768/1280、DPR 1/1.25/2、三语、明暗、工具栏、后台任务刷新恢复和递归目录拖放通过；报告 SHA-256=`619de5231f38c065f54e9ea6e39663ca5a111e272f02b454fc7413e40806bd03`。
- 隔离 Agent 未挂生产 Docker socket，相关 503 属预期环境边界；业务 console error=0，生产未执行下载或目录上传写验收。
- 测试容器、网络、临时 Profile、隧道、凭据和运行数据已清理；失败轮仅保留脱敏诊断证据。

## 发布产物与公开仓库复核

- GitHub Release：[v0.93.0](https://github.com/kejilion/KPanel/releases/tag/v0.93.0)，非 draft、非 prerelease，Tag 解引用精确指向 `66199b7...`。
- Docker `0.93.0` 与 `latest` OCI index 均为 `sha256:3ffdd29f78cba50d10c2efe2140af46dee2104bc6151d91c6f0031b1449bee2b`。
- `linux/amd64`=`sha256:ac2998f22c9da97c1ace9610d6927cc84eea521111b463a842d03e11b2eeb6f8`；`linux/arm64`=`sha256:de1d6e4408cd3e344f91aa7ed8a2ca9d7d9f5dca5166f6adc2711d012863fc3c`。
- arena-154 独立公开回拉验证 version=`0.93.0`、revision=`66199b7...`、非 root、受限容器、内置 `kejilion.sh` revision=`ca91f4d...`、SHA-256=`7549cb639b1c874c0e4460863ef621af05d7ff008e507b8892a2eedbaaf64a19`；公开证据清单 SHA-256=`3f7ad61a45613d768cfeadc87e7494adad04d93b48fdbe1e771e3e5fcb263c56`。
- `packaging/kejilion-app/kpanel.conf` 相对 v0.92.0 零差异；生产 `kejilion/apps` 工作树干净，无应用市场提交。

## 生产部署安全核对

- 唯一生产目标为 arena-154；`prod-108` 本次未连接、未读取、未测试、未备份、未部署。
- 部署前 v0.92.0 healthy/active、restart=0、OOM=false；新停写一致性备份为 `/root/kpanel-backups/v0.93.0-preupgrade-arena154-20260823T044848Z`，`SHA256SUMS` SHA-256=`86c4cd4d0ea5a4146832cfbc5370fd0b39ca5f22fecdb2185373acf749be5fbf`。
- 备份已独立解包并比较 Compose、`.env`、Agent unit/token，验证 20 个 JSON、2 个 SQLite 与旧 OCI 归档，随后重新启动 v0.92.0 healthy 才进入升级。
- 使用 `KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update bash /home/docker/kpanel/bin/kejilion.sh app kpanel` 标准入口升级；更新日志 SHA-256=`36f24a80463a09840f60e42265e7e1e36b796aaa2ec04ba457e482cc8024692a`。
- 部署后 Panel v0.93.0 healthy、Agent active、restart=0、OOM=false、NRestarts=0、NeedDaemonReload=no；未登录后台下载请求返回 403，公网 health 为 200。
- `.env`、Compose、Agent 配置和 service 文件组合哈希升级前后同为 `d60e21246904d61f3eb67c3b35cb5deb945d987766cf50d72fcd3d5e9b31e6f6`；21 个 JSON、1 个非空 SQLite 完整。
- 三次采样 CPU 0.02%～0.03%、内存 73.1 MiB/256 MiB、7 PIDs，未见单调增长。

## 回滚

- 源码/tag：`v0.92.0` / `8da419349ed195123a665c46e274f07aa295407a`。
- 旧 OCI：`sha256:3daedf937586c3ca5fde46c8c89b53feebf61dbd081dcc75e65f98117b3a708c`。
- 数据/配置备份：`/root/kpanel-backups/v0.93.0-preupgrade-arena154-20260823T044848Z`。
- 回滚必须停写并成套恢复旧镜像、完整 `/home/docker/kpanel`、Compose、`.env`、数据、Agent unit 和二进制；禁止只替换镜像。
- 备份独立恢复检查、旧 OCI 归档校验和旧版重新启动均通过；生产无需回滚，当前保持 v0.93.0 healthy。

## 交付节奏数据

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-08-23T11:54:00+08:00
- 候选冻结时间：2026-08-23T11:57:28+08:00
- 生产完成时间：2026-08-23T12:49:51+08:00
- 提交到生产用时：0.93 小时
- 是否回滚、紧急热修复或重复发布：否
- 若发生失败，发现时间、恢复时间和逃逸门禁：不适用
<!-- kpanel-release-metrics:end -->

<!-- kpanel-release-process-metrics:start -->
- 已记录发布流程异常或无效证据拦截次数：7
- 其中生产写操作开始后异常次数：0
<!-- kpanel-release-process-metrics:end -->

### 流程异常明细

<!-- kpanel-release-process-incidents:start -->
[
  {
    "fingerprint": "git-fetch/local-tag/conflict",
    "position": "before-production-write",
    "count": 1,
    "impact": "本地历史 v0.86.2 标签与远端同名对象冲突，广泛 fetch 被安全停止；远端和产品未改变。",
    "recoveryEvidence": "改用 --no-tags 精确 fetch 与公开 API 核对 main/Tag，未删除或改写用户本地标签。",
    "permanentAction": "发布候选仅精确 fetch 必需 ref，历史本地标签冲突另行治理。",
    "historicalReleases": []
  },
  {
    "fingerprint": "l3-orchestration/image-revision/default-unknown",
    "position": "before-production-write",
    "count": 1,
    "impact": "首轮聚合器用默认 unknown 构建验证镜像，产品测试虽全绿但精确 revision 断言失败。",
    "recoveryEvidence": "从同一冻结 bundle 以 REVISION=66199b7... 重建，重跑脚本契约、镜像扫描和运行标签核验。",
    "permanentAction": "L3 镜像构建显式传入冻结完整 SHA。",
    "historicalReleases": []
  },
  {
    "fingerprint": "l3-orchestration/scratch-image/sha256sum",
    "position": "before-production-write",
    "count": 1,
    "impact": "聚合器试图在 scratch 运行时镜像内执行 sha256sum，因镜像按设计无工具而停止。",
    "recoveryEvidence": "同一 r2 产物改用 docker create/cp 后在宿主校验内置脚本，L3 完整证据重新收口。",
    "permanentAction": "distroless/scratch 产物一律从宿主提取后校验，不在运行时镜像执行调试工具。",
    "historicalReleases": []
  },
  {
    "fingerprint": "background-e2e/error-envelope/field-selection",
    "position": "before-production-write",
    "count": 1,
    "impact": "首轮队列溢出断言读取 RFC7807 type 而非产品 code，误判已正确返回的 429。",
    "recoveryEvidence": "按真实响应契约检查 code=remote_download_queue_full，fresh r2 全链路通过。",
    "permanentAction": "API E2E 从注册类型或前端 ApiError 契约提取错误字段。",
    "historicalReleases": []
  },
  {
    "fingerprint": "browser-e2e/directory-upload/completion-timing",
    "position": "before-production-write",
    "count": 1,
    "impact": "首轮在首个上传项完成前检查根目录，测试断言过早。",
    "recoveryEvidence": "等待全部预期条目完成后再核对递归结构，fresh r3 通过。",
    "permanentAction": "批量上传验收以每项终态和最终目录快照作为完成条件。",
    "historicalReleases": []
  },
  {
    "fingerprint": "browser-e2e/console/expected-network-classification",
    "position": "before-production-write",
    "count": 1,
    "impact": "首轮把未登录 session 401、隔离 Docker 503 和 no-overwrite 探针 404 误计为业务 console error。",
    "recoveryEvidence": "按精确 URL/状态分类环境预期消息，fresh r3 业务 console error=0。",
    "permanentAction": "浏览器门禁分栏记录 product-error 与 expected-network-boundary。",
    "historicalReleases": []
  },
  {
    "fingerprint": "remote-diagnostic/powershell-ssh/variable-expansion",
    "position": "before-production-write",
    "count": 1,
    "impact": "一次只读 evidence find 命令中的远端变量被本地 PowerShell 提前展开，未得到目标清单。",
    "recoveryEvidence": "改用不含远端变量的单引号固定绝对路径重新核对，未产生远端写入。",
    "permanentAction": "复杂 SSH 取证优先上传固定脚本或使用无变量绝对路径。",
    "historicalReleases": []
  }
]
<!-- kpanel-release-process-incidents:end -->

## 遗留风险与后续准入

- 后台下载会在 Panel 重启后标记 `interrupted`，不会自动续传；管理员可确认目标不存在后重新发起。
- 目录递归上传受浏览器目录句柄、文件数量、单文件/总量和同名 no-overwrite 约束；部分失败会逐项呈现，不会覆盖已有文件。
- 镜像源优化现在是非阻断步骤；系统更新本身仍严格失败即停，生产未执行破坏性一条龙调优。
- 本版无 P0/P1/P2 遗留项，无需回滚；生产写验收与隔离真机分层记录。
