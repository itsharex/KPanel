# KPanel v0.32.0 发布验收

## 发布范围

- 文件管理恢复符合 Windows 使用习惯的选择、复制、剪切与粘贴交互。
- 新增轻量主机与容器历史监控，按颗粒度和保留周期控制存储占用。
- 历史监控入口统一移动到概览实时资源卡片，不增加左侧导航负担。
- 应用市场 KPanel 配置同步放开经审计的文件管理写操作。
- `kejilion.sh` 协议版本与固定摘要保持不变，本次发布不要求同步升级脚本。

## 验收结果

- [x] Web 类型检查、161 项单元测试、生产构建。
- [x] Go 全量测试、`go vet` 和核心包竞态检测。
- [x] `govulncheck` 使用发布工具链 Go 1.26.5 扫描，无可达漏洞。
- [x] Shell 语法、安装安全测试和生态规则检查。
- [x] Linux `amd64` / `arm64` 的 Panel、Agent 与 CLI 交叉构建。
- [x] 浏览器复核历史监控入口、图表映射和控制台错误。
- [x] GitHub 主线 CI、Release、版本镜像和 `latest` 镜像发布。

## 发布证据

- 文件管理提交：`94d7fab`
- 历史监控提交：`bbc2507`
- 历史入口提交：`4e73d19`
- 发布提交：`47f2eba`
- 应用市场配置提交：`2391353`
- 主线 CI：https://github.com/kejilion/KPanel/actions/runs/30608839426
- Release：https://github.com/kejilion/KPanel/actions/runs/30609121419
- GitHub Release：https://github.com/kejilion/KPanel/releases/tag/v0.32.0
- Docker Hub `0.32.0` / `latest`：
  `sha256:cfe5b2433736820ea1a8299c4e2db21a3ddbcdc3d4d6747a144b133b20daaf36`
- 平台：`linux/amd64`、`linux/arm64`

## 兼容与回滚

- 本次为兼容新增，不修改现有 Agent、应用任务和 `kejilion.sh` 固定协议。
- 历史监控数据采用独立存储；回滚不会删除已有业务、容器或网站数据。
- 回滚版本：`v0.31.3`
- 回滚镜像：
  `docker.io/kjlion/kejilion-panel@sha256:4b74d04d84089cd3e2ac1b1f0f87be46c8b5f7531ec680ac452574078cea8601`
