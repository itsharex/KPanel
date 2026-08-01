# KPanel v0.34.1 发布验收

发布日期：2026-08-01

## 发布范围

- 文件管理批量操作栏统一为桌面端和移动端底部悬浮操作区，保持路径导航、搜索和文件行位置稳定。
- 轻量代码编辑器增加查找替换、跳转行、代码折叠、光标行列状态和自动换行开关，并修复深色主题输入光标不明显的问题。
- CodeMirror 继续按需加载，编辑内容仅在保存时读取完整文本，避免每次按键复制整份文件。
- 全局滚动条统一为 KPanel 深浅主题配色，代码编辑器、交互终端和日志视图使用深色专用样式。

## 源码与自动化

- 发布提交：`2599f7d9174af463afe10990d4987c673adf5666`
- 标签：`v0.34.1`
- 候选 CI：[30679715734](https://github.com/kejilion/KPanel/actions/runs/30679715734) — 成功
- 主线 CI：[30679795041](https://github.com/kejilion/KPanel/actions/runs/30679795041) — 成功
- Release：[30679869025](https://github.com/kejilion/KPanel/actions/runs/30679869025) — 成功
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.34.1>

Release 附件已确认包含：

- `kejilion-agent-linux-amd64`
- `kejilion-agent-linux-arm64`
- `kejilion-panel-deploy-0.34.1.tar.gz`
- `SHA256SUMS`
- `LICENSE`
- `THIRD_PARTY_NOTICES.md`

## L3 验收

- Go `1.26.5`：Linux 环境 `go test ./...`、`go vet ./...` 通过。
- Web：26 个测试文件、175 个测试通过；类型检查和生产构建通过。
- `deploy/tests/install-safety.sh`：GitHub 干净 Ubuntu 环境通过。
- `packaging/tests/app-conf-lifecycle.sh`：一次性 Alpine 容器通过。
- linux/amd64、linux/arm64 的 Panel、Agent、`kpctl` 构建通过。
- 固定脚本提交和 SHA-256、OCI 版本、许可证、非 root 用户、健康检查契约核对通过。
- `govulncheck`：0 个可达漏洞；npm 官方源审计：0 个漏洞。

## Docker Hub

- `docker.io/kjlion/kejilion-panel:0.34.1`
- `docker.io/kjlion/kejilion-panel:latest`
- 两个标签的 OCI index 摘要相同：
  `sha256:5821b9e9574caf3fdb262f83dfa5c63fe0d5d61818a310a24f74239a02493de3`
- linux/amd64：
  `sha256:0abd4b0e73851510418399b4e58a6073446a4057c6fb491d618e480aeab03bda`
- linux/arm64：
  `sha256:d4deb6e6c4085f4c7aec51d32c53133420b1bff6e8061d02db5dc86cf7a43371`

`unknown/unknown` 条目为每个平台对应的 provenance/SBOM attestation，不是缺失架构。

## 应用市场与兼容性

本次没有修改 Panel/Agent API、协议、部署 Compose、安装更新脚本或
`packaging/kejilion-app/kpanel.conf`。应用市场继续使用 `latest`，并从 OCI 标签及镜像内
`/release/VERSION` 动态校验版本，因此不需要修改或提交 `kejilion/apps`。

## 回滚

- 源码：`v0.34.0`（`1dfde09bf70446208fe9ef65f9924605dffe8fcb`）
- 镜像：`docker.io/kjlion/kejilion-panel:0.34.0`
- 回滚不会修改或删除 Panel 数据、宿主机文件、站点、Docker 资源或文件回收站；本次没有状态格式或数据库迁移。
