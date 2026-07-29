# KPanel v0.27.0 发布验收

- 发布日期：2026-07-29
- 发布提交：`0b3f1e952fccc64006543acc225a2975bb0353d7`
- 功能基点：`23980020eb7c610a0698ab8f9fa35fdfe775cc42`
- 标签：`v0.27.0`
- 回滚版本：`v0.26.1`

## 发布范围

本版本增加 KPanel 集群主机看板和联邦只读监控：

- 当前 KPanel 自动作为本机节点展示；
- 每个节点均可同时作为中心端和被监控端；
- 支持公网 HTTPS，以及 Noise 端到端加密的 `http://IP:非80端口`；
- 展示 CPU、内存、磁盘、网络、系统、地区、延迟和在线状态；
- 保留既有 v1 HTTPS 联邦状态和凭据兼容；
- 标准部署增加独立 Panel 出站网络，不改变 Agent Unix Socket 与宿主机权限边界。

## 自动化验收

本地复核通过：

- `go test ./...`
- `go vet ./...`
- `go test -race ./internal/cluster ./internal/panel`
- `make build-linux`
- 前端 `typecheck`、90 个测试和生产构建
- `govulncheck`：无可达漏洞
- `npm audit --audit-level=high`：0 个漏洞
- `deploy/tests/install-safety.sh`
- `scripts/check-ecosystem-policy.sh`

构建体积符合当前预算：

- 主入口 JS gzip：`55.24 KiB`
- 集群路由 JS gzip：`8.39 KiB`

GitHub Actions：

- 候选 CI：
  <https://github.com/kejilion/KPanel/actions/runs/30461798009>
- 主线 CI：
  <https://github.com/kejilion/KPanel/actions/runs/30461992395>
- Release：
  <https://github.com/kejilion/KPanel/actions/runs/30462239206>

三个流程均成功；Release 流程额外通过：

- 发行输入与版本一致性；
- 源码、漏洞和前端依赖检查；
- `kejilion.sh` 应用生命周期容器测试；
- amd64/arm64 Agent 构建；
- 原生镜像非 root、只读根、无 capability 运行契约；
- 双架构镜像推送、`latest` 提升和 GitHub Release 公开。

## 公开产物

GitHub Release：

<https://github.com/kejilion/KPanel/releases/tag/v0.27.0>

附件：

- `kejilion-agent-linux-amd64`
- `kejilion-agent-linux-arm64`
- `kejilion-panel-deploy-0.27.0.tar.gz`
- `SHA256SUMS`
- `LICENSE`
- `THIRD_PARTY_NOTICES.md`

生产镜像：

```text
docker.io/kjlion/kejilion-panel@sha256:f9a8e7ef459fdda1be6890b93c6a5a3fc82de3bd41763abb1f8f50b338109984
```

Docker Registry 原生清单验证：

- `0.27.0` 与 `latest` 指向同一 manifest digest；
- 包含 `linux/amd64` 和 `linux/arm64`；
- 额外 `unknown/unknown` 清单为 SBOM/provenance attestation。

## 公开镜像实机验收

在 `arena-154` 使用公开 `0.27.0` 镜像运行
`packaging/tests/image-e2e.sh`，结果：

```text
image_e2e=pass
```

验收使用唯一临时容器、网络和回环端口 `18083`，覆盖：

- 镜像版本和健康检查；
- 可信 HTTPS 反向代理 Host；
- Bootstrap、Secure Cookie；
- 只读根、无 capability 的运行约束。

验收前后生产 KPanel 容器 ID 均为：

```text
ebb14cb7195060a3559597b96710c2da2a47938d3da464610c5dbdc0bf389305
```

生产 KPanel 保持 `running`，Agent 保持 `active`；临时容器、网络和脚本无残留。

## 应用市场契约

本版部署契约新增：

```text
KEJILION_PANEL_CLUSTER_PRIVATE_CIDRS
```

`kejilion/apps` 已同步 KPanel 配置：

- 提交：`914e226`
- 文件：`kpanel.conf`
- KPanel 仓库与应用市场配置已归一化一致

普通后续 KPanel 版本若部署契约不变，不需要修改应用市场配置。

## 回滚

源码回滚：

```text
v0.26.1
```

镜像回滚：

```text
docker.io/kjlion/kejilion-panel@sha256:07cd9bee0367effd279beaf923f986cd508e74cfe8efa2deb5f701338aef5752
```

回滚旧镜像时保留 v2 集群状态和凭据文件；旧版本会忽略这些文件，网站、应用、Docker、
`/home/web` 和 Agent 数据不受影响。回滚前仍应先在各节点撤销不再需要的控制端授权。
