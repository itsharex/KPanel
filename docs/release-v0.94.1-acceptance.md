# KPanel v0.94.1 发布验收记录

日期：2026-08-23

发布级别：L3

候选提交 / 标签：`cdaf6b82d9c2f1169f0411ee0d783fbff491bdc4` / `v0.94.1`

上一稳定版本 / 回滚点：`v0.94.0` / `105f4366dea10aa386767b6eebdd41845e820fb6`

## 发布画像

- 业务域：Files、Docker、终端和桌面模式的右键菜单布局与键盘交互。
- 变更面：仅 Web 展示与交互；不改后端、API、依赖、端口、Compose、数据格式、受管脚本或应用市场安装契约。
- 受影响用户旅程：在窗口、任务栏、浏览器底部/右侧、窄屏与长菜单中打开右键菜单；键盘 Home/End/Escape 与焦点恢复。
- 风险等级及理由：低；统一复用一个可回滚的安全边界定位 helper，风险集中在菜单位置、层级和焦点行为。

## 发布范围与未纳入内容

- `0a67d28`：Files/Docker 菜单 Teleport 到 body；统一桌面、任务栏、窗口和 visualViewport 安全范围；限制长菜单滚动；增加键盘导航、焦点恢复和失效关闭；修复 transition scale 造成的测量偏差；提高菜单字号可读性。
- `cdaf6b8`：准备 0.94.1 版本、CHANGELOG 和升级/回滚说明。
- 治理候选 `e8b6404ade4003f97ec0902d7edd9e448fc35857` 未纳入；本次继续使用冻结候选中的 `release-kpanel v2.8`，未在发布中途更换规范。

## 多维质量结论

| 维度 | 状态 | 证据 | 未验证风险或不适用依据 |
| --- | --- | --- | --- |
| 业务正确性 | 已验证 | Web 115 文件/919 项；Files、真实 Docker 容器、桌面图标和任务栏菜单真实 Chrome 门禁 | 终端真实 Shell 会话未在生产创建；TerminalContextMenu 自动化已覆盖 |
| 安全与供应链 | 已验证 | govulncheck 可达漏洞 0、npm audit 0、Trivy source/image 0、正式 OCI revision/version 固定 | 无新增网络或权限边界 |
| 稳定性与恢复 | 已验证 | resize 关闭旧菜单、Home/End/Escape、焦点与滚动策略、标准应用更新事务、停写备份 | UI 补丁未做长 soak，生产短采样更符合风险 |
| 性能与资源 | 已验证 | 5 次生产采样 CPU 0.02%～0.03%、内存 70.92～70.98 MiB/256 MiB、7 PIDs | 无新增后台任务 |
| 用户体验与可访问性 | 已验证 | Chrome 151，1365×900、1200×760、390×844；菜单未越界、无页面横向溢出、应用级控制台错误 0 | 未做物理显示器 200% 缩放人工验收 |
| 数据、配置与迁移 | 已验证 | `.env` 与 `panel-state.json` 升级前后逐字节一致；Compose config 通过 | 本版无数据迁移 |

## 自动门禁

- 最终 bundle：`kpanel-v0.94.1-cdaf6b8.bundle`，SHA-256=`f4576502f28c287ea452e04b64519d091bf848c2b6ab17964436899fd2112aaa`。
- 固定 Runner：`kpanel-release-gate:go1.26.6-node24` / `sha256:b593c0ffe32e80a6a6ae8fcac38ed916587dbf698a6b6a55fa33887298737148`。
- 完整 L3 exit 0，日志 SHA-256=`f1e996377da125fce7964263df118fa0c12f9434c29b24348aedfdb4b6ceada5`；Go 全包、核心 race、vet、Linux amd64/arm64、Web 115/919、i18n 2465/21、typecheck、生产构建、安装安全、治理和应用生命周期均通过。
- 候选 CI：`32634347795` completed/success；候选依赖检查：`32634347769` completed/success。
- 主线 CI：`32634550001` completed/success；主线依赖检查：`32634549968` completed/success。
- Release workflow：`32634774035` completed/success，精确绑定 `v0.94.1` / `cdaf6b82...`。
- 规范入口说明：Windows PATH 没有 `make`，本地变更门禁直接调用仓库等价实现 `node scripts/run-repo-bash.mjs scripts/verify-change.sh`；Linux 固定 Runner 的权威 L3 使用项目标准脚本并一次完成。

## 依赖与技术栈变化

- 无 Go/npm 依赖、Action SHA、基础镜像或期限例外变化。
- 受管脚本仍固定到 `9fec61b50cc6ef798dfac1edf11c2ec60ca6b0d1`，OCI LF SHA-256=`54ceb0e72c4c342382500fc35da636fa436c484a12c4766fb9c7f806a23ae8fa`。

## 隔离真机与浏览器验收

