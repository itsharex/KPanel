# KPanel v0.34.0 发布验收

发布日期：2026-08-01

## 发布范围

- 增加可选登录安全入口，升级用户默认关闭；支持生成、复制、修改和轮换。
- 文件管理扩展到宿主机根目录，同时隔离 KPanel 数据、凭据目录和内核虚拟文件系统写操作。
- 侧栏版本区直接复用 KPanel 应用更新流程，更新后等待服务恢复并自动刷新。
- 容器监控列表高度与右侧趋势图区域对齐。
- Go 最低工具链提升到 `1.26.5`，修复旧标准库中已可达的安全漏洞。

## 源码与自动化

- 发布提交：`1dfde09bf70446208fe9ef65f9924605dffe8fcb`
- 标签：`v0.34.0`
- 候选 CI：[30650445514](https://github.com/kejilion/KPanel/actions/runs/30650445514) — 成功
- 主线 CI：[30650620478](https://github.com/kejilion/KPanel/actions/runs/30650620478) — 成功
- Release：[30650805471](https://github.com/kejilion/KPanel/actions/runs/30650805471) — 成功
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.34.0>

Release 附件已确认包含：

- `kejilion-agent-linux-amd64`
- `kejilion-agent-linux-arm64`
- `kejilion-panel-deploy-0.34.0.tar.gz`
- `SHA256SUMS`
- `LICENSE`
- `THIRD_PARTY_NOTICES.md`

## L3 验收

- Go `1.26.5`：`go test ./...`、`go vet ./...` 通过。
- Web：26 个测试文件、174 个测试通过；类型检查和生产构建通过。
- `deploy/tests/install-safety.sh`：`install_safety=pass`。
- `packaging/tests/app-conf-lifecycle.sh`：`app_conf_lifecycle=pass`。
- linux/amd64、linux/arm64 的 Panel、Agent、`kpctl` 构建通过。
- 固定脚本提交和 SHA-256、OCI 版本、许可证、非 root 用户、健康检查契约核对通过。
- `govulncheck`：0 个可达漏洞；npm 官方源审计：0 个漏洞。
- 本地候选镜像及 Docker Hub 公开镜像分别执行 `packaging/tests/image-e2e.sh`，均输出
  `image_e2e=pass`。

## Docker Hub

- `docker.io/kjlion/kejilion-panel:0.34.0`
- `docker.io/kjlion/kejilion-panel:latest`
- 两个标签的 OCI index 摘要相同：
  `sha256:e245eb9dd7dfe8475b8924b28d871688cd3f68000a62d091835a73cd0262c60e`
- linux/amd64：
  `sha256:fd034ad54e071b2f1a33f917d412389faad23c496ec7e7c44898e869153c767d`
- linux/arm64：
  `sha256:fc3cbb855f1dca7706b068bd7a3d3033e2bf1451465acda122cf189800c81b43`

`unknown/unknown` 条目为每个平台对应的 provenance/SBOM attestation，不是缺失架构。

## 应用市场与兼容性

`packaging/kejilion-app/kpanel.conf` 自 `v0.33.2` 起没有安装、更新或卸载契约变化；当前配置
继续使用 `latest` 并从 OCI 标签及镜像内 `/release/VERSION` 动态校验版本，因此本次不需要
修改或提交 `kejilion/apps`。

安全入口对升级用户默认关闭；启用前不会改变现有登录 URL。全盘文件管理不改变 Panel 的
非特权容器身份，也不新增任意 Shell 或公网监听端口。

## 回滚

- 源码：`v0.33.2`（`aaf14baae14043ea2692eaeb5a72045108307278`）
- 镜像：`docker.io/kjlion/kejilion-panel:0.33.2`
- 回滚不会删除 `/home`、站点、Docker 资源、Panel 数据或文件回收站；新版本中启用的安全入口
  配置保留在 Panel 状态中，回滚后旧版本会忽略该可选字段。
