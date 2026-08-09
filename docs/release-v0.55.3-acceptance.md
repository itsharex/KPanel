# KPanel v0.55.3 发布验收记录

日期：2026-08-09

发布级别：L3

生产目标：`arena-154`（`154.36.153.9:8080`）

## 发布范围

本版本只包含以下已冻结内容：

- `15b8fb4`：修复浅色桌面中脚本终端窗口为外层滚动条预留白色槽位的问题，并补充视觉契约测试；
- `846744a`：统一版本字段并准备 KPanel `0.55.3`。

正式源码、标签和通过 CI 的提交均为：

```text
846744a0183836762d02d0dd28f28f6414568d89
```

本版本没有系统中心、SSH、`kejilion.sh`、数据库、端口、Compose、Agent 权限或应用市场安装契约变更。旧概览工作树和 apps 工作树中的未提交内容未纳入，也未被覆盖。

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
- 重新拉取公开镜像后 `image_e2e=pass`。

远端门禁：

- 候选 CI：<https://github.com/kejilion/KPanel/actions/runs/31303596194>，成功；
- 主线 CI：<https://github.com/kejilion/KPanel/actions/runs/31303738024>，成功；
- Release：<https://github.com/kejilion/KPanel/actions/runs/31303867983>，成功。

## 发布产物

GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.55.3>

附件已核对：

- `kejilion-agent-linux-amd64`
- `kejilion-agent-linux-arm64`
- `kejilion-node-linux-amd64`
- `kejilion-node-linux-arm64`
- `kejilion-panel-deploy-0.55.3.tar.gz`
- `SHA256SUMS`

Docker Hub：`docker.io/kjlion/kejilion-panel`

```text
0.55.3 / latest index:
sha256:a980e3bf5adfa9395f45de507a223d706945c09c541341b5078b69b375eccdae

linux/amd64:
sha256:d96f6de8c492257f3076aacecd13fd06b964e94c63635d325faae34227969371

linux/arm64:
sha256:a90cb0dfe0a3a7a31d9bee666eda5a3654b63142bd2d40f6304b38f897e04515
```

版本镜像与 `latest` 索引摘要一致。`packaging/kejilion-app/kpanel.conf` 未变更，与
`kejilion/apps` 远端 `main` 的配置 blob `d49383667cea8c3b7294bf40ba1e272370a2cb87`
一致，无需应用市场提交。

## 生产部署与验收

部署前生产版本为 `0.55.2`，Panel healthy、0 重启，Agent active、0 重启。通过标准应用市场更新入口部署：

```text
KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update /home/docker/kpanel/bin/kejilion.sh app kpanel
```

停写一致性备份：

```text
/root/kpanel-backups/v0.55.3-preupgrade-20260809T084148Z
```

归档校验：

```text
be135f7ae4afdff2e0cae63e12553fd23f86ecd234180427a3c120691bab5830  kpanel.tar.gz
```

备份完成后先恢复 `0.55.2` 并确认 Panel/Agent 正常，再执行正式更新。上线后验证：

- `/api/v1/health`：`status=ok`、`version=0.55.3`、`protocolVersion=v1alpha1`；
- Panel：running、healthy、0 重启，镜像索引为 `sha256:a980e3bf...5eccdae`；
- Agent：active、0 重启、`NeedDaemonReload=no`，安装二进制与正式镜像内 Agent 的 SHA-256 均为 `289b9acea9be5d9de17be717a8ec4cda30bb1225ec104eb7e1ab3ed744ecaf75`；
- `/home/docker/kpanel/data/panel/panel-state.json` JSON 校验通过，两份 `ai.db` 的 SQLite `PRAGMA integrity_check` 均返回 `ok`；
- 更新稳定后 Panel 与 Agent 日志无 `panic`、`fatal` 或 `error`；
- 公开首页返回 HTTP 200；
- 2 分钟 60 次健康请求全部成功，版本始终为 `0.55.3`。

未使用生产管理员凭据，因此未进入生产账户中的脚本终端窗口；该修复由冻结候选的全量测试、10 项桌面视觉契约、正式镜像 E2E 和公网健康检查覆盖。

## 回滚点

源码回滚：`v0.55.2`（`7a6057634b7b46c7aaeb28ac60602ae503eaa66e`）。

镜像回滚：

```text
docker.io/kjlion/kejilion-panel@sha256:4dd38b6b66f031285dd16d63482de92ead59ce582e64e17d9098668ee4cfd211
```

普通代码回滚可把 Compose 镜像固定到上述摘要后重建 Panel。只有数据或配置同时需要回退时，才停止 Panel、校验备份 SHA-256，并恢复 `kpanel.tar.gz`。

## 遗留风险

- 生产管理员登录后的脚本终端视觉未直接操作，避免读取或传输生产凭据；
- 旧概览工作树仍有未提交样式内容，apps 工作树仍有未提交 `kpanel.conf`，二者均未纳入或覆盖，需由各自原任务独立处理。
