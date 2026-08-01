# KPanel v0.34.5 发布验收

发布日期：2026-08-01

## 发布范围

- 普通文件区仅允许将文件移入回收站；“彻底删除”和“清空回收站”仅在回收站管理界面提供。
- Panel 与 Agent 同步拒绝普通目录永久删除动作，避免旧前端或直接 API 请求绕过界面约束。
- 修复文件页批量操作样式泄漏到主内容容器、导致桌面端侧栏收起后页面偏移和卡顿的问题。

## 源码与自动化

- 功能提交：`39fc258e495ec5e82244bc42810aa5d2cd5768cf`
- 发布提交：`da0f8b6d7dccabe1c8ad08b9a060728d6447b767`
- 标签：`v0.34.5`
- 候选 CI：[30689657212](https://github.com/kejilion/KPanel/actions/runs/30689657212) — 成功
- 主线 CI：[30689728227](https://github.com/kejilion/KPanel/actions/runs/30689728227) — 成功
- Release：[30689805573](https://github.com/kejilion/KPanel/actions/runs/30689805573) — 成功
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.34.5>

Release 附件已确认包含：

- `kejilion-agent-linux-amd64`
- `kejilion-agent-linux-arm64`
- `kejilion-panel-deploy-0.34.5.tar.gz`
- `SHA256SUMS`
- `LICENSE`
- `THIRD_PARTY_NOTICES.md`

## 验收结果

- Web：27 个测试文件、180 项测试通过；类型检查和生产构建通过。
- Go：文件管理与 Panel 文件动作目标测试通过；Linux 候选 CI 和主线 CI 全绿。
- 真实浏览器分别在桌面展开/收起布局及文件页验证，无横向溢出，普通文件区不再出现彻底删除入口。
- 154 验收机从 Docker Hub 重新拉取公开镜像并执行隔离运行时 E2E，输出 `image_e2e=pass`；未替换其现有 KPanel。

## Docker Hub

- `docker.io/kjlion/kejilion-panel:0.34.5`
- `docker.io/kjlion/kejilion-panel:latest`
- 两个标签的 OCI index 摘要一致：
  `sha256:b7b1179a85938300e0fc24082491d1f608ea143a2197aee620d51260cc12b306`
- linux/amd64：
  `sha256:33e8b2e254b25ab64f4e37948b6a250623d5153814137bbe777dafa9e36074d0`
- linux/arm64：
  `sha256:4f8bd25fa960f7190942804799a434798e303b680652911cf371c3ca783e5454`

清单中的 `unknown/unknown` 项为 provenance/SBOM attestation，不是缺失架构。

## 应用市场兼容性

- 本次未修改 `kejilion.sh`、Compose 或 `packaging/kejilion-app/kpanel.conf`。
- 应用市场继续使用 `latest`，并从 OCI 标签及镜像内 `/release/VERSION` 动态校验版本；无需修改或提交 `kejilion/apps`。
- 本次没有数据库、站点、Docker 业务资源或部署参数迁移。

## 回滚

- 源码：`v0.34.4`（`fd5a24b334b08f64101d603890d9ec9127451e05`）
- 镜像：`docker.io/kjlion/kejilion-panel:0.34.4`
- 回滚不会删除回收站内容；旧版普通文件区会重新显示直接永久删除入口，重新升级到 `v0.34.5` 后恢复本版本规则。
