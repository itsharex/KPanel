# KPanel v0.20.5 建站任务持久化验收

## 问题结论

- `clear` 仅发送 ANSI 清屏序列，xterm.js 与建站 PTY 均支持，不会结束脚本。
- v0.20.4 的建站执行器仍由 Agent 主进程等待 `systemd-run --pipe`；Agent 短暂不可用或
  重启时，任务会被标记为 `interrupted`，前端又把一次 503 轮询错误直接显示为搭建失败。

## 修复验收

- [x] 建站改由 `systemd-run --no-block` 启动独立 `site-pty-run` worker
- [x] Agent 重启后按 systemd 单元恢复运行中任务
- [x] Agent 与 worker 通过原子任务文件同步实时状态，避免进程内旧状态
- [x] 502、503、504 与网络错误只进入自动重连，不产生虚假失败事件
- [x] PTY 原始 ANSI 颜色、清屏序列和任务 FIFO 输入保持可用
- [x] WordPress、IP+端口反代及固定成品站参数由持久任务重新校验后恢复
- [x] 相关 Go 测试、Go vet、前端 34 项测试、前端生产构建与 Linux 二进制构建
- [x] 安装安全测试和 KPanel 应用安装、更新、卸载、失败回滚、未托管保护全链路
- [x] 功能分支 CI `30288495116`
- [x] main CI `30288586781`
- [x] Release `30288717965`
- [x] `kejilion/apps` 同步提交 `cbddaeb`
- [ ] 目标服务器在 WordPress 安装中主动重启 Agent，并复核终端续传与最终站点产物

## 发布证据

- GitHub Release：`https://github.com/kejilion/KPanel/releases/tag/v0.20.5`
- Docker `0.20.5` / `latest` manifest：
  `sha256:4af36c7e0b3a8eee1ace02852568cd4abf224c0ea706ce6c2b44b4f757a5c3bc`
- 平台：`linux/amd64`、`linux/arm64`
- 资产：双架构 Agent、部署归档、`SHA256SUMS`

## 目标机复核

1. 更新到 v0.20.5，确认面板与 Agent 均显示 `0.20.5`。
2. 创建 WordPress，在安装依赖阶段执行 `systemctl restart kejilion-agent`。
3. 页面应显示“Agent 重连中”，不能显示“搭建失败”；重连后继续读取原任务。
4. 在终端执行脚本提示所需输入并确认 ANSI 颜色、`clear` 和滚动回放正常。
5. 完成后确认网站、证书、数据库和 `/home/web` 产物可被面板重新发现。
