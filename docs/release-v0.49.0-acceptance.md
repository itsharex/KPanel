# KPanel v0.49.0 发布验收

## 发布范围

本次独立发布桌面模式：新增 Windows 风格桌面工作区、桌面应用图标、窗口与任务栏交互、已安装应用和网站入口、服务器监控、窗口生命周期及轮询控制。上线前额外修复桌面窗口懒加载页面组件被 Vue 深层响应式代理的问题，并加入回归测试。

本版本不包含数据库结构、端口、Compose、Panel/Agent 协议、`kejilion.sh` 或应用市场配置迁移；可直接从 `v0.48.4` 升级并保留现有数据。

## 版本与产物

- 发布提交：`203285f175c4071228b36c2f1ecce7340648cb58`。
- 标签：`v0.49.0`。
- 候选分支 CI：[31253230635](https://github.com/kejilion/KPanel/actions/runs/31253230635)，结论 `success`。
- 主分支 CI：[31253365005](https://github.com/kejilion/KPanel/actions/runs/31253365005)，结论 `success`。
- Release 工作流：[31253502824](https://github.com/kejilion/KPanel/actions/runs/31253502824)，结论 `success`。
- GitHub Release：[v0.49.0](https://github.com/kejilion/KPanel/releases/tag/v0.49.0)，已公开、非 prerelease，共 8 个发布资产。
- Docker OCI index：`sha256:e9af766e2ab3ffa4079fae71e0dd06f0071047734c54e194f061ae47dfa22048`；`0.49.0` 与 `latest` 一致。
- `linux/amd64`：`sha256:c1cd049220b04cedb897f2d27f0a92294c3b4a7d6cfe1835fcc16bdeccc0df38`。
- `linux/arm64`：`sha256:fb7d45ec3060347503af1cd4c068225c31f68cfac81647e8dbadb6119845dfac`。
- `kejilion/apps` 无契约变化；本地与远端 `main` 均为 `1f2740666a55ccbb3749ce83168e073c1ea08431`，两仓库 `kpanel.conf` Git blob 均为 `d49383667cea8c3b7294bf40ba1e272370a2cb87`。

## 上线前验证

- 前端 68 个测试文件、445 项测试通过；typecheck、1,655 条国际化短语校验和生产构建通过。
- 候选分支、主分支和 Release 三层 GitHub Actions 门禁全部成功；覆盖 Go 全量测试、核心 race、`govulncheck`、`npm audit`、Trivy 源码/镜像扫描、部署安全、应用配置生命周期、多架构二进制与最终镜像构建。
- 桌面模式人工验收覆盖进入桌面、桌面入口、文件窗口打开、最大化、还原、最小化和任务栏恢复；修复后干净页面的浏览器 `warn/error` 日志为 0。
- 154 从 Docker Hub 按不可变摘要重新拉取公开镜像，在隔离网络和端口 `18090` 完成冷启动 E2E，输出 `image_e2e=pass`。

## 154 上线结果

- 升级前 Panel 与 Agent 均为 `0.48.4`，Panel healthy、重启 0，镜像摘要为 `sha256:863c271dcde1527aad2e0f3f7098220ed8b2febef05119683c6bf1296d164de8`。
- 升级前备份：`/root/kpanel-backups/v0.49.0-preupgrade-20260808T105352Z`，目录权限 0700；SQLite 在线备份 122,880 B、`integrity_check=ok`、SHA-256 为 `5b05d3bfb21d628de4ab2fcba415961099faaf686358dd969705e8db014e3220`；完整应用归档 21,517,332 B、SHA-256 为 `7275d551e9256b7bfb945865cd31368b10426309f55ee19754cdc465a681efe7`。
- 使用 `KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update k app kpanel` 完成标准应用市场更新，实际拉取摘要与 Release OCI index 完全一致。
- 升级后 Panel 健康接口连续 3 次返回 `status=ok`、`version=0.49.0`；容器 healthy、重启 0，OCI `revision=203285f175c4071228b36c2f1ecce7340648cb58`。
- Panel 继续使用 `65532:65532`、只读根文件系统、256 MiB、1 CPU、128 PID、`cap-drop ALL` 和 `no-new-privileges`；Agent 为 `0.49.0 v1alpha1`、active/enabled、重启 0、`NeedDaemonReload=no`，宿主机 Agent 与镜像内置二进制一致。
- 生产 SQLite 再次检查为 `integrity_check=ok`；Panel 与 Agent 最近 10 分钟错误日志计数均为 0；`http://154.36.153.9:8080/` 返回 200，公网健康接口返回 `status=ok`、`version=0.49.0`。
- 升级前后 `kejilion.sh` SHA-256 一致；本次上线未修改脚本。
- `kp.kejilion.pro` 是独立源站，不属于 154 部署目标，本次未修改该入口。

## 回滚

- 源码与标签回滚点：`v0.48.4` / `61a498e4ae7542749f9ae49ca4e2893724605bf4`。
- 镜像回滚点：`docker.io/kjlion/kejilion-panel@sha256:863c271dcde1527aad2e0f3f7098220ed8b2febef05119683c6bf1296d164de8`。
- 现场回滚使用上述旧镜像及升级前备份恢复 `/home/docker/kpanel`，再重启 Compose 与 Agent。本版本没有数据库结构迁移，可直接回滚并保留现有数据格式。
