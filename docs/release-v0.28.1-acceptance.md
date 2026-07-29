# KPanel v0.28.1 发布验收

- 发布日期：2026-07-30
- 发布提交：`c85e151296d4655789eebe234dc94925adc571af`
- 标签：`v0.28.1`
- 源码回滚点：`v0.28.0`
- 镜像回滚摘要：`sha256:e7f87d8e976a4341dba49e502a0f22f116c568daaab63c8238e9211bc3561bf4`

## 发布范围

本版本完善集群监控入口和主机列表体验：

- 左侧导航将“集群”移动到“体检”下方。
- 本机接入授权窗口显示并支持复制浏览器当前可访问的主机 URL。
- 添加主机时关闭浏览器账号密码自动填充，配对码仍按一次性凭据处理。
- 集群主机支持行列表与卡片两种布局，默认使用行列表并保存本机偏好。
- 增加对应前端回归测试和 `0.28.1` 版本元数据。

## 自动化与安全验收

Windows 前端验证：

- `npm ci`
- TypeScript 类型检查
- `17` 个测试文件、`112` 个测试全部通过
- 生产构建通过
- `npm audit --audit-level=high`：`0` 个高危漏洞
- `scripts/check-ecosystem-policy.sh`

隔离 Linux 验证：

- `go test ./...`
- `go vet ./...`
- `go test -race ./internal/cluster ./internal/panel`
- amd64/arm64 的 Panel、Agent 和 kpctl 构建
- `govulncheck v1.6.0 ./...`：无可达漏洞
- `deploy/tests/install-safety.sh`
- 候选镜像只读根、非 root、无 capability 的运行契约
- Trivy `0.72.0` 扫描候选镜像：`HIGH`/`CRITICAL` 为 `0`

前端构建体积：

- 主入口 JS gzip：约 `55.26 KiB`
- 集群页面 JS gzip：约 `9.09 KiB`

GitHub Actions：

- 候选 CI：<https://github.com/kejilion/KPanel/actions/runs/30473258917>
- 主线 CI：<https://github.com/kejilion/KPanel/actions/runs/30473422013>
- Release：<https://github.com/kejilion/KPanel/actions/runs/30473578006>

三个流程均成功。Release 流程额外通过：

- 版本和发布输入一致性验证
- Go 漏洞扫描与 Node 依赖审计
- `kejilion.sh` 应用生命周期验证
- amd64/arm64 Agent 构建
- 运行时镜像契约验证
- 双架构镜像推送、`latest` 提升和 GitHub Release 发布

## 公开产物

GitHub Release：

<https://github.com/kejilion/KPanel/releases/tag/v0.28.1>

附件：

- `kejilion-agent-linux-amd64`
- `kejilion-agent-linux-arm64`
- `kejilion-panel-deploy-0.28.1.tar.gz`
- `SHA256SUMS`
- `LICENSE`
- `THIRD_PARTY_NOTICES.md`

公开下载三个运行产物后执行 `sha256sum -c SHA256SUMS`，全部通过；amd64 Agent 返回：

```text
0.28.1 v1alpha1
```

生产镜像：

```text
docker.io/kjlion/kejilion-panel@sha256:3a152a99f4372c60b17b00bbc7ef50377f7f2558407bebcd0693a3535a6b2105
```

Docker Registry 原生清单验证：

- `0.28.1` 与 `latest` 指向相同 manifest digest。
- 包含 `linux/amd64` 和 `linux/arm64`。
- 额外 `unknown/unknown` 清单为构建证明。
- Docker Hub 网页标签 API 发布后仍短暂返回旧索引；Registry 原生查询与实际拉取均已返回新摘要，不影响 `docker pull`。

## 公开镜像实机验收

在 `arena-154` 使用公开 `0.28.1` 镜像、独立临时目录、网络和回环端口 `18085` 运行：

```text
packaging/tests/image-e2e.sh
image_e2e=pass
```

验证覆盖：

- 镜像版本和健康检查
- Bootstrap 和 Secure Cookie
- 可信 HTTPS 反向代理 Host
- 只读根、非 root、无 capability 的运行约束

验收后临时容器和网络无残留。生产 `kejilion-panel` 容器保持 `healthy`，`kejilion-agent.service` 保持 `active`，未执行生产更新或重启。

## 应用市场契约

`kejilion/apps` 当前 `kpanel.conf` 提交为：

```text
1695cacbdf7fb5193d506ad61708aa3292b91b4c
```

配置继续使用：

```text
docker.io/kjlion/kejilion-panel:latest
```

版本、Agent 和脚本契约均从镜像标签及 `/release` 产物读取。本版本没有改变部署参数或目录契约，因此不需要修改应用市场配置；用户可继续通过应用市场安全更新和失败回滚。

## 回滚

源码回滚：

```text
v0.28.0
```

镜像回滚使用不可变摘要：

```text
docker.io/kjlion/kejilion-panel@sha256:e7f87d8e976a4341dba49e502a0f22f116c568daaab63c8238e9211bc3561bf4
```

该摘要已通过 Docker Registry 原生清单复核，包含 amd64 和 arm64。回滚仅替换 Panel 镜像和对应 Agent，保留网站、应用、Docker、`/home/web`、集群主机和配对状态数据。
