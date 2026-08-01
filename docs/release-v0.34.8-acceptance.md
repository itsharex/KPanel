# KPanel v0.34.8 发布验收

发布日期：2026-08-01

## 发布范围

- 交互式终端统一空闲与运行状态的深色画布、输入区和滚动条，降低 ANSI 彩色输出偏色，品牌绿色仅用于光标、选区和交互状态。
- 代码编辑器、Docker 日志与控制台、应用日志和体检输出统一使用 KPanel 深色工作区色板。
- 深色工作区统一为 12px 圆角、细边框和轻量内描边；浅色主题下保留浅色弹窗外壳，终端和编辑器画布维持固定深色。

## 源码与自动化

- 终端色彩提交：`a91b121c8ffa5183c40477d0551454a53612b101`
- 工作区统一提交：`3a3fd7db0b6523bb9c909b717127219b95849122`、`ff4858f6c16283248db925714f34a2007417fbd2`
- 发布提交：`ecece66eac1eb473ff57f2d6ed263cae70ffdd09`
- 标签：`v0.34.8`
- 候选 CI：[30694753804](https://github.com/kejilion/KPanel/actions/runs/30694753804) — 成功
- 主线 CI：[30697358397](https://github.com/kejilion/KPanel/actions/runs/30697358397) — 成功
- Release：[30697433533](https://github.com/kejilion/KPanel/actions/runs/30697433533) — 成功
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.34.8>

GitHub Release 已公开并包含 6 个附件：amd64/arm64 Agent、部署归档、`SHA256SUMS`、许可证和第三方声明。

## 功能、安全与性能验收

- Web：28 个测试文件、188 项测试通过；类型检查、生产构建和预压缩通过。
- 主题契约：自动测试覆盖交互终端、体检终端、代码编辑器、Docker 日志与控制台、应用日志的统一工作区变量、边框和圆角。
- 浏览器实机：在 154 隔离候选环境分别检查体检空闲/运行终端、Docker 日志、Docker 控制台和 Nginx 代码编辑器；深浅主题外壳、固定深色画布、圆角和滚动条符合设计，浏览器控制台无项目错误。
- 发布工作流中的 Go 全量测试、Go vet、amd64/arm64 发布构建、`govulncheck`、npm 高危依赖审计、`kejilion.sh` 应用生命周期及镜像运行契约检查全部通过。
- 本次仅调整前端主题样式和版本元数据，没有新增轮询、外部依赖、后台任务或运行时数据写入，资源占用模型不变。

## Docker Hub

- `docker.io/kjlion/kejilion-panel:0.34.8`
- `docker.io/kjlion/kejilion-panel:latest`
- 两个标签的 OCI index 摘要一致：
  `sha256:028a2d5db4ace51b19d688233e8598154042e878eb5d3c0cafbddd2f003e2beb`
- linux/amd64：
  `sha256:6e54d4dac828314d5a6bf9dafabaf78aee594b9afd35c670086001d86b520331`
- linux/arm64：
  `sha256:5f0cec7b534eaba89464e8ef3975ca80a76a22a9a4f90274cbbfcb722f74d649`

清单中的 `unknown/unknown` 项为 provenance/SBOM attestation，不是缺失架构。OCI 标签已核对版本 `0.34.8`、源码修订 `ecece66eac1eb473ff57f2d6ed263cae70ffdd09` 和 AGPL-3.0-only 许可。

## 实机与应用市场兼容性

- 154 验收机从 Docker Hub 拉取上述不可变摘要并执行隔离运行时 E2E，输出 `image_e2e=pass`；镜像内 Agent 返回 `0.34.8 v1alpha1`。
- 验收后生产 `kejilion-panel` 仍为 `running/healthy`，`kejilion-agent.service` 为 `active`；临时容器、网络、Agent、数据目录和 SSH 隧道均已清理，未替换或重启生产 KPanel。
- 本次未修改 `kejilion.sh`、Compose 或 `packaging/kejilion-app/kpanel.conf`。应用市场继续使用 `latest`，无需修改或提交 `kejilion/apps`。
- 没有数据库、站点、Docker 业务资源、文件管理格式或部署参数迁移。

## 回滚

- 源码：`v0.34.7`（`747528adcb70966d1b257109500ced7a4dd3d1d7`）
- 镜像：`docker.io/kjlion/kejilion-panel:0.34.7`
- 镜像摘要：
  `sha256:98c6ee8ed4a501d4cb0481f1a8d7ebbba078c61c90ed6e17b126d003c5383254`

本次没有数据格式或配置迁移；回滚只恢复上一版界面样式，不修改用户文件、容器、站点和面板状态。
