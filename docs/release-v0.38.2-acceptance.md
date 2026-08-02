# KPanel v0.38.2 发布验收

发布日期：2026-08-02

## 发布范围

- Docker 容器、镜像、网络和存储卷的操作按钮在桌面端统一靠右排列，窄屏时自动换行。
- 网站环境管理首屏只等待环境摘要；版本目录和备份列表进入对应分区后按需加载，任务记录独立刷新。
- 保留现有中英文路由级资源，不增加常驻依赖、轮询频率、后端接口或持久化状态。

## 源码与自动化

- Docker 工具栏提交：`5d718bf`
- 环境管理加载提交：`f12ed76`
- 发布准备提交：`809391851bf2b8c78250e277904aaed7c4cb9920`
- 标签：`v0.38.2`
- 主线 CI：<https://github.com/kejilion/KPanel/actions/runs/30746808577> — 成功
- Release：<https://github.com/kejilion/KPanel/actions/runs/30746884322> — 成功
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.38.2> — 已公开

## 功能、安全与性能验收

- Web 定向测试：Docker 工具栏和环境管理 2 个文件、5 项测试通过。
- Web 全量测试：32 个文件、214 项测试通过；类型检查、1411 条多语言资源完整性和生产构建通过。
- 环境管理首屏测试确认不会请求版本目录和备份列表；备份接口失败不阻断环境摘要和其他管理功能。
- npm 高危依赖审计为 0；Linux Release 门禁完成 Go 全量测试、`go vet`、`govulncheck`、部署安全测试、脚本生命周期、双架构构建和镜像运行契约验证。
- 本次没有新增网络入口、权限、宿主机写入或敏感数据处理，真实状态来源和安全边界不变。

## 发布产物

- `kejilion-agent-linux-amd64`：`4cea13c8fc1282850b57cfa6e8bdf51d030db041497672ec8f83f096c381daf7`
- `kejilion-agent-linux-arm64`：`750acd7cedd1e90ac8baedbd2e4002882bbaf2ab9b9e862f27c3c7a058629b0e`
- `kejilion-node-linux-amd64`：`b0928c6dba914e5f837bff3021fe52431165c53464c8b66565011636ca75802f`
- `kejilion-node-linux-arm64`：`d3ed4cb7b6fd232d2577ca110b881250ec189ef60600998c2ad280407428a36d`
- `kejilion-panel-deploy-0.38.2.tar.gz`：`b140dffcc98151ea5a18ecdce2f65d37f03c34dd71cedfa9714e54542810702b`

## Docker Hub

- `docker.io/kjlion/kejilion-panel:0.38.2`
- `docker.io/kjlion/kejilion-panel:latest`
- 两个标签经 Docker Hub API 回读确认指向同一 OCI index：
  `sha256:7237d7a181c7be2c3583c0abb9db44972c3702097b6ebd9279a8514665cd88ff`
- linux/amd64：`sha256:56e5d6b72c347604e6595271eaff7f2e6478d61d8dfb7016d862697e7f671c8b`
- linux/arm64：`sha256:b1585659499a079eb918065498b073d5c6bebe1b0405174913b397e665f0b391`

清单中的两个 `unknown/unknown` 项为 provenance/SBOM attestation，不是缺失架构。

## 154 隔离实机与回滚

- 154 验收机从 Docker Hub 拉取上述不可变摘要并执行隔离运行时 E2E，输出 `image_e2e=pass`。
- 验收后生产 KPanel 仍为 `running/healthy`，`kejilion-agent.service` 为 `active`；临时容器、网络和目录均已清理。
- 本轮没有替换或重启 154 的生产 KPanel；生产实例继续使用原版本，用户可按应用市场更新流程选择升级。
- 无数据、配置、端口、脚本协议或持久化格式迁移；回滚源码和镜像为 `v0.38.1`，镜像摘要为
  `sha256:2986324461f6919806f0aabba1dacfc42ee98bba01ee114da451842ee9e6f4ed`。
