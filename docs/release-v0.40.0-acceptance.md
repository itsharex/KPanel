# KPanel v0.40.0 发布验收

发布日期：2026-08-03

## 发布范围

- 设置页支持修改登录用户名，并保持现有密码、2FA 与会话安全边界。
- 网站列表中的本地目录可直接跳转到文件管理对应目录；只接受文件管理已开放的规范化路径。
- Docker Hub 更新检测在中国大陆、香港及区域未知时增加受控回源：官方源短超时后依次尝试 `docker.1ms.run` 与 `gh.kejilion.pro`。
- 加速仅用于公开 Docker Hub 镜像的更新检测；私有 Registry、摘要固定镜像和实际更新拉取流程保持原逻辑。
- 本次没有端口、数据格式、部署配置、应用市场契约或 kejilion.sh 协议迁移。

## 自动化与 Linux 验收

- 154 隔离仓库使用精确提交 `e906f3241891f58cf5bf78e15b05b4cf4352ce5c` 执行完整 `make verify-release`，L3 通过。
- Go 全量测试、核心包 race、`go vet`、`govulncheck`、amd64/arm64 交叉构建通过。
- Web 35 个测试文件、229 项测试、类型检查、1458 条多语言资源检查和生产构建通过。
- `npm audit`、Trivy 源码与最终镜像扫描通过，未发现阻断发布的已调用漏洞、高危依赖或镜像问题。
- 应用生命周期测试输出 `app_conf_lifecycle=pass`；候选镜像与公开版本镜像均输出 `image_e2e=pass`。
- Docker Hub 官方源、`docker.1ms.run` 与 `gh.kejilion.pro` 在 154 上对 KPanel、Redis 和 Cloudreve 镜像返回一致摘要；单元测试覆盖官方成功不回源、加速源顺序回退、海外节点、私有 Registry 与摘要镜像不使用代理。

## 发布产物

- 功能提交：`af691784b71423b122a88f450b44b9aba181f9bd`。
- 版本准备提交：`b35fdf5fa9e64d430b08ce4e4b5b693dd9c82f5e`。
- Docker Hub 更新检测修复提交及 `v0.40.0` 标签：`e906f3241891f58cf5bf78e15b05b4cf4352ce5c`。
- 候选分支 CI：[30791869086](https://github.com/kejilion/KPanel/actions/runs/30791869086)。
- 主线 CI：[30791949287](https://github.com/kejilion/KPanel/actions/runs/30791949287)。
- Release：[30792146571](https://github.com/kejilion/KPanel/actions/runs/30792146571)。
- GitHub Release：[v0.40.0](https://github.com/kejilion/KPanel/releases/tag/v0.40.0)，非草稿、非预发布。
- `docker.io/kjlion/kejilion-panel:0.40.0` 与 `latest` 均指向 OCI 摘要 `sha256:20bdabf841635020ff4cc1a611e05d2429898821ce1e727aee0145e07791b4bc`，包含 linux/amd64 与 linux/arm64。
- Release 附件 SHA-256：Agent amd64 `1add09faa16fb112f143f8c80102fc1185dfaa4cfc6edff95a7b074b07fc1ed7`，Agent arm64 `bff11c6d22eb6886b32decb58f17fd9d59e6e658504018ddc93a0fd66f3719c2`。
- 轻量节点 SHA-256：amd64 `f08e4eaad730b26466a7e7e3da8955a59028a964659c083cf023c920e622ae08`，arm64 `b89d69270a69c7fa5b64123da24ba753e2372fb0af08bb852b51abc062cae3e7`。
- 部署包 SHA-256：`04a3caee6c8484216a5cd2c4fb9da4b5016c48ae97bafd3b8b14ce60cc802b3e`。

## 154 线上复核

- 通过既有 `k app kpanel` 更新流程从 `0.39.2` 升级到 `0.40.0`，容器源码修订为 `e906f3241891f58cf5bf78e15b05b4cf4352ce5c`。
- `kejilion-panel` 为 `running/healthy`，8080 端口映射保持不变，本机 HTTP 返回 200。
- `kejilion-agent` 保持 `active/enabled`；Panel 与 Agent 更新后近 10 分钟日志未发现 `panic`、`fatal` 或 `error`。
- Panel 数据目录顶层文件数量更新前后均为 5；未删除面板数据、配置或审计记录。
- 线上 Agent SHA-256 与公开 `v0.40.0` 镜像内 `/release/kejilion-agent` 一致，均为 `85b0a7add8e957b5b4d58de1cd943090a8dc38dfa5877d7528702b08628292a9`。
- 非交互管道调用应用菜单在更新完成后会返回上级菜单等待输入；验收确认无 Docker 子进程后只终止了菜单进程，未终止更新任务或服务。

## 兼容、风险与回滚

- `packaging/kejilion-app/kpanel.conf` 相对 `v0.39.2` 未变化，无需同步修改 `kejilion/apps` 仓库。
- 154 的 `gm.kejilion.eu.org` 在本次独立探测中未通过 curl 的 TLS/HTTP2 校验；由于更新前未记录该域名基线，不能认定由本次镜像更新引起。本机 8080、容器健康、Agent 与数据均已通过，域名证书与反向代理链路应作为独立事项复核。
- 回滚点为 `v0.39.2`，对应镜像摘要 `sha256:d6107799bdd98329b2c9530164753d0d6ca46bd9f5018a7c233eb58e25ac406d`。回滚只替换 Panel/Agent 版本，不删除现有业务数据与配置。
