# KPanel v0.55.5 发布验收记录

日期：2026-08-09

发布级别：L3

生产目标：`arena-154`（`154.36.153.9:8080`）

## 发布范围

本版本只包含以下已冻结内容：

- `7887ff4`：统一浅色桌面模式选中图标标签语义，使用品牌绿色底色、边框和阴影，并保持深色高对比文字；
- `87c4653`：统一版本字段并准备 KPanel `0.55.5`。

正式源码、标签和通过 CI 的提交均为：

```text
87c465371854623d4a33b06e658cb1d6d9c76801
```

间距试验提交的最终树与 `77b4620` 完全一致，因此未纳入本次候选；未跟踪的
`web/public/desktop-icon-preview.html` 仅为预览文件，也未纳入。本版本没有系统中心、SSH、
`kejilion.sh`、数据库、端口、Compose、Agent 权限或应用市场安装契约变更。

## 发布前验证

在独立候选工作树与 154 隔离目录执行完整 L3：

- 生态策略、版本一致性和工作流 YAML 静态检查通过；
- `go test ./...`、`go vet ./...` 通过；
- 核心特权包 race 检查通过；
- `govulncheck`、`npm audit`、Trivy 源码和最终镜像扫描通过；
- Linux amd64/arm64 Panel、Agent、Node、kpctl 构建通过；
- 前端 74 个测试文件、518 项测试通过，其中桌面视觉契约 10 项通过；
- TypeScript、国际化检查和生产构建通过；
- 受限容器运行门禁与应用配置生命周期检查通过；
- 公开镜像版本、修订标签、Panel 和 Agent 二进制提取检查通过。

远端门禁：

- 候选 CI：<https://github.com/kejilion/KPanel/actions/runs/31307080506>，成功；
- 主线 CI：<https://github.com/kejilion/KPanel/actions/runs/31307208602>，成功；
- Release：<https://github.com/kejilion/KPanel/actions/runs/31307363857>，成功。

## 发布产物

GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.55.5>

附件已核对：

- `kejilion-agent-linux-amd64`
- `kejilion-agent-linux-arm64`
- `kejilion-node-linux-amd64`
- `kejilion-node-linux-arm64`
- `kejilion-panel-deploy-0.55.5.tar.gz`
- `LICENSE`
- `SHA256SUMS`
- `THIRD_PARTY_NOTICES.md`

Docker Hub：`docker.io/kjlion/kejilion-panel`

```text
0.55.5 / latest index:
sha256:a5a62898dac421ce6829394769d026c520f0d65e8de7c8503d52f86f4c897cc1

linux/amd64:
sha256:f059855b2a2de7d75f0d60dc6c2d1e529a07202b5f6446b76cc1f44dc5e93f1b

linux/arm64:
sha256:f2df1f9264fbc239dda1844e90c7a0a96a4803d1599a9628e3cf028f4ea8aeb8
```

版本镜像与 `latest` 索引摘要一致。`packaging/kejilion-app/kpanel.conf` 未变更，与
`kejilion/apps` 远端 `main` 的配置 blob `d49383667cea8c3b7294bf40ba1e272370a2cb87`
一致，无需应用市场提交。

## 生产部署与验收

部署前生产版本为 `0.55.4`，Panel healthy、0 重启，Agent active、0 重启。通过标准应用市场更新入口部署：

```text
KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update /home/docker/kpanel/bin/kejilion.sh app kpanel
```

停写一致性备份：

```text
/root/kpanel-backups/v0.55.5-preupgrade-20260809T101111Z
```

归档校验：

```text
cc1f2ff855a3ab52c39010d60c8b95788f22d90ec95ebffdcc699d83c544283e  kpanel.tar.gz
```

备份完成后先恢复 `0.55.4` 并确认 Panel/Agent 正常，再执行正式更新。上线后验证：

- `/api/v1/health`：`status=ok`、`version=0.55.5`、`protocolVersion=v1alpha1`；
- Panel：running、healthy、0 重启，镜像索引为 `sha256:a5a62898...897cc1`；
- Agent：active、0 重启、`NeedDaemonReload=no`，安装二进制与正式镜像内 Agent 的 SHA-256 均为 `a2cbb4c7c970bec21c9dbe690c886e37b7dfd008bfa01638dc11cc29a2815cb3`；
- Panel 二进制 SHA-256 为 `cb1bb8a7d86385e9651c626fb49b1a2a2eb2d1066a2ceb57267eae26a8668992`；
- `panel-state.json` 和集群状态 JSON 均可解析，两份 `ai.db` 的 SQLite `PRAGMA integrity_check` 均返回 `ok`；
- 更新稳定后 Panel 和 Agent 日志无 `panic`、`fatal` 或 `error`；
- 正式访问目标的首页、登录页和健康接口均返回 HTTP 200；
- 2 分钟 60 次本地及公开健康采样全部成功，版本始终为 `0.55.5`；
- 部署后再次执行前端测试，74 个测试文件、518 项测试全部通过。

## 回滚点

源码回滚：`v0.55.4`（`9e5b7a5ee9e4a8648857dd0433064efd94cd46b8`）。

镜像回滚：

```text
docker.io/kjlion/kejilion-panel@sha256:629d3c3cf66951beec1bb36cdf786de0f78a63c167cb5ade849d8a4a095f1ebf
```

本版本没有数据或配置迁移，普通回滚只需把 Compose 镜像固定到上述摘要后重建 Panel，并从该镜像恢复 Agent。
只有数据或配置同时需要回退时，才停止 Panel、校验备份 SHA-256，并恢复 `kpanel.tar.gz`。

## 遗留风险

- 未使用生产管理员凭据进入桌面工作区直接点击验收，避免读取或传输生产凭据；视觉变更由冻结候选的全量测试、10 项桌面视觉契约、正式镜像提取检查和公网健康检查覆盖；
- 旧概览工作树仍有未提交样式内容，apps 工作树仍有未提交 `kpanel.conf`，功能分支仍有未跟踪预览 HTML；这些内容均未纳入或覆盖，需由各自任务独立处理。
