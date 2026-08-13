# KPanel v0.60.0 发布验收记录

日期：2026-08-10

发布级别：L3

生产目标：`arena-154`

## 发布范围

本版本只包含桌面监控与桌面窗口适配改进：

- `50d077f`：桌面服务器监控增加累计接收、累计发送流量，并补中英文文案与组件测试；
- `eba7552`、`9f464a3`：统一监控指标间距并平衡紧凑行布局；
- `a4e26ec`：终端、应用交互终端和体检窗口适配桌面窗口高度，聚焦时使用 `preventScroll`，保留手机端滚动边界；
- `6d6028a`：统一版本字段并准备 KPanel `0.60.0`。

正式源码、标签和通过 CI 的提交均为：

```text
6d6028a90c2912bc4c7c3f5d53b44e95687fb65b
```

配套 `kejilion/sh` 主线提交保持为：

```text
3972217d4d4a51d473b7375f5b850870e066be92
```

原始 `kejilion.sh` SHA-256 保持为：

```text
90da9f13da056b133079fa6fcb673004f377a6b2c0e69e01015040f234e51921
```

本版本没有 API、数据库、端口、Compose、Agent 权限、脚本协议或应用市场安装契约迁移。

## 未上线内容审计

从 `main@4360b91` 审计全部本地工作树和近期分支后，真正未上线且仍有效的净内容只有上述 4 个桌面改进提交；它们已逐提交迁移到最新主线并冻结为 `v0.60.0` 候选。

未纳入范围：

- 旧体检救援工作树的未提交内容已被后续正式设计覆盖；
- 旧概览服务状态工作树的差异在主线已有等价实现；
- 功能分支中的未跟踪预览 HTML 仅用于视觉验收，不属于产品文件；
- 登录/初始化、集群系统图标、九线路悬停、SSH 端口、桌面模式及此前各批 UI 改进均已在历史版本发布，不重复上线。

上述旧工作树均未被覆盖或提交。

## 发布前验证

在独立候选工作树、本地浏览器和 154 完整 Linux L3 环境执行验证：

- 变更边界共 15 个文件，版本字段一致，`git diff --check` 通过；
- `go test ./...`、`go vet ./...`、核心特权包 race 检查通过；
- `govulncheck`、`npm audit`、Trivy 源码和最终镜像扫描通过，Node 依赖审计为 0 漏洞；
- Linux amd64/arm64 Panel、Agent、Node 和最终镜像构建通过；
- 前端 81 个测试文件、586 项测试通过；
- i18n 检查通过，共 1891 条文案和 19 个按页加载语言包；
- TypeScript、生产构建、受限容器运行门禁和应用配置生命周期检查通过，生命周期输出 `app_conf_lifecycle=pass`；
- 1280px 暗色桌面和 390px 浅色窄窗口视觉验收通过，累计流量完整可见且页面无横向溢出或控制台错误；
- 公开版本镜像端到端检查输出 `image_e2e=pass`；
- GitHub Release 中 5 个二进制/部署归档均通过公开 `SHA256SUMS` 校验。

远端门禁：

- 候选 CI：<https://github.com/kejilion/KPanel/actions/runs/31387528002>，成功；
- 主线 CI：<https://github.com/kejilion/KPanel/actions/runs/31387776923>，成功；
- Release：<https://github.com/kejilion/KPanel/actions/runs/31388060077>，成功。

## 发布产物

GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.60.0>

Release 为非 draft、非 prerelease，附件已核对：

- `kejilion-agent-linux-amd64`
- `kejilion-agent-linux-arm64`
- `kejilion-node-linux-amd64`
- `kejilion-node-linux-arm64`
- `kejilion-panel-deploy-0.60.0.tar.gz`
- `LICENSE`
- `SHA256SUMS`
- `THIRD_PARTY_NOTICES.md`

Docker Hub：`docker.io/kjlion/kejilion-panel`

