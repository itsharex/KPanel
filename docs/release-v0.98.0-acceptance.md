# KPanel v0.98.0 发布验收记录

日期：2026-08-25

发布级别：L3

产品候选提交 / 标签：`c4d1b5f4813397ae812b4d3704fad3264f7c5a1e` / `v0.98.0`

发布后治理提交：`0f3213e0c8dabbbd3a0684519eda24837a41d46a`、`95a0c01c9ef7c8750709cd1d399aefb2bdd3193d`

上一稳定版本 / 回滚点：`v0.97.3` / `sha256:1d1e5fe884c9580ac7ce357397d78323ca7905b8e1d13f8367c25091d2f6ad49`

## 发布画像

- 业务域：桌面主题、壁纸联动和交互式 Shell 终端配色。
- 变更面：新增五套协调配色方案、自定义浅深主题色、近黑深色基础层，并让交互式终端随当前深色主题变化；ANSI 语义色保持固定。
- 未变化契约：API、数据库、Agent 协议、端口、Compose、受管 `kejilion.sh` 和应用市场安装契约均不变。
- 治理增强：新增固定生产证据入口；生产备份实测暴露恢复就绪竞态后，以两个聚焦提交补齐 HTTP 有界重试和 Docker health 等待，不改产品 Tag/OCI。
- 风险等级：中低；产品仅 Web 主题层变化，但涉及全局视觉基础面和终端可读性，须同时验证桌面、390px、键盘操作、终端语义色和回滚。

## 发布范围与未纳入内容

- `4c70196`：五套协调配色方案与壁纸主题联动。
- `481ff81`：交互式终端在深色模式下继承当前主题，亮色模式保留经典 Shell 配色。
- `63c5228`：加深暗色基础表面，不改变亮色基础。
- `d645b11`、`739eb17`、`c4d1b5f`：固定生产证据入口、版本准备和备份前置加固。
- `0f3213e`、`95a0c01`：发布后仅治理的生产健康采样与恢复容器就绪修复，已分别经候选/main CI；未重写 `v0.98.0`。
- 旧治理提交 `4a432a4` 未重复合入；能力已由主线祖先 `3ccf14a` 等价吸收且主线含后续增强。
- 未纳入其他旧工作树、未提交草稿或 108 环境。

## 多维质量结论

| 维度 | 状态 | 证据 | 未验证风险或不适用依据 |
| --- | --- | --- | --- |
| 业务正确性 | 已验证 | Web 122 文件/1031 项、五套配色精确值、主题键盘切换、终端深浅主题与 ANSI 固定色 | 无后端业务逻辑变化 |
| 网络与供应链安全 | 已验证 | govulncheck 可达漏洞 0、npm audit 0、Trivy source/image/config 0、OCI revision/version 固定 | 无新增网络入口或权限 |
| 稳定性与恢复 | 已验证 | canonical L3、候选/main/Tag 门禁、公开 OCI E2E、停写备份、旧镜像加载与旧版恢复、标准更新、postdeploy 全通过 | 备份门禁首轮暴露的恢复就绪竞态已在治理提交中闭环并用新 run ID 真机通过 |
| 性能与资源 | 已验证 | 生产快照 CPU 0.02%、内存 73.87 MiB/256 MiB、7 PIDs、restart=0、OOM=false | 确定性主题更新，不执行长时间 soak |
| 用户体验与无障碍 | 已验证 | 1280 与 390x844 真浏览器；无横向溢出；五套 radio 键盘切换；终端窗口和活动主机在主题切换后仍保留 | 视觉验收为冻结候选 mock UI，不冒充生产登录态点击 |
| 数据、配置与迁移 | 已验证 | 受保护配置哈希不变、SQLite quick_check=ok、数据清单可读；无数据迁移 | 不适用额外迁移 |

## 自动门禁

- canonical L3：`v0980-c4d1b5f-l3-r1`，exit 0；远端证据目录 `/root/kpanel-release-evidence/v0980-c4d1b5f-l3-r1`。
- bundle SHA-256=`8160dbd5b0a3721387b64b5bd1ad60ef7d767e40346ff5866f7676a2db23e24a`；L3 日志 SHA-256=`8bb07b4013ba060798898b1e47d92947ced831cdf93968d827c7db97d838f7ae`；manifest SHA-256=`49af86c1c86ce8a2a9dda99c27c47f3e8c85d06dbdb32e36e36fe556bce5109e`。
- 固定 Runner：`kpanel-release-gate:go1.26.6-node24`，image ID=`sha256:b593c0ffe32e80a6a6ae8fcac38ed916587dbf698a6b6a55fa33887298737148`。
- 候选 CI `32752598019`、Dependency freshness `32752597900`；main CI `32752832619`、Dependency freshness `32752832585`；Release `32753307295`、Tag Dependency freshness `32753307289`，均 completed/success 且绑定产品 SHA。
- 最终治理修复候选 CI `32755267705`、main CI `32755491643` completed/success，绑定 `95a0c01c9ef7c8750709cd1d399aefb2bdd3193d`。
- Windows change-aware 按设计因缺少 Go/gofmt 停止；Linux 固定 Runner 提供权威 L3，未把 Windows 平台缺口写成通过。

