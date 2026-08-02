# KPanel v0.36.0 发布验收

发布日期：2026-08-02

## 发布范围

- 建立前端多语言基础架构，首批支持 `zh-CN` 与 `en-US`。
- 未保存用户偏好时，浏览器首个有效语言以 `zh` 开头则使用中文，其余语言统一使用英文。
- 用户手动切换语言后写入本地偏好，后续访问固定使用该选择；同时同步 `document.lang` 与页面标题。
- 中文资源随首屏加载，英文资源按需异步加载，避免为默认中文用户增加不必要的初始体积。
- 登录、初始化、全局导航、公共错误、设置语言入口等全局界面已接入翻译；业务页面按既定迁移规范继续分批接入，不能将本版本描述为“全站翻译完成”。
- 未识别的服务端错误保持原文，不以错误翻译掩盖真实故障信息。

## 源码与自动化

- 多语言功能提交：`843ecaa946fe40beab754a65ae91e0aafc3bdde4`
- 发布准备提交：`2b6750d7e9ebcbd46e91c681cf8375c20496a8b9`
- 标签：`v0.36.0`
- 候选分支 CI：<https://github.com/kejilion/KPanel/actions/runs/30730387046> — 成功
- 主线 CI：<https://github.com/kejilion/KPanel/actions/runs/30730439450> — 成功
- Release：<https://github.com/kejilion/KPanel/actions/runs/30730507495> — 成功
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.36.0>

GitHub Release 已公开并包含 6 个附件：amd64/arm64 Agent、部署归档、`SHA256SUMS`、许可证和第三方声明。

## 功能、性能与安全验收

- Web：30 个测试文件、205 项测试通过；类型检查、生产构建和 `npm audit` 通过，审计结果为 0 个已知漏洞。
- 多语言：覆盖浏览器语言归一化、默认语言、手动选择持久化、动态资源加载、页面标题与全局界面翻译。
- 体积：英文语言包独立切分，约 2.24 KiB；首屏 gzip 增量处于项目规定的 8 KiB 预算内。
- Go：Linux 全量 `go test ./...`、`go vet ./...`、amd64/arm64 的 Panel、Agent、`kpctl` 构建通过。
- 漏洞扫描：`govulncheck v1.6.0` 未发现可达调用链漏洞；报告的 1 个模块漏洞不在当前程序调用路径中。
- 部署：Shell 语法、安装安全测试、候选镜像非 root/健康检查/只读运行/网络与 capability 约束均通过。
- 线上浏览器：154 登录页实际加载成功，标题为 `KPanel`，中文自动选择、语言入口与安全登录文案正常渲染。浏览器控制连接在点击语言切换时超时，因此切换后的持久化以已通过的自动化测试为验收依据。

## Docker Hub

- `docker.io/kjlion/kejilion-panel:0.36.0`
- `docker.io/kjlion/kejilion-panel:latest`
- 两个标签的 OCI index 摘要一致：
  `sha256:0bba49179d9ea0787836154c8bbea0f59d666a418620b61c9ddf8bfd5f8fb736`
- linux/amd64：
  `sha256:f07b783a076fa0e508dc97ca8f2abefc98d10d9dd77b2aebf962bfc337d6f696`
- linux/arm64：
  `sha256:a01a981ea9ed76fa2ee429f400ed0ba844bfa9aa67892e4be340259637a0e41a`

清单中的 `unknown/unknown` 项为 provenance/SBOM attestation，不是缺失架构。

## 154 实机与应用市场兼容

- 154 隔离目录完成 Web、Go、安全、双架构构建和候选镜像运行验收。
- 154 通过现有 `k app kpanel` 应用市场更新核心流程升级，未改写生产 Compose、端口、数据目录或反向代理配置。
- 升级后生产容器为 `running/healthy`，镜像版本 `0.36.0`，源码修订 `2b6750d7e9ebcbd46e91c681cf8375c20496a8b9`。
- `kejilion-agent.service` 为 `active`，协议版本 `v1alpha1`，无需重新加载 systemd 配置。
- `/api/v1/health` 返回 `initialized: true`、`status: ok`、`version: 0.36.0`；升级后 30 分钟内未发现 Panel 或 Agent 的 `panic`、`fatal`、`error`、`failed` 日志。
- 本版本不要求同步更新 `kejilion.sh` 或 `kejilion/apps`；应用市场继续使用 `latest` 标签并保持现有业务入口。
- 本次没有迁移用户、站点、容器、文件或登录数据；语言偏好仅保存在当前浏览器 `localStorage`。

## 回滚

- 源码：`v0.35.0`（`a09fa961b2487b1d012d89422f2137a2818a9e21`）
- 镜像：`docker.io/kjlion/kejilion-panel:0.35.0`
- 镜像摘要：
  `sha256:fa0f1d24370ea0d93543ea2fa4ec69dbec2375cf85b9fc7f13b88b8d3c6f5c2b`

回滚不会删除用户、站点、容器或文件。旧版本会忽略浏览器中保存的多语言偏好；再次升级后可继续读取该偏好。
