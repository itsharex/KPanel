# KPanel v0.39.1 发布验收

发布日期：2026-08-03

## 发布范围

- 修复三网延迟在不同网络环境下全部超时、旧成功值误显示以及宽屏图表没有铺满的问题。
- 修复本机和远端终端轮询、Agent 重启后的会话收敛、输入保留与异常窗口尺寸问题。
- 本次没有数据、端口、部署配置或脚本协议迁移。

## 本地与 Linux 验收

- Web 全量测试 33 个文件、221 项通过；1447 条多语言资源检查、类型检查和生产构建通过。
- Windows Go 测试中，本次涉及的 `monitoring`、`panel` 与 `terminal` 测试通过；全量测试仍包含项目既有的 Linux 路径和 systemd 假设，不作为 Linux 发布结论。
- 154 隔离 Linux 环境通过 `go test ./...` 与 `go vet ./...`。
- 154 隔离真机三网采样连续两轮均为 9/9 成功；采样延迟约为 49–76 ms，具体数值仅代表验收时网络，不作为产品承诺。
- 154 浏览器真机终端显示“已加密连接”，执行 `echo KPANEL-TERM-UI-OK` 成功；重启隔离 Agent 后 6.5 秒内明确收敛为“会话已结束”，未出现无限重连。

## 安全与性能边界

- 九条线路使用内置固定端点；每个目标仅尝试非特权 ICMP、TCP/53 和 UDP DNS，不接受任意目标或任意命令。
- 最多 3 个目标工作线程、每目标 1.5 秒总超时、每 5 分钟一轮；最坏同时 9 个短生命周期套接字。
- 终端继续使用独立权限、登录态、Agent Unix Socket 和 Noise 加密通道；输入、输出、会话、窗口尺寸和重连速率均有界。

## 发布产物与线上复核

- 功能提交：`8bc15a6`；版本提交与 `v0.39.1` 标签：`c084080`。
- 候选分支 CI：[`30778931111`](https://github.com/kejilion/KPanel/actions/runs/30778931111)；主线 CI：[`30779057628`](https://github.com/kejilion/KPanel/actions/runs/30779057628)；Release：[`30779223572`](https://github.com/kejilion/KPanel/actions/runs/30779223572)，均成功。
- GitHub Release：[`v0.39.1`](https://github.com/kejilion/KPanel/releases/tag/v0.39.1)，非草稿、非预发布。
- Agent SHA-256：amd64 `9ee000dcddc51882639fd8e0068b07a87fdd8304dacab5e5ffb8b84f255e76fb`；arm64 `e7408051dd8ebcc8612a48cbab76d5f64209f94513d35ea5ecf8b8214c2c9275`。
- 轻量节点 SHA-256：amd64 `ed6bbe2c2248d711e10237f4f8f6da31bcde18f0f79e2772c2d92e20bd808d95`；arm64 `a9b359fc620059c384ee6e3393e41332ecbc948a12e624a3d649d55b63df7295`。
- 部署包 SHA-256：`6aa9edd6bc0fbeac78539c70acee8ba4af4afe51f0e9b3ab00ce75b8a7ef9f60`。
- `docker.io/kjlion/kejilion-panel:0.39.1` 与 `latest` 均指向 OCI 摘要 `sha256:2093416aabc2226a9b257106abfeec18c3a7720eb6f5798ea5af74f2882f5c5e`，包含 linux/amd64 与 linux/arm64。
- 154 从 Docker Hub 重新拉取公开镜像后，隔离 `image-e2e.sh` 验收通过；未修改 154 正式 KPanel 实例及业务数据。
- `packaging/kejilion-app/kpanel.conf` 相对 `v0.39.0` 未变化，应用市场更新契约无需同步修改。

回滚点为 `v0.39.0`；回滚只替换 Panel/Agent 版本，不删除现有业务数据。
