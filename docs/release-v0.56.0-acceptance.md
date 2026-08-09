# KPanel v0.56.0 发布验收记录

日期：2026-08-10

发布级别：L3

生产目标：`arena-154`（`154.36.153.9:8080`）

## 发布范围

本版本只包含以下已冻结内容：

- `d57cd07`：桌面任务栏窗口项增加完整标题提示和右键关闭操作；
- `880f9dd`：压缩页面标题和操作区，重排应用、集群、体检、终端等工作区，补齐手机与窄桌面窗口适配及英文文案；
- `4fd157a`：统一版本字段并准备 KPanel `0.56.0`。

正式源码、标签和通过 CI 的提交均为：

```text
4fd157abf632b217de65cc6a02bfb54d380718c6
```

本版本没有后端 API、数据库、端口、Compose、Agent 权限、`kejilion.sh` 协议或应用市场安装契约变更。
本地预览服务、依赖目录、构建目录、缓存和截图留档均未纳入。

## 发布前验证

在独立候选工作树与 154 隔离目录执行完整 L3：

- 生态策略、版本一致性和 `release-kpanel v1.4` 工作流检查通过；
- `go test ./...`、`go vet ./...` 通过；
- 核心特权包 race 检查通过；
- `govulncheck`、`npm audit`、Trivy 源码和最终镜像扫描通过；
- Linux amd64/arm64 Panel、Agent、Node、kpctl 构建通过；
- 前端 78 个测试文件、534 项测试通过；
- i18n 检查通过，共 1748 条文案和 19 个按页加载语言包；
- TypeScript 和生产构建通过；
- 受限容器运行门禁与应用配置生命周期检查通过；
- 公开镜像端到端检查输出 `image_e2e=pass`。

远端门禁：

- 候选 CI：<https://github.com/kejilion/KPanel/actions/runs/31325594996>，成功；
- 主线 CI：<https://github.com/kejilion/KPanel/actions/runs/31325755667>，成功；
- Release：<https://github.com/kejilion/KPanel/actions/runs/31325928250>，成功。

## 发布产物

GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.56.0>

附件已核对：

- `kejilion-agent-linux-amd64`
- `kejilion-agent-linux-arm64`
- `kejilion-node-linux-amd64`
- `kejilion-node-linux-arm64`
- `kejilion-panel-deploy-0.56.0.tar.gz`
- `LICENSE`
- `SHA256SUMS`
- `THIRD_PARTY_NOTICES.md`

Docker Hub：`docker.io/kjlion/kejilion-panel`

```text
0.56.0 / latest index:
sha256:e3e4fdd3249df8e4a1acf10d0946c8d77a263599362e8f44dbcebf8490e639f5

linux/amd64:
sha256:d68ec4cf3c74077c5ca3b77d5fc3086c2580d3b8198bf3db04943a05626ac467

linux/arm64:
sha256:7794a477e23296bdb059674acb6cb1538d299f43bbd22b1d231950b639c1f230
```

版本镜像与 `latest` 索引摘要一致。`packaging/kejilion-app/kpanel.conf` 未变更，与
`kejilion/apps` 远端 `main` 的配置 blob `d49383667cea8c3b7294bf40ba1e272370a2cb87`
一致，无需应用市场提交。

## 生产部署与验收

部署前生产版本为 `0.55.5`，Panel healthy、0 重启，Agent active、0 重启。通过标准应用市场更新入口部署：

```text
KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update /home/docker/kpanel/bin/kejilion.sh app kpanel
```

停写一致性备份：

```text
/root/kpanel-backups/v0.56.0-preupgrade-20260809T172157Z
```

归档校验：

```text
4f416a9ea334da8d0dd1295f8ad9289a99ee096429e4cd2251092c11ee814134  kpanel.tar.gz
```

备份完成后先恢复 `0.55.5` 并确认 Panel/Agent 正常，再执行正式更新。上线后验证：

- `/api/v1/health`：`status=ok`、`version=0.56.0`、`protocolVersion=v1alpha1`；
- Panel：running、healthy、0 重启，镜像索引为 `sha256:e3e4fdd3...639f5`；
- Agent：active、0 重启、`NeedDaemonReload=no`，安装二进制与正式镜像内 Agent 的 SHA-256 均为 `e8997f4a112229c826e5fcc523fa7a8d4ac9a73b6f0b4c2a4bea2345906b57db`；
- `panel-state.json` 和集群状态 JSON 均可解析，两份 `ai.db` 的 SQLite `PRAGMA integrity_check` 均返回 `ok`；
- 更新稳定后 Panel 和 Agent 日志无 `panic`、`fatal` 或 `error`；
- 正式访问目标的首页、登录页和健康接口均返回 HTTP 200；
- 2 分钟 60 次本地及公开健康采样全部成功，版本始终为 `0.56.0`；
- 部署后再次执行前端测试，78 个测试文件、534 项测试全部通过。

## 回滚点

源码回滚：`v0.55.5`（`87c465371854623d4a33b06e658cb1d6d9c76801`）。

镜像回滚：

```text
docker.io/kjlion/kejilion-panel@sha256:a5a62898dac421ce6829394769d026c520f0d65e8de7c8503d52f86f4c897cc1
```

本版本没有数据或配置迁移，普通回滚只需把 Compose 镜像固定到上述摘要后重建 Panel，并从该镜像恢复 Agent。
只有数据或配置同时需要回退时，才停止 Panel、校验备份 SHA-256，并恢复 `kpanel.tar.gz`。

## 遗留风险

- 未使用生产管理员凭据进入桌面工作区直接点击验收，避免读取或传输生产凭据；UI 变更由多语言审查、经典/桌面深浅主题截图、390px 手机验收、布局契约、全量测试和正式镜像 E2E 覆盖；
- 集群与活动记录在开发验收环境中缺少真实业务数据，仅覆盖空态和错误态；生产健康、数据完整性与服务稳定性均已验证，但真实数据密集布局仍需后续日常使用观察。