## 浏览器与视觉验收

- 冻结候选 mock acceptance 证据：`C:\GitHub\_release-artifacts\v0.98.0-c4d1b5f-visual-r1`。
- 五套方案均能选中并生成预期三色值；键盘 ArrowLeft 能在相邻 radio 间移动并同步选择。
- 暮光棱镜深色模式计算值：`--bg=#09070b`、`--surface=#100c13`、`--surface-raised=#1f1826`、`--text=#f0edf2`、`--border=#2b272e`、`--accent=#a189cc`。
- 390x844 下 document/client/scroll width 均为 390，设置窗口及所有预设卡片未越界。
- 深色终端背景/表面/文字/边框与根主题一致；亮色终端保留经典 Shell 配色；ANSI red 在两种模式均为 `#d86f74`。
- 切换主题后终端窗口、主机列表与活动会话仍在；preview 与 mock API 日志为空。证据不替代生产 Panel/Agent 门禁。

## 发布产物

- GitHub Release：[v0.98.0](https://github.com/kejilion/KPanel/releases/tag/v0.98.0)，Release workflow completed/success；annotated Tag object=`98e406c0623137dccb20df518836460075fc2bf5`，peel=`c4d1b5f4813397ae812b4d3704fad3264f7c5a1e`。
- Docker `0.98.0` 与 `latest` OCI index：`sha256:23fcaf791c574b774a8eba8c008a03d7f16925950566458c60c70f2d26c71815`。
- `linux/amd64`=`sha256:6ba4b0d676d68e56877c67d777330d20e49b4f569e81fdb1f5a410169f5b0d58`；`linux/arm64`=`sha256:f3b2640a75add8fd9b77fdc2e0209fe2a1630be5567e49165c71bd3d500467c9`。
- 镜像 labels：version=`0.98.0`、revision=`c4d1b5f...`、受管脚本 revision=`9fec61b...`、脚本 SHA-256=`54ceb0e72c4c342382500fc35da636fa436c484a12c4766fb9c7f806a23ae8fa`。
- arena-154 从 Docker Hub 回拉正式镜像并执行固定 `packaging/tests/image-e2e.sh`，输出 `image_e2e=pass`。
- `packaging/kejilion-app/kpanel.conf` 相对 `v0.97.3` 无差异，无 apps 提交。

## 生产部署与回滚

- 唯一目标：`arena-154`；108 未连接、未检查、未测试、未备份、未部署。
- 固定证据 run ID：`v0980-production-r4`。preflight、backup、postdeploy 均 `status=passed`、exit 0。
- 停写一致性备份：`/root/kpanel-backups/pre-v0.98.0-20260824T171753Z`；`SHA256SUMS` SHA-256=`6616f027410448b3031918fff0ac5c6bb04ea7ad66c9450bad86c154b2cb1f00`。归档、旧镜像加载、校验和、旧版恢复健康均通过。
- 标准入口：`KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update bash /home/docker/kpanel/bin/kejilion.sh app kpanel`；输出到达 `KPanel 更新完成`、`KPANEL_PROGRESS 100`，exit 0。
- 部署后 health=`0.98.0`；Panel healthy、Agent active/enabled、NeedDaemonReload=no、restart=0、OOM=false；OCI/revision 精确匹配，受保护配置无差异，SQLite quick_check=ok，近 10 分钟日志无 panic/fatal/OOM。
- 回滚点：`v0.97.3`、旧 OCI `sha256:1d1e5fe884c9580ac7ce357397d78323ca7905b8e1d13f8367c25091d2f6ad49` 与上述停写备份。回滚须成套恢复旧镜像、Compose、`.env`、Agent unit、apps 配置和数据；未执行生产回滚，备份阶段已验证旧镜像可加载和旧版可健康重启。

## 交付节奏数据

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-08-25T00:16:49+08:00
- 候选冻结时间：2026-08-25T00:27:30+08:00
- 生产完成时间：2026-08-25T01:18:49+08:00
- 提交到生产用时：1.03 小时
- 是否回滚、紧急热修复或重复发布：是（发布工具紧急修复；未回滚或重复发布产品）
- 若发生失败，发现时间、恢复时间和逃逸门禁：发现时间：2026-08-25T01:04:00+08:00; 恢复时间：2026-08-25T01:18:03+08:00; 逃逸门禁：未逃逸：固定生产备份门禁在部署前失败关闭
<!-- kpanel-release-metrics:end -->

<!-- kpanel-release-process-metrics:start -->
- 已记录发布流程异常或无效证据拦截次数：9
- 其中生产写操作开始后异常次数：3
<!-- kpanel-release-process-metrics:end -->

### 流程异常明细

<!-- kpanel-release-process-incidents:start -->
[
  {
    "fingerprint": "release-browser/local-preview/ambiguous-theme-radio",
    "position": "before-production-write",
    "count": 1,
    "impact": "真实浏览器首次用非精确名称定位深色 radio 时同时命中预览与模式选项；未改变候选或生产。",
    "recoveryEvidence": "改为精确可访问名称后完成同一冻结候选的深浅主题与 390px 验收。",
    "permanentAction": "主题浏览器门禁对重复语义控件使用完整可访问名称，不使用宽松文本选择器。",
    "historicalReleases": []
  },
  {
    "fingerprint": "release-operator/github-public-evidence/rate-or-transport",
    "position": "before-production-write",
    "count": 2,
    "impact": "未认证 GitHub API 轮询触发限流，随后逐附件回拉遇到公网连接超时；Tag、Release 与 OCI 未被改写。",
    "recoveryEvidence": "改用公开 Actions 页面确认精确 run 成功，并以 Docker Hub manifest、latest 同摘要及 arena-154 正式镜像 E2E 完成产物核验。",
    "permanentAction": "公开 CI 使用有界页面查询，附件存在性以 Release workflow 与固定产物清单为准，避免高频未认证 API 轮询。",
    "historicalReleases": []
  },
  {
    "fingerprint": "release-operator/oci-inspect/remote-quoting",
    "position": "before-production-write",
    "count": 2,
    "impact": "两次远端 Docker inspect 格式串被 PowerShell/SSH 转义破坏，只产生无效只读命令。",
    "recoveryEvidence": "改为读取远端原始 JSON并在本地 PowerShell 解析，精确得到 version、revision 与受管脚本 labels。",
    "permanentAction": "跨 PowerShell/SSH 的 OCI labels 不再内联 jq/template，统一回传 JSON后本地解析。",
    "historicalReleases": []
  },
  {
    "fingerprint": "release-operator/production-entrypoint/unsupported-help-probe",
    "position": "before-production-write",
    "count": 1,
    "impact": "对不支持 --help 的固定入口执行了一次只读探测并按设计失败，未上传计划或操作生产。",
    "recoveryEvidence": "改读工作流登记的规范参数并以 prepare/preflight 真实入口执行。",
    "permanentAction": "固定入口参数只从 workflow 真源读取，不猜测可选参数。",
    "historicalReleases": []
  },
  {
    "fingerprint": "release-production/backup/http-health-snapshot-race",
    "position": "after-production-write",
    "count": 2,
    "impact": "前两次停写备份均已完整生成并校验，但旧服务恢复后的单次 health 快照遇到 connection reset，门禁失败关闭且生产仍为 v0.97.3。",
    "recoveryEvidence": "提交 0f3213e 为 health 快照增加 10 次有界重试，候选/main CI 成功；后续证据证明该层能跨过瞬态 reset。",
    "permanentAction": "生产快照所有 HTTP health 采集统一使用有界重试并保持最终状态断言。",
    "historicalReleases": []
  },
  {
    "fingerprint": "release-production/backup/docker-health-starting",
    "position": "after-production-write",
    "count": 1,
    "impact": "第三次备份已跨过 HTTP reset，但快照时容器 Docker health 仍为 starting，固定门禁失败关闭且未升级。",
    "recoveryEvidence": "提交 95a0c01 将 HTTP、Agent、容器 running 与 Docker health=healthy 合并为有界 production_ready；候选/main CI 后 r4 停写备份真机通过。",
    "permanentAction": "恢复旧服务后必须等待四项就绪条件同时成立，再生成不可变生产快照。",
    "historicalReleases": []
  }
]
<!-- kpanel-release-process-incidents:end -->

## 遗留风险

- 当前无已知产品 P0/P1/P2；主题选择与自定义颜色仍是当前浏览器本地偏好，不跨设备同步。
- v0.98.0 Tag 保持产品候选不可变；生产证据工具修复位于 Tag 后的 main，验收记录明确区分产品 revision 与发布治理 revision。
- L3 仍有既存 jsdom/响应式组件警告和 npm `glob@10.5.0` deprecation 提示，但测试、安全审计与生产门禁通过；不得表述为 warning-free。
