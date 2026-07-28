# KPanel v0.23.0 验收记录

## 交付范围

- 应用市场对未进入统一管理函数的脚本应用开放原生交互终端。
- 交互终端降低键盘输入延迟，并为大段 UTF-8 粘贴提供有界 FIFO 写入。
- 概览基础系统设置增加 SSH 防御开关，读取并调用可信 `kejilion.sh`
  的 `k f2b status|enable|disable` 固定协议。
- SSH 防御使用独立 systemd 后台任务，关闭页面不会中断；关闭防御时保留
  Fail2Ban 软件、配置和日志。

## 发布前验收

- `kejilion.sh`：Shell 语法及 6 组 KPanel 协议 smoke。
- Go：Linux `go test ./...`、`go vet ./...`。
- Agent：Linux amd64、arm64 静态编译。
- Web：TypeScript 检查、40 项 Vitest、生产构建。
- 部署：生态规则、全部 Shell 语法、安装安全测试、容器内应用生命周期测试。
- 固定脚本：提交 `8ad705bc48f56eda2ce5b39ca324ae662c126c364481`；
  SHA-256 `11891afcc2a985383899d9632d2258bbf46ccfb68fdabe5bad745683ce5cae43`。

## 线上核查（2026-07-28）

- 功能分支 CI `30331625933`、主分支 CI `30331715350` 和 Release
  `30331835729` 全部成功。
- GitHub Release `v0.23.0` 已发布且不是草稿，包含 amd64/arm64 Agent、
  部署包和 `SHA256SUMS`。
- Docker Hub `0.23.0` 与 `latest` 均指向
  `sha256:e26a35860c36acb662eca918d20e4c0758875f4b1d285f04c9e8413316bd6414`，
  并包含 linux/amd64 与 linux/arm64。
- 从 Docker Hub 重新拉取 linux/amd64 镜像后，OCI 版本 `0.23.0`、源码提交
  `d7b3ca6bbb05e46ca58a20d243e40412ed10e59c`、脚本提交与脚本 SHA-256
  均与发布记录一致；非 root 用户及只读运行时健康检查通过。
- 发布后发现 `kejilion/apps` 仍硬编码上一版 Agent 版本和脚本摘要；应用市场提交
  `4892d75913d7d74c6dbaf3a248a5be0b66a08838` 已改为读取并交叉校验镜像自描述契约。
  真实 `latest` 镜像提取测试及版本、摘要不一致拒绝测试均通过。

## 回滚

- KPanel 回滚到 `v0.22.0` / `ac34272`。
- `kejilion.sh` 回滚到 `ded5af1`。
- 回滚 KPanel/Agent 不删除 `/home/web`、`/home/web_*.tar.gz` 或 Fail2Ban 配置。
