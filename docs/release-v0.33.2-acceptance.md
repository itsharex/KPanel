# KPanel v0.33.2 发布验收

## 发布范围

- 历史趋势图按 SVG 实际屏幕坐标矩阵换算鼠标位置，宽屏宿主机图与容器图均可到达首尾采样点。
- 多曲线图使用全部曲线时间点作为悬停锚点，不再依赖第一条曲线是否具有对应采样。
- 不属于最新容器采样批次的序列以灰色“历史”状态展示，避免同名旧容器被误认为仍在运行。
- 容器图表统一为 CPU、内存、磁盘 I/O、网络顺序，与宿主机指标布局一致。

## 验收结果

- [x] Web 类型检查、168 项单元测试和生产构建通过。
- [x] 坐标换算、首尾时间锚点和历史容器判定单元测试通过。
- [x] 1920px 宽屏浏览器实测宿主机与容器图首尾游标分别落在 `x=64` 和 `x=708`，时间与首末采样一致。
- [x] 同名当前容器保持正常样式，旧容器灰化并显示“历史”；容器四图顺序完成视觉复核。
- [x] 生态规则、Go 全量测试与 `go vet`、`govulncheck`、Node 高危依赖审计、安装器、应用生命周期和镜像运行时契约通过。
- [x] GitHub Release、`linux/amd64`、`linux/arm64` 版本镜像和 `latest` 发布完成。

## 发布证据

- 功能提交：`0171a56`
- 发布提交：`aaf14ba`
- 功能分支 CI：https://github.com/kejilion/KPanel/actions/runs/30636147130
- 功能主线 CI：https://github.com/kejilion/KPanel/actions/runs/30636230699
- 版本主线 CI：https://github.com/kejilion/KPanel/actions/runs/30636426609
- Release：https://github.com/kejilion/KPanel/actions/runs/30636559915
- GitHub Release：https://github.com/kejilion/KPanel/releases/tag/v0.33.2
- Docker Hub `0.33.2` / `latest`：
  `sha256:1d61d36964761d09e21f759855b64b4c7fd86fa424625c9b8027ec8571029bda`
- 平台：`linux/amd64`、`linux/arm64`
- Release 附件：双架构 Agent、部署归档、`SHA256SUMS`、许可证与第三方声明均可下载。

## 兼容、观察与回滚

- 本次只修改前端展示和交互，不修改监控采样频率、存储格式、API、Agent 权限或 `kejilion.sh` 协议。
- 历史状态依据同一次查询中的最新采样批次判定，只用于展示数据新旧，不替代 Docker Engine 实时运行状态。
- 发布后 Docker Hub 公共目录 API 存在索引延迟；Release 流水线已直接通过 Registry 校验版本标签和 `latest` 摘要一致。
- 回滚版本：`v0.33.1`
- 回滚镜像：
  `docker.io/kjlion/kejilion-panel@sha256:74d29da681c1377c77ed062d363a02d26abaafc748c80a2e988e8fd5729d96e8`
