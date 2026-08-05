# KPanel v0.48.2 发布验收

## 发布范围

本补丁精简监控页容器对比条：容器列表与已选容器条通过鼠标悬停、键盘焦点联动图表高亮，长容器名和镜像名支持省略显示与完整提示，指标改为紧凑网格对齐，并补充可聚焦列表语义和无障碍标签。

新增 3 项布局与交互回归测试，覆盖列表到图表的指针/键盘联动、可聚焦列表语义和紧凑横向布局。本次没有数据库迁移、端口变化、Panel/Agent 协议变化、Compose 变化、`kejilion.sh` 变化或应用市场配置变化。

## 版本与产物

- 发布提交：`bd8bf8cff7a550c3738731438881acb07976c027`。
- 标签：`v0.48.2`。
- 候选分支 CI：[31054076833](https://github.com/kejilion/KPanel/actions/runs/31054076833)，结论 `success`。
- 主分支 CI：[31054258772](https://github.com/kejilion/KPanel/actions/runs/31054258772)，结论 `success`。
- Release 工作流：[31054438172](https://github.com/kejilion/KPanel/actions/runs/31054438172)，结论 `success`。
- GitHub Release：[v0.48.2](https://github.com/kejilion/KPanel/releases/tag/v0.48.2)，已公开、不是 prerelease，共 8 个发布资产。
- Docker OCI index：`sha256:c66a7ac27fd72433eb6514a7f9362d683068c0194b6f0a6e5321acb54460a2fa`；`0.48.2` 与 `latest` 一致。
- linux/amd64：`sha256:459972ea33c4f96226c83e9ad85246dbbec4892af3e69dec41ac310f0ce59df0`。
- linux/arm64：`sha256:925751de63e3b8e1517e8601ae82c73366b9b0fd3925d407605a35a0c2ea45c2`。
- `kejilion/apps` 无契约变化；仓库仍为 `1f2740666a55ccbb3749ce83168e073c1ea08431`，`kpanel.conf` 与 KPanel 发布配置的 Git blob 均为 `d49383667cea8c3b7294bf40ba1e272370a2cb87`。

## 上线前验证

- Windows 候选验证：51 个前端测试文件、322 项测试通过；typecheck、1,652 条国际化短语检查和生产构建通过；主入口 JS 为 `22.60 KiB gzip`，主 CSS 为 `18.64 KiB gzip`，监控页 JS 为 `12.74 KiB gzip`、CSS 为 `3.13 KiB gzip`。
- 154 在精确提交 `bd8bf8c` 上完成完整 L3：Go 全量测试、核心 race、`govulncheck`、npm audit、Trivy 源码与镜像扫描、Linux 构建、部署隔离、应用配置生命周期和最终镜像构建全部通过。
- 候选分支、主分支和 Release 三层 GitHub Actions 门禁均成功；Release 工作流完成多架构镜像构建、稳定标签提升及公开发布。
- 154 从 Docker Hub 按不可变 OCI 摘要重新拉取公开镜像，在独立端口 `18082` 完成冷启动 E2E，输出 `image_e2e=pass`。

## 154 上线结果

- 升级前 Panel 与 Agent 均为 `0.48.1`，Panel healthy、重启 0、OOM=false；旧镜像 revision 为 `df32aff6780143263303382c9aefad33b061a92e`，摘要为 `sha256:132f311530d9782d14f569510fd27fd8be2a6f87f84e580b21e85a15f9d9947d`。
- 升级前备份：`/root/kpanel-backups/v0.48.2-preupgrade-20260805T225948Z`。SQLite 在线备份 122,880 B，`integrity_check=ok`，SHA-256 为 `5b05d3bfb21d628de4ab2fcba415961099faaf686358dd969705e8db014e3220`；完整应用归档 20,973,921 B，SHA-256 为 `751f97eea5fbbf603560644f221fccca93d8031241a493b76317e4603e07b283`。
- 使用 `KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update k app kpanel` 完成标准应用市场更新，实际拉取摘要与 Release OCI index 完全一致。
- 升级后 Panel 与 Agent 均为 `0.48.2`，镜像 revision 为 `bd8bf8cff7a550c3738731438881acb07976c027`；Panel healthy、重启 0、OOM=false，Agent active/enabled、重启 0，且 `NeedDaemonReload=no`。
- 生产 SQLite `integrity_check=ok`；连续 3 轮本机健康检查、`http://154.36.153.9:8080` 公网直连根页和健康接口均返回 200；Panel 与 Agent 最近 5 分钟错误日志计数均为 0。
- Panel 继续使用 `65532:65532`、只读根文件系统、256 MiB、1 CPU、128 PID、`cap-drop ALL`、`no-new-privileges` 和内外双网络。
- 浏览器实际进入 `http://154.36.153.9:8080/login?redirect=/overview`，页面标题为“登录 · KPanel”，用户名/密码输入框和安全登录按钮正常，浏览器控制台错误计数为 0。监控页需要登录态，本轮未读取或传输任何认证信息；其交互与布局由发布前自动化测试覆盖。
- `kp.kejilion.pro` 是独立源站，不属于 154 部署目标；本次未访问其认证信息，也未修改该入口。

## 回滚

- 源码与标签回滚点：`v0.48.1` / `df32aff6780143263303382c9aefad33b061a92e`。
- 镜像回滚点：`docker.io/kjlion/kejilion-panel@sha256:132f311530d9782d14f569510fd27fd8be2a6f87f84e580b21e85a15f9d9947d`。
- 现场回滚使用上述旧镜像及升级前备份恢复 `/home/docker/kpanel`，再重启 Compose 与 Agent。本版本没有数据结构迁移，可直接回滚并保留现有数据格式。
