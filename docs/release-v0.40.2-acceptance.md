# KPanel v0.40.2 发布验收

发布日期：2026-08-03

## 发布范围

- 修复应用市场安装或更新 KPanel 后，文件管理在 `/root`、`/var`、`/opt`、`/srv`、`/mnt` 等已开放宿主机目录执行创建、上传、粘贴、保存和删除时出现 `read-only file system` 的问题。
- systemd 写视图与文件 API 的 `/` 管理根保持一致；实际文件操作仍经过固定类型、路径校验、符号链接拒绝、敏感目录隔离和审计，不开放任意 Shell 写入接口。
- 本次不包含系统中心的页面、API、协议或文档变更。

## 自动化与运行时验收

- 完整 L3 发布验证通过：Go 全量测试、核心包 race、`go vet`、`govulncheck`、Web 233 项测试、类型检查、生产构建、i18n、amd64/arm64 构建均通过。
- `npm audit`、Trivy 源码与最终镜像扫描通过，未发现阻断发布的已调用漏洞、高危镜像漏洞、密钥或配置问题。
- 应用安装、更新、回滚和卸载生命周期测试输出 `app_conf_lifecycle=pass`。
- Docker Hub 公共镜像按不可变摘要重新拉取，隔离运行时 E2E 输出 `image_e2e=pass`；测试容器、网络和临时数据均已清理。
- 当前自动化环境没有可用的 154 验收机 SSH 凭据，因此未将本地隔离验证记录为 154 真机验收。

## 发布产物

- 功能提交：`1ee79d8`。
- 版本准备与 `v0.40.2` 标签提交：`a408ce491b135b4f25f3f931dceba434eccaf466`。
- 候选分支 CI：[30810011852](https://github.com/kejilion/KPanel/actions/runs/30810011852)。
- 主线 CI：[30810159937](https://github.com/kejilion/KPanel/actions/runs/30810159937)。
- Release 工作流：[30810329320](https://github.com/kejilion/KPanel/actions/runs/30810329320)。
- GitHub Release：[v0.40.2](https://github.com/kejilion/KPanel/releases/tag/v0.40.2)，非草稿、非预发布。
- `docker.io/kjlion/kejilion-panel:0.40.2` 与 `latest` 均指向 OCI 摘要 `sha256:37069b9f67ff34af20a8958b9374c1a54fa3b8d192c5558376a5050f076982bb`，包含 linux/amd64 与 linux/arm64。
- Agent SHA-256：amd64 `a12c74813a42ae93b587a0ea570046815a1e648146114b1027cab5818e5ed2ea`，arm64 `5c6e22cd9cdde86ad50ebc254249b42c8d698e16c64bd075f8f4ab74fa0b4f55`。
- 轻量节点 SHA-256：amd64 `2713459a660a605c34012ea62e3575a8f32ebf629881e668fae79dfa2ee2a572`，arm64 `a4840489e058b8d0507aba43f97682cce9dbf5a80fbd9cf50f31d3d15ed5a386`。
- 部署包 SHA-256：`063e465755c9c0c86046eb6e550d8b2e4a46560ffda1271febc2e528b13b97f2`。

## 应用市场契约与回滚

- `packaging/kejilion-app/kpanel.conf` 已同步至 `kejilion/apps` 主线提交 `2e40c0e`，规范化内容与发布产物一致，并通过 Shell 语法检查。
- 回滚点为 `v0.40.1`。回滚只替换 Panel/Agent 版本，不删除用户文件、业务数据、配置或审计记录。
