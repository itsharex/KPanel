# KPanel v0.40.1 发布验收

发布日期：2026-08-03

## 发布范围

- 终端工作区将主机、连接状态和关闭操作整合到多标签栏，移除重复的长条标题区。
- 终端支持占满浏览器内容区的全屏模式；全屏时保留标签切换、状态提示、预输入框和退出入口，并支持 `Escape` 退出。
- 设置页的两步验证移动到安全入口下方，保持安全设置的阅读和操作顺序一致。
- 修改用户名时不再由 KPanel 预填当前密码；密码框经用户聚焦后允许浏览器密码管理器主动填充。
- 本次没有端口、持久化数据格式、部署配置、应用市场契约或 `kejilion.sh` 协议迁移。

## 自动化与 Linux 验收

- 本地版本一致性检查、`git diff --check`、Web 类型检查和生产构建通过。
- Web 35 个测试文件、233 项测试全部通过；1458 条多语言资源检查通过。
- 154 隔离仓库使用精确提交 `4a6ca1011ce9ace8d40cc6f8f5c32ecf6deb85a3` 执行完整 `make verify-release`，L3 通过。
- Go 全量测试、核心包 race、`go vet`、`govulncheck`、amd64/arm64 构建通过；未发现已调用漏洞。
- `npm audit`、Trivy 源码、Dockerfile 与最终镜像扫描通过，未发现阻断发布的问题。
- 应用生命周期复核输出 `app_conf_lifecycle=pass`；公开版本镜像复核输出 `image_e2e=pass`。

## 发布产物

- 功能提交：`dec9ebd`。
- 密码管理器兼容修复：`94d669b`。
- 版本准备与标签提交：`4a6ca1011ce9ace8d40cc6f8f5c32ecf6deb85a3`。
- 候选分支 CI：[30796589684](https://github.com/kejilion/KPanel/actions/runs/30796589684)。
- 主线 CI：[30796858947](https://github.com/kejilion/KPanel/actions/runs/30796858947)。
- Release 工作流：[30797041946](https://github.com/kejilion/KPanel/actions/runs/30797041946)。
- GitHub Release：[v0.40.1](https://github.com/kejilion/KPanel/releases/tag/v0.40.1)，非草稿、非预发布。
- `docker.io/kjlion/kejilion-panel:0.40.1` 与 `latest` 均指向 OCI 摘要 `sha256:6ca170f69ce8291be1b4499296faa31810a6002ce7af21716b9670f23a0c08cb`。
- 镜像包含 `linux/amd64` 和 `linux/arm64`；平台清单摘要分别为 `sha256:a92efd7b210abf28452fb13cadb386c3c1393e0da7b9546af45313cebb796a89` 与 `sha256:b6b5f96d836d2af91b6e025cebcc3b3d1cf6a30e63a73ca0ef9a9c88dfa676ea`。
- Agent SHA-256：amd64 `21b68a1bc1e34d13afe8522e632be2a50a41b9edd9f0a59e67ce5256d3f702b6`，arm64 `1cac1bbe1b712526c30a72f281174985a3d62ec09870c83eea2072460086ebb4`。
- 轻量节点 SHA-256：amd64 `2e3cc69850149b0ca6c37f5cecf8dff23cdf7a63fa050ab9ce878df8b33aaf22`，arm64 `e045337a87de3bd7e853d32d28ba983ae1a58af37a6d8565042af0f5b43cb69d`。
- 部署包 SHA-256：`9bc1dde0bd4dde6b17abf6205bb20eafe333facbfdaf90781586eaf3706cdfca`。

## 154 线上复核

- 通过现有非交互应用协议 `KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update k app kpanel` 从 `0.40.0` 升级到 `0.40.1`。
- `kejilion-panel` 为 `running/healthy`，重启次数为 0；8080 端口映射保持不变，本机 HTTP 返回 200。
- 容器 OCI revision 为 `4a6ca1011ce9ace8d40cc6f8f5c32ecf6deb85a3`，version 为 `0.40.1`。
- `kejilion-agent` 为 `0.40.1 v1alpha1`，保持 `active/enabled`，`NeedDaemonReload=no`。
- Panel 数据目录顶层项目数量升级前后均为 8；未删除现有业务数据、配置或审计记录。
- 线上 Agent SHA-256 与公开镜像 `/release/kejilion-agent` 一致，均为 `46a818b2353c498281189d45e377fd2ea8f9d1ca6c3244a7ddfcc22a75fccecb`。
- 更新后近期 Panel 与 Agent 错误日志均为 0，临时终端 systemd 单元为 0。

## 兼容、风险与回滚

- `packaging/kejilion-app/kpanel.conf` 相对 `v0.40.0` 未变化，因此本次无需修改或发布 `kejilion/apps`。
- 自动化、构建和 154 运行态均已验证；需要登录态的终端全屏与设置页视觉交互未使用生产账号进行独立浏览器验收，由前端测试、生产构建和 154 服务健康检查覆盖基础回归。
- 回滚点为 `v0.40.0`、提交 `e906f3241891f58cf5bf78e15b05b4cf4352ce5c`，对应镜像摘要 `sha256:20bdabf841635020ff4cc1a611e05d2429898821ce1e727aee0145e07791b4bc`。
- 回滚只替换 Panel/Agent 版本，不删除现有业务数据、配置和审计记录。
