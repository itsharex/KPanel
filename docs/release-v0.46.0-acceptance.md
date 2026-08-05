# KPanel v0.46.0 发布验收

## 发布范围

本版本只上线历史监控长期趋势：

1. 新增 3 个月、6 个月和 12 个月窗口，长期数据分别按 6、12、24 小时聚合展示。
2. Agent 按小时有界汇总主机、容器和三网延迟数据；长期 CPU 同时展示峰值和加权平均值，延迟包含桶内成功率。
3. 原始数据与长期趋势共用既有监控目录，长期月分片限制 16 MiB、长期目录限制 128 MiB、保留期限制 365 天。
4. 修复小时汇总写入失败后累加器无法进入下一小时，以及低于 `1 B/s` 时图表纵轴显示无效单位的问题。

本次没有数据库、端口、Compose、Agent 权限、`kejilion.sh` 协议或应用市场配置迁移；`packaging/kejilion-app/kpanel.conf` 与 Dockerfile 相对 `v0.45.2` 均无变化。升级前已经删除的历史数据不会回填，长期窗口从升级后的采样开始逐步补齐。

## 版本与产物

- 功能提交：`0c8273b`。
- 发布准备提交：`0a95fc4`。
- 低速率单位修复提交：`cc3944638cb7590d41e0c7ea643231d75d890766`。
- 标签：`v0.46.0`。
- 候选分支 CI：<https://github.com/kejilion/KPanel/actions/runs/31009833812>，结论为 `success`。
- 主分支 CI：<https://github.com/kejilion/KPanel/actions/runs/31010033566>，结论为 `success`。
- Release 工作流：<https://github.com/kejilion/KPanel/actions/runs/31010291059>，结论为 `success`。
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.46.0>，已公开且不是 prerelease。
- Docker OCI index：`sha256:c7b71889a48ee9b1be5c1b370b2f3c1ca003bd673b68f9e1dacda7d2c623c8ec`；`0.46.0` 与 `latest` 一致。
- linux/amd64：`sha256:5f52c184fd99ae19f0ab4d340f1d3a2b31badfb77b64d8e29ea919f2688049c6`。
- linux/arm64：`sha256:720ee3de7aaa324091f4427fe97dd981d31a1f194e5e2bc3aa81e958bec2078c`。

## 自动化、154 与浏览器验收

- 本地通过监控与 Agent Go 测试、`go vet`、前端 46 个测试文件共 282 项测试、typecheck 和生产构建；本地化共 1,630 条短语。
- 154 在精确发布提交 `cc39446` 上完整运行 L3：Go 全量测试、核心 race、`go vet`、`govulncheck`、npm audit、Trivy 源码与镜像扫描、双架构构建、部署验证及应用配置生命周期全部通过。
- 公开 `docker.io/kjlion/kejilion-panel:0.46.0` 在 154 重新拉取后通过 `packaging/tests/image-e2e.sh`，确认验收对象是公开发布镜像而不是本地候选镜像。
- 隔离候选环境在 154 Agent 上完成真实浏览器验收：3/6/12 月切换、365 天趋势、CPU 峰值与平均值、存储状态均正常；修复后页面不再出现 `undefined/s`，浏览器错误日志为 0。
- 32 容器、12 个月最大数据基准的响应约 6.08 MiB、查询约 1.01 s；单次 32 容器小时汇总约 36.6 μs，均低于项目既有上限。

## 154 生产上线

- 升级前 Panel 为 `0.45.2`，镜像摘要 `sha256:440585da65ce231fea54fed9739d5bba22ad478f9d046555ad74f0ce542a65a1`；容器 healthy、重启 0、OOM=false，Agent active、重启 0。
- 升级前备份：`/root/kpanel-backups/v0.46.0-preupgrade-20260805T133800Z`，目录权限 0700。SQLite 在线备份 122,880 B，`integrity_check=ok`，SHA-256 为 `5b05d3bfb21d628de4ab2fcba415961099faaf686358dd969705e8db014e3220`；完整 KPanel 归档 20,852,768 B，SHA-256 为 `7e85a013b5e004de6a4720e25453df328d3c19b278c53f5d6ec47c2bc2b86728`。
- 使用 `KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update k app kpanel` 完成标准应用市场更新，实际拉取摘要与 Release OCI index 完全一致。
- 升级后 Panel OCI 版本为 `0.46.0`、修订为 `cc39446`；Agent 日志确认版本 `0.46.0`。Panel healthy、重启 0、OOM=false，Agent active、重启 0。
- 容器继续使用 `65532:65532`、只读根文件系统、256 MiB 和 128 PID 限制；`ai.db integrity_check=ok`，数据库与密钥权限保持 0600。
- 生产 Agent 的 3/6/12 月查询分别返回 6/12/24 小时桶宽，并识别 20 个容器；上线当小时各有 1 个汇总点，符合“不回填、从升级后开始积累”的设计。
- 连续 8 轮稳定性采样通过，Panel 与 Agent 的 panic/fatal/error 计数均为 0。隔离候选容器、候选 Agent、154 临时目录、本地 SSH 隧道和临时凭据均已清理。

## 回滚

- 镜像回滚点为 `v0.45.2` / `sha256:440585da65ce231fea54fed9739d5bba22ad478f9d046555ad74f0ce542a65a1`。
- 现场回滚可使用升级前归档和 SQLite 在线备份成对恢复 `/home/docker/kpanel`，不得覆盖其他业务容器、网站、证书或数据库。
- 本版本没有数据结构迁移；直接切回 `v0.45.2` 时可保留数据目录，旧版本会忽略新增小时分片。若需要保留长期趋势，回滚前应先备份监控目录。
