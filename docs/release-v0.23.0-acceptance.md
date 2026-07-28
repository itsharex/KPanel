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

## 发布流水线核查

正式标签发布后核查：

- GitHub CI 与 Release 全部成功。
- Docker Hub `0.23.0` 与 `latest` 指向同一双架构镜像摘要。
- GitHub Release 已发布，附件及 `SHA256SUMS` 可下载。
- 镜像 OCI 版本、源码提交、脚本提交和脚本 SHA-256 与发布记录一致。

## 回滚

- KPanel 回滚到 `v0.22.0` / `ac34272`。
- `kejilion.sh` 回滚到 `ded5af1`。
- 回滚 KPanel/Agent 不删除 `/home/web`、`/home/web_*.tar.gz` 或 Fail2Ban 配置。
