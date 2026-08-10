# KPanel v0.57.0 发布验收记录

日期：2026-08-10

发布级别：L3

生产目标：`arena-154`（`154.36.153.9:8080`）

## 发布范围

本版本只包含以下已冻结内容：

- `92ba14e`：桌面窗口支持顶部最大化、左右半屏贴靠、目标预览、拖离恢复，以及有界的按应用窗口尺寸记忆；
- `761c14c`：补齐手机和平板触摸/手写笔单击打开、长按菜单、多指误触防护、窄屏单窗口工作区和任务栏切换；
- `c8d3ac8`：统一版本字段并准备 KPanel `0.57.0`。

正式源码、标签和通过 CI 的提交均为：

```text
c8d3ac8496d07496680b19fa14792a51b3efba43
```

本版本没有后端 API、数据库、端口、Compose、Agent 权限、`kejilion.sh` 协议或应用市场安装契约变更。系统/网络资源工具、诊断页备选布局、桌面浏览器归档实验和其他脏工作区均未纳入。

## 发布前验证

在独立候选工作树与 154 隔离目录执行完整 L3：

- 生态策略、版本一致性和 `release-kpanel v1.4` 工作流检查通过；
- `go test ./...`、`go vet ./...` 通过；
- 核心特权包 race 检查通过；
- `govulncheck`、`npm audit`、Trivy 源码和最终镜像扫描通过；
- Linux amd64/arm64 Panel、Agent、Node、kpctl 构建通过；
- 前端 79 个测试文件、560 项测试通过；
- i18n 检查通过，共 1748 条文案和 19 个按页加载语言包；
- TypeScript 和生产构建通过；
- 受限容器运行门禁与应用配置生命周期检查通过；
- 公开镜像端到端检查输出 `image_e2e=pass`。

远端门禁：

- 候选 CI：<https://github.com/kejilion/KPanel/actions/runs/31358990597>，成功；
- 主线 CI：<https://github.com/kejilion/KPanel/actions/runs/31359160664>，成功；
- Release：<https://github.com/kejilion/KPanel/actions/runs/31359343540>，成功。

## 发布产物

GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.57.0>

附件已核对：

- `kejilion-agent-linux-amd64`
- `kejilion-agent-linux-arm64`
- `kejilion-node-linux-amd64`
- `kejilion-node-linux-arm64`
- `kejilion-panel-deploy-0.57.0.tar.gz`
- `LICENSE`
- `SHA256SUMS`
- `THIRD_PARTY_NOTICES.md`

Docker Hub：`docker.io/kjlion/kejilion-panel`

```text
0.57.0 / latest index:
sha256:1668b1e14dc441f6304c8698a438602a4f6e13976bf5bbc78b05432770d6a8cd

linux/amd64:
sha256:19bed836350b879d8143a2c661b74b300ad219c035dbe7d61670f8905cf8f1be

linux/arm64:
sha256:7124f1d8c454d1455dd306ea79bc99ff7c0e4a752ca00c68886166d985e7d2e5
```

版本镜像与 `latest` 索引摘要一致。`packaging/kejilion-app/kpanel.conf` 未变更，与 `kejilion/apps` 远端 `main` 的配置 blob `d49383667cea8c3b7294bf40ba1e272370a2cb87` 一致，无需应用市场提交。

## 生产部署与验收

部署前生产版本为 `0.56.0`，Panel healthy、0 重启，Agent active、0 重启。通过标准应用市场更新入口部署：

```text
KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update /home/docker/kpanel/bin/kejilion.sh app kpanel
```

停写一致性备份：

```text
/root/kpanel-backups/v0.57.0-preupgrade-20260810T054833Z
```

归档校验：

```text
7d7bd08d6c459e5289eb5bad9bfabc8070499f20830422f6c644fbbd7fe53222  kpanel.tar.gz
```

备份完成后先恢复 `0.56.0` 并确认 Panel/Agent 正常，再执行正式更新。上线后验证：

- `/api/v1/health`：`status=ok`、`version=0.57.0`、`protocolVersion=v1alpha1`；
- Panel：running、healthy、0 重启，镜像索引为 `sha256:1668b1e1...a8cd`；
- Agent：active、0 重启、`NeedDaemonReload=no`，安装二进制与正式镜像内 Agent 的 SHA-256 均为 `c9f09ff658f364dde1b074b95238143fe6fa7e402aa129e6e12c1766f30ac6bd`；
- `panel-state.json` 和三份集群状态 JSON 均可解析，两份 `ai.db` 的 SQLite `PRAGMA integrity_check` 均返回 `ok`；
- 更新稳定后 Panel 和 Agent 日志无 `panic`、`fatal` 或 `error`；
- 正式访问目标的首页、登录页和健康接口在本地及公网均返回 HTTP 200；
- 2 分钟 60 次本地及公开健康采样全部成功，版本始终为 `0.57.0`；
- 部署后再次执行前端测试，79 个测试文件、560 项测试全部通过。

## 回滚点

源码回滚：`v0.56.0`（`4fd157abf632b217de65cc6a02bfb54d380718c6`）。

镜像回滚：

```text
docker.io/kjlion/kejilion-panel@sha256:e3e4fdd3249df8e4a1acf10d0946c8d77a263599362e8f44dbcebf8490e639f5
```

本版本没有数据或配置迁移，普通回滚只需把 Compose 镜像固定到上述摘要后重建 Panel，并从该镜像恢复 Agent。只有数据或配置同时需要回退时，才停止 Panel、校验备份 SHA-256，并恢复 `kpanel.tar.gz`。

## 遗留风险

- 未使用生产管理员凭据进入桌面工作区直接点击验收，避免读取或传输生产凭据；桌面鼠标、触摸/手写笔、长按、多指取消、窗口贴靠和尺寸记忆由单元测试、布局契约、完整 L3 和正式镜像 E2E 覆盖；
- 本轮没有在独立手机真机上重新执行手势验收，生产健康、数据完整性和服务稳定性均已验证，移动端交互仍需后续日常使用观察；
- 测试输出存在 Vue 测试桩的非失败告警，不影响 79/560 全量通过，生产 Panel/Agent 日志未出现对应错误。
