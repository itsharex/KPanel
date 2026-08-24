# KPanel v0.97.0 发布验收记录

日期：2026-08-24

发布级别：L3

候选提交 / 标签：`807ccca352a10d62a5aa8037a1dcafb7fae19b0c` / `v0.97.0`

上一稳定版本 / 回滚点：`v0.96.0` / `sha256:1c5148551fa3bdf02a07c398705a1423b317aa74f23145d9fb0d4dd0e1cb632a`

## 发布画像

- 业务域：Web 主题系统、桌面图标交互反馈、网站无图标回退呈现。
- 变更面：新增浅色/深色/跟随系统和三项颜色意图，设置页支持局部预览、校验、应用、恢复；淡化未选中桌面图标阴影并让选中态优先于 hover；移除网站无图时多余的 Globe fallback badge，保留单一字母图形。
- 不变边界：不修改 Panel/Agent API、权限、端口、Compose、应用市场安装契约或受管 `kejilion.sh` 固定版本；状态绿、警告橙和危险红不被自定义主题覆盖。
- 风险等级：中；变化集中在全局视觉语义层，若 token 或消费者不一致会影响多个页面，因此按兼容功能版本执行完整 L3 和真实浏览器跨视口验收。

## 精确提交范围

- `6ac192c`：全新语义主题框架及设置页配色工作室。
- `5d060fa`：准备 `0.97.0` 版本和 CHANGELOG。
- `c93be61`：对主线既有 3 个 Go 文件执行纯 `gofmt`，满足新治理门禁，不改变业务语义。
- `d543a73`：淡化桌面图标阴影，选中态在悬停下立即生效。
- `a3af182`：移除网站无图片时的冗余 fallback badge。
- `807ccca`：冻结最终发布范围。
- 未纳入其他工作树、未提交草稿、旧候选证据或 108 环境。

## 多维质量结论

| 维度 | 状态 | 证据 | 边界 |
| --- | --- | --- | --- |
| 业务正确性 | 已验证 | 主题模式、颜色校验/应用/恢复、选中与 hover 优先级、网站 fallback DOM 契约 | 主题配置仅保存在当前浏览器 |
| 网络与供应链安全 | 已验证 | govulncheck 可达漏洞 0、npm audit 0、Trivy source/image 0、OCI revision/version 固定 | 无新增网络入口 |
| 稳定性与兼容 | 已验证 | canonical L3、候选/main/Tag CI、公开 OCI E2E、停写备份和标准更新全绿 | 无数据 schema 变化 |
| 性能与资源 | 已验证 | 生产 5 次采样 CPU 0.02–0.03%、内存 73.12 MiB/256 MiB、7 PIDs，restart=0、OOM=false | 本轮无长时间 soak |
| 用户体验与可访问性 | 已验证 | Chrome 151 桌面/390×844、浅深主题、键盘 radio 契约、非法颜色反馈；手机 clientWidth=scrollWidth=390 | 隔离 Agent 离线产生的 401/403/503 不计为 UI 回归 |
| 数据与配置 | 已验证 | `.env` 与 panel-state 升级前后摘要一致；21 个 JSON、2 个 SQLite 完整 | 无安装契约或迁移 |

## 自动门禁

- canonical L3：`v0.97.0-807ccca-l3-r1`，exit 0；Go 全包、核心 race、vet、Web 121 文件/1014 项、i18n 2582/21、typecheck、production build、双架构构建、安装安全、应用生命周期、govulncheck、npm audit、Trivy source/image 全绿。
- bundle SHA-256=`b0d79998f6d4c98eb7fcab85aa548a78d3bfb2baa72e888a29abda855e0c910b`；L3 日志 SHA-256=`d50a12ee408c8e43480cb77d62bc41b8dc63d9c7057d3b2d0006b27e80ba9038`；manifest SHA-256=`07757e05e4cdf90eae5e6ae2d790f4b817936dcc119a7d1e1f7abec92355ac67`。
- 固定 Runner：`kpanel-release-gate:go1.26.6-node24`，image ID=`sha256:b593c0ffe32e80a6a6ae8fcac38ed916587dbf698a6b6a55fa33887298737148`。
- 候选 CI `32714497086`、Dependency freshness `32714497254`；main CI `32714743554`、Dependency freshness `32714743562`；Release workflow `32715242684`、Tag Dependency freshness `32715242701`，均 completed/success 且绑定精确 SHA。

