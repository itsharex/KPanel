# KPanel v0.24.1 验收记录

## 交付范围

- 标准化 KPanel 品牌、PWA/SEO 与本地轻量发行版图标。
- 限制登录密码校验、Docker 批量探测和宿主机采集的并发与重复请求。
- 静态资源增加预压缩、不可变缓存，API 响应按安全类型启用 gzip。
- 增加 HSTS、COOP、CORP 等网络响应头，同时保留可信反向代理下的 `k fd`
  域名访问、Secure Cookie 与动态域名行为。
- 建立运行时性能基线、安全加固报告及长期开发规范。
- 发布流水线统一使用 Node.js 24.18.0，并增加 Go 可达漏洞和 npm 高危依赖审计。

## 自动与本地验收

- Linux Go：`go test ./...`、`go vet ./...` 全部通过。
- 竞态：`go test -race ./internal/panel ./internal/auth ./internal/dockerx` 通过。
- Web：Node.js 24.18.0 下 TypeScript 检查、55 项 Vitest 和生产构建全部通过。
- Agent：linux/amd64、linux/arm64 静态编译通过。
- 部署：安装安全测试、Shell 语法、生态策略和 `kejilion.sh` 应用生命周期测试通过。
- 镜像：以 `read-only`、`cap-drop ALL`、`no-new-privileges`、无网络和受限
  `tmpfs` 启动后健康检查通过。
- `k fd`：`TestTrustedHTTPSProxyAllowsKFDOriginAndSecureCookies` 回归测试通过。
- 主分支 CI：
  <https://github.com/kejilion/KPanel/actions/runs/30356260820>
- Release：
  <https://github.com/kejilion/KPanel/actions/runs/30356350389>

## 安全验收

- `govulncheck v1.6.0`：0 个可达漏洞；依赖模块中有 1 个未被代码调用的漏洞。
- `npm audit --audit-level=high`：0 个漏洞。
- Trivy 0.72.0 最终镜像：`paneld`、`kejilion-agent` 均为 0 High/Critical。
- Trivy 源码：Go/npm 依赖 0 High/Critical、0 secret，Dockerfile 0 高危配置。
- Trivy 数据库来自官方 `ghcr.io/aquasecurity/trivy-db:2`，OCI 数据层
  `sha256:efd9a10cbee40f87d5717e8efbc29824881104509de44bcc0347f3c5dd881174`
  在使用前完成 SHA-256 校验。

## 线上核查（2026-07-28）

- GitHub Release [v0.24.1](https://github.com/kejilion/KPanel/releases/tag/v0.24.1)
  已发布且不是草稿，包含 linux/amd64、linux/arm64 Agent、部署包和 `SHA256SUMS`。
- 发布附件已重新下载并逐项核验：
  - `kejilion-agent-linux-amd64`：
    `12554b8cbcbff9125ca5d346b5cb5e1abaa4c2d44314e49ce6b3b2d775d29012`
  - `kejilion-agent-linux-arm64`：
    `2ca043466a01443c94a71dbc0f1354fe6a3d3dac58c940cdb7b02a0851ab7f11`
  - `kejilion-panel-deploy-0.24.1.tar.gz`：
    `bf03859b81563ab65260f90bde141b5cbf52b8b6f8fbe8a3d9ccbde2592935fc`
- linux/amd64 Agent 实际执行 `version` 输出 `0.24.1 v1alpha1`；部署归档
  `VERSION` 为 `0.24.1`。
- Docker Hub `0.24.1` 与 `latest` 均指向
  `sha256:d6da7b6be360520a6847d7a39168520ed1e16db22c8c5c16ff4a82089b3e55ce`。
- 平台镜像：
  - linux/amd64：
    `sha256:2772ba0391a05bf7d858b943bb5c9ca22f1247abf13031b766cf81abad210f67`
  - linux/arm64：
    `sha256:e6242b503a34b9aac9f914c94c72b5752a7d92e10cb7cd0bf1aa7364173494e5`
- Docker Hub OCI 配置确认版本为 `0.24.1`、源码提交为
  `0b4564eda5522cc627948706f67a7dd55f556e9a`、运行用户为 `65532:65532`、
  健康检查为 `CMD /paneld healthcheck`，并包含两个 SBOM/Provenance 证明清单。
- 镜像固定脚本提交为 `cd4a97823e95f4029f6cb3a82249f2adf5d53763`，
  SHA-256 为 `dec802845150762a977c2dbf300a7ccf20e2e95135f9a4e4e751069d1a834259`。

## 回滚

- KPanel 镜像可回滚到 `0.24.0` /
  `sha256:d9d54d7945d0a3510780134e7a65475560d1cc69e1fd20e1cb9a571b15829d02`。
- Git 回滚点为标签 `v0.24.0`。
- 本版本没有数据库格式迁移；回滚 KPanel/Agent 不删除 `/home/web`、应用容器、
  域名配置或环境备份。

## 边界说明

- 本轮发布更新 GitHub Release 与 Docker Hub，没有直接替用户升级任一生产主机。
- 干净环境的安装安全、镜像启动和更新产物已自动验证；不同发行版宿主机的实际
  nftables、systemd、DNS 和 Docker 策略仍以各机器更新后的实机结果为准。
