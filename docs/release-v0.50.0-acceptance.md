# KPanel v0.50.0 发布验收

## 发布范围

本版本只上线三项最近未上线变更：

1. 终端主机选择区支持收起与展开，记住本机偏好，并补充移动端、减少动态效果和无障碍状态支持。
2. 概览时区优先跟随宿主机 `/etc/localtime` 来源，无法识别时不再误报上海时区。
3. 桌面上动态生成的已安装应用和网站图标完整填充入口图块。

旧文件管理分支中的滚动恢复补丁未进入本版本，因为相同能力已由 `e9982bd` 实现并包含在
`v0.49.0`。本版本不包含系统中心、治理提交、数据库结构、端口、Compose、Agent 权限、
`kejilion.sh` 协议或应用市场配置迁移。

## 版本与产物

- 发布提交：`13da6b53b6efd79cb2e09b0a531b80131d61d745`。
- 标签：`v0.50.0`。
- 候选分支 CI：[31256961435](https://github.com/kejilion/KPanel/actions/runs/31256961435)，结论 `success`。
- 主分支 CI：[31257084019](https://github.com/kejilion/KPanel/actions/runs/31257084019)，结论 `success`。
- Release 工作流：[31257200700](https://github.com/kejilion/KPanel/actions/runs/31257200700)，结论 `success`。
- GitHub Release：[v0.50.0](https://github.com/kejilion/KPanel/releases/tag/v0.50.0)，已公开、非 prerelease，共 8 个发布资产。
- Docker OCI index：`sha256:f54bd72c6d01465de294326329fc131e30c1c04603d9e12f97e5b4763560904c`；`0.50.0` 与 `latest` 一致。
- `linux/amd64`：`sha256:3dcc96cca98655aa71458417c8f19b2732903960f450d78650d1fbdd352e4a47`。
- `linux/arm64`：`sha256:282b67e11afbe5f1a850a6397e5d4e6949abe3a3dbb7985c4793306f417edf15`。
- `kejilion/apps` 无契约变化；本地与远端 `main` 均为 `1f2740666a55ccbb3749ce83168e073c1ea08431`，两仓库 `kpanel.conf` Git blob 均为 `d49383667cea8c3b7294bf40ba1e272370a2cb87`。

## 上线前验证

- 干净安装 252 个前端依赖，npm 审计为 0 个漏洞。
- 前端 68 个测试文件、447 项测试通过；typecheck、1,655 条国际化短语校验和生产构建通过。
- 候选分支、主分支和 Release 三层 GitHub Actions 门禁全部成功；覆盖 Go 全量测试、核心 race、
  `govulncheck`、npm audit、Trivy 源码与镜像扫描、部署安全、应用配置生命周期、多架构二进制、
  最终镜像构建和受限运行时契约。
- 154 从 Docker Hub 按正式标签拉取公开镜像，在隔离网络和端口 `18091` 两次完成冷启动 E2E，
  最终命令退出码为 0，输出 `image_e2e=pass`。

## 154 上线结果

- 升级前 Panel 为 `0.49.0`，容器 healthy，镜像摘要为
  `sha256:e9af766e2ab3ffa4079fae71e0dd06f0071047734c54e194f061ae47dfa22048`。
- 升级前备份：`/root/kpanel-backups/v0.50.0-preupgrade-20260808T123243Z`，目录权限 0700；
  SQLite 在线备份 122,880 B、`integrity_check=ok`、SHA-256 为
  `5b05d3bfb21d628de4ab2fcba415961099faaf686358dd969705e8db014e3220`；完整应用归档
  21,532,263 B、SHA-256 为 `55a71e6a67a91108a09dd2427c02fae51361acf799a228e315a04c109fd5104c`。
- 使用 `KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update k app kpanel` 完成标准应用市场更新，实际拉取
  摘要与 Release OCI index 完全一致。
- 升级后 Panel 健康接口连续 3 次返回 `status=ok`、`version=0.50.0`；容器 healthy、重启 0、
  OOM=false，OCI `revision=13da6b53b6efd79cb2e09b0a531b80131d61d745`。
- Panel 继续使用 `65532:65532`、只读根文件系统、256 MiB、1 CPU、128 PID、`cap-drop ALL`
  和 `no-new-privileges`。
- Agent 为 `0.50.0 v1alpha1`、active/enabled、重启 0、`NeedDaemonReload=no`；宿主机 Agent 与
  镜像内置二进制 SHA-256 均为 `adf3e71563d5624ed1210b00fca252843cb43adb37fe7e8c56b348b2106326d9`。
- 生产 SQLite 再次检查为 `integrity_check=ok`；Panel 与 Agent 最近 10 分钟
  `panic/fatal/error` 计数均为 0；`http://154.36.153.9:8080/` 返回 200。
- 生产登录页在真实浏览器加载正常，控制台 `warn/error` 为 0；由于现有浏览器没有 154 登录会话，
  本次未执行登录后的终端收起和桌面动态图标视觉点击，相关交互由布局、组件和全量回归测试覆盖。
- `/usr/local/bin/k` 修改时间仍为 2026-08-05，本次应用更新未修改宿主机 `kejilion.sh`；本地 sh
  仓库与远端 `main` 均为 `dd26cf7eb962f985f94773c15f9b643677b4471c`。
- `kp.kejilion.pro` 是独立源站，不属于 154 部署目标，本次未修改该入口。

## 回滚

- 源码与标签回滚点：`v0.49.0` / `203285f175c4071228b36c2f1ecce7340648cb58`。
- 镜像回滚点：`docker.io/kjlion/kejilion-panel@sha256:e9af766e2ab3ffa4079fae71e0dd06f0071047734c54e194f061ae47dfa22048`。
- 现场回滚使用上述旧镜像及升级前备份恢复 `/home/docker/kpanel`，再重启 Compose 与 Agent。
  本版本没有数据结构迁移，可直接回滚并保留现有数据格式。
