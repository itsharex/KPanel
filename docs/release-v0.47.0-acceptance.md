# KPanel v0.47.0 发布验收

## 发布范围

本版本只上线历史监控范围缩放与三网延迟图表交互整理：

1. 在任意历史趋势图横向拖动后，用一次带严格 `start`、`end` 的查询同步放大全部主机、容器和三网延迟图表。
2. 提供有界缩放历史、返回和重置操作；根时间范围仍支持 1 小时至 12 个月，点数不足时不会错误进入缩放状态。
3. 三网延迟线路继续使用紧凑开关，移除图表内重复线路图例，九线路悬停提示改为紧凑双列。

本次没有数据库、文件格式、端口、Compose、Agent 权限、`kejilion.sh`、Dockerfile 或应用市场配置迁移；`packaging/kejilion-app/kpanel.conf` 相对 `v0.46.0` 无变化。发布范围不包含 SSH 端口修改、Docker 管理右键菜单或 apps 仓库改动。

## 版本与产物

- 同步缩放功能提交：`fac7106`。
- 延迟图表交互整理提交：`160a0dd`。
- 发布准备提交：`668df46ddf439c5398e9bff69731e5eaeef58830`。
- 标签：`v0.47.0`。
- 候选分支 CI：<https://github.com/kejilion/KPanel/actions/runs/31015031372>，结论为 `success`。
- 主分支 CI：<https://github.com/kejilion/KPanel/actions/runs/31015268987>，结论为 `success`。
- Release 工作流：<https://github.com/kejilion/KPanel/actions/runs/31015599432>，结论为 `success`。
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.47.0>，已公开且不是 prerelease，共 8 个发布资产。
- Docker OCI index：`sha256:a2dd6c96eb32163a725a054bd3ddb0b8914b01cadd690f6673a01271e5a02ce2`；`0.47.0` 与 `latest` 一致。
- linux/amd64：`sha256:33205dd1ee573e104fdacaffd56c2f4a244cdbc8294856c144fe63dd1f5cb1a0`。
- linux/arm64：`sha256:e4f338aeef810d8d44c3c2c2b48529eb77be8f111e2d9b9ac8a97866ff3a3565`。

## 自动化、154 与浏览器验收

- 本地通过监控与 Agent 定向 Go 测试；前端 46 个测试文件共 288 项测试、typecheck、生产构建和 npm audit 均通过；本地化共 1,636 条短语、16 个目录。
- 154 在精确发布提交 `668df46` 上完整运行 L3：Go 全量测试、核心 race、`go vet`、`govulncheck`、npm audit、Trivy 源码与镜像扫描、双架构构建、部署验证及应用配置生命周期全部通过。
- 隔离候选环境使用真实监控数据完成浏览器验收：24 小时和 30 天范围可拖拽同步放大，返回与重置正确；12 个月仅有一个聚合点时不触发无效缩放；九线路悬停框为紧凑双列；浏览器错误日志为 0，导航与 `v0.46.0` 线上基线一致。
- 公开 `docker.io/kjlion/kejilion-panel:0.47.0` 在 154 重新拉取后完成独立冷启动 E2E：容器 healthy、初始化返回 201、Agent 健康返回 200、24 小时与自定义 `start`/`end` 历史查询均返回 200。

## 154 生产上线

- 升级前 Panel 为 `0.46.0`，镜像摘要 `sha256:c7b71889a48ee9b1be5c1b370b2f3c1ca003bd673b68f9e1dacda7d2c623c8ec`；容器 healthy、重启 0、OOM=false，Agent active、重启 0。
- 升级前备份：`/root/kpanel-backups/v0.47.0-preupgrade-20260805T144049Z`，目录权限 0700。SQLite 在线备份 122,880 B，`integrity_check=ok`，SHA-256 为 `5b05d3bfb21d628de4ab2fcba415961099faaf686358dd969705e8db014e3220`；完整 KPanel 归档 20,882,513 B，SHA-256 为 `3dcade57689f81425c54581dc4ba719f0011e18736e13c58b387aecd2d3ecc03`。
- 使用 `KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update k app kpanel` 完成标准应用市场更新，实际拉取摘要与 Release OCI index 完全一致。
- 升级后 Panel OCI 版本为 `0.47.0`、修订为 `668df46`；Agent 健康接口与日志均确认版本 `0.47.0`。Panel healthy、重启 0、OOM=false，Agent active、重启 0。
- 容器继续使用 `65532:65532`、只读根文件系统、256 MiB、1 CPU、128 PID、全部 capability 丢弃及 `no-new-privileges`；生产 `ai.db integrity_check=ok`。
- `.env`、`agent.env`、Compose 和 systemd service 与升级前逐字节一致；数据库与密钥权限保持 0600，Agent token 保持 0640。
- 生产 Agent 的 3/6/12 月查询均成功，分别返回当前小时聚合点、24 个容器和 9 组延迟线路，异常记录为 0；自定义 5 小时窗口返回 298 个主机点、28 个容器和 9 组延迟线路。
- 连续 8 轮稳定性采样通过，Panel 与 Agent 的 `panic`、`fatal`、`error` 计数均为 0。

## 回滚

- 镜像回滚点为 `v0.46.0` / `sha256:c7b71889a48ee9b1be5c1b370b2f3c1ca003bd673b68f9e1dacda7d2c623c8ec`。
- 现场回滚可使用升级前归档和 SQLite 在线备份成对恢复 `/home/docker/kpanel`，不得覆盖其他业务容器、网站、证书或数据库。
- 本版本没有数据结构迁移；直接切回 `v0.46.0` 可保留现有数据目录，旧版本会忽略新版本保存的界面缩放状态。
