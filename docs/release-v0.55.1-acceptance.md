# KPanel v0.55.1 发布验收记录

日期：2026-08-09

发布级别：L3

生产目标：`arena-154`（`154.36.153.9:8080`）

## 发布范围

本版本包含以下已冻结内容：

- `542144b`：普通桌面窗口接入浏览器原生前进/后退并移除重复返回箭头；
- `b49dd9c`：隐藏桌面上的 KPanel 自身入口，保留应用市场管理能力；
- `3165980`：统一版本字段并准备 KPanel `0.55.1`。

正式源码、标签和通过 CI 的提交均为：

```text
316598067c63f44c0eafdb55d5ebf29ecf3876fd
```

旧概览工作树的未提交样式改动未纳入本版本，也未被覆盖。

## 发布前验证

在独立候选工作树与 154 隔离目录执行完整 L3：

- 生态策略、版本一致性和工作流 YAML 静态检查通过；
- `go test ./...`、`go vet ./...` 通过；
- 核心特权包 race 检查通过；
- `govulncheck`、`npm audit`、Trivy 源码和最终镜像扫描通过；
- Linux amd64/arm64 Panel、Agent、Node、kpctl 构建通过；
- 前端 74 个测试文件、512 项测试通过；
- TypeScript、国际化检查和生产构建通过；
- 受限容器运行门禁与 `app_conf_lifecycle=pass`；
- 重新拉取公开镜像后 `image_e2e=pass`。

远端门禁：

- 候选 CI：<https://github.com/kejilion/KPanel/actions/runs/31300547477>，成功；
- 主线 CI：<https://github.com/kejilion/KPanel/actions/runs/31300659313>，成功；
- Release：<https://github.com/kejilion/KPanel/actions/runs/31300780377>，成功。

## 发布产物

GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.55.1>

附件已核对：

- `kejilion-agent-linux-amd64`
- `kejilion-agent-linux-arm64`
- `kejilion-node-linux-amd64`
- `kejilion-node-linux-arm64`
- `kejilion-panel-deploy-0.55.1.tar.gz`
- `SHA256SUMS`

Docker Hub：`docker.io/kjlion/kejilion-panel`

```text
0.55.1 / latest index:
sha256:2fee895054bbc97419d12d6788b07404245ef1b3d4327fc86b6f7784eb1e4233

linux/amd64:
sha256:ce0b02d4b8d763cf4c92f2cd2f6a8c816c6cafb67a9c7db288484b12cebaa2d1

linux/arm64:
sha256:c15d00f80c092b2d2bd8fad7a908177d1419e40022dd771581d88beb4cfec20f
```

版本镜像与 `latest` 索引摘要一致。`packaging/kejilion-app/kpanel.conf` 未变更，与
`kejilion/apps@1f2740666a55ccbb3749ce83168e073c1ea08431` 的配置 blob
`d49383667cea8c3b7294bf40ba1e272370a2cb87` 一致，无需应用市场提交。

## 生产部署与验收

部署前生产版本为 `0.55.0`，Panel healthy、0 重启，Agent active、0 重启。通过标准应用市场更新入口部署：

```text
KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update /home/docker/kpanel/bin/kejilion.sh app kpanel
```

停写一致性备份：

```text
/root/kpanel-backups/v0.55.1-preupgrade-20260809T072651Z
```

归档校验：

```text
b6ffee7adae69db39e2b237a4dd12d954ad346e43c9fd20ecdf66b55e5d96881  kpanel-home.tar.gz
```

上线后验证：

- `/api/v1/health`：`status=ok`、`version=0.55.1`、`protocolVersion=v1alpha1`；
- Panel：running、healthy、0 重启，镜像索引为 `sha256:2fee8950...1e4233`；
- Agent：active、0 重启、`NeedDaemonReload=no`，安装二进制与正式镜像内 Agent 哈希一致；
- `panel-state.json` JSON 校验通过，`ai.db` SQLite `PRAGMA integrity_check` 返回 `ok`；
- 更新稳定后 Panel 与 Agent 日志无 `panic`、`fatal` 或 `error`；
- 公开首页返回 HTTP 200；
- 2 分钟 60 次健康请求全部成功，版本始终为 `0.55.1`；
- 真实浏览器进入 `/login?redirect=/overview`，标题为“登录 · KPanel”，用户名、密码、安全登录与“忘记密码？”入口均正常渲染。

未使用生产管理员凭据，因此未进入生产账户中的桌面工作区；桌面功能由冻结候选的全量测试、原开发任务视觉验收、正式镜像 E2E 和公开登录页加载共同覆盖。

## 回滚点

源码回滚：`v0.55.0`（`7986cedb63dad50b13f10103e7bbdc6f45a82527`）。

镜像回滚：

```text
docker.io/kjlion/kejilion-panel@sha256:849e157ecf39ec7cba6da8c854c51cec5c6618e8bd86de0457712e6e8f0b8ba7
```

普通代码回滚可把 Compose 镜像固定到上述摘要后重建 Panel。只有数据或配置同时需要回退时，才停止 Panel、校验备份 SHA-256，并恢复 `kpanel-home.tar.gz`。

## 遗留风险

- 生产管理员登录后的桌面交互未直接操作，避免读取或传输生产凭据；
- 旧概览工作树仍保留 2 行未提交改动，未纳入、未覆盖，需由其原任务单独处理。
