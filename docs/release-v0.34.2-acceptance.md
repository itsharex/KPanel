# KPanel v0.34.2 发布验收

发布日期：2026-08-01

## 发布范围

- 代码编辑器的查找与自动换行入口改为右上角图标按钮，减少底部操作区占用。
- 查找替换改为接近 VS Code 操作习惯的紧凑悬浮面板。
- 支持 `Ctrl+F`、`Ctrl+H`、`F3`、`Shift+F3`、`Esc`，以及大小写、全字、正则、单次替换和全部替换。
- 查找面板继续随 CodeMirror 按需加载，不增加首屏依赖。

## 源码与自动化

- 功能提交：`4cae5a79e33c012776ec97bb4841ddd8aea051b6`
- 发布提交：`f49969d6b3f4b1d4ad0c99a63bfecca22923b739`
- 标签：`v0.34.2`
- 候选 CI：[30681512968](https://github.com/kejilion/KPanel/actions/runs/30681512968) — 成功
- 主线 CI：[30681605768](https://github.com/kejilion/KPanel/actions/runs/30681605768) — 成功
- Release：[30681704763](https://github.com/kejilion/KPanel/actions/runs/30681704763) — 成功
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.34.2>

Release 附件已确认包含：

- `kejilion-agent-linux-amd64`
- `kejilion-agent-linux-arm64`
- `kejilion-panel-deploy-0.34.2.tar.gz`
- `SHA256SUMS`
- `LICENSE`
- `THIRD_PARTY_NOTICES.md`

## L3 验收

- Web：26 个测试文件、175 个测试通过；类型检查和生产构建通过。
- 实际浏览器验证查找入口、替换展开、`Ctrl+F` 和 `Esc` 行为通过。
- 首屏主包为 `92.26 kB`，gzip 后 `34.27 kB`；本次没有新增运行时依赖。
- Linux Release 工作流中的 Go 测试、Go Vet、部署安全测试和脚本应用生命周期测试通过。
- `govulncheck` 和 npm 高危依赖审计通过，npm 报告 0 个漏洞。
- 原生镜像运行时契约、非 root 用户、健康检查、固定脚本摘要和许可证核对通过。
- linux/amd64、linux/arm64 Agent 与多架构镜像构建通过。

## Docker Hub

- `docker.io/kjlion/kejilion-panel:0.34.2`
- `docker.io/kjlion/kejilion-panel:latest`
- 两个标签的 OCI index 摘要相同：
  `sha256:4059e4423344c8364cfe86e3ce831874659dad9278d4c02b428d838efed29105`
- linux/amd64：
  `sha256:27a7c2c01c9a47754bf4edd69977ff5534d4d2e86412f4fd45c1e1bec5c1b284`
- linux/arm64：
  `sha256:aebb3bb49b2a43f5d66ab731792185e7c2e836a2ef9976b66d65e7249f30ba3a`

`unknown/unknown` 条目为每个平台对应的 provenance/SBOM attestation，不是缺失架构。

## 应用市场与兼容性

本次没有修改 Panel/Agent API、协议、部署 Compose、安装更新脚本或
`packaging/kejilion-app/kpanel.conf`。应用市场继续使用 `latest` 并按现有流程识别更新，
不需要修改或提交 `kejilion/apps`。

## 回滚

- 源码：`v0.34.1`（`2599f7d9174af463afe10990d4987c673adf5666`）
- 镜像：`docker.io/kjlion/kejilion-panel:0.34.1`
- 本次没有状态格式、数据库、Agent 权限或部署契约迁移；回滚不会修改 Panel 数据、宿主机文件、站点或 Docker 业务资源。
