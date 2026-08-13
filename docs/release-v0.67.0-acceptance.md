# KPanel v0.67.0 发布验收记录

日期：2026-08-12

发布级别：L3

正式提交 / 标签：`6aaabeacd38e2d7b4c51e70b3aeff82db5523866` / `v0.67.0`

上一稳定版本 / 回滚点：`v0.66.0` / `b96f38dfada3fafa8286dbf2d7ee4be8937fc804` / OCI index `sha256:2abfb267da7a1ead720dbe3d44928a2fe76e3f3ef2efe727b7082c62718bbd99`

## 上线范围

- Docker“新建容器”支持直接粘贴单条 `docker run` 命令或完整 Compose YAML；命令解析、结构化预览、同名冲突检查、任务互斥、输出脱敏、有界留存、失败回滚及人工恢复标记均进入正式版本。
- 手机端主机终端和应用交互终端新增稳定的纵向历史滚动处理；窄桌面窗口不再为抽屉化的主机列表保留空白行。
- 本版本没有数据库、端口、密钥、Agent 协议或 `kejilion.sh` 协议变化。

## 候选冻结与门禁

- 候选由 `v0.66.0` 主线 `b96f38d` 创建，仅纳入 `d443e62`、`2b224a5` 和版本提交 `6aaabea`；未从其他脏工作树拼装内容。
- bundle：`C:\GitHub\_release-artifacts\v0.67.0-core\kpanel-v0.67.0-6aaabea.bundle`，SHA-256 `cc1abf39057049bed9969135c6a5ba19444dc039a3a9e91e1c050d33f55cf8c6`。
- Web 完整门禁通过：89 个测试文件、633 项测试，i18n 2144 条、typecheck、生产构建和 `npm audit` 全部通过。
- 154 Linux L3 完成 Go 全量测试、核心包 race、`go vet`、Linux amd64/arm64 构建、Web 全量门禁、部署安全、源码/镜像 Trivy、Agent/Panel/Node 构建和应用配置生命周期验证。日志 `/root/kpanel-release-v0670-core-6aaabea/l3-6aaabea.log`，SHA-256 `fee3983fd8a67b8215a50ebc95c1d27eb4bd843f008da4a6b1970fdc4e18d03b`，最终标记为 `L3 release verification completed`。
- 首次 L3 仅因验证容器缺少 Buildx、`BUILDPLATFORM` 为空而停止；补齐固定版本 Buildx 的验证镜像后从头重跑全套 L3 通过，候选源码未修改。
- 154 Docker L2 使用候选 Agent 实际完成结构化 `container_create`、镜像拉取、Compose 部署、同名冲突 409、宿主端口占用失败回滚、任务收据跨 Agent 重启持久化及临时资源清理；未在生产创建测试容器。
- Chrome 候选验收通过：Docker Run 解析和预览、Compose 识别、shell 链接拒绝、390px 无横向溢出；手机桌面终端滚动与窗口布局正常，控制台无新增错误。

## CI、Release 与公开镜像

