# KPanel v0.31.1 发布验收

## 范围

- 文件选中后的批量操作栏覆盖现有路径工具栏，不再插入新行推动文件列表。
- 保持文件列表位置稳定，避免首次点击造成条目位移并打断双击打开目录。
- 本次不修改后端协议、Agent、数据结构、文件权限或部署配置。

## 验收结果

- [x] Web 类型检查、155 项单元测试和生产构建。
- [x] 生态规则检查。
- [x] Linux CI 全量源码验证、Go 测试与 vet、部署安装安全测试。
- [x] Go 已知漏洞扫描与 Node 高危依赖审计。
- [x] kejilion.sh 应用生命周期测试。
- [x] Linux amd64/arm64 镜像构建、运行时健康检查与镜像契约检查。
- [x] GitHub Release、版本镜像和 `latest` 镜像发布。

## 发布证据

- 发布提交：`46a97c9dbe21eef19a48a741990840528f204b26`
- 主线 CI：<https://github.com/kejilion/KPanel/actions/runs/30592457208>
- Release：<https://github.com/kejilion/KPanel/actions/runs/30592574271>
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.31.1>
- Docker Hub `0.31.1` / `latest`：
  `sha256:3048ee2c4b693261107a1c739239e7a23f4d7313c5136aee85b28c846190bfbd`
- 平台：`linux/amd64`、`linux/arm64`

## 性能、安全与兼容结论

- 只改变前端布局方式，不增加请求、轮询、缓存、日志、CPU、内存或磁盘占用。
- 批量操作按钮和原有权限边界不变，不新增网络入口或不可信输入路径。
- 浏览器刷新或回滚到 `v0.31.0` 不涉及数据迁移，不影响 `/home` 文件和现有任务。

## 回滚

- 上一稳定版本：`v0.31.0`
- 上一稳定镜像：
  `docker.io/kjlion/kejilion-panel@sha256:1d9a73d105a11059e6b909433a1927fe6400ca4198c7211faa0786205ef45c0a`
