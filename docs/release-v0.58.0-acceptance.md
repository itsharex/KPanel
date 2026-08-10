# KPanel v0.58.0 发布验收记录

日期：2026-08-10

发布级别：L3

生产目标：`arena-154`（`154.36.153.9:8080`）

## 发布范围

本版本只包含以下已冻结内容：

- `5a6cbad`、`6ac488e`：统一 AI 输入区的权限、思考和模型选择菜单，补充运行中追加/停止操作及手机定位加固；
- `4783f12`：网站管理页通过现有受控终端会话启动固定 `k web` 原生菜单；
- `23dac4a`：修复手机全屏终端输入区溢出；
- `32c522c`：修复手机顶部栏桌面入口图标居中；
- `bdd969c`：修复桌面体检区块拉伸及终端自动聚焦引发的外层滚动；
- `da1c7be`：统一版本字段并准备 KPanel `0.58.0`。

正式源码、标签和通过 CI 的提交均为：

```text
da1c7be9d21891504a19f4e58dac5f078e9ce955
```

本版本没有数据库、端口、Compose、Agent 权限、`kejilion.sh` 协议或应用市场安装契约变更。浏览器 fallback 的修改与回滚净差异为零；概览系统/网络资源工具、诊断备选布局及其他旧/脏工作区均未纳入。

## 发布前验证

在独立候选工作树与 154 隔离目录执行完整 L3：

- 生态策略、版本一致性和 `release-kpanel v1.4` 工作流检查通过；
- `go test ./...`、`go vet ./...` 通过；
- 核心特权包 race 检查通过；
- `govulncheck`、`npm audit`、Trivy 源码和最终镜像扫描通过；
- Linux amd64/arm64 Panel、Agent、Node、kpctl 构建通过；
- 前端 80 个测试文件、571 项测试通过；
- i18n 检查通过，共 1765 条文案和 19 个按页加载语言包；
- TypeScript 和生产构建通过；
- 受限容器运行门禁与应用配置生命周期检查通过；
- 公开镜像端到端检查输出 `image_e2e=pass`。

远端门禁：

- 候选 CI：<https://github.com/kejilion/KPanel/actions/runs/31375167344>，成功；
- 主线 CI：<https://github.com/kejilion/KPanel/actions/runs/31375434577>，成功；
- Release：<https://github.com/kejilion/KPanel/actions/runs/31375710420>，成功。

## 发布产物

GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.58.0>

Release 正文已核对版本更新、升级注意事项、升级方式、完整性、测试和回滚说明。附件已核对：

- `kejilion-agent-linux-amd64`
- `kejilion-agent-linux-arm64`
- `kejilion-node-linux-amd64`
- `kejilion-node-linux-arm64`
- `kejilion-panel-deploy-0.58.0.tar.gz`
- `LICENSE`
- `SHA256SUMS`
- `THIRD_PARTY_NOTICES.md`

Docker Hub：`docker.io/kjlion/kejilion-panel`

```text
0.58.0 / latest index:
sha256:3c28f8930da2fa1b96b735394a333bcca4d3f3d6680800001c2d537fc2cdf8b6

linux/amd64:
sha256:bb44c789b0e1ba7ff24ba55e07588007deaa02ee590931fdb87724615c3ced5d

linux/arm64:
sha256:8c0b793d8f6b636cdffd28506811dbcd75eb2b090a6714b96a2914e6148f72f6
```

版本镜像与 `latest` 索引摘要一致。`packaging/kejilion-app/kpanel.conf` 未变更，与 `kejilion/apps` 远端 `main` 的配置 blob `d49383667cea8c3b7294bf40ba1e272370a2cb87` 一致，无需应用市场提交。

## 生产部署与验收

部署前生产版本为 `0.57.0`，Panel healthy、0 重启，Agent active、0 重启。通过标准应用市场更新入口部署：

```text
KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update /home/docker/kpanel/bin/kejilion.sh app kpanel
```

停写一致性备份：

```text
/root/kpanel-backups/v0.58.0-preupgrade-20260810T094716Z
```

归档校验：

```text
45e96e8aeb52cadc0ecc56e910659ca71c0d6fbe618ecb2b957152380c2d0ae5  kpanel.tar.gz
```

备份完成后先恢复 `0.57.0` 并确认 Panel/Agent 正常，再执行正式更新。上线后验证：

- `/api/v1/health`：`status=ok`、`version=0.58.0`、`protocolVersion=v1alpha1`；
- Panel：running、healthy、0 重启，镜像索引为 `sha256:3c28f893...df8b6`；
- Agent：active、0 重启、`NeedDaemonReload=no`，安装二进制与正式镜像内 Agent 的 SHA-256 均为 `56d22e02dbc34b7d4d7cc2de359e9d97598f87da1a2d1f7acc4671995767ed65`；
- `panel-state.json` 和三份集群状态 JSON 均可解析，两份 `ai.db` 的 SQLite `PRAGMA integrity_check` 均返回 `ok`；
- 本机存在 `/usr/local/bin/k`，网站终端入口使用固定 `k web` 输入；
- 更新稳定后 Panel 和 Agent 日志无 `panic`、`fatal` 或 `error`；
- 正式访问目标的首页、登录页和健康接口在本地及公网均返回 HTTP 200；
- 2 分钟 60 次本地及公开健康采样全部成功，版本始终为 `0.58.0`；
- 部署后再次执行前端测试，80 个测试文件、571 项测试全部通过。

## 回滚点

源码回滚：`v0.57.0`（`c8d3ac8496d07496680b19fa14792a51b3efba43`）。

镜像回滚：

```text
docker.io/kjlion/kejilion-panel@sha256:1668b1e14dc441f6304c8698a438602a4f6e13976bf5bbc78b05432770d6a8cd
```

本版本没有数据或配置迁移，普通回滚只需把 Compose 镜像固定到上述摘要后重建 Panel，并从该镜像恢复 Agent。只有数据或配置同时需要回退时，才停止 Panel、校验备份 SHA-256，并恢复 `kpanel.tar.gz`。

## 遗留风险

- 未使用生产管理员凭据进入 AI、网站和终端页面直接点击验收，避免读取或传输生产凭据；交互由组件测试、布局契约、完整 L3、公开镜像 E2E 和生产服务验收覆盖；
- 未在无人值守验收中实际进入 `k web` 交互菜单，避免误操作生产网站；已验证 `/usr/local/bin/k` 存在，前端测试确认只向新建本机会话发送固定 `k web\r` 并正确处理关闭竞态；
- 测试输出存在既有 Vue 测试桩生命周期告警，以及一个缺少必填 `approvalMode` 的旧测试 fixture 告警；类型契约和后端 JSON 均要求并返回该字段，80/571 全量测试通过，生产日志无对应错误。
