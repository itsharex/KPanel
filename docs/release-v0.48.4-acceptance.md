# KPanel v0.48.4 发布验收

## 发布范围

本补丁发布 AI 工具拒绝后的安全重规划、Responses 历史消息 ID 兼容、弹窗焦点与无障碍加固，以及概览服务状态布局修复。桌面模式等新功能不在本版本范围内，后续按 `v0.49.0` 规划。

本次没有数据库、端口、Compose、Agent 权限、`kejilion.sh` 协议或应用市场配置迁移，可直接从 `v0.48.3` 升级并保留现有数据。

## 版本与产物

- 发布提交：`61a498e4ae7542749f9ae49ca4e2893724605bf4`。
- 标签：`v0.48.4`。
- 候选分支 CI：[31251311376](https://github.com/kejilion/KPanel/actions/runs/31251311376)，结论 `success`。
- 主分支 CI：[31251367943](https://github.com/kejilion/KPanel/actions/runs/31251367943)，结论 `success`。
- Release 工作流：[31251432437](https://github.com/kejilion/KPanel/actions/runs/31251432437)，结论 `success`。
- GitHub Release：[v0.48.4](https://github.com/kejilion/KPanel/releases/tag/v0.48.4)，已公开、不是 prerelease，共 8 个发布资产。
- Docker OCI index：`sha256:863c271dcde1527aad2e0f3f7098220ed8b2febef05119683c6bf1296d164de8`；`0.48.4` 与 `latest` 一致。
- `linux/amd64`：`sha256:de324ab2d0e4fdc281320a6fc577e9937605d5e9ba2c617473c918a95f13f566`。
- `linux/arm64`：`sha256:05491160ffba93681fe060dece5ee9595113931243e9f7640a7787ffb22e560d`。
- `kejilion/apps` 无契约变化；远端 `main` 仍为 `1f2740666a55ccbb3749ce83168e073c1ea08431`，其 `kpanel.conf` 与本仓库发布配置的 Git blob 均为 `d49383667cea8c3b7294bf40ba1e272370a2cb87`。

## 上线前验证

- AI Runtime 测试补齐后台 worker 收尾，`internal/ai` 连续 50 轮稳定性测试通过；修复提交为 `61a498e`。
- 精确发布提交在 154 隔离目录完成完整 L3：Go 全量测试、核心 race、`govulncheck`、`npm audit`、Trivy 源码与镜像扫描、安装安全、应用配置生命周期和最终镜像构建全部通过。
- 前端 54 个测试文件、330 项测试通过；typecheck、1,653 条国际化短语检查和生产构建通过。主入口 JS 为 `22.60 KiB gzip`，主 CSS 为 `18.64 KiB gzip`。
- 候选分支、主分支和 Release 三层 GitHub Actions 门禁均成功。
- 154 从 Docker Hub 重新拉取公开 `0.48.4` 镜像，在隔离端口 `18084` 完成冷启动 E2E，输出 `image_e2e=pass`。

## 154 上线结果

- 升级前 Panel 与 Agent 均为 `0.48.3`；Panel healthy、重启 0、OOM=false，镜像摘要为 `sha256:b79a5305de2ea273e6217fa16f824bffd4f6ef3fd7bc9301131021b114c7cca1`。
- 升级前备份：`/root/kpanel-backups/v0.48.4-preupgrade-20260808T100024Z`，目录权限 0700。SQLite 在线备份 122,880 B，`integrity_check=ok`，SHA-256 为 `5b05d3bfb21d628de4ab2fcba415961099faaf686358dd969705e8db014e3220`；完整应用归档 21,507,498 B，SHA-256 为 `27cb1e4f7707ed3d5a0acd47620fc1749f8d8aef48c08e884fabcd0ccfbf46c3`。
- 使用 `KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update k app kpanel` 完成标准应用市场更新，实际拉取摘要与 Release OCI index 完全一致。
- 升级后 Panel 健康接口连续 3 次返回 `status=ok`、`version=0.48.4`；容器 healthy、重启 0、OOM=false，继续使用 `65532:65532` 和只读根文件系统。
- 镜像 OCI 标签为 `version=0.48.4`、`revision=61a498e4ae7542749f9ae49ca4e2893724605bf4`。Agent 二进制与镜像内 `/release/kejilion-agent` 一致，`kejilion-agent version` 返回 `0.48.4 v1alpha1`；服务 active/enabled、重启 0，`NeedDaemonReload=no`。
- 生产 SQLite 再次检查为 `integrity_check=ok`；`http://154.36.153.9:8080/` 返回 200，公网健康接口返回 `status=ok`、`version=0.48.4`；Panel 与 Agent 最近 10 分钟错误日志计数均为 0。
- `kp.kejilion.pro` 是独立源站，不属于 154 部署目标，本次未修改该入口。

## 回滚

- 源码与标签回滚点：`v0.48.3` / `2b39e6d7dbfd288801c8381af8aeec8811ff0a2f`。
- 镜像回滚点：`docker.io/kjlion/kejilion-panel@sha256:b79a5305de2ea273e6217fa16f824bffd4f6ef3fd7bc9301131021b114c7cca1`。
- 现场回滚使用上述旧镜像以及升级前备份恢复 `/home/docker/kpanel`，再重启 Compose 与 Agent。本版本无数据结构迁移，可直接回滚并保留现有数据格式。