- 环境：`arena-154`；精确候选镜像 `sha256:44ba9885f822b7f034193658e4190c191598f8794dbe7fc55b5629078f3290d6`，revision/version 精确匹配 `cdaf6b82...` / `0.94.1`。
- Chrome 151 后台独立临时 Profile：Files 右下边缘菜单、真实 Docker 容器菜单、桌面图标键盘菜单、任务栏菜单和 390px Files 菜单全部位于安全范围内；Home/End/Escape、scroll policy、resize 关闭旧菜单均通过。
- 390px：菜单 rect=`left 8 / right 382 / bottom 836`，document `clientWidth=scrollWidth=390`。
- 浏览器报告 SHA-256=`9352aee79db86b0b920e67db26c18b0c738e9439c12185e72b461861f7a39795`；截图和报告位于 `C:\GitHub\_release-artifacts\v0.94.1\browser-context-menu`。
- HTTP 候选的 COOP secure-context 警告属于隔离入口既有属性；产品应用级 blocking console errors 为 0。

## 发布产物与公开仓库复核

- GitHub Release：[v0.94.1](https://github.com/kejilion/KPanel/releases/tag/v0.94.1)。
- Docker 版本与 `latest` OCI index：`sha256:e666a8e67680e9a838e752ac1828286fa504a770afd224d439aa6452f38eb62a`。
- `linux/amd64`=`sha256:527f01bbc64b80568f25485b2a1cba84c13d1a270f26bf85fc4d6ed9d88d1fb1`；`linux/arm64`=`sha256:f44ad00305ac94b06294feed8c27c82e1f425c91aef5da4c38bc5b9779bc1ed4`；额外 unknown/unknown 为 provenance attestations。
- 公开镜像按不可变摘要回拉，version/revision/non-root/read-only 契约通过；项目标准 `image_e2e=pass`，日志 SHA-256=`827d8c2a3ddce9dbc89e0a9a6f8e463eda7460755e0dd8e238b2b4591ff98c5a`。
- `packaging/kejilion-app/kpanel.conf` 与 154 `/root/apps/kpanel.conf` SHA-256 均为 `82f06ca32ce827ef8d0c9c72e65eed9180841a23cbc507237072b58a0807ef04`；安装契约无变化，因此不创建 apps 空提交。

## 生产部署安全核对

- 唯一生产目标为 `arena-154`；`prod-108` 禁用全部 KPanel 操作，本次未连接、未读取、未测试、未备份、未部署、未核对。
- 部署前 v0.94.0 healthy/active、restart=0、OOM=false；旧 OCI=`sha256:041d39cbd4fa042fad71d9a180177c7eb01cd017ac5cca36f228b204f0870d00`。
- 停写一致性备份：`/root/kpanel-backups/pre-v0.94.1-20260823T110021Z`；`SHA256SUMS` SHA-256=`a7c7ddec23104a4d4bbe864270027c8aad9c5f0c311ffb7eab8d3868bb92e303`。完整 `/home/docker/kpanel` 和旧 OCI 通过校验、独立解包比较与 `docker image load` 恢复核验。
- 标准部署入口：`KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update bash /home/docker/kpanel/bin/kejilion.sh app kpanel`；日志 SHA-256=`647b288f1f56ef3ae36e7a46e04d3d73529312fcee6795aca26db0744fdff69e`。
- 部署后 Panel v0.94.1 healthy、Agent v0.94.1 active、restart=0、OOM=false；OCI version/revision/digest 精确匹配；Compose config 通过，无 `.new`/`.rollback` 残留。
- `.env` 和 `panel-state.json` 与停写备份逐字节一致。宿主脚本仅存在安装器约定的 `permission_granted=true` 参数化，归一化 `canshu`/`permission_granted` 后与 OCI `/release/kejilion.sh` 逐字节一致。
- 资源采样 SHA-256=`85c61388338d24ad6305153f076a771c6495deda4aa8ff1e672deb94cbb6fe5a`；Panel 日志仅包含 v0.94.1 启动信息，无 panic/fatal/OOM。

## 回滚

- 源码/tag：`v0.94.0` / `105f4366dea10aa386767b6eebdd41845e820fb6`。
- 旧 OCI：`sha256:041d39cbd4fa042fad71d9a180177c7eb01cd017ac5cca36f228b204f0870d00`。
- 数据、配置、二进制和旧 OCI 备份：`/root/kpanel-backups/pre-v0.94.1-20260823T110021Z`。
- 回滚必须停写并成套恢复旧 image、完整 `/home/docker/kpanel`、Compose、`.env`、数据、Agent unit/二进制，然后 daemon-reload、启动 Agent/Panel 并复核 v0.94.0/health/digest；禁止只换镜像。
- 备份恢复检查已验证；生产无需回滚，当前保持 v0.94.1 healthy。

## 交付节奏数据

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-08-23T17:59:32+08:00
- 候选冻结时间：2026-08-23T18:00:02+08:00
- 生产完成时间：2026-08-23T19:04:02+08:00
- 提交到生产用时：1.08 小时
- 是否回滚、紧急热修复或重复发布：否
- 若发生失败，发现时间、恢复时间和逃逸门禁：不适用
<!-- kpanel-release-metrics:end -->

<!-- kpanel-release-process-metrics:start -->
- 已记录发布流程异常或无效证据拦截次数：18
- 其中生产写操作开始后异常次数：1
<!-- kpanel-release-process-metrics:end -->

### 流程异常明细

<!-- kpanel-release-process-incidents:start -->
[
  {
    "fingerprint": "local-entrypoint/repo-bash/windows-tool-resolution",
    "position": "before-production-write",
    "count": 2,
    "impact": "Windows PATH 缺少 make，且首次使用了已不存在的 mjs 版本检查文件名；均在候选测试前停止，没有修改候选。",
    "recoveryEvidence": "使用仓库 run-repo-bash 入口与现行 check-version-consistency.sh 后对应门禁通过；固定 Linux L3 exit 0。",
    "permanentAction": "只调用仓库当前存在的唯一脚本，不把本机 PATH 当项目门禁结论。",
    "historicalReleases": []
  },
  {
    "fingerprint": "release-v2.8/bundle-verify/repository-context",
    "position": "before-production-write",
    "count": 1,
    "impact": "v2.8 在非仓库目录执行 git bundle verify 被 Git 拒绝，产品代码尚未运行。",
    "recoveryEvidence": "在独立 bare Git context 验证同一 bundle 后通过，bundle SHA 未变化，随后完整 L3 一次通过。",
    "permanentAction": "治理 v2.9 候选已覆盖该入口，但按规则未在本次冻结发布中途更换规范。",
    "historicalReleases": []
  },
  {
    "fingerprint": "powershell/exec-command/exit-and-quoting",
    "position": "before-production-write",
    "count": 3,
    "impact": "旧镜像精确清理、main 祖先检查和 tag peel 三条命令分别被 PowerShell 展开或返回值语义误判；危险命令未执行或写入已停止在下一步前。",
    "recoveryEvidence": "改用远端 literal script、显式 LASTEXITCODE 和 quoted rev 后均核对成功；main 为普通 fast-forward，tag 精确指向候选。",
    "permanentAction": "PowerShell Git/SSH 操作立即检查退出码，复杂远端命令使用 literal stdin script。",
    "historicalReleases": []
  },
  {
    "fingerprint": "browser-gate/background-chrome/evidence-channel-assumptions",
    "position": "before-production-write",
    "count": 9,
    "impact": "浏览器控制通道超时及独立 Chrome 夹具的 import URL、登录 URL、桌面选择器/触发和经典模式状态假设导致九次证据脚本失败；候选服务与已通过页面始终正常。",
    "recoveryEvidence": "最终同一候选、同一正式 Chrome 151 报告 passed=true；Files/Docker/桌面/任务栏/390px 全部有精确 rect 与键盘证据。",
    "permanentAction": "证据脚本使用独立临时 Profile、精确 pathname、真实 DOM class 和显式桌面模式切换；测试后清理 Profile。",
    "historicalReleases": []
  },
  {
    "fingerprint": "public-e2e/image-e2e/container-health-interval",
    "position": "before-production-write",
    "count": 1,
    "impact": "公开 OCI 应用 health 已为 200/v0.94.1 时，首次立即断言 Docker 30 秒 health 周期导致脚本退出。",
    "recoveryEvidence": "同一不可变容器等待 health=healthy 后 restart=0/OOM=false；项目标准 image-e2e 一次通过。",
    "permanentAction": "公开运行时核验先等待容器 health 周期再断言。",
    "historicalReleases": []
  },
  {
    "fingerprint": "production-preflight/compose-label/compose-filename",
    "position": "before-production-write",
    "count": 1,
    "impact": "只读盘点首次请求不存在的 compose.yml，未发生生产写入。",
    "recoveryEvidence": "按 Compose 元数据确认正式文件为 docker-compose.yml，后续备份和 docker compose config 通过。",
    "permanentAction": "从容器 Compose labels 获取 config_files，不猜测文件名。",
    "historicalReleases": []
  },
  {
    "fingerprint": "post-deploy/managed-script-contract/normalization",
    "position": "after-production-write",
    "count": 1,
    "impact": "上线后首次把宿主受管脚本原始 SHA 直接等同 OCI blob，忽略安装器 permission_granted 参数化；生产服务始终 healthy。",
    "recoveryEvidence": "归一化 canshu 与 permission_granted 后宿主与 OCI 脚本逐字节一致，Agent/Panel/正式摘要全部正确。",
    "permanentAction": "生产脚本核验固定使用安装器允许字段归一化规则，同时保留原始 SHA。",
    "historicalReleases": ["v0.94.0"]
  }
]
<!-- kpanel-release-process-incidents:end -->

## 遗留风险与后续准入

- 终端真实交互会话和物理显示器 200% 缩放未人工执行；受影响代码由 TerminalContextMenu 与安全范围自动化覆盖，本版没有终端协议变化。
- 右键菜单属于低频、无数据写入的 UI 行为；完整自动门禁、真实浏览器矩阵、正式 OCI 和生产短采样共同覆盖本次风险。
- 本次没有新增可复用产品工作流；沿用项目现有 release-kpanel v2.8。治理 v2.9 候选等待本发布完全收口后再独立复核。
