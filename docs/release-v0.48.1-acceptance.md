# KPanel v0.48.1 发布验收

## 发布范围

本补丁修复 Vue Router 滚动恢复：浏览器前进/后退时保留已保存位置，同一路径仅查询参数变化时保持当前滚动位置，跨页面导航回到顶部。新增 3 项单元测试覆盖以上分支。

上线前门禁复现到既有 AI 重试测试未等待后台 Runtime 结束，导致临时目录偶发清理竞态；独立提交仅增加 `defer runtime.Close()`，不修改生产运行逻辑。

本次没有数据库迁移、端口变化、Panel/Agent 协议变化、Compose 变化、`kejilion.sh` 变化或应用市场配置变化。

## 版本与产物

- 发布提交：`df32aff6780143263303382c9aefad33b061a92e`。
- 标签：`v0.48.1`。
- 候选分支 CI：[31049392768](https://github.com/kejilion/KPanel/actions/runs/31049392768)，结论 `success`。
- 主分支 CI：[31049527493](https://github.com/kejilion/KPanel/actions/runs/31049527493)，结论 `success`。
- Release 工作流：[31049707170](https://github.com/kejilion/KPanel/actions/runs/31049707170)，结论 `success`。
- GitHub Release：[v0.48.1](https://github.com/kejilion/KPanel/releases/tag/v0.48.1)，已公开、不是 prerelease，共 8 个发布资产。
- Docker OCI index：`sha256:132f311530d9782d14f569510fd27fd8be2a6f87f84e580b21e85a15f9d9947d`；`0.48.1` 与 `latest` 一致。
- linux/amd64：`sha256:aad9bfbbbaad1022a098678943b36273860722baefe85adbb1fdbb64a1191bfb`。
- linux/arm64：`sha256:d41294a783e1cfdbcfb869a24542c654838e0278862854232f374271cf84327f`。
- `kejilion/apps` 无契约变化；仓库仍为 `1f2740666a55ccbb3749ce83168e073c1ea08431`，`kpanel.conf` 与 KPanel 发布配置的 Git blob 均为 `d49383667cea8c3b7294bf40ba1e272370a2cb87`。

## 上线前验证

- Windows 候选验证：50 个前端测试文件、319 项测试通过；typecheck、1,653 条国际化短语检查和生产构建通过；主入口 JS 为 `22.61 KiB gzip`，主 CSS 为 `18.64 KiB gzip`。
- AI 重试测试清理修复在隔离 Linux 环境普通模式重复 50 次、`-race` 模式重复 10 次，均通过。
- 154 在精确提交 `df32aff` 上完成完整 L3：Go 全量测试、核心 race、`govulncheck`、npm audit、Trivy 源码与镜像扫描、Linux 构建、部署隔离、应用配置生命周期和最终镜像构建全部通过。
- 154 从 Docker Hub 按不可变 OCI 摘要重新拉取公开镜像并完成独立冷启动 E2E，输出 `image_e2e=pass`。

## 154 上线结果

- 升级前 Panel 与 Agent 均为 `0.48.0`，Panel healthy、重启 0、OOM=false；旧镜像摘要为 `sha256:144a64a77768e23b7d266d0d66ac944c77b2a968d11dbb5e9d2dd7d1b148a491`。
- 升级前备份：`/root/kpanel-backups/v0.48.1-preupgrade-20260805T214701Z`。SQLite 在线备份 122,880 B，`integrity_check=ok`，SHA-256 为 `5b05d3bfb21d628de4ab2fcba415961099faaf686358dd969705e8db014e3220`；完整应用归档 20,962,803 B，SHA-256 为 `738210bb4dec1ed67856268ae399e97ce70682eb0fed659cb905a099821b8667`。
- 使用 `KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update k app kpanel` 完成标准应用市场更新，实际拉取摘要与 Release OCI index 完全一致。
- 升级后 Panel 与 Agent 均为 `0.48.1`，镜像 revision 为 `df32aff`；Panel healthy、重启 0、OOM=false，Agent active、重启 0，且无需 systemd daemon reload。
- 生产 SQLite `integrity_check=ok`；连续 3 轮本机健康检查、`http://154.36.153.9:8080` 公网直连根页和健康接口均返回 200；Panel 与 Agent 最近 5 分钟错误日志计数均为 0。
- Panel 继续使用 `65532:65532`、只读根文件系统、256 MiB、1 CPU、128 PID、`cap-drop ALL`、`no-new-privileges` 和内外双网络。
- 桌面浏览器实际进入 `http://154.36.153.9:8080/login?redirect=/overview`，标题、用户名/密码输入框和安全登录按钮正常。手机端布局沿用发布前隔离截图验收；本轮浏览器视口覆盖未生效，因此没有将其计为新增线上证据。
- `https://kp.kejilion.pro` 实时健康接口仍返回另一源站的 `0.45.1`。154 没有该域名 vhost，历史验收也已记录二者不是同一部署目标；本次未越权修改该入口。

## 回滚

- 源码与标签回滚点：`v0.48.0` / `b298c9647b07d50709b753a25beb85af72274027`。
- 镜像回滚点：`docker.io/kjlion/kejilion-panel@sha256:144a64a77768e23b7d266d0d66ac944c77b2a968d11dbb5e9d2dd7d1b148a491`。
- 现场回滚使用上述旧镜像及升级前备份恢复 `/home/docker/kpanel`，再重启 Compose 与 Agent。本版本没有数据结构迁移，可直接回滚并保留现有数据格式。