- 候选 CI `31567539710`、候选 Dependency freshness `31567539684`、主线 CI `31567774260`、主线 Dependency freshness `31567774278`、Release `31567959937` 和标签 Dependency freshness `31567959968` 全部成功。
- [GitHub Release v0.67.0](https://github.com/kejilion/KPanel/releases/tag/v0.67.0) 已公开，非 draft、非 prerelease；Agent、Node、部署归档及许可附件完整，重新下载后 `SHA256SUMS` 全部通过。
- annotated tag object 为 `54c66c99791f5c1c18f7352d12b831c4bfb0dc3c`，指向正式提交 `6aaabea`。
- `docker.io/kjlion/kejilion-panel:0.67.0` 与 `latest` 均指向 OCI index `sha256:f90c4c667eb180b407dccec4901c48d2cf691578fb7abdcfac92a72df7c32b03`。
- `linux/amd64` 子清单为 `sha256:9f8f4825db7f045eb47df130c7c1fc51ed1678e52ce13d197dbcb0c777e001c2`；`linux/arm64` 子清单为 `sha256:3edbbff00a379b2daad88994117f41415e82bcd72f2ba175eebc5864b2af7a96`，其余 `unknown/unknown` 条目为构建证明。
- 154 从 Docker Hub 重新拉取 `0.67.0` 后，公开镜像 E2E 输出 `image_e2e=pass`，并确认测试容器、网络和临时目录已清理。
- apps 仓库 `b34a3992806b65fac789a0a28cb7018b1cbec501` 的 `kpanel.conf` 与本候选完全一致，文件 SHA-256 均为 `201ae7babced0e9a5efb7833c8491a81c61f8603323067b91ed9e2b1ec2890fe`，本版本无需 apps 提交。

## 生产部署与持续观察

- 生产真源为 `arena-154` 的 `http://154.36.153.9:8080`。`kp.kejilion.pro` 不属于本次生产目标。
- 升级前 Panel 为 `0.66.0`、healthy、0 restart、OOM false；Agent 为 `0.66.0 v1alpha1`、active，systemd 无待 reload。
- 停写一致性备份目录：`/root/kpanel-backups/v0.67.0-preupgrade-20260812T060803Z`；归档：`/root/kpanel-backups/v0.67.0-preupgrade-20260812T060803Z.tar.gz`；SHA-256 `e56d99bf91f7b5144a362d0661c4f2a970b6cbf651818607af167845e23d3c6d`。源数据、归档清单、逐文件哈希、权限、SQLite、JSON 和 JSONL 均在独立恢复目录复核通过，恢复 `v0.66.0` 健康后才继续升级。
- 使用标准入口 `KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update bash /home/docker/kpanel/bin/kejilion.sh app kpanel` 更新成功，输出 `KPanel 更新完成 / Update Complete`，没有触发自动回滚。更新日志 SHA-256 `4b8a043be07ae66db6e85623ad36c65b6747714ccbcc784a0a17d6aa719b6e76`。
- 上线后本机与公网 Panel 均为 `0.67.0/ok/v1alpha1`、healthy、0 restart、OOM false；Agent 为 `0.67.0 v1alpha1`、active。
- 生产镜像 revision、version、公开 OCI index、固定脚本提交/摘要、Agent 二进制、应用市场配置、权限与资源边界均核对通过。Agent SHA-256 `66938f8f7e6faccd0e1737cc00083108ee3da4d397fcc076e30cb0f890c20b22`。
- 固定脚本仍为 `kejilion/sh@28f89c1b34df4b25e6ef9b144c328fdea75dbac9`，raw SHA-256 `0583f7cd5be1f0bb6ec48d92e2cf224bfabfafada5788658bda4414ba9561229`；宿主安装脚本只存在预期的许可/区域/统计设置差异。
- `.env`、Compose、Agent 配置、systemd、Agent token、AI secret 和集群身份均与升级前备份一致。SQLite quick check、5 个 JSON 和 15342 条 JSONL 解析通过；最近 Panel/Agent 日志无 panic、fatal、segmentation fault 或 OOM。生产验证日志 SHA-256 `0454c0822a77dd7f30ac07a7f73e2d8f2a6f1628007156c8f9e98acc7a4c721d`。
- 60 次、2 秒间隔持续采样从 `2026-08-12T06:10:55Z` 至 `06:13:11Z` 全部通过，同时覆盖本机/公网 Panel、容器健康/重启/OOM 和 Agent；采样 SHA-256 `7660061ae81340e563b78615d7c26e64ba080bbed401d058a6cc63598c464b9a`。

## 回滚

- 源码 / Tag：`v0.66.0` / `b96f38dfada3fafa8286dbf2d7ee4be8937fc804`。
- 镜像：OCI index `sha256:2abfb267da7a1ead720dbe3d44928a2fe76e3f3ef2efe727b7082c62718bbd99`。
- 停止 Panel 与 Agent，按已验证备份恢复 `/home/docker/kpanel`、`/root/apps/kpanel.conf` 及 systemd 链接，执行 `systemctl daemon-reload`，将 `latest` 固定回上一镜像摘要后恢复 Agent 与 Panel。
- 回滚后复核 `0.66.0` 版本、Panel 健康、容器重启/OOM、Agent、数据完整性和公网入口。历史 `v0.67.0` Tag、Release 与版本镜像保持不可变。

## 限制与沉淀

- 本次生产没有执行 Docker 新建/删除、Compose 写部署或手机端终端写操作；功能写路径由 154 隔离 L2 和真实 Chrome 候选验收覆盖。
- 本轮复用 `release-kpanel` 与既有版本治理流程，没有新增重复工作流；发布事实与精确回滚信息沉淀在本文档。
