# KPanel v0.45.2 发布验收

## 发布范围

本版本只上线终端网页铺满调整：

1. 主机终端将会话标签和终端操作合并为同一工具栏，网页铺满时仍可切换会话、回到顶部和恢复窗口。
2. 主机终端和应用交互终端统一使用网页视口铺满，不再进入浏览器或操作系统原生全屏。
3. 切换会话和铺满状态后重新适配当前终端尺寸，关闭最后一个主机终端会话时恢复普通页面布局。

本次没有数据库、Compose、Agent 权限、`kejilion.sh`、应用市场配置或网络迁移。

## 版本与产物

- 功能提交：`37e893e`。
- 发布提交：`b4bec8c43ea4a587344ee3c34866a95a110103eb`。
- 标签：`v0.45.2`。
- Release 工作流：<https://github.com/kejilion/KPanel/actions/runs/30995206537>，结论为 `success`。
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.45.2>，已公开且不是 prerelease。
- Docker OCI index：`sha256:440585da65ce231fea54fed9739d5bba22ad478f9d046555ad74f0ce542a65a1`。
- linux/amd64：`sha256:2c58f337d736267861f923cf036c7ca872a52d2f4421042048edfa6276c4f582`。
- linux/arm64：`sha256:eb7192f55a2d7ca4a34729b9087edcafdc1781a114cb40ff03e0ba6d5fcd0d06`。

## 自动化、154 与浏览器验收

- 本地重新执行 `npm ci` 和 `npm audit`，依赖漏洞为 0；46 个 Vitest 文件共 282 项测试、1,623 条本地化短语与 16 个目录、typecheck 和生产构建全部通过。
- 154 在精确发布提交和基线 `3a7eafc` 上运行 `VERIFY_LEVEL=release bash scripts/verify-change.sh`，完整 L3 返回 0；Go 全量测试、核心 race、`go vet`、`govulncheck`、部署安装安全测试、npm audit、Trivy 源码和镜像扫描、双架构 Panel/Agent/Node/kpctl 构建全部通过。
- 隔离候选实例以只读根文件系统和非特权用户连接 154 Agent，健康接口返回 `0.45.2/ok`。真实主机终端验证了会话标签与工具栏同排、网页铺满、回到顶部、恢复窗口、`Esc` 退出和右键菜单；铺满期间标签、终端及恢复按钮保持可见，恢复后 `html/body` 无滚动锁残留，浏览器错误日志为 0。
- 应用交互终端的共享铺满逻辑通过组件布局测试和组合式函数测试；为避免在生产 Agent 上下载并执行第三方体检脚本，隔离浏览器未启动真实第三方交互任务。
- OSC 52 拦截、用户剪贴板手势、会话鉴权、Origin/CSRF、Agent 和后端命令边界均未改变。

## 154 生产上线

- 升级前 Panel 为 `0.45.1/ok`，容器 healthy、重启 0、OOM=false，Agent active、重启 0；回滚镜像摘要为 `sha256:d1bf20d3644de84fd67332e60a2fdc474d53a9f276a55b356de5c2aeff4e110f`。
- 一致性备份：`/root/kpanel-backups/v0.45.2-preupgrade-20260805T095947Z`，目录权限 0700、文件权限 0600。
  SQLite 在线备份 122,880 B，`integrity_check=ok`，SHA-256 为
  `5b05d3bfb21d628de4ab2fcba415961099faaf686358dd969705e8db014e3220`；完整 KPanel 归档 20,821,161 B，
  SHA-256 为 `45931662eddbcee6537b1f30653d6ba47ac1a1b2400b2c00793bf2200dbb6ffa`。
- 使用 `KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update k app kpanel` 完成标准应用市场更新，实际拉取摘要与 Release OCI index 完全一致。
- 升级后本机健康接口为 `0.45.2/ok`，Agent 日志确认版本 `0.45.2`；连续 8 次稳定采样全部通过，Panel healthy、重启 0、OOM=false，Agent active、重启 0，Panel 与 Agent 的 panic/fatal/error 计数均为 0。
- 容器继续使用 `65532:65532`、只读根文件系统、非特权、`cap-drop ALL`、256 MiB 和 128 PID；`ai.db integrity_check=ok`，数据库与密钥权限保持 0600。
- 镜像内 `kejilion.sh` SHA-256 为 `cf835ec01a10955e91d49a5653870a8cab33288b32ee5cab40405bd69ef77bf4`。受管脚本仅按既有约定保留首行 `permission_granted=true`；将该行标准化后摘要与镜像内脚本一致，脚本逻辑没有变化。
- 隔离候选容器、门禁镜像、154 临时目录和本地 SSH 隧道均已清理。

## 公网入口待同步

`https://kp.kejilion.pro/api/v1/health` 在本次 154 升级后返回 `0.45.1/ok`。该地址已经从上一轮记录的 `0.45.0` 独立更新，但 154 本机没有对应域名的 Nginx vhost，因此它不是 154 当前容器的直接反向代理。本次不把该公网地址计入 154 上线成功证据；同步 `v0.45.2` 前必须确认实际源站和发布授权。

## 回滚

- 公共镜像回滚点为 `v0.45.1` / `sha256:d1bf20d3644de84fd67332e60a2fdc474d53a9f276a55b356de5c2aeff4e110f`。
- 现场回滚使用升级前归档和 SQLite 在线备份成对恢复 `/home/docker/kpanel`，不覆盖其他业务容器、网站、证书或数据库。
- 本版本没有数据结构迁移，直接切回 `v0.45.1` 镜像时数据目录可保持不动。
