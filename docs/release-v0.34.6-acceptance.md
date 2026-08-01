# KPanel v0.34.6 发布验收

发布日期：2026-08-01

## 发布范围

- 交互式终端继续使用固定深色工作区，光标、选区、边框、状态、输入框和按钮统一使用 KPanel 品牌与语义色。
- 终端保留 ANSI 红、黄、蓝、紫、青等原生输出颜色，不改变 PTY、输入、轮询、重连或 URL 跳转协议。
- 代码编辑器继续使用固定深色工作区，统一光标、选区、活动行、查找替换和滚动条样式，同时保留多色语法高亮。
- 本次没有数据库迁移、Agent/Panel API 变更、部署参数变更或 `kejilion.sh` 变更。

## 源码与自动化

- 功能提交：`a652d612ee2f93fcd638c403dff7f42663d58a38`
- 发布提交：`5a465bf479efed0936d6a0fa2325032799eab2e5`
- 标签：`v0.34.6`
- 候选 CI：[30691085960](https://github.com/kejilion/KPanel/actions/runs/30691085960) — 成功
- 主线 CI：[30691175394](https://github.com/kejilion/KPanel/actions/runs/30691175394) — 成功
- Release：[30691248702](https://github.com/kejilion/KPanel/actions/runs/30691248702) — 成功
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.34.6>

GitHub Release 已公开并包含 8 个附件，包括 Agent 的 amd64/arm64 二进制、部署归档、`SHA256SUMS`、许可证和第三方声明。

## 验收结果

- Web：28 个测试文件、182 项测试通过；类型检查和生产构建通过。
- 主题回归：固定深色工作区、KPanel 品牌变量接入及旧紫色/蓝色强调色移除均有自动测试覆盖。
- 安全：Release 工作流中的 `govulncheck`、`npm audit --audit-level=high`、应用生命周期和镜像运行契约检查全部通过；本地 `npm audit` 同样为 0 个漏洞。
- 154 验收机重新拉取公开不可变摘要并执行隔离运行时 E2E，输出 `image_e2e=pass`。
- 154 验收前后生产 `kejilion-panel` 均为 `running/healthy`，`kejilion-agent.service` 均为 `active`；临时容器和网络无残留，未替换或重启生产 KPanel。

## Docker Hub

- `docker.io/kjlion/kejilion-panel:0.34.6`
- `docker.io/kjlion/kejilion-panel:latest`
- 两个标签的 OCI index 摘要一致：
  `sha256:3c6656a21de1dfca5a949ca7ea34d64df387730836a285108ac7d9e6aa424682`
- linux/amd64：
  `sha256:e68819aaa7771fcf504416ded375d31e6bf0086724cd7620a56f454ac90f087c`
- linux/arm64：
  `sha256:26cf24a9c9a67bb039e761dbc9241b984fcae036741b470d6913256a9dfa07f0`

清单中的 `unknown/unknown` 项为 provenance/SBOM attestation，不是缺失架构。

## 应用市场兼容性

- 本次未修改 `kejilion.sh`、Compose 或 `packaging/kejilion-app/kpanel.conf`。
- 应用市场继续使用 `latest`，版本与 Agent 契约由 OCI 标签及镜像内 `/release/VERSION` 动态校验，无需修改或提交 `kejilion/apps`。
- 更新不会改动网站、应用、Docker 业务资源、`/home/web`、集群凭据或 Panel 数据。

## 回滚

- 源码：`v0.34.5`（`da0f8b6d7dccabe1c8ad08b9a060728d6447b767`）
- 镜像：`docker.io/kjlion/kejilion-panel:0.34.5`
- 镜像摘要：
  `sha256:b7b1179a85938300e0fc24082491d1f608ea143a2197aee620d51260cc12b306`

本次仅涉及前端静态资源，回滚不需要数据迁移或配置转换。