## 真实浏览器视觉验收

- 独立正式 Google Chrome `151.0.7922.172`、随机临时 Profile、后台 headless；未读取或接管用户日常浏览器。
- 1440×900 浅色和深色设置页层级协调；主题色、界面基调、点缀色的局部预览与应用清晰，非法 `red` 输入显示精确 Hex 提示，刷新后自定义颜色仍生效。
- 390×844 设置页 `innerWidth=clientWidth=scrollWidth=390`，卡片边界保持 12px 安全边距，无横向溢出。
- 桌面 1440×900 选中图标在 hover 下保持选中反馈，阴影更克制；`.desktop__site-fallback-badge` 运行中数量为 0，精确测试覆盖桌面入口和外部打开确认窗。
- 视觉结果 SHA-256=`fe26c83ee13491258b2104fb51aff4408895b3311797ca185bb134261fe7c6a6`；截图与聚合证据位于 `C:\GitHub\_release-artifacts\v0.97.0-807ccca-visual`。

## 发布产物

- GitHub Release：[v0.97.0](https://github.com/kejilion/KPanel/releases/tag/v0.97.0)，非 draft、非 prerelease，8 个附件；annotated Tag object=`7237d38122d99350e6eda1d02fc2f9b42fb300f2`，peel=`807ccca352a10d62a5aa8037a1dcafb7fae19b0c`。
- Docker `0.97.0` 与 `latest` OCI index：`sha256:ae60f409410af74153f20d0e47265ff59e80f4a6938ca1018d074609136d6410`。
- `linux/amd64`=`sha256:1435e483540652c07129bc453f8f9062c805f8aa51ca3ab6cc884b441dbcc761`；`linux/arm64`=`sha256:e8bdfaa2e2654584182daf34b3a0a824c16b02d70eb081f277f7eb68cee0f539`。
- OCI labels：version=`0.97.0`、revision=`807ccca...`；managed script revision=`9fec61b50cc6ef798dfac1edf11c2ec60ca6b0d1`、SHA-256=`54ceb0e72c4c342382500fc35da636fa436c484a12c4766fb9c7f806a23ae8fa`。
- `0.97.0` 与 `latest` digest 一致；arena-154 公开回拉、镜像标签和镜像内 VERSION 核验通过。安装契约无差异，不创建 apps 空提交。

## 生产部署与回滚

- 目标仅 `arena-154`；108 未连接、未测试、未备份、未部署。
- 部署前：Panel `0.96.0` healthy、Agent active、restart=0、OOM=false；旧 OCI=`sha256:1c5148551fa3bdf02a07c398705a1423b317aa74f23145d9fb0d4dd0e1cb632a`。
- 停写一致性备份：`/root/kpanel-backups/pre-v0.97.0-20260824T101818Z`；`SHA256SUMS` SHA-256=`9086b7e9e73511e068f6730ca9f4fe56a32a14bf667f08654825e9e008b947d5`。状态目录、旧 OCI、Compose、`.env`、Agent 文件、21 个 JSON 和 2 个 SQLite 均完成独立恢复与旧版重启核验。
- 标准更新入口一次成功：`KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update bash /home/docker/kpanel/bin/kejilion.sh app kpanel`；日志 SHA-256=`151ac11870abda42410cc18983b4be7933cf532bde728c180a4a81f244ae581f`。
- 部署后：Panel `0.97.0`、revision/OCI 精确匹配、Agent active、Panel healthy、restart=0、OOM=false；公网 health 返回 `0.97.0`，数据和配置完整，日志无 panic/fatal/OOM。
- 宿主 `kejilion-agent` 与镜像逐字节一致；宿主 `kejilion.sh` 仅保留既有 `permission_granted=true` 本地状态，归一化后与镜像固定脚本逐字节一致，未覆盖用户配置。
- 回滚：停写并校验备份 `SHA256SUMS`；加载旧 OCI；成套恢复 KPanel 目录、Compose、`.env`、apps 配置和 Agent 文件；`systemctl daemon-reload` 后启动 Agent/Panel；复核 `0.96.0`、旧 digest、health、restart/OOM 和数据。禁止只换镜像。

## 交付节奏数据

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-08-24T17:05:48+08:00
- 候选冻结时间：2026-08-24T17:34:50+08:00
- 生产完成时间：2026-08-24T18:20:41+08:00
- 提交到生产用时：1.25 小时
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
    "fingerprint": "candidate-governance/canonical-l3/preexisting-gofmt-baseline",
    "position": "before-production-write",
    "count": 1,
    "impact": "首轮阶段 L3 被新全仓 gofmt 门禁发现主线既有 3 文件格式差异，产品测试前 fail-closed。",
    "recoveryEvidence": "仅执行机械 gofmt 形成 c93be61，最终 SHA 从零运行 canonical L3 全绿。",
    "permanentAction": "候选冻结前执行与当前治理版本一致的全仓 gofmt 预检。",
    "historicalReleases": []
  },
  {
    "fingerprint": "release-orchestration/local-preflight/tool-assumptions",
    "position": "before-production-write",
    "count": 3,
    "impact": "一次错误候选 SHA 被外层拒绝、一次 WSL 缺少 gofmt、一次 PowerShell 远端格式命令在执行前解析失败；均未修改候选或服务器。",
    "recoveryEvidence": "固定精确 SHA、Linux Runner 和本地聚焦提交后完成全量门禁。",
    "permanentAction": "候选 SHA 从 git rev-parse 读取；格式化只在具备 gofmt 的固定环境执行，复杂远端命令使用受审脚本。",
    "historicalReleases": []
  },
  {
    "fingerprint": "visual-gate/browser-and-dependencies/preflight-missing",
    "position": "before-production-write",
    "count": 2,
    "impact": "本地定向测试首次缺 node_modules；内置浏览器控制 kernel reset，均未形成产品结论。",
    "recoveryEvidence": "npm ci 后 92 项定向回归通过；独立正式 Chrome 临时 Profile 完成同 SHA 视觉验收。",
    "permanentAction": "视觉验收先执行依赖和控制通道 preflight，控制通道失败时使用隔离正式 Chrome，不接管用户 Profile。",
    "historicalReleases": []
  },
  {
    "fingerprint": "public-image-e2e/version-file/distroless-shell-assumption",
    "position": "before-production-write",
    "count": 1,
    "impact": "首次尝试用 /bin/sh 读取无 shell 正式镜像中的 VERSION，容器未启动且未触碰生产。",
    "recoveryEvidence": "改用 docker create + docker cp 只读核验，VERSION、revision 和 digest 全部一致。",
    "permanentAction": "极简 OCI 的文件身份核验固定使用不启动容器的 docker cp。",
    "historicalReleases": []
  }
]
<!-- kpanel-release-process-incidents:end -->

## 遗留风险

- 自定义配色按当前浏览器持久化，不跨浏览器或设备同步；清理站点数据会恢复默认配色。
- 主题框架只允许项目生成的安全 token；新增颜色消费者时必须继续使用语义变量并重跑明暗主题、状态色和 390px 实际几何验收。
- 网站无图片时只显示字母图形；后续如增加新 fallback 资产，不得重新叠加独立角标造成双重视觉语义。
