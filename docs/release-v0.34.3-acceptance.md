# KPanel v0.34.3 发布验收

发布日期：2026-08-01

## 发布范围

- 侧栏更新提示改为明确显示“更新可用”，不再把当前 Agent 版本误显示为目标版本。
- 概览自动刷新改为在完整数据就绪后一次性更新页面，消除周期性卡片闪动和布局跳动。
- 文件列表多选时阻止浏览器同时选中文字，保留编辑器、预览和输入框的正常文本选择能力。

## 源码与自动化

- 功能提交：`c667554`、`f894e23`、`20abcbe`
- 发布提交：`333d0a26fdfc0165c08fc79a1793e7242fbe6545`
- 标签：`v0.34.3`
- 候选 CI：[30682979157](https://github.com/kejilion/KPanel/actions/runs/30682979157) — 成功
- 主线 CI：[30683041498](https://github.com/kejilion/KPanel/actions/runs/30683041498) — 成功
- Release：[30683125943](https://github.com/kejilion/KPanel/actions/runs/30683125943) — 成功
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.34.3>

Release 附件已确认包含：

- `kejilion-agent-linux-amd64`
- `kejilion-agent-linux-arm64`
- `kejilion-panel-deploy-0.34.3.tar.gz`
- `SHA256SUMS`
- `LICENSE`
- `THIRD_PARTY_NOTICES.md`

## L3 验收

- Web：27 个测试文件、178 项测试通过；类型检查和生产构建通过。
- Go 测试、Go Vet、部署安全测试和 `kejilion.sh` 应用生命周期测试通过。
- `govulncheck` 与 npm 高危依赖审计通过；npm 报告 0 个漏洞。
- 原生镜像运行时契约、非 root 用户、健康检查、固定脚本摘要和许可证核对通过。
- linux/amd64、linux/arm64 Agent 与多架构镜像构建通过。

## Docker Hub

- `docker.io/kjlion/kejilion-panel:0.34.3`
- `docker.io/kjlion/kejilion-panel:latest`
- 两个标签的 OCI index 摘要相同：
  `sha256:a7ee44b854b5b8253d961c66729792bde994a13a3f2da9b8a85f34a19c12bc64`
- linux/amd64：
  `sha256:5237e255b5184e6e48379acad9c343135eb2e0b4b568096e2d2a073b5d966dd1`
- linux/arm64：
  `sha256:5a701f544ec5df8c58108cc2e9e801a69c3156221ae78d35129b06a4476db094`

`unknown/unknown` 条目为各平台对应的 provenance/SBOM attestation，不是缺失架构。

## 应用市场与兼容性

本次没有修改 Panel/Agent API、协议、部署 Compose、安装更新脚本或
`packaging/kejilion-app/kpanel.conf`。应用市场继续使用 `latest` 并按现有流程识别更新，
不需要修改或提交 `kejilion/apps`。

## 回滚

- 源码：`v0.34.2`（`f49969d6b3f4b1d4ad0c99a63bfecca22923b739`）
- 镜像：`docker.io/kjlion/kejilion-panel:0.34.2`
- 本次没有状态格式、数据库、Agent 权限或部署契约迁移；回滚不会修改 Panel 数据、宿主机文件、站点或 Docker 业务资源。
