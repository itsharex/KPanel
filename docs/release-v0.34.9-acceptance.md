# KPanel v0.34.9 发布验收

发布日期：2026-08-01

## 发布范围

- 文件管理新增列表/大图标两种视图，并记住用户选择。
- 大图标视图对 JPEG、PNG、GIF 生成真实缩略图；其他文件继续使用轻量类型图标。
- 批量选择新增“反选”，仅作用于当前可见条目，避免误操作隐藏文件。
- 文件、代码、数据库、演示文稿、压缩包、密钥与配置文件使用更明确的类型图标和颜色。

## 源码与自动化

- 大图标与缩略图提交：`778cd765016630505604e82a54d30c3af5c6e884`
- 选择与文件图标提交：`0b63d3ad491d67fecac0293b8f431da05175cc22`
- 发布提交：`b7c25f4e7dc5b82779a0eedd358ffe8ff785c7c8`
- 标签：`v0.34.9`
- 候选 CI：[30702954425](https://github.com/kejilion/KPanel/actions/runs/30702954425) — 成功
- 主线 CI：[30703030376](https://github.com/kejilion/KPanel/actions/runs/30703030376) — 成功
- Release：[30703104044](https://github.com/kejilion/KPanel/actions/runs/30703104044) — 成功
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.34.9>

GitHub Release 已公开并包含 6 个附件：amd64/arm64 Agent、部署归档、`SHA256SUMS`、许可证和第三方声明。

## 功能、安全与性能验收

- Web：28 个测试文件、194 项测试通过；类型检查、生产构建、预压缩和 npm 高危依赖审计通过。
- Go：全量测试、`go vet`、`govulncheck` 及 amd64/arm64 的 Panel、Agent、`kpctl` 构建通过。
- 部署：Shell 语法、生态规则、安装安全测试、`kejilion.sh` 应用生命周期和镜像运行契约检查通过。
- 浏览器实机：在 154 隔离候选环境验证列表/大图标切换、偏好持久化、目录双击、真实 PNG 缩略图、文件类型图标、单选和反选；浏览器控制台无项目错误。
- 缩略图仅支持 JPEG、PNG、GIF；源文件最大 12 MiB、像素最大 800 万、输出最大 320×210，并发上限 2、超时 20 秒，不写磁盘缓存，前端按视口懒加载。
- 缩略图接口复用登录鉴权、路径边界和符号链接校验；不接受任意输出路径，不新增后台轮询、数据库迁移或长期缓存。

## Docker Hub

- `docker.io/kjlion/kejilion-panel:0.34.9`
- `docker.io/kjlion/kejilion-panel:latest`
- 两个标签的 OCI index 摘要一致：
  `sha256:70ad6be0e8602e3f49bcc71905bfe763f5b4ed71acd6b66c6c696310a7fd9037`
- linux/amd64：
  `sha256:817f126c516c7b8d77d21e7c713eb984aa38427f2658a14b18292393454156d7`
- linux/arm64：
  `sha256:94413b6d61702709c7c650b65ba3d568de16da67a12890b27dd5ba9dc82a9a54`

清单中的 `unknown/unknown` 项为 provenance/SBOM attestation，不是缺失架构。OCI 标签已核对版本 `0.34.9`、源码修订 `b7c25f4e7dc5b82779a0eedd358ffe8ff785c7c8` 和 AGPL-3.0-only 许可。

## 实机与应用市场兼容性

- 154 验收机从 Docker Hub 拉取上述不可变摘要并执行隔离运行时 E2E，输出 `image_e2e=pass`。
- 验收前后生产 `kejilion-panel` 均为 `running/healthy`，`kejilion-agent.service` 均为 `active`；候选容器、网络、Agent、测试目录和 SSH 隧道已清理，未替换或重启生产 KPanel。
- 本次未修改 `kejilion.sh`、Compose 或 `packaging/kejilion-app/kpanel.conf`。应用市场继续使用 `latest`，无需修改或提交 `kejilion/apps`。
- 没有面板状态、用户文件、站点、容器或部署参数迁移；列表视图仍为原有行为。

## 回滚

- 源码：`v0.34.8`（`ecece66eac1eb473ff57f2d6ed263cae70ffdd09`）
- 镜像：`docker.io/kjlion/kejilion-panel:0.34.8`
- 镜像摘要：
  `sha256:028a2d5db4ace51b19d688233e8598154042e878eb5d3c0cafbddd2f003e2beb`

本次没有数据格式或配置迁移；回滚只恢复上一版文件管理界面与接口，不修改用户文件、容器、站点和面板状态。
