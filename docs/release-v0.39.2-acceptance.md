# KPanel v0.39.2 发布验收

发布日期：2026-08-03

## 发布范围

- 修复多主机数量较多时终端区域被挤压、底部输入区不可见的问题，连接列表与终端区域改为独立滚动。
- 统一本机、远程主机、体检、应用、建站和环境管理交互终端的滚动回看、智能跟随、回到底部及预输入体验。
- 终端输入采用有界短批次发送与失败重试，降低高延迟链路连续输入卡顿，同时保留逐键交互能力。
- 本机终端通过临时 systemd PTY 在独立权限边界运行；主 Agent 继续保持 `ProtectSystem=strict`。
- 本次没有数据格式、端口、部署配置、应用市场契约或 kejilion.sh 协议迁移。

## 自动化与 Linux 验收

- 候选分支 CI、主线 CI 和 Release 工作流均成功。
- 154 隔离目录使用明确基线 `372f49063671a1d8f5e4cff1aa90e03bcce7e67b` 执行完整 `make verify-release`，识别 19 个变更文件并通过 L3。
- Go 全量测试、核心包 race、`go vet`、`govulncheck`、amd64/arm64 交叉构建通过。
- Web 35 个测试文件、225 项测试、类型检查、1447 条多语言资源检查和生产构建通过。
- `npm audit`、Trivy 源码与最终镜像扫描通过，未发现阻断发布的已调用漏洞或高危镜像问题。
- 应用生命周期测试输出 `app_conf_lifecycle=pass`；公开镜像执行 `image-e2e.sh` 输出 `image_e2e=pass`。
- 154 隔离环境在 `ProtectSystem=strict` 父服务中完成真实 Linux PTY 验收，临时终端可在限定权限内写入所需系统目录，退出后无 `kpanel-terminal-*` 残留单元。

## 发布产物

- 功能提交：`6344756`；版本提交与 `v0.39.2` 标签：`649711082d03ab3b4dfe4ea9e84b11c1360c8bef`。
- 候选分支 CI：[`30784721827`](https://github.com/kejilion/KPanel/actions/runs/30784721827)。
- 主线 CI：[`30784843981`](https://github.com/kejilion/KPanel/actions/runs/30784843981)。
- Release：[`30784958833`](https://github.com/kejilion/KPanel/actions/runs/30784958833)。
- GitHub Release：[`v0.39.2`](https://github.com/kejilion/KPanel/releases/tag/v0.39.2)，非草稿、非预发布。
- `docker.io/kjlion/kejilion-panel:0.39.2` 与 `latest` 均指向 OCI 摘要 `sha256:d6107799bdd98329b2c9530164753d0d6ca46bd9f5018a7c233eb58e25ac406d`，包含 linux/amd64 与 linux/arm64。
- Release 附件 Agent SHA-256：amd64 `e65af352d0e70e771d72c2318a35d9619c44b2fc6a3f4a5500bb1d1594c26ed2`，arm64 `9565d1663b7f279c12161a6de07c340c2f014ec920603f01619dd83f38f1a0e8`。
- 轻量节点 SHA-256：amd64 `df9dd8773c558ab0ac07095cbbda560b25dec126571d90d5678925fdcfc5a681`，arm64 `30ca707aa9f6e07b6dc58e0afa3ab502b4ce77652f6d4e1da63bf3241c91b43b`。
- 部署包 SHA-256：`0dd8dbe951206bca7dd818e27e8077257c3eeaf1a354a5e5a37b34770326e7cf`。

Release 附件和容器内分发 Agent 由不同构建步骤生成，因此不能跨产物比较文件摘要。154 线上 Agent SHA-256 为 `a7e77001aafa24d333a94808c804b4b290dfd96291087145c2da2b4ff1f3a348`，与公开 `latest` 镜像内 `/release/kejilion-agent` 逐字节一致。

## 154 线上复核

- 通过既有 `k app kpanel` 更新流程升级，更新后容器版本标签为 `0.39.2`，源码修订为 `649711082d03ab3b4dfe4ea9e84b11c1360c8bef`。
- `kejilion-panel` 容器状态为 `running/healthy`，本机 HTTP 返回 200，现有 8080 端口映射保持不变。
- `kejilion-agent` 保持 `active/enabled`，更新后日志未发现 panic、fatal 或 error，未发现残留终端单元。
- 线上 Agent 与当前公开镜像内分发 Agent 摘要一致。
- 当前自动化环境没有可复用的已登录 KPanel 浏览器会话，因此未把线上人工浏览器视觉检查写成已完成；终端布局由前端测试、生产构建和 154 隔离真机 PTY 验收覆盖。

## 兼容与回滚

- `packaging/kejilion-app/kpanel.conf` 相对 `v0.39.1` 未变化，且与 `kejilion/apps` 主线契约一致，无需同步修改应用市场仓库。
- 回滚点为 `v0.39.1`。回滚只替换 Panel/Agent 版本，不删除现有业务数据、终端审计或配置。
