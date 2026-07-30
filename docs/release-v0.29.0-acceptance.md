# KPanel v0.29.0 发布验收

## 范围

- 概览增加 BBRv3 管理卡片，和普通 BBR 并排展示。
- 支持 BBRv3 状态读取、安装、更新、卸载及待重启提示。
- 建站任务启动后自动聚焦紧凑终端，任务结束后保留输出并由用户手动关闭。
- 固定 `kejilion/sh@4a6e5b5b6054fd4da47d0362bc759c43cf83c06a`，
  脚本 SHA-256 为
  `4291e382b9b6d728f9884bcb162de443262a5430af03b2a0f66c471723a1a085`。

## 自动验收

- [x] `kejilion.sh` 和 `cn/kejilion.sh` Shell 语法检查。
- [x] BBRv3 固定协议 smoke：状态、未知动作拒绝、原交互菜单保留。
- [x] Go 新增严格状态解析、可信脚本调用、固定 systemd 任务和 Panel 枚举校验测试。
- [x] Windows 可执行的 Go 编译、定向测试与 `go vet ./...`。
- [x] Web TypeScript、121 项单元测试和生产构建。
- [ ] Ubuntu CI 全量 `make verify-change`、`govulncheck` 与 `npm audit`。
- [ ] Release 工作流镜像契约、安装生命周期、多架构镜像与 Docker Hub 摘要。

## 安全与兼容结论

- Web 与 Agent 不接受任意命令、包名、URL 或路径；只接受三种 BBRv3 写动作。
- 安装业务继续由 `kejilion.sh` 负责，KPanel 只负责可信协议、后台任务和真实状态展示。
- BBRv3 与系统更新、系统清理、SSH 防御共享维护锁，不并发修改宿主机软件包。
- 任务不会自动重启服务器。ARM64 原外部安装器未固定摘要，因此不从 Web 执行。
- 原 `k bbrv3`、普通 BBR、应用、建站、DNS、SSH 防御和 LDNMP 协议保持兼容。

## 回滚

- 回滚点：`v0.28.5` / `03d47f1`。
- 回滚 KPanel 镜像不会卸载已安装内核，也不会修改 `/home/web`、Docker、站点或数据库。
- 若 BBRv3 安装后尚未重启，回滚面板不影响系统继续使用当前运行内核。
