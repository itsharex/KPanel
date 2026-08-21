# KPanel v0.90.0 发布验收记录

日期：2026-08-21

发布级别：L3（兼容功能版本）

候选提交 / 标签：`040005ddd784187fc3effc339d21682a8b36a769` / `v0.90.0`

上一稳定版本 / 回滚点：`v0.89.1` / `sha256:755cad031673e9bc44dc8b43ed7b62012a240d708cdf8d14c6c593810e1a73d8`

## 发布范围

- 终端窗口修正底部状态行与输入栏可见性，并用串行发送队列避免快速输入时丢字符。
- 体检页面优化窄屏布局，并细化 IP 质量评分展示与回读。
- 桌面模式扩大可用拖拽表面，同时保持现有桌面边界和落位语义。
- 受管 `kejilion.sh` 固定到 `2d1a37416c574c7398445a54cd6d9d3a0d4bc124`，公开 LF 脚本 SHA-256=`17c1544b826c45f070e49df2f71a5e152fedc922a1de201da5bfa0393d250a4d`；脚本仅同步 VMRack 联盟优惠内容，根/CN 语法与同步门禁通过。
- 没有数据库 schema、外部 API、端口、权限模型或应用市场契约迁移；`kejilion/apps@6d86eee24a477320f4d8ffb32d9e85b785cf3c2c` 的 `kpanel.conf` 与候选逐字节一致，无需空提交，生产本地配置未覆盖。
- 未纳入脏管理工作树、旧候选、未提交草稿或其它无关历史；未连接、读取、测试或部署 108。

## 自动门禁

- 最终 L3：arena-154 固定 Runner `sha256:b593c0ffe32e80a6a6ae8fcac38ed916587dbf698a6b6a55fa33887298737148`，标记 `L3 release verification completed`；日志 SHA-256=`e7b590bf953d9e5340ef9cc88e7efc4b07d5ae9c157f6d409b7678170982ed78`。
- 覆盖：Go 全包、`panel/auth/dockerx` 竞态、govulncheck 可达漏洞 0、npm audit 0、Trivy 源码/镜像 0、Web 106 文件/821 测试、i18n 2427/20、typecheck、production build、linux/amd64 与 linux/arm64、安装安全和应用生命周期。
- 候选 CI `32447814483`、候选依赖新鲜度 `32447814471`、main CI `32448070292`、main 依赖新鲜度 `32448070432` 均 success。
- Release workflow `32448330802` success；发布分支按工作流自动清理，Tag 与公开产物保持不可变。

## 界面与真实链路验收

- 精确候选在正式浏览器中复核体检、终端和桌面模式：1280px 桌面与 390px 手机视口均无页面级横向溢出；终端底部状态行和发送栏完整可见；体检评分层级、窄屏卡片和桌面拖拽表面正常。
- Mock 终端夹具因未提供四个可选字段产生 4 条 Vue warning；真实候选链路和生产日志未出现对应错误，未将夹具告警误记为产品缺陷。
- arena-154 精确候选镜像 `sha256:9da685399f92ea733b7be822c53c175603fe61b40bf982b977db05d9d2f7c6ab`：Panel→Agent 原生综合体检成功；Panel→Agent→PTY 快速 Unicode 输入、输出回读和关闭成功；候选容器 healthy、restart=0、OOM=false。
- 正式公开 OCI 独立回拉后核验 version=`0.90.0`、revision=`040005d...`、script revision=`2d1a374...`、脚本摘要及镜像内 `/release/VERSION` 全部匹配。

## 发布产物

- GitHub Release：[v0.90.0](https://github.com/kejilion/KPanel/releases/tag/v0.90.0)，非 draft、非 prerelease；Tag 解引用到 `040005ddd784187fc3effc339d21682a8b36a769`。
- Docker `0.90.0` 与 `latest` OCI index：`sha256:e935d612c7bb8df9c8b66f3b69c91fb3a99b4c7a3162b3da46fc6e1c381eefeb`。
- `linux/amd64`：`sha256:142834a7f4e15d4f986340d697e88184e32f8c62c0594965083ef518d74df8e4`；`linux/arm64`：`sha256:46969263e70903a41126d3803254d466440a8536e96c0d0466e5bd84547da9f6`。
- Release `SHA256SUMS` asset digest=`sha256:57ad312a854fe880a2c2756464c82663f05cdb6ae963553e97153f7a34527b81`；公开部署包 digest=`sha256:1d47f9d3b39ef976a0e199d4d283f73a4b4e14a79f1069ab4bfaa6cedbc35d72`。

## 生产部署与回滚

- 生产目标仅 arena-154。部署前为 v0.89.1，Panel healthy、Agent active、restart=0、OOM=false。
- 停写一致性备份：`/root/kpanel-backups/v0.90.0-preupgrade-arena154-20260821T045845Z`；状态包与旧镜像归档均通过 `SHA256SUMS`，备份已独立解包、关键文件对比、SQLite integrity、Compose 解析、旧镜像加载及 v0.89.1 恢复健康核验，结果 `backup_verified=pass`。
- 标准应用市场 `docker_app_update` 完成升级；没有覆盖 `/root/apps/kpanel.conf` 或用户现有应用登记状态。
- 部署后 Panel v0.90.0 healthy、Agent v0.90.0 active、restart=0、OOM=false，公网 HTTPS 健康正常；三次资源采样 CPU 0.02%～0.06%、内存 74.36～74.37 MiB/256 MiB；SQLite integrity=ok，部署后日志无 panic/fatal/OOM/协议错误。
- 回滚时停写，加载上述旧 OCI，成套恢复备份中的 `/home/docker/kpanel`、应用市场配置和 systemd unit，执行 `systemctl daemon-reload` 后启动 Agent/Panel，并复核 v0.89.1、SQLite、Compose、restart/OOM 与公网健康；禁止只替换镜像或单独恢复配置。

## 交付节奏数据

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-08-21T10:46:57+08:00
- 候选冻结时间：2026-08-21T12:20:05+08:00
- 生产完成时间：2026-08-21T13:02:07+08:00
- 提交到生产用时：2.25 小时
- 是否回滚、紧急热修复或重复发布：否
- 若发生失败，发现时间、恢复时间和逃逸门禁：不适用
<!-- kpanel-release-metrics:end -->

<!-- kpanel-release-process-metrics:start -->
- 已记录发布流程异常或无效证据拦截次数：2
- 其中生产写操作开始后异常次数：0
<!-- kpanel-release-process-metrics:end -->

两次拦截均发生在生产前：公开 distroless 镜像不能通过不存在的 `version` 子命令或 `cat` 入口取证，随后改用不启动容器的 `docker create` + `docker cp` 完成等价核验。没有门禁绕过、数据损坏、生产回滚或 108 操作。
