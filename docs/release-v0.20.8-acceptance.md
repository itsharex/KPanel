# KPanel v0.20.8 DNS 与后台维护隔离验收

## 根因与修复

1. `kejilion.sh` 的启动初始化会改写 `/root/kejilion.sh` 和 Shell 配置；Agent 的
   `ProtectHome=read-only` 正确阻止了写入，但旧 DNS 固定协议没有跳过这些交互副作用。
2. DNS 静态事务使用 `cp -p` 保存 `/etc/resolv.conf`，Agent 的能力集缺少保留属主所需的
   `CAP_CHOWN`。
3. 应用市场安装将维护状态放在 `/home/docker/kpanel/data/agent`，维护 transient unit
   启用了 `ProtectHome=read-only` 却没有放行自己的状态目录，因此实际任务结束后无法写入
   完成凭据。154 使用 `/var/lib/...` 状态目录，所以没有触发该差异。
4. v0.20.8 的 KPanel 固定协议跳过脚本交互初始化，拒绝没有隔离入口的旧脚本；Agent 只增加
   `CAP_CHOWN`，维护与交换空间单元只增加自己的 `ReadWritePaths`，没有放宽其他宿主机路径。

## 验收

- [x] `kejilion/sh` 非交互协议在只读根文件系统中不再尝试写入 `/root`
- [x] WSL systemd 实测 `ProtectHome=read-only` 下仅放行 Agent 状态目录可写
- [x] WSL systemd 实测受限组身份加入 `CAP_CHOWN` 后 `cp -p` 正确保留 `root:root`
- [x] Linux 全量 Go 测试与 `go vet ./...`
- [x] 前端类型检查、34 项测试与生产构建
- [x] `linux/amd64`、`linux/arm64` Agent 构建
- [x] 应用安装、更新、卸载、失败清理与未托管保护生命周期
- [x] 安装安全与生态同源策略检查
- [x] 本地生产镜像构建、只读运行时健康检查与固定脚本摘要校验
- [x] 功能分支 CI `30317413880`
- [x] main CI `30317417820`
- [x] Release `30317532501`
- [x] `kejilion/sh` 固定提交 `a06bbc0662a426841c28bef9d692c12f13cc1d9d`
- [x] `kejilion/apps` 同步提交 `81c712e7598553517e25ebf09c76c204bbcd934a`

## 发布证据

- GitHub Release：`https://github.com/kejilion/KPanel/releases/tag/v0.20.8`
- Docker `0.20.8` / `latest`：
  `sha256:7e3139158a4c6b70180e75129896ce67dcc60f3aa20ef00f560c556031090028`
- 平台：`linux/amd64`、`linux/arm64`
- 应用市场在线配置 SHA-256：
  `cb60b6b8b1fc1d49bf2b1a40e2b2a240d25aaf4c27d3590d72cbf521ebaab1e9`

## 目标机复核

在应用市场更新 KPanel 到 `0.20.8` 后，Agent 与固定 `kejilion.sh` 会一起更新。目标机应重新
执行一次 DNS 设置、缓存清理和系统更新；旧的失败记录仍会保留，但新任务应写入完成凭据。
