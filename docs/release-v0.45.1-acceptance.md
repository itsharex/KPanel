# KPanel v0.45.1 发布验收

## 发布范围

本版本只上线终端交互增强：

1. 主机终端和应用交互终端新增右键复制、粘贴与全选菜单。
2. 两类终端统一增加回到顶部与全屏工具栏，支持浏览器原生全屏、视口回退和 `Esc` 退出。
3. 全屏状态收敛到实际终端组件，移除体检、环境任务和主机终端的重复外层控制。
4. 清理上一版本验收记录中的历史模块残留说明。

本次没有数据库、Compose、Agent 权限、`kejilion.sh`、应用市场配置或网络迁移。

## 版本与产物

- 功能提交：`13ef98b`。
- 发布提交：`8fe46c4eeba4fffa42d15d8715da554948c07802`。
- 标签：`v0.45.1`。
- Release 工作流：<https://github.com/kejilion/KPanel/actions/runs/30989946493>，结论为 `success`。
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.45.1>，已公开且不是 prerelease。
- Docker OCI index：`sha256:d1bf20d3644de84fd67332e60a2fdc474d53a9f276a55b356de5c2aeff4e110f`。
- linux/amd64：`sha256:e8c1a94c696330f80d83751a0b97797d36ea491cff6a18b92155ab871f474b6b`。
- linux/arm64：`sha256:13749e957ca556d4acf242196fa572392af906d098467c31fdf437a2bb1a2625`。

## 自动化、154 与浏览器验收

- 本地 `npm ci`、`npm audit`、46 个 Vitest 文件共 283 项测试、1,623 条本地化短语与 16 个目录、typecheck 和生产构建全部通过。
- 154 在精确发布提交运行 `VERIFY_LEVEL=release bash scripts/verify-change.sh`，完整 L3 返回 0；Go 全量测试、核心 race、`go vet`、`govulncheck`、npm audit、Trivy 源码和镜像扫描、双架构 Panel/Agent/Node/kpctl 构建全部通过。
- 隔离候选实例只读连接 154 Agent，真实主机终端验证了右键菜单禁用态、选区复制、受控文本粘贴、普通 `Ctrl+C` 语义、回到顶部、全屏进入和 `Esc` 退出；浏览器运行错误为 0。
- OSC 52 远端剪贴板写入仍被拦截，复制与粘贴只由用户手势触发，未增加后端命令入口。

## 154 生产上线

- 升级前 Panel 为 `0.45.0/ok`，Agent active、重启 0，回滚镜像摘要为 `sha256:6676752672d069e0c5659b31d4960b4bf11415e5d4e8d624fce6397dc0020d1c`。
- 一致性备份：`/root/kpanel-backups/v0.45.1-preupgrade-20260805T084522Z`，目录权限 0700、文件权限 0600。
  SQLite 在线备份 122,880 B，`integrity_check=ok`，SHA-256 为
  `5b05d3bfb21d628de4ab2fcba415961099faaf686358dd969705e8db014e3220`；完整 KPanel 归档 20,809,916 B，
  SHA-256 为 `dd111a1513c66904d6a999fd4775a529cfc96f7926505c6086b5894ea7abfa18`。
- 使用 `KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update k app kpanel` 完成标准应用市场更新，实际拉取摘要与 Release OCI index 完全一致。
- 升级后本机健康接口为 `0.45.1/ok`，Agent 日志确认版本 `0.45.1`；连续 8 次稳定采样全部通过，Panel healthy、重启 0、OOM=false，Agent active、重启 0。
- 容器继续使用 `65532:65532`、只读根文件系统、非特权、`cap-drop ALL`、256 MiB 和 128 PID；`ai.db integrity_check=ok`，数据库与密钥权限保持 0600。
- 镜像内 `kejilion.sh` SHA-256 为 `cf835ec01a10955e91d49a5653870a8cab33288b32ee5cab40405bd69ef77bf4`。受管脚本仅按既有约定保留首行 `permission_granted=true`；将该行标准化后摘要与镜像内脚本一致，脚本逻辑没有变化。
- 隔离候选容器、测试镜像、154 临时目录和本地 SSH 隧道均已清理。

## 公网入口待确认

`https://kp.kejilion.pro/api/v1/health` 在 154 升级后仍返回 `0.45.0/ok`，响应为动态生成且带 `cache-control: no-store`、`cf-cache-status: BYPASS`。154 本机没有该域名的 Nginx vhost，因此该地址不是 154 当前容器的直接反向代理。本次不把该公网地址计入 154 上线成功证据；如需同步该入口，必须先确认它的实际源站和发布授权。

## 回滚

- 公共镜像回滚点为 `v0.45.0` / `sha256:6676752672d069e0c5659b31d4960b4bf11415e5d4e8d624fce6397dc0020d1c`。
- 现场回滚使用升级前归档和 SQLite 在线备份成对恢复 `/home/docker/kpanel`，不覆盖其他业务容器、网站、证书或数据库。
- 本版本没有数据结构迁移，直接切回 `v0.45.0` 镜像时数据目录可保持不动。
