# KPanel v0.34.4 发布验收

发布日期：2026-08-01

## 发布范围

- 修复应用市场部署中回收站写入只读 `/var/lib/kejilion-panel`、导致文件无法删除的问题；回收站现在跟随 Agent 实际持久化状态目录。
- 文件管理增加可视化回收站、原路径恢复、选择性彻底删除、清空回收站，以及普通文件列表中的彻底删除入口。
- 修复桌面端左侧栏收起后主内容仍按错误宽度布局、页面异常偏移和裁切的问题。

## 源码与自动化

- 功能提交：`9ee1404b739ed0d839886220764043a60948f600`
- 发布提交：`fd5a24b334b08f64101d603890d9ec9127451e05`
- 标签：`v0.34.4`
- 功能候选 CI：[30687509859](https://github.com/kejilion/KPanel/actions/runs/30687509859) — 成功
- 版本候选 CI：[30687606746](https://github.com/kejilion/KPanel/actions/runs/30687606746) — 成功
- 主线 CI：[30687686260](https://github.com/kejilion/KPanel/actions/runs/30687686260) — 成功
- Release：[30687765148](https://github.com/kejilion/KPanel/actions/runs/30687765148) — 成功
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.34.4>

Release 附件已确认包含：

- `kejilion-agent-linux-amd64`
- `kejilion-agent-linux-arm64`
- `kejilion-panel-deploy-0.34.4.tar.gz`
- `SHA256SUMS`
- `LICENSE`
- `THIRD_PARTY_NOTICES.md`

## L3 验收

- Web：27 个测试文件、179 项测试通过；类型检查和生产构建通过。
- Linux Go 全包测试和 Go Vet 通过；文件管理与 Agent 竞态检测通过。
- 部署安全测试和 `kejilion.sh` 应用生命周期测试通过。
- `govulncheck` 与 npm 高危依赖审计通过；npm 报告 0 个漏洞。
- 原生镜像运行时契约、非 root 用户、健康检查、固定脚本摘要和许可证核对通过。
- linux/amd64、linux/arm64 Agent 与多架构镜像构建通过。

新增文件生命周期测试覆盖：

- 移入回收站、列出、恢复、选择性彻底删除和清空。
- stale `resourceVersion` 冲突、恢复不覆盖同名文件、受保护目录拒绝删除。
- Agent 状态目录派生、Panel 鉴权代理和前端恢复请求。

## Docker Hub

- `docker.io/kjlion/kejilion-panel:0.34.4`
- `docker.io/kjlion/kejilion-panel:latest`
- 两个标签的 OCI index 摘要相同：
  `sha256:41ea191312b6edaac198092c8c547c2a3d362f68c689265f9d4c8cf03ca7e7e5`
- linux/amd64：
  `sha256:4e0251762d9f971928edbf378d63876873d0e106b058d1d7b7e13eec4f84a525`
- linux/arm64：
  `sha256:e135dd27622b058c8b4134e0ca52981ebc929fab1859d5238f3268dcb919c88d`

`unknown/unknown` 条目为各平台对应的 provenance/SBOM attestation，不是缺失架构。

## 应用市场与兼容性

- 本次未修改 `kejilion.sh`、部署 Compose 或 `packaging/kejilion-app/kpanel.conf`；应用市场继续使用 `latest`，不需要修改 `kejilion/apps`。
- 回收站内部新增原路径元数据；旧版无元数据项目仍可显示并彻底删除，但不能恢复。
- 回收站和 KPanel 状态目录继续对普通文件 API 隔离，彻底删除使用固定动作、资源版本校验、路径保护和审计。

## 回滚

- 源码：`v0.34.3`（`333d0a26fdfc0165c08fc79a1793e7242fbe6545`）
- 镜像：`docker.io/kjlion/kejilion-panel:0.34.3`
- 回滚不会删除已经回收的文件或回收站元数据；旧版界面不提供恢复入口，重新升级到 `v0.34.4` 后可继续管理。
- 本次没有 Panel 数据库、站点、Docker 业务资源或 `kejilion.sh` 协议迁移。
