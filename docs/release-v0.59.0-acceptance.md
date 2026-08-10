# KPanel v0.59.0 发布验收记录

日期：2026-08-10

发布级别：L3

生产目标：`arena-154`（`154.36.153.9:8080`）

## 发布范围

本版本只包含概览系统/网络资源工具及其受控宿主机适配链路：

- `a9be061`、`aa99c37`、`a9a3c45`：定义适配边界、修正调用说明并记录 154 L2 闭环；
- `a88d105`：在概览加入 Hosts、root 定时任务、网卡和防火墙工具；
- `2774051`：增加 Panel/Agent 的类型化宿主机资源 API；
- `03ad7d4`：加固资源版本冲突、跨进程锁、写后回读、持久化和失败回滚；
- `16b7ee2`：固定 `kejilion.sh` system-resource v3 协议提交与摘要；
- `5e650d0`：统一版本字段并准备 KPanel `0.59.0`。

正式源码、标签和通过 CI 的提交均为：

```text
5e650d03bb15db348d141014868cb697a147a68d
```

配套 `kejilion/sh` 主线提交为：

```text
3972217d4d4a51d473b7375f5b850870e066be92
```

原始 `kejilion.sh` SHA-256：

```text
90da9f13da056b133079fa6fcb673004f377a6b2c0e69e01015040f234e51921
```

本版本没有数据库、端口、Compose 或应用市场安装契约迁移。防火墙“全部开放/关闭”保留脚本既有的高影响语义，因此 UI、API 和脚本均要求明确确认并使用资源版本保护。

## 发布前验证

在独立候选工作树、154 隔离 L2 环境和 154 完整 Linux L3 环境执行验证：

- `release-kpanel v1.4` 工作流校验和参数渲染通过，无未替换占位符；
- `go test ./...`、`go vet ./...`、核心特权包 race 检查通过；
- `govulncheck`、`npm audit`、Trivy 源码和最终镜像扫描通过，未发现可达漏洞、依赖漏洞、密钥或配置问题；
- Linux amd64/arm64 Panel、Agent、Node 和最终镜像构建通过；
- 前端 81 个测试文件、582 项测试通过；
- i18n 检查通过，共 1891 条文案和 19 个按页加载语言包；
- TypeScript、生产构建、受限容器运行门禁和应用配置生命周期检查通过；
- 154 隔离候选 Panel/Agent 完成 Hosts、Cron、网卡和防火墙真实读写、回读、删除与原状恢复；生产 Panel 全程保持 `0.58.0` 健康；
- 公开镜像端到端检查输出 `image_e2e=pass`。

远端门禁：

- 候选 CI：<https://github.com/kejilion/KPanel/actions/runs/31380578690>，成功；
- 主线 CI：<https://github.com/kejilion/KPanel/actions/runs/31380814030>，成功；
- Release：<https://github.com/kejilion/KPanel/actions/runs/31381049413>，成功。

## 发布产物

GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.59.0>

Release 为非 draft、非 prerelease，附件已核对：

- `kejilion-agent-linux-amd64`
- `kejilion-agent-linux-arm64`
- `kejilion-node-linux-amd64`
- `kejilion-node-linux-arm64`
- `kejilion-panel-deploy-0.59.0.tar.gz`
- `LICENSE`
- `SHA256SUMS`
- `THIRD_PARTY_NOTICES.md`

Docker Hub：`docker.io/kjlion/kejilion-panel`

```text
0.59.0 / latest index:
sha256:a2a8a6f28a87536b254539accaf5570ed8c2011c807de7e36fa2bedc29fcd13c

linux/amd64:
sha256:94a8b1cece82040a62ac5f6a34251bc6ebdff372d1b36832357cb762334139d1

linux/arm64:
sha256:0fec226aea19c8c64b1f086a8c75b015e11a9a1002f3df78efe0de817612cc3d
```

版本镜像与 `latest` 索引摘要一致。`packaging/kejilion-app/kpanel.conf` 未变更，与 `kejilion/apps` 远端 `main` 的配置 blob `d49383667cea8c3b7294bf40ba1e272370a2cb87` 一致，无需应用市场提交。

## 生产部署与验收

部署前生产版本为 `0.58.0`，Panel healthy、0 重启，Agent active、0 重启。通过标准应用市场更新入口部署：

```text
KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update /home/docker/kpanel/bin/kejilion.sh app kpanel
```

停写一致性备份：

```text
/root/kpanel-backups/v0.59.0-preupgrade-20260810T105951Z
```

归档校验：

```text
0bc6f6f4317574064248378cd56b456bf86c5d2eb09ff01e3eafcfdc18df49e2  kpanel.tar.gz
```

备份归档已独立解包，并与停写源目录逐文件 SHA-256 对比一致；随后恢复 `0.58.0` 服务健康，再执行正式更新。上线后验证：

- `/api/v1/health`：`status=ok`、`version=0.59.0`、`protocolVersion=v1alpha1`；
- Panel：running、healthy、0 重启，镜像索引为 `sha256:a2a8a6f2...cd13c`，OCI 源码提交为 `5e650d0`；
- Agent：active、0 重启、`NeedDaemonReload=no`，安装二进制与正式镜像内 Agent 的 SHA-256 均为 `80e06801f5034e49b900d7aca1050b47fd341a97930021957615e5a5d833e4ba`；
- 受管 `kejilion.sh` 仅在 `permission_granted` 安装标记上与固定原始脚本不同，归一化后逐字节一致；system-resource v3 标记完整；
- Agent Unix Socket 上的 Hosts、Cron、网卡和防火墙四类只读接口均返回有效的 64 位资源版本；
- 5 份状态 JSON 均可解析，2 份 SQLite 数据库的 `PRAGMA integrity_check` 均返回 `ok`；
- 更新后 Panel 和 Agent 日志无 `panic`、`fatal` 或 `error`；
- 本机及公网的首页、登录页和健康接口均返回 HTTP 200；
- 2 分钟 60 次本地及公网健康采样全部成功，版本始终为 `0.59.0`，Panel/Agent 始终为 0 重启。

## 回滚点

源码回滚：`v0.58.0`（`da1c7be9d21891504a19f4e58dac5f078e9ce955`）。

镜像回滚：

```text
docker.io/kjlion/kejilion-panel@sha256:3c28f8930da2fa1b96b735394a333bcca4d3f3d6680800001c2d537fc2cdf8b6
```

本版本没有数据或配置迁移。普通回滚只需把 Compose 镜像固定到上述摘要后重建 Panel，并从该镜像恢复 Agent 和受管脚本。只有数据或配置也需要回退时，才停止 Panel/Agent、校验备份 SHA-256，并恢复 `kpanel.tar.gz`。

回滚 KPanel 不会撤销管理员上线后主动完成的 Hosts、Crontab、网卡或防火墙修改；这类宿主机资源变更应使用各自的审计记录和备份单独恢复。

## 遗留风险

- 上线后未再次写入生产 Hosts、Crontab、网卡或防火墙，避免为验收制造生产配置变更；真实写入、冲突、回滚及原状恢复已在同一台 154 的隔离 L2 候选环境完成；
- 未使用生产管理员凭据进入浏览器逐按钮验收，避免读取或传输生产凭据；UI 行为由组件测试、全量前端测试、L3、公开镜像 E2E 和生产只读资源 API 验收覆盖；
- 防火墙“全部开放/关闭”会清空 filter 表规则和自定义链后重建基础规则，仍属于高影响操作；使用前必须核对云安全组、现有规则和可用回滚路径。