```text
0.60.0 / latest index:
sha256:09b0ed3be95b2db4c4a2dfa4aceb31fd369f6ea473e38e06d713d9ec8d6174af

linux/amd64:
sha256:54154a7030f8141520c5e2f1a116cc6064f47941ac0da640bbaaecee07cd6089

linux/arm64:
sha256:092df273a433f6882fff9fd46e065e7f471f1efc3b9635d991dc380553738e51
```

版本镜像与 `latest` 索引摘要一致。`packaging/kejilion-app/kpanel.conf` 相对发布基线未变更，与 `kejilion/apps` 远端 `main` 的配置 blob `d49383667cea8c3b7294bf40ba1e272370a2cb87` 一致；apps 主线保持 `1f2740666a55ccbb3749ce83168e073c1ea08431`，无需应用市场提交。

## 生产部署与验收

部署前生产版本为 `0.59.0`，权威公网入口和本机健康接口均正常，Panel healthy、0 重启，Agent active、0 重启。通过标准应用市场更新入口部署：

```text
KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update /home/docker/kpanel/bin/kejilion.sh app kpanel
```

停写一致性备份：

```text
/root/kpanel-backups/v0.60.0-preupgrade-20260810T124304Z
```

归档校验：

```text
2769bdd1b102dc1b380203a0c1b66bd043fd7ba737713ee0d95595714059c124  kpanel.tar.gz
```

备份归档已独立解包，并与停写源目录逐文件 SHA-256 对比一致；随后恢复 `0.59.0` 服务健康，再执行正式更新。上线后验证：

- `/api/v1/health`：`status=ok`、`version=0.60.0`、`protocolVersion=v1alpha1`；
- Panel：running、healthy、0 重启，镜像索引为 `sha256:09b0ed3b...174af`，OCI 源码提交为 `6d6028a`；
- Agent：active、0 重启、`NeedDaemonReload=no`，安装二进制与正式镜像内 Agent 的 SHA-256 均为 `08c2d9d0018a274b1908a86aeac8613cc081f9adedb8362697b6e089d8f66997`；
- 受管 `kejilion.sh` 仅在 `permission_granted` 安装标记上与固定原始脚本不同，其他内容一致；
- Agent Unix Socket 上的 Hosts、Cron、网卡和防火墙四类只读接口均返回有效的 64 位资源版本；
- 5 份状态 JSON 均可解析，2 份 SQLite 数据库的 `PRAGMA integrity_check` 均返回 `ok`；
- 更新后 Panel 和 Agent 日志无 `panic`、`fatal` 或 `error`；
- 本机及权威公网入口的首页、登录页和健康接口均返回 HTTP 200；
- 2 分钟 60 次本地及公网健康采样全部成功，版本始终为 `0.60.0`，Panel/Agent 始终为 0 重启。

## 回滚点

源码回滚：`v0.59.0`（`5e650d03bb15db348d141014868cb697a147a68d`）。

镜像回滚：

```text
docker.io/kjlion/kejilion-panel@sha256:a2a8a6f28a87536b254539accaf5570ed8c2011c807de7e36fa2bedc29fcd13c
```

本版本没有数据或配置迁移。普通回滚只需把 Compose 镜像固定到上述摘要后重建 Panel，并从该镜像恢复 Agent 和受管脚本。只有数据或配置也需要回退时，才停止 Panel/Agent、校验备份 SHA-256，并恢复 `kpanel.tar.gz`。

## 遗留风险

- 未使用生产管理员凭据进入浏览器逐按钮验收，避免读取或传输生产凭据；UI 行为由组件测试、全量前端测试、本地视觉验收、L3、公开镜像 E2E 和生产健康采样覆盖；
- 旧工作树仍保留各自未提交或未跟踪内容，未纳入本次发布；后续开发应继续从同步后的 `main` 新建分支，避免把这些历史差异误判为新需求；
- 本版本仅调整展示与窗口布局，未在生产执行写操作型业务验收。
