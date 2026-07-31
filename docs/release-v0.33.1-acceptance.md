# KPanel v0.33.1 发布验收

## 发布范围

- 历史监控容器列表先按最后采样时间排列，保证当前运行或数据最新的容器优先。
- 同一采样批次内按实际内存占用降序、CPU 使用率降序和名称升序排列。
- 无历史点的容器置底；采样频率、32 容器上限、API、存储格式、前端布局与左侧导航保持不变。

## 验收结果

- [x] 排序单元测试覆盖最新数据、内存、CPU、名称、过期数据和空数据优先级。
- [x] Go 相关包测试和 `go vet` 通过。
- [x] Web 类型检查、164 项单元测试和生产构建通过。
- [x] 7 天历史查询五次基准中位数约 `33.42 ms/op`，未发现相对 `v0.33.0` 的性能回退。
- [x] 生态规则、`govulncheck`、Node 依赖审计、应用生命周期及镜像运行时契约通过。
- [x] GitHub Release、`linux/amd64`、`linux/arm64` 版本镜像和 `latest` 发布完成。

## 发布证据

- 功能提交：`c08892e`
- 发布提交：`8fab713`
- 功能分支 CI：https://github.com/kejilion/KPanel/actions/runs/30628170596
- 功能主线 CI：https://github.com/kejilion/KPanel/actions/runs/30628265299
- 版本主线 CI：https://github.com/kejilion/KPanel/actions/runs/30628432674
- Release：https://github.com/kejilion/KPanel/actions/runs/30628535415
- GitHub Release：https://github.com/kejilion/KPanel/releases/tag/v0.33.1
- Docker Hub `0.33.1` / `latest`：
  `sha256:74d29da681c1377c77ed062d363a02d26abaafc748c80a2e988e8fd5729d96e8`
- 平台：`linux/amd64`、`linux/arm64`

## 兼容与回滚

- 排序仅发生在最多 32 条已读取历史序列上，不增加 Docker Stats 请求、采样量或磁盘占用。
- 本次不修改 `kejilion.sh` 协议，不要求同步更新脚本。
- 回滚版本：`v0.33.0`
- 回滚镜像：
  `docker.io/kjlion/kejilion-panel@sha256:a0f84f38dd3cee370d37490e09c25a8b00ccd3b1d684cea1cf3a4c445ba759c6`
