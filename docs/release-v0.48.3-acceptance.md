# KPanel v0.48.3 发布验收

## 发布范围

本补丁修复概览 CPU 采样稳定性、网络重复计数、磁盘占用口径和首次网络速率展示：系统摘要先独立完成 CPU 区间采样，再加载页面可选数据；网络累计值优先统计 IPv4/IPv6 默认路由接口，无路由信息时排除 loopback 与已知虚拟接口；磁盘已用容量和使用率改为与 `df` 一致的 `Bfree`/`Bavail` 口径；首次采样及计数回绕时显示“等待下一次采样”。

新增默认路由、虚拟网卡回退、文件系统保留块、CPU 请求时序和前端相邻网络样本测试，并补齐英文资源。本次没有数据库迁移、端口变化、Panel/Agent 协议变化、Compose 变化、`kejilion.sh` 变化或应用市场配置变化。

## 版本与产物

- 补丁提交：`bbf30e6ef31ab77eec3b290c817ad1bf184c5a93`。
- 发布提交：`2b39e6d7dbfd288801c8381af8aeec8811ff0a2f`。
- 标签：`v0.48.3`。
- 候选分支 CI：[31063862062](https://github.com/kejilion/KPanel/actions/runs/31063862062)，结论 `success`。
- 主分支 CI：[31064012815](https://github.com/kejilion/KPanel/actions/runs/31064012815)，结论 `success`。
- Release 工作流：[31064144643](https://github.com/kejilion/KPanel/actions/runs/31064144643)，结论 `success`。
- GitHub Release：[v0.48.3](https://github.com/kejilion/KPanel/releases/tag/v0.48.3)，已公开、不是 prerelease，共 8 个发布资产；正文明确列出 CPU、网络、磁盘与首次采样修复及升级/回滚说明。
- Docker OCI index：`sha256:b79a5305de2ea273e6217fa16f824bffd4f6ef3fd7bc9301131021b114c7cca1`；`0.48.3` 与 `latest` 一致。
- linux/amd64：`sha256:96e951a3b19996a73a0e9e404579cc6c961a2f70a4f28c5e98b29553792d9c37`。
- linux/arm64：`sha256:c6f12bf9f8845dc1c0e789a449348820b943682f18d65cc599c1e7256c78ba4a`。
- `kejilion/apps` 无契约变化；仓库仍为 `1f2740666a55ccbb3749ce83168e073c1ea08431`，`kpanel.conf` 与 KPanel 发布配置的 Git blob 均为 `d49383667cea8c3b7294bf40ba1e272370a2cb87`。

## 上线前验证

- Windows 候选验证：51 个前端测试文件、322 项测试通过；typecheck、1,653 条国际化短语检查和生产构建通过；主入口 JS 为 `22.60 KiB gzip`，主 CSS 为 `18.64 KiB gzip`，概览页 JS 为 `16.74 KiB gzip`。
- 154 隔离环境在补丁提交 `bbf30e6` 上完成 Go 格式检查与 `internal/systeminfo` 定向测试。
- 154 在精确发布提交 `2b39e6d` 上完成完整 L3：Go 全量测试、核心 race、`govulncheck`、npm audit、Trivy 源码与镜像扫描、Linux 双架构构建、安装隔离、应用配置生命周期和最终镜像构建全部通过。
- 候选分支、主分支和 Release 三层 GitHub Actions 门禁均成功；154 从 Docker Hub 按不可变 OCI 摘要重新拉取公开镜像，在独立端口 `18083` 完成冷启动 E2E，输出 `image_e2e=pass`。

## 154 上线结果

- 升级前 Panel 与 Agent 均为 `0.48.2`，Panel healthy、重启 0、OOM=false；旧镜像 revision 为 `bd8bf8cff7a550c3738731438881acb07976c027`，摘要为 `sha256:c66a7ac27fd72433eb6514a7f9362d683068c0194b6f0a6e5321acb54460a2fa`。
- 升级前备份：`/root/kpanel-backups/v0.48.3-preupgrade-20260806T020241Z`。SQLite 在线备份 122,880 B，`integrity_check=ok`，SHA-256 为 `5b05d3bfb21d628de4ab2fcba415961099faaf686358dd969705e8db014e3220`；完整应用归档 21,004,920 B，SHA-256 为 `f50700d3683dd1d48cb33440f175a9b4d84bf4de5e6f65efcbd6cc02c57cac50`。
- 使用 `KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update k app kpanel` 完成标准应用市场更新，实际拉取摘要与 Release OCI index 完全一致。
- 升级后 Panel 与 Agent 均为 `0.48.3`，镜像 revision 为 `2b39e6d7dbfd288801c8381af8aeec8811ff0a2f`；Panel healthy、重启 0、OOM=false，Agent active/enabled、重启 0，且 `NeedDaemonReload=no`。
- 生产 SQLite `integrity_check=ok`；连续 3 轮本机健康检查、`http://154.36.153.9:8080` 公网直连根页和健康接口均返回 200；Panel 与 Agent 最近 5 分钟错误日志计数均为 0。
- Panel 继续使用 `65532:65532`、只读根文件系统、256 MiB、1 CPU、128 PID、`cap-drop ALL`、`no-new-privileges` 和内外双网络。
- 浏览器访问生产根页后正常渲染 KPanel 登录界面，用户名/密码输入框和安全登录按钮存在，页面控制台错误计数为 0。概览内部需要登录态，本轮未读取或传输任何认证信息；CPU、网络和磁盘逻辑由发布前自动化及隔离环境覆盖，未将未认证浏览器检查计为概览实机视觉证据。
- `kp.kejilion.pro` 是独立源站，不属于 154 部署目标；本次未访问其认证信息，也未修改该入口。

## 协作限制

- GitHub 公开 Issue 中没有本补丁对应的 AI Task Issue，本机 GitHub CLI 也未配置写入认证；本轮依据用户明确上线授权，由唯一发布工作树串行执行，提交、候选分支、CI、标签和验收记录作为可追溯真源。

## 回滚

- 源码与标签回滚点：`v0.48.2` / `bd8bf8cff7a550c3738731438881acb07976c027`。
- 镜像回滚点：`docker.io/kjlion/kejilion-panel@sha256:c66a7ac27fd72433eb6514a7f9362d683068c0194b6f0a6e5321acb54460a2fa`。
- 现场回滚使用上述旧镜像及升级前备份恢复 `/home/docker/kpanel`，再重启 Compose 与 Agent。本版本没有数据结构迁移，可直接回滚并保留现有数据格式。
