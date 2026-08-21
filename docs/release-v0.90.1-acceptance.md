# KPanel v0.90.1 发布验收记录

日期：2026-08-21

发布级别：L3（纯前端补丁）

候选提交 / 标签：`44211d11c40c8617e333f3ff65f951ef15d745ea` / `v0.90.1`

上一稳定版本 / 回滚点：`v0.90.0` / `sha256:e935d612c7bb8df9c8b66f3b69c91fb3a99b4c7a3162b3da46fc6e1c381eefeb`

## 发布范围

- 桌面“服务总览”改为两行摘要，清晰区分总量与运行、待核对、在线状态，进度条固定在卡片底部。
- 应用卡仅展示已安装与运行数量；集群卡使用在线比例；简中、繁中与英文摘要同步。
- 保留最新主线的 13px 辅助字号、15px 指标值和既有桌面滚动/布局契约；没有 API、数据库、Agent、Compose、端口、受管脚本或应用市场契约变更。
- 未纳入脏管理工作树、旧候选或未提交草稿；未连接、读取、测试或部署 108。

## 自动门禁

- 最终 L3：arena-154 固定 Runner `sha256:b593c0ffe32e80a6a6ae8fcac38ed916587dbf698a6b6a55fa33887298737148`，标记 `L3 release verification completed`；日志 SHA-256=`7662821dbd0054447d69fdca39063cff9d408e8fe9d25619436d580549f5d067`。
- 覆盖：Go 全包、`panel/auth/dockerx` 竞态、govulncheck 可达漏洞 0、npm audit 0、Trivy 源码/镜像 0、Web 106 文件/821 测试、i18n 2427/20、typecheck、production build、linux/amd64 与 linux/arm64、安装安全和应用生命周期。
- 候选 CI `32453706623`、候选依赖新鲜度 `32453706638`、main CI `32454049677`、main 依赖新鲜度 `32454049674`、Tag 依赖新鲜度 `32454340700` 均 success。
- Release workflow `32454340702` success；发布分支已按工作流自动清理，Tag、Release 与 OCI 保持不可变。

## 界面与真实链路验收

- 精确候选 `44211d1...` 的标准 acceptance mock 预览在 1280×720 下通过：组件 `clientHeight=scrollHeight=292`、最小字号 13px、页面横向溢出 0；工作区向下滚动后组件完整位于任务栏上方。
- 简中、英文、浅色、深色与中窄窗口复核通过；卡片与时钟、服务器监控小组件层级协调，没有文字超框或进度条漂移。
- arena-154 精确候选镜像 `sha256:e9831576bc0951789d80f912010f082a6f99d61d1192bd12e79c23c532edefc7` 通过真实 Chrome：从生产 Agent 只读回读网站 `5/待核对 3`、容器 `17/运行中 16`、应用 `5/运行中 5`、集群 `1/1 在线`，组件完整可达；候选容器 healthy、restart=0、OOM=false。
- 隔离浏览器首个未登录 session 401、第三方应用更新检查 409 和缺失站点外观 404 均被分类为现有状态/数据响应，不是本补丁白屏、布局或主链路失败；隔离容器、临时账号与数据已清理。
- 生产公网 HTTPS 登录页由正式 Chrome 加载通过，样式表完整、页面无横向溢出；未使用或污染用户日常 Chrome Profile。

## 发布产物

- GitHub Release：[v0.90.1](https://github.com/kejilion/KPanel/releases/tag/v0.90.1)，非 draft、非 prerelease；Tag 解引用到 `44211d11c40c8617e333f3ff65f951ef15d745ea`。
- Docker `0.90.1` 与 `latest` OCI index：`sha256:41a5757ccf041469e8bdeed56574b4c75539142d483c2b5554b2cb103137cd7d`。
- `linux/amd64`：`sha256:9b5ba7f96fc939bbd9804f440cf393df2486784e371a854e65bc3c5d161f8307`；`linux/arm64`：`sha256:6d7763d660f151f26db06a3b0e71eaca75daaa8e162dc162763e2545ca9ed648`。
- Release `SHA256SUMS` asset digest=`sha256:0d8af6e2fee7775f09614efa5d25dcfabc01a6186366cbea6aeb359498acba80`；公开部署包 digest=`sha256:ad1995f2618d403a99214553ff23acbc943364cdd1c229cfe784bddf3e7c2506`。
- 公开 OCI 独立回拉后核验 version=`0.90.1`、revision=`44211d1...`、受管脚本 revision=`2d1a374...`、镜像内脚本 SHA-256=`17c1544b826c45f070e49df2f71a5e152fedc922a1de201da5bfa0393d250a4d`。
- `kejilion/apps@6d86eee24a477320f4d8ffb32d9e85b785cf3c2c` 的 `kpanel.conf` 与候选逐字节一致，无需空提交。

## 生产部署与回滚

- 生产目标仅 arena-154。部署前 Panel/Agent 为 v0.90.0，healthy/active、restart=0、OOM=false。
- 停写一致性备份：`/root/kpanel-backups/v0.90.1-preupgrade-arena154-20260821T063540Z`；状态包和旧镜像归档已完成独立解包、关键文件对比、双 SQLite integrity、Compose 解析、旧镜像加载及 v0.90.0 恢复健康核验，结果 `backup_verified=pass`。
- 标准应用市场 `docker_app_update` 完成升级；没有覆盖 `/root/apps/kpanel.conf` 或用户应用登记状态。
- 部署后 Panel v0.90.1 healthy、Agent v0.90.1 active、restart=0、OOM=false，公网 HTTPS 健康正常；三次资源采样 CPU 0.02%～0.06%、内存 74.34 MiB/256 MiB，随后复核 74.65 MiB/256 MiB；两个 SQLite integrity=ok，部署后日志无 panic/fatal/OOM/协议错误。
- 回滚时停写，加载备份中的 v0.90.0 OCI，成套恢复 `/home/docker/kpanel`、应用市场配置和 systemd unit，执行 `systemctl daemon-reload` 后启动 Agent/Panel，并复核 v0.90.0、SQLite、Compose、restart/OOM 与公网健康；禁止只替换镜像或单独恢复配置。

## 交付节奏数据

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-08-21T13:40:11+08:00
- 候选冻结时间：2026-08-21T13:49:39+08:00
- 生产完成时间：2026-08-21T14:36:20+08:00
- 提交到生产用时：0.94 小时
- 是否回滚、紧急热修复或重复发布：否
- 若发生失败，发现时间、恢复时间和逃逸门禁：不适用
<!-- kpanel-release-metrics:end -->

<!-- kpanel-release-process-metrics:start -->
- 已记录发布流程异常或无效证据拦截次数：8
- 其中生产写操作开始后异常次数：0
<!-- kpanel-release-process-metrics:end -->

八次拦截均发生在生产前，分别涉及定向测试路径、Windows/WSL linked-worktree Bash 环境、Windows 缺少 Go、PowerShell SSH 命令替换、L3 bundle 缺少业务基线标签、隔离端口/状态与清理命令转义、历史本地标签抓取冲突；均在继续前改用确定性命令或补齐证据，没有跳过门禁、污染候选、损坏数据、触发生产回滚或操作 108。
