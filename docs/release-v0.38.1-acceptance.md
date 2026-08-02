# KPanel v0.38.1 发布验收

发布日期：2026-08-02

## 发布范围

- 修复 `kejilion.sh` 内部 `systemctl` 包装器截断轻量节点服务参数的问题。
- 修复部分 locale 下主机名清洗触发 `tr` 反向范围错误的问题。
- 新授权继续保持 5 分钟有效期，同时兼容滚动升级期间旧中心签发的 30 分钟授权。
- 接入成功但服务启用失败时保留节点凭据，重新执行命令可续装且不重复消费一次性授权。

## 源码与自动化

- `kejilion.sh` 修复提交：`3f91034c50158d701132c4adce2fea35802b50e9`
- 根脚本 SHA-256：`8e9f2f1e367a71bc0e97be0c901727522d853eb82b105442d8424cbba2d24fbc`
- KPanel 兼容修复提交：`51495409fa8b51862dd6262332164c55f3ee260c`
- 发布准备提交：`77a2ae81b32d7b430d3708d5c2ae28452f3d3ca2`
- 标签：`v0.38.1`
- 主线 CI：<https://github.com/kejilion/KPanel/actions/runs/30745185361> — 成功
- Release：<https://github.com/kejilion/KPanel/actions/runs/30745429432> — 成功
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.38.1> — 已公开

## 功能与安全验收

- Shell：根脚本与中文脚本通过语法检查、安装器 smoke test 和首次启动失败后续装测试；模拟接入计数确认授权只消费一次。
- Go：服务端和轻量节点客户端覆盖 5 分钟新授权、30 分钟旧授权兼容以及超过 1 小时拒绝测试。
- Linux CI：完成全量测试、`go vet`、漏洞扫描、双架构构建、部署生命周期和镜像运行契约检查。
- 安全边界：仍要求一次性授权、目标中心绑定和可信 HTTPS；没有增加入站端口、远程命令或节点权限。

## 发布产物

- `kejilion-agent-linux-amd64`：`709c1828b14e4fd3ebad5fc0a69e882d3342d97a64a061579bfe7c1917aff556`
- `kejilion-agent-linux-arm64`：`5f7d352d4449781000d38b4a0501d46a0986cedf46d7939486f6c898e3ecefe2`
- `kejilion-node-linux-amd64`：`4aff784b92dccaff007cc3a46f691d9d4c0fa1753e01299559d5c0510fa48fae`
- `kejilion-node-linux-arm64`：`5c21d7349f0f582db7a6dd8ea44c7e4a67539f1a5d0c272455ab01e5221d1433`
- `kejilion-panel-deploy-0.38.1.tar.gz`：`b3157deabe47a5464af6453368971730b11dbce1afcb953ed102b211643641a9`

## Docker Hub

- `docker.io/kjlion/kejilion-panel:0.38.1`
- `docker.io/kjlion/kejilion-panel:latest`
- 两个标签经 Docker Registry 回读确认指向同一 OCI index：
  `sha256:2986324461f6919806f0aabba1dacfc42ee98bba01ee114da451842ee9e6f4ed`

## 兼容、实机复核与回滚

- 兼容现有 KPanel 数据、端口、反向代理、集群配置和已接入节点，无需迁移。
- 用户已公开在会话中的授权必须作废并重新生成；正式实机闭环以新授权再次接入成功为最终确认。
- 回滚源码、标签和镜像：`v0.38.0`。
- 回滚仅切换 KPanel 镜像和程序版本，不删除用户数据或节点身份。
