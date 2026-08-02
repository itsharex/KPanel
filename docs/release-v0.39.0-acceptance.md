# KPanel v0.39.0 发布验收

发布日期：2026-08-03

## 发布范围

- 新增轻量多主机终端：支持本机与重新授予终端权限的 v2 KPanel 主机，不新增公网监听端口。
- 终端支持多主机页签、预输入整行发送、原生交互输入、窗口缩放、断线续读和安全 URL 跳转。
- 历史监控新增中国电信、联通、移动在北京、上海、广州的九路 DNS 延迟趋势，并支持逐路显示或隐藏。
- 本次没有数据、端口、部署配置或脚本协议迁移；远端终端权限不向旧配对自动扩展。

## 源码与自动化

- 多主机终端提交：`8f336f8`
- 三网延迟历史提交：`3e73b27`
- 发布准备提交：`c72d19609411ff50c5dde5195eaddfac57a770e7`
- 标签：`v0.39.0`
- 候选分支 CI：<https://github.com/kejilion/KPanel/actions/runs/30763832391> — 成功
- 主线 CI：<https://github.com/kejilion/KPanel/actions/runs/30763937811> — 成功
- Release：<https://github.com/kejilion/KPanel/actions/runs/30764042714> — 成功
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.39.0> — 已公开，非预发布

## 功能、安全与性能验收

- Linux Go 全量测试通过；`panel`、`auth`、`dockerx`、`terminal`、`cluster`、`monitoring` 竞态检测通过。
- Web 全量测试 33 个文件、218 项通过；类型检查、1447 条多语言资源完整性和生产构建通过。
- `amd64`、`arm64` 的 Panel、Agent、轻量节点和 `kpctl` 构建通过；应用市场生命周期输出 `app_conf_lifecycle=pass`。
- `govulncheck v1.6.0` 未发现可达漏洞；npm audit 为 0；固定摘要 Trivy 对源码、Secret、Dockerfile 和最终镜像的 High/Critical 检查均为 0。
- 三网延迟使用固定 DNS 端点、3 个工作线程和 5 分钟采样间隔；失败样本记录为缺失，不增加 `CAP_NET_RAW`。
- 七天历史监控基准为 `36.469266 ms/op`、`8,432,847 B/op`、`80,788 allocs/op`；最大每日落盘投影 `3,194,208` 字节，最大七天响应 `4,869,233` 字节，均在既有预算内。
- 终端会话、单用户并发、输入、输出、内存缓冲、空闲时间和最长生命周期均有界；OSC 8 任意链接和 OSC 52 剪贴板控制被阻止，审计不保存命令及终端内容。

## 发布产物

- `kejilion-agent-linux-amd64`：`96fe7f21544766f1c198c7c1f683b2c637c8f1ab2b195496ae44d08005ef5ae7`
- `kejilion-agent-linux-arm64`：`f3346f461b722e001c42c9ab04a48a4a7f1e68f4176e493e54e6d9a921e8b2e5`
- `kejilion-node-linux-amd64`：`28009821b07d93ccb757f588a28776fdcb265e029eaed5d2a9faecc15c78b1a7`
- `kejilion-node-linux-arm64`：`a9ba98923687144c60dfefe4bf500ab4f73829f5bd1b7863e41ef9bbed662e4a`
- `kejilion-panel-deploy-0.39.0.tar.gz`：`5e63c556a7e4e685a83196e5c27fa8786a854fb040d0e632f7c1cac05d3bc2e2`

## Docker Hub

- `docker.io/kjlion/kejilion-panel:0.39.0`
- `docker.io/kjlion/kejilion-panel:latest`
- 两个标签指向同一 OCI index：`sha256:dcc8b66fd893000610326eed346f88fdacd2a285fd5bf4185a2f61b735abfef9`
- linux/amd64：`sha256:8a15a1347508740b0b53e4dc8c32a190934074e10bcce5045f58e49c2f16db37`
- linux/arm64：`sha256:abcf148f2d35550c798c8e1c85324de1abc391b4a3fd6ab626c3177fa38d85af`

清单中的 `unknown/unknown` 项为 provenance/SBOM attestation，不是缺失架构。

## 154 隔离实机、应用市场与回滚

- 154 验收机从 Docker Hub 重新拉取上述不可变摘要并执行隔离运行时 E2E，输出 `image_e2e=pass`。
- 验收前后生产 KPanel 均为 `running/healthy`，`kejilion-agent.service` 均为 `active`；没有替换或重启生产 KPanel。
- `packaging/kejilion-app/kpanel.conf` 相对 `v0.38.3` 无变化，继续使用 `latest` 与镜像内版本契约，无需提交 `kejilion/apps`。
- 代码合并前回滚点为 `0dee3cef7b98c0065776d72db9b6fad6791f8724`；公开镜像和源码回滚点为 `v0.38.3`（`4244abc2da0a54d09f59cd527c43633316f4c484`）。
- 回滚不会删除现有网站、应用、Docker、文件、集群关系或监控历史；旧版本忽略新终端 scope，远端终端需双方重新升级和配对后恢复。
