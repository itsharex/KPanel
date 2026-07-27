# KPanel v0.20.7 系统维护状态验收

## 实机结论

- 154 测试机为 Debian 13，Agent `0.18.1`。
- 2026-07-27 的标准清理从 15:21:14 执行到 15:21:30；APT history 与 dpkg log
  均记录了真实的 `apt-get autoremove --purge` 和软件包移除。
- 2026-07-26 的系统更新执行了固定 `apt-get ... full-upgrade`，升级了 Docker、
  containerd 和 Python 等软件包。
- 154 没有后续版本增加的 systemd 对账逻辑，因此不会用旧启动快照覆盖 worker 终态。

## 根因与修复

1. 页面轮询读取到 `2%` 启动快照。
2. worker 完成并原子写入 `succeeded`，transient unit 随后被 `--collect` 回收。
3. 旧对账流程仍使用先前读取的启动快照，把 `LoadState=not-found` 解释为失败并覆盖成功。
4. v0.20.7 在写入推断终态前重新读取状态文件；worker 已推进或完成时必须保留新状态。
5. systemd 的默认 `Result=success / ExecMainStatus=0` 不能替代 worker 完成凭据。

## 验收

- [x] 并发回归测试模拟 worker 在 `systemctl show` 期间写入成功，成功凭据未被覆盖
- [x] 无 worker 凭据的已回收单元不能伪报成功
- [x] RHEL 9.8 / systemd 252 使用生产同款 transient unit 参数执行 DNF 缓存清理成功
- [x] Linux 全量 Go 测试与 `go vet ./...`
- [x] 前端类型检查、34 项测试与生产构建
- [x] `linux/amd64`、`linux/arm64` 双架构二进制构建
- [x] 应用安装、更新、卸载、失败清理与未托管保护生命周期
- [x] 功能分支 CI `30307955028`
- [x] main CI `30307958389`
- [x] Release `30308078908`
- [x] `kejilion/apps` 同步提交 `575e522`

## 发布证据

- GitHub Release：`https://github.com/kejilion/KPanel/releases/tag/v0.20.7`
- Docker `0.20.7` / `latest`：
  `sha256:fccb85d7b3daf209f88c81d87f73fb9240925326959099e2e1b8f951baabeb4e`
- 平台：`linux/amd64`、`linux/arm64`

## 待目标机复核

截图对应的 `223.26.59.216` 拒绝现有 SSH 公钥，因此未在该机读取日志或执行维护。
更新面板与 Agent 到 `0.20.7` 后，应分别执行一次缓存清理和系统更新，确认快速任务不再
从真实成功变为 `LoadState=not-found` 失败；任务完成消息应包含实际步骤数和耗时。
