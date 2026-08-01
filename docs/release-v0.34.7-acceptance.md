# KPanel v0.34.7 发布验收

发布日期：2026-08-01

## 发布范围

- 文件管理支持批量压缩为 TAR.GZ、ZIP 或 TAR，默认推荐适合 Linux 服务器的 TAR.GZ。
- TAR.GZ、TGZ、ZIP 和 TAR 可从右键菜单解压到全新的文件夹，不覆盖已有目录或文件。
- 压缩和解压支持主动停止；页面关闭、网络断开或用户停止后，请求上下文会终止流式处理并清理临时产物。
- 将“先按 KPanel 规范独立设计，再复核 `kejilion.sh` 与竞品差距，最后开发、验收、提交和发布”的顺序固化到永久工程规范。

## 源码与自动化

- 功能提交：`3afbfd8dd3a3a0a88aa0924e80d52afbfe262ca8`
- 发布提交：`747528adcb70966d1b257109500ced7a4dd3d1d7`
- 标签：`v0.34.7`
- 候选 CI：[30693552002](https://github.com/kejilion/KPanel/actions/runs/30693552002) — 成功
- 主线 CI：[30693662299](https://github.com/kejilion/KPanel/actions/runs/30693662299) — 成功
- Release：[30693744379](https://github.com/kejilion/KPanel/actions/runs/30693744379) — 成功
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.34.7>

GitHub Release 已公开并包含 6 个附件：amd64/arm64 Agent、部署归档、`SHA256SUMS`、许可证和第三方声明。

## 功能、安全与性能验收

- Web：28 个测试文件、185 项测试通过；类型检查、生产构建和预压缩通过。
- Go：文件管理、Agent、Panel 文件动作测试通过；Linux CI 全量测试、Go vet、amd64/arm64 发布构建通过。
- 格式闭环：TAR.GZ、ZIP、TAR 的压缩/解压、单目录去除双层嵌套、多选顶层名称、空目录和修改时间保留均有自动测试。
- 攻击面：路径穿越、反斜杠路径、重复条目、符号链接、硬链接、设备/特殊文件、ZIP 中央目录超限、解压膨胀、目标冲突和取消清理均有拒绝或回滚测试。
- 发布工作流中的 `govulncheck`、npm 高危依赖审计、`kejilion.sh` 应用生命周期及镜像运行契约检查全部通过。
- 本地 Windows amd64 基准（8 MiB 固定样本、3 次迭代）为：TAR.GZ 压缩 332.51 MB/s、约 0.85 MiB 分配；解压 405.56 MB/s、约 0.16 MiB 分配。该数据用于确认固定缓冲和无全文件内存加载，不作为生产主机性能承诺。

## Docker Hub

- `docker.io/kjlion/kejilion-panel:0.34.7`
- `docker.io/kjlion/kejilion-panel:latest`
- 两个标签的 OCI index 摘要一致：
  `sha256:98c6ee8ed4a501d4cb0481f1a8d7ebbba078c61c90ed6e17b126d003c5383254`
- linux/amd64：
  `sha256:8caac283f2be2e2bcabd8d1731d2879f9f622bdd360d52bc62e964c530530b25`
- linux/arm64：
  `sha256:8659174d80ebc4069decf30dcc6da39afc9eef7373c5de227fdd80c0b7d56b48`

清单中的 `unknown/unknown` 项为 provenance/SBOM attestation，不是缺失架构。

## 实机与应用市场兼容性

- 154 验收机从 Docker Hub 拉取上述不可变摘要并执行隔离运行时 E2E，输出 `image_e2e=pass`。
- 验收后生产 `kejilion-panel` 仍为 `running/healthy`，`kejilion-agent.service` 为 `active`；临时容器和网络均为 0，未替换或重启生产 KPanel。
- 本次未修改 `kejilion.sh`、Compose 或 `packaging/kejilion-app/kpanel.conf`。应用市场继续使用 `latest`，并从 OCI 标签及镜像内 `/release/VERSION` 动态校验版本，无需修改或提交 `kejilion/apps`。
- 没有数据库、站点、Docker 业务资源、回收站格式或部署参数迁移。

## 回滚

- 源码：`v0.34.6`（`5a465bf479efed0936d6a0fa2325032799eab2e5`）
- 镜像：`docker.io/kjlion/kejilion-panel:0.34.6`
- 镜像摘要：
  `sha256:3c6656a21de1dfca5a949ca7ea34d64df387730836a285108ac7d9e6aa424682`

回滚不会修改或删除现有文件和归档；旧版仅不再显示压缩/解压操作，已有压缩包仍可作为普通文件下载和管理。
