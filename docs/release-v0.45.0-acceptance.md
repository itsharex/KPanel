# KPanel v0.45.0 发布验收

## 发布范围

v0.45.0 合并并上线以下 6 项：

1. 概览 SSH 端口修改改为固定 `kejilion.sh ssh-port` 协议，并在成功前验证 `sshd` 配置与真实监听。
2. 体检工作区全屏模式，以及体检、应用交互终端和主机终端的滚动边界修复。
3. 历史监控三网九线路悬停提示改为紧凑双列布局。
4. 集群主机操作系统图标识别与 Linux 回退规则修复。
5. 登录、初始化错误本地化、校验、重试跳转和无障碍反馈优化。
6. `kejilion.sh` 多网络容器 `DOCKER-USER` 规则写入、识别和清理修复。

## 版本与制品

- KPanel 发布提交：`e24f20010fd31f6447323eefd2c52d5779cb9a37`。
- `kejilion/sh` 提交：`dd26cf7eb962f985f94773c15f9b643677b4471c`。
- 镜像内原始 `kejilion.sh` SHA-256：
  `cf835ec01a10955e91d49a5653870a8cab33288b32ee5cab40405bd69ef77bf4`。
- 标签：`v0.45.0`。
- Release 工作流：<https://github.com/kejilion/KPanel/actions/runs/30984686974>。
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.45.0>。
- Docker OCI index：`sha256:6676752672d069e0c5659b31d4960b4bf11415e5d4e8d624fce6397dc0020d1c`。
- linux/amd64：`sha256:e06324342d3bf10fbc52ad5b26690d4f421a77fdebb629e03af8c29f7921b98c`。
- linux/arm64：`sha256:1583f471f9e3d609c923e0c6cc55d84d0b0306b8a7c264c99ea5934120e2240c`。

Release 已公开且不是 prerelease，Agent、Node、部署归档、许可证、第三方声明和 `SHA256SUMS` 共 8 个附件完整。
`0.45.0` 与 `latest` 指向同一 OCI index；额外 `unknown/unknown` 清单为 amd64/arm64 的 SBOM/Provenance
attestation。

## 自动化、154 与浏览器验收

- 154 在精确发布提交运行仓库原生 `VERIFY_LEVEL=release bash scripts/verify-change.sh`，完整 L3 返回 0。
  全量 Go 测试、核心特权包 race、`go vet ./...`、`govulncheck`、npm audit、Trivy 源码与镜像扫描均通过；
  可达漏洞、npm 漏洞及 HIGH/CRITICAL 结果均为 0。
- 前端 typecheck、生产构建、43 个 Vitest 文件共 269 项测试和 16 个目录共 1,623 条本地化短语通过；
  linux/amd64 与 linux/arm64 的 Panel、Agent、Node、kpctl 均完成 CGO-free 构建。
- `kejilion.sh` 根级 16 项 smoke 全部通过；root/cn 同步、SSH 固定协议、应用生命周期和多网络防火墙均通过。
  OpenClaw 子目录的一项独立测试因测试夹具缺少 `openclaw_get_config_file` 失败，同样可在未修改的
  `82aaa87b3dea671e6fc5e1a60f536c40352c47ad` 复现，不属于本次变更回归。
- 154 使用两个临时 Docker 网络和真实 `DOCKER-USER` 链验证多网络规则：两个 IPv4 各写入 7 条规则，
  重复执行无重复项，清理后候选规则为 0，临时容器和网络均已删除。
- 154 当前 SSH 有效端口为 22。真实部署脚本执行 `ssh-port 22` 返回 `KPANEL_SSH_RESULT unchanged`，非法端口
  `0`、`65536` 和 `abc` 均被拒绝；配置、INPUT 链及防火墙包状态未变化。由于测试机没有云安全组控制面和
  带外登录通道，未在生产主机执行真实端口切换；成功路径、配置语法和监听失败关闭由脚本 smoke、Go 测试与
  L3 覆盖。
- 隔离候选实例只读连接 154 Agent 完成浏览器验证：初始化弱密码/不一致提示、错误登录、正确登录、会话跳转、
  体检全屏与 `Esc` 退出、终端滚动锁、Debian 图标和九线路双列悬停提示均符合预期，浏览器错误日志为 0。
- `packaging/kejilion-app/kpanel.conf` 与 `kejilion/apps` 远端 `origin/main` 的 Git blob
  `0a603abfe77beb045c4e7648dd60f5e4a1876e4d` 完全一致，应用市场配置无需新提交。

## 154 生产上线

- 升级前 Panel 为 `0.44.4/ok`，容器 healthy、重启 0、OOM=false，Agent active、重启 0；回滚镜像摘要为
  `sha256:3fe4b02713514e3ccbde01f14c60423a60088621fb27ffbb28e95def0b4c46a8`。
- 一致性备份：`/root/kpanel-backups/v0.45.0-preupgrade-20260805T073037Z`，目录权限 0700、文件权限 0600。
  SQLite 在线备份 122,880 B，`integrity_check=ok`，SHA-256 为
  `5b05d3bfb21d628de4ab2fcba415961099faaf686358dd969705e8db014e3220`；完整 KPanel 归档 20,803,556 B，
  SHA-256 为 `b111b67d7191105a684f15fecf808d44cf900b5b9615925a41be06e71a539426`。
- 使用 `KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update k app kpanel` 完成标准应用市场更新，实际拉取正式摘要
  `sha256:6676752672d069e0c5659b31d4960b4bf11415e5d4e8d624fce6397dc0020d1c`，输出
  `KPanel 更新完成 / Update Complete`。
- 升级后 Panel 与 Agent 均为 `0.45.0 v1alpha1`；本机、公网 IP 和
  `https://kp.kejilion.pro/api/v1/health` 均返回 `0.45.0/ok`。容器保持 `65532:65532`、只读根文件系统、
  非特权、`cap-drop ALL`、256 MiB 和 128 PID。
- 升级后 `ai.db integrity_check=ok`，`ai.db` 与 `ai-secrets.key` 权限保持 0600。应用市场按既有约定仅把受管
  脚本的 `permission_granted` 改为 `true`；标准化还原该行后 SHA-256 与 Release 固定摘要完全一致。
- 约 2 分钟内连续 8 次采样全部为 healthy、Panel/Agent 重启 0、OOM=false；CPU 0.02%–0.05%，内存约
  72.42–74.82 MiB，PID 7，Panel/Agent 的 panic、fatal、error 计数均为 0。

## 回滚

- 公共镜像回滚点为 v0.44.4：
  `sha256:3fe4b02713514e3ccbde01f14c60423a60088621fb27ffbb28e95def0b4c46a8`。
- 现场回滚使用上述升级前归档与 SQLite 在线备份，成对恢复 KPanel 目录、Compose、Agent、受管脚本和旧镜像；
  不覆盖其他业务容器、网站、证书或数据库。
- 镜像回滚不会撤销管理员后续主动执行的 SSH 端口变更；如已变更端口，应使用该操作返回的配置备份路径恢复，
  并在切换会话前同时验证 `sshd -t`、真实监听和云安全组。
