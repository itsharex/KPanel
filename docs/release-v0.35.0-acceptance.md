# KPanel v0.35.0 发布验收

发布日期：2026-08-02

## 发布范围

- 新增基于 TOTP 的双因素认证，默认关闭，由管理员在设置页主动启用。
- 支持验证器二维码、手工密钥、一次性恢复码、恢复码轮换和关闭 2FA。
- 登录流程在密码验证后进入独立的二次验证阶段；未完成 TOTP 或恢复码验证前不会创建登录会话。
- 恢复码仅保存不可逆摘要，使用后立即失效；启用、轮换和关闭 2FA 会撤销既有会话。
- 优化登录成功后的控制台切换，预加载首屏并减少生硬的空白等待；提交 `7043c35` 的等价补丁已由 `fabf094` 纳入本次发布。

## 源码与自动化

- 登录切换提交：`fabf0948cd3660b0c786b05f05e51a2781407cc0`
- 2FA 功能提交：`19edc3e641717cc4b44b8d3698a949f855e8c9cc`
- 发布准备提交：`95f3c471f8e65fd57430332551cda19442759130`
- TOTP 路径隔离测试提交及发布修订：`a09fa961b2487b1d012d89422f2137a2818a9e21`
- 标签：`v0.35.0`
- 候选 CI：[30708672125](https://github.com/kejilion/KPanel/actions/runs/30708672125) — 成功
- 主线 CI：[30708736864](https://github.com/kejilion/KPanel/actions/runs/30708736864) — 成功
- Release：[30708830543](https://github.com/kejilion/KPanel/actions/runs/30708830543) — 成功
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.35.0>

GitHub Release 已公开并包含 6 个附件：amd64/arm64 Agent、部署归档、`SHA256SUMS`、许可证和第三方声明。

## 功能、安全与性能验收

- Web：29 个测试文件、198 项测试通过；类型检查、生产构建、预压缩和 npm 高危依赖审计通过。
- Go：全量测试、`go vet`、认证/存储/Panel 竞态检测、`govulncheck v1.6.0` 及 amd64/arm64 的 Panel、Agent、`kpctl` 构建通过。
- 部署：全部 Shell 语法、安装安全测试、`kejilion.sh` 应用生命周期和候选镜像运行契约检查通过。
- 浏览器实机：在 154 隔离候选环境验证密码后 TOTP 挑战、正确 TOTP 登录、恢复码轮换、恢复码单次登录与重复使用拒绝；浏览器控制台无项目错误。
- TOTP 密钥使用独立受保护路径，测试覆盖目录权限、文件权限、符号链接拒绝及与普通 Panel 状态文件的隔离。
- 登录密码和第二因素均由服务端独立验证；二次验证状态短时有效、不可直接升级为登录会话，恢复码不会以明文持久化。

## Docker Hub

- `docker.io/kjlion/kejilion-panel:0.35.0`
- `docker.io/kjlion/kejilion-panel:latest`
- 两个标签的 OCI index 摘要一致：
  `sha256:fa0f1d24370ea0d93543ea2fa4ec69dbec2375cf85b9fc7f13b88b8d3c6f5c2b`
- linux/amd64：
  `sha256:982c0819897082190d2d43311561688f3a172331c3b00eef6ec91c0b0d8fa293`
- linux/arm64：
  `sha256:f8d4eef048c1e96c7a21524684905083f420aed8e6ce1eaa2e764e985a14d32e`

清单中的 `unknown/unknown` 项为 provenance/SBOM attestation，不是缺失架构。OCI 标签已核对版本 `0.35.0`、源码修订 `a09fa961b2487b1d012d89422f2137a2818a9e21` 和 AGPL-3.0-only 许可。

## 实机与应用市场兼容性

- 154 验收机从 Docker Hub 拉取上述不可变摘要并执行隔离运行时 E2E，输出 `image_e2e=pass`。
- 154 随后通过现有 `k app kpanel` 应用市场更新链路升级；生产容器版本与 Agent 版本均为 `0.35.0`，容器为 `running/healthy`，`kejilion-agent.service` 为 `active`。
- 本次未修改 `kejilion.sh`、Compose 或 `packaging/kejilion-app/kpanel.conf`；应用市场继续使用 `latest`，无需修改或提交 `kejilion/apps`。
- 没有站点、容器、用户文件或部署参数迁移；2FA 默认关闭，不改变现有用户的密码登录方式。

## 回滚

- 源码：`v0.34.9`（`b7c25f4e7dc5b82779a0eedd358ffe8ff785c7c8`）
- 镜像：`docker.io/kjlion/kejilion-panel:0.34.9`
- 镜像摘要：
  `sha256:70ad6be0e8602e3f49bcc71905bfe763f5b4ed71acd6b66c6c696310a7fd9037`

回滚前应确认管理员已保存恢复码。回滚到不支持 TOTP 的版本会恢复密码登录界面，不删除用户、站点、容器或文件；再次升级后仍以持久化状态为准。
