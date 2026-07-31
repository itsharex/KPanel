# KPanel v0.33.0 发布验收

## 发布范围

- 历史监控维持四组宿主机视图：CPU 与负载、内存、磁盘容量与 I/O、网络流量与连接数。
- 图表增加纵向刻度、悬停游标和精确时间值；历史监控仍从概览进入，不增加左侧导航分类。
- 新增宿主机磁盘读写速率，并统一宿主机与 Docker 容器的 CPU、内存、网络和磁盘 I/O 指标语义。
- Docker 管理默认进入容器页，统一状态概览、导航、搜索和排序，运行资源及正在使用的资源优先显示。
- `kejilion.sh` 协议版本与固定摘要保持不变，本次发布不要求同步升级脚本。

## 验收结果

- [x] Web 类型检查、164 项单元测试和生产构建。
- [x] Go 相关包测试、`go vet`、磁盘设备识别及历史聚合回归测试。
- [x] `govulncheck` 使用 Go 1.26.5 扫描，无可达漏洞；`npm audit` 无漏洞。
- [x] Linux 安装安全检查和应用生命周期检查。
- [x] 浏览器复核深浅主题、四组历史视图、悬停刻度、切换控件、Docker 默认页与智能排序。
- [x] 历史查询基准中位数由 34.14 ms 变为 34.50 ms，差异约 1.1%，内存分配基本不变。
- [x] `/proc/diskstats` 采集基准中位数 25.09 µs/op、2417 B/op、13 allocs/op。
- [x] GitHub 主线 CI、Release、版本镜像和 `latest` 镜像发布。

## 发布证据

- 功能提交：`ae3e98f`
- 发布提交：`637a2f1`
- 功能主线 CI：https://github.com/kejilion/KPanel/actions/runs/30624299366
- 版本主线 CI：https://github.com/kejilion/KPanel/actions/runs/30624444280
- Release：https://github.com/kejilion/KPanel/actions/runs/30624574923
- GitHub Release：https://github.com/kejilion/KPanel/releases/tag/v0.33.0
- Docker Hub `0.33.0` / `latest`：
  `sha256:a0f84f38dd3cee370d37490e09c25a8b00ccd3b1d684cea1cf3a4c445ba759c6`
- 平台：`linux/amd64`、`linux/arm64`

## 兼容与回滚

- 新增历史字段均为可选字段，旧 Agent 滚动升级期间可继续读取现有历史记录。
- 磁盘 I/O 缺失时不伪造速率；老记录与新记录混合查询不会制造异常峰值。
- 本次不修改侧栏信息架构、`kejilion.sh` 协议、网站、容器或业务数据。
- 回滚版本：`v0.32.0`
- 回滚镜像：
  `docker.io/kjlion/kejilion-panel@sha256:cfe5b2433736820ea1a8299c4e2db21a3ddbcdc3d4d6747a144b133b20daaf36`
