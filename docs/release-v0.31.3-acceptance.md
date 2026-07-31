# KPanel v0.31.3 发布验收

## 范围

- 左侧栏收起时保持品牌与导航图标横向锚点稳定，避免图标先跳位再随宽度移动。
- 侧栏文字、间距、按钮、在线状态、用户区与内容区采用一致的过渡节奏。
- 保留 `prefers-reduced-motion` 兼容，移动端抽屉导航行为不变。

## 验收结果

- [x] 展开、收起两种终态的视觉复核；导航图标横向位置由 26px 平滑收敛到 30px。
- [x] Web 类型检查、156 项单元测试和生产构建。
- [x] 生态规则检查。
- [x] Linux CI、Go 测试与 vet、部署安装安全测试。
- [x] 漏洞扫描、依赖审计和 kejilion.sh 应用生命周期测试。
- [x] Linux amd64/arm64 镜像构建、运行时健康检查与镜像契约检查。
- [x] GitHub Release、版本镜像和 `latest` 镜像发布。

## 发布证据

- 修复提交：`131a8be`
- 发布提交：`de52587`
- 主线 CI：https://github.com/kejilion/KPanel/actions/runs/30594420318
- Release：https://github.com/kejilion/KPanel/actions/runs/30594546406
- GitHub Release：https://github.com/kejilion/KPanel/releases/tag/v0.31.3
- Docker Hub `0.31.3` / `latest`：
  `sha256:4b74d04d84089cd3e2ac1b1f0f87be46c8b5f7531ec680ac452574078cea8601`
- 平台：`linux/amd64`、`linux/arm64`

## 兼容与回滚

- 本次仅修改浏览器端桌面侧栏样式，不改变路由、API、Agent、数据和任务协议。
- 回滚版本：`v0.31.2`
- 回滚镜像：
  `docker.io/kjlion/kejilion-panel@sha256:960ac7c7ced5dd22bd71d2db6ec7f9cfbb29af809850e05e9b7299b67bffb265`
