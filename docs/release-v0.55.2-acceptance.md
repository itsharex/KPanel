# KPanel v0.55.2 发布验收记录

日期：2026-08-09

发布级别：L3

生产目标：`arena-154`（`154.36.153.9:8080`）

## 发布范围

本版本只包含以下已冻结内容：

- `dde1632`：统一内置浏览器与其他桌面窗口的视觉层级，提升桌面图标和名称清晰度并移除动态图标白色高光遮罩；
- `27e6d85`：降低浅色桌面的壁纸漂白蒙版、暗角和光晕强度，恢复原始壁纸层次，保持深色主题及原始壁纸文件不变；
- `7a60576`：统一版本字段并准备 KPanel `0.55.2`。

正式源码、标签和通过 CI 的提交均为：

```text
7a6057634b7b46c7aaeb28ac60602ae503eaa66e
```

本版本没有 `kejilion.sh`、数据库、端口、Compose、Agent 权限或应用市场安装契约变更。旧概览工作树和 apps 工作树中的未提交内容未纳入，也未被覆盖。

## 发布前验证

在独立候选工作树与 154 隔离目录执行完整 L3：

- 生态策略、版本一致性和工作流 YAML 静态检查通过；
- `go test ./...`、`go vet ./...` 通过；
- 核心特权包 race 检查通过；
- `govulncheck`、`npm audit`、Trivy 源码和最终镜像扫描通过；
- Linux amd64/arm64 Panel、Agent、Node、kpctl 构建通过；
- 前端 74 个测试文件、517 项测试通过，其中桌面视觉契约 9 项通过；
- TypeScript、国际化检查和生产构建通过；
- 受限容器运行门禁与应用配置生命周期检查通过；
- 重新拉取公开镜像后 `image_e2e=pass`。

远端门禁：

- 候选 CI：<https://github.com/kejilion/KPanel/actions/runs/31302234424>，成功；
- 主线 CI：<https://github.com/kejilion/KPanel/actions/runs/31302352064>，成功；
- Release：<https://github.com/kejilion/KPanel/actions/runs/31302484090>，成功。

## 发布产物

GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.55.2>

附件已核对：

- `kejilion-agent-linux-amd64`
- `kejilion-agent-linux-arm64`
- `kejilion-node-linux-amd64`
- `kejilion-node-linux-arm64`
- `kejilion-panel-deploy-0.55.2.tar.gz`
- `SHA256SUMS`

Docker Hub：`docker.io/kjlion/kejilion-panel`

```text
0.55.2 / latest index:
sha256:4dd38b6b66f031285dd16d63482de92ead59ce582e64e17d9098668ee4cfd211

linux/amd64:
sha256:3b4f31120942d6622d216c19e3109203e189ef507d4fe0cf1f817ad3017758aa

linux/arm64:
sha256:3f1295177c9c473d5e925c79f7dd24f98c40f20d4e5631aba03d03963aa0561e
```

版本镜像与 `latest` 索引摘要一致。`packaging/kejilion-app/kpanel.conf` 未变更，与
`kejilion/apps` 远端 `main` 的配置 blob `d49383667cea8c3b7294bf40ba1e272370a2cb87`
一致，无需应用市场提交。

## 生产部署与验收

部署前生产版本为 `0.55.1`，Panel healthy、0 重启，Agent active、0 重启。通过标准应用市场更新入口部署：

```text
KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update /home/docker/kpanel/bin/kejilion.sh app kpanel
```

停写一致性备份：

```text
/root/kpanel-backups/v0.55.2-preupgrade-20260809T080737Z
```

归档校验：

```text
1b3f8ca3ec89156fb799ce332c6a6dde8684a51ad57a6178090a21f9e31a111e  kpanel.tar.gz
```

备份完成后先恢复 `0.55.1` 并确认 Panel/Agent 正常，再执行正式更新。上线后验证：

- `/api/v1/health`：`status=ok`、`version=0.55.2`、`protocolVersion=v1alpha1`；
- Panel：running、healthy、0 重启，镜像索引为 `sha256:4dd38b6b...cfd211`；
- Agent：active、0 重启、`NeedDaemonReload=no`，安装二进制与正式镜像内 Agent 的 SHA-256 均为 `db7131cb20751c81d3aee580fdafe66c3e4f0123764bf1ff7a94779927b4def6`；
- `/home/docker/kpanel/data/panel/panel-state.json` JSON 校验通过，两份 `ai.db` 的 SQLite `PRAGMA integrity_check` 均返回 `ok`；
- 更新稳定后 Panel 与 Agent 日志无 `panic`、`fatal` 或 `error`；
- 公开首页返回 HTTP 200；
- 2 分钟 60 次健康请求全部成功，版本始终为 `0.55.2`。

未使用生产管理员凭据，因此未进入生产账户中的桌面工作区；桌面功能由冻结候选的全量测试、9 项视觉契约、开发阶段浅色/深色真机截图验收、正式镜像 E2E 和公网健康检查共同覆盖。

## 回滚点

源码回滚：`v0.55.1`（`316598067c63f44c0eafdb55d5ebf29ecf3876fd`）。

镜像回滚：

```text
docker.io/kjlion/kejilion-panel@sha256:2fee895054bbc97419d12d6788b07404245ef1b3d4327fc86b6f7784eb1e4233
```

普通代码回滚可把 Compose 镜像固定到上述摘要后重建 Panel。只有数据或配置同时需要回退时，才停止 Panel、校验备份 SHA-256，并恢复 `kpanel.tar.gz`。

## 遗留风险

- 生产管理员登录后的桌面浅色/深色切换未直接操作，避免读取或传输生产凭据；
- 旧概览工作树仍有未提交样式内容，apps 工作树仍有未提交 `kpanel.conf`，二者均未纳入或覆盖，需由各自原任务独立处理。
