# KPanel v0.22.0 验收记录

## 交付范围

- `kejilion.sh` LDNMP 环境兼容协议与原菜单兼容测试。
- KPanel 网站二级导航、LDNMP 环境总览、防护、优化、更新、备份还原和卸载。
- systemd 后台任务、可交互 ANSI PTY、完成凭据、资源版本、共享锁和敏感信息隔离。
- 冷备、传统备份发现、安全扫描、原子还原与失败回滚。

## 已完成的发布前验证

- Shell：`bash -n kejilion.sh`、LDNMP 协议 smoke、原网站非交互 smoke。
- Go（Linux）：`go test ./...`、`go vet ./...`。
- Agent：Linux `amd64`、`arm64` 静态编译。
- Web：TypeScript 检查、36 项 Vitest、生产构建。
- 部署：`deploy/tests/install-safety.sh`、生态规则检查。
- 页面：本地真实浏览器检查总览、防护、任务横幅、ANSI 颜色、弹窗关闭和全屏终端。
- 固定资源：脚本提交 `ded5af10d04ba4f9b39ca324ae662c126c364481`；
  SHA-256 `5632ee311a573b22a9c4f7fc7488ef1998119dc8225e5a583035cc4a1a5c627f`。

## 发布流水线待核查

- GitHub CI 的 Linux 变更感知验证与容器化脚本生命周期测试。
- Release 的原生镜像契约、双架构镜像、Docker Hub `0.22.0`/`latest` 和 GitHub Release。
- 独立 Debian、Ubuntu、Rocky/AlmaLinux 主机上的破坏性安装、还原、卸载矩阵。

未完成上述在线结果核查前，不将 v0.22.0 标记为完整 L3 验收通过。

## 回滚

KPanel 回滚到 `v0.21.0` / `4b41740`；`/home/web` 与 `/home/web_*.tar.gz` 保持不动。
脚本和 KPanel 独立提交，回滚 KPanel 不改变现有 LDNMP 业务产物。
