# KPanel v0.38.3 发布验收

发布日期：2026-08-02

## 发布范围

- 修复宿主机历史监控图表在宽屏和左右边缘处的悬停提示定位偏差。
- 宿主机与 Docker 图表统一依据 SVG 实际显示坐标定位提示框，并自动避让当前刻度线。
- 发布门禁增加核心特权包竞态检测、固定摘要 Trivy 源码与最终镜像扫描，以及受限容器运行验证。
- 统一版本一致性、隔离部署安全、变更感知核验和完整质量审计入口。
- 本次没有数据、配置、端口、脚本协议或持久化格式迁移。

## 源码与自动化

- 质量门禁提交：`0f0988f`
- 历史监控修复提交：`f1baad2`
- 发布准备提交：`4244abc2da0a54d09f59cd527c43633316f4c484`
- 标签：`v0.38.3`
- 候选分支 CI：<https://github.com/kejilion/KPanel/actions/runs/30751377702> — 成功
- 主线 CI：<https://github.com/kejilion/KPanel/actions/runs/30751896595> — 成功
- Release：<https://github.com/kejilion/KPanel/actions/runs/30752003677> — 成功
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.38.3> — 已公开，非预发布

## 功能、安全与性能验收

- 历史监控坐标定向测试 6 项通过；验证左右边缘避让、宽屏坐标换算和宿主机/Docker 共用定位契约。
- Web 全量测试 32 个文件、216 项通过；类型检查、1411 条多语言资源完整性和生产构建通过。
- Go 全量测试、核心特权包竞态检测、安装安全、脚本生命周期和双架构构建通过。
- `govulncheck` 未发现可达或已导入漏洞；npm audit 为 0；Trivy 源码和最终镜像 High/Critical 为 0。
- 候选镜像已验证可在非 root、只读根、`256 MiB`、`1 CPU`、`128 PID`、`cap-drop ALL` 和 `no-new-privileges` 条件下启动。

## 发布产物

- `kejilion-agent-linux-amd64`：`46651f2966cdc1f2a71105f3836d7cd2b25cfe4dd27ca362d83e82f1a9c31457`
- `kejilion-agent-linux-arm64`：`40a2a3f36a953b943ca24216c97fd3d1b8351068962b8da2e5db880459debe69`
- `kejilion-node-linux-amd64`：`5c3bf4413a3779b247bf4d0593eeae1079620e4ffb6d07dd8f9d21add051d238`
- `kejilion-node-linux-arm64`：`23f084166acaaa01b889af857d7533a33c376d414508aacbfb92709315a718f0`
- `kejilion-panel-deploy-0.38.3.tar.gz`：`371902ac79ad8e41d9e52bce569a32cf97aae69b51c6cfd1148f0e871a59ae40`

## Docker Hub

- `docker.io/kjlion/kejilion-panel:0.38.3`
- `docker.io/kjlion/kejilion-panel:latest`
- 两个标签指向同一 OCI index：`sha256:a4f88aafbb434460689035941d1056b76fb5cb40cf90b2c94fa82b228f4dcd3d`
- linux/amd64：`sha256:d9288ad83cb5d73378ea7ccebe2c7a10e6b6fb926323495303e3bed2f952f0cd`
- linux/arm64：`sha256:dc5f6d5a85775b0f3dc587e557193817755f5bac8041df064d919166428db15d`

清单中的 `unknown/unknown` 项为 provenance/SBOM attestation，不是缺失架构。

## 154 隔离实机、应用市场与回滚

- 154 验收机从 Docker Hub 重新拉取 `0.38.3` 公共镜像并执行隔离运行时 E2E，输出 `image_e2e=pass`。
- 验收后生产 KPanel 仍为 `running/healthy`，`kejilion-agent.service` 为 `active`；临时容器和网络已清理。
- 本轮没有替换或重启 154 的生产 KPanel，用户仍可通过应用市场自行选择升级时间。
- `packaging/kejilion-app/kpanel.conf` 相对 `v0.38.2` 无变化；继续使用 `latest` 和镜像内版本契约，无需提交 `kejilion/apps`。
- 源码和镜像回滚点为 `v0.38.2`，提交 `fc89ee94d03a324ac325516c593c97cc9b9a1435`。
