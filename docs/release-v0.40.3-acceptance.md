# KPanel v0.40.3 发布验收

发布日期：2026-08-03

## 发布范围

- 设置页将“修改密码”移动到“修改用户名”下方，使账户管理顺序更符合操作习惯。
- 手机端文件管理使用紧凑工具栏和底部选择操作区，优化路径导航、搜索、批量选择及窄屏连续操作体验。
- 桌面端文件管理布局和交互保持不变。
- 本次不包含系统中心页面、API、协议或文档变更。
- 本次没有端口、持久化数据格式、部署配置、应用市场契约或 `kejilion.sh` 协议迁移。

## 自动化与安全验收

- Windows 前端依赖安装、定向测试、完整生产构建、类型检查和 1458 条多语言资源检查通过；定向测试共 38 项。
- Linux L3 验收使用精确基线 `v0.40.2`，Go 全量测试、核心包 race、前端 35 个测试文件与 235 项测试、安装安全检查、`govulncheck` 和 `npm audit` 均通过。
- 候选分支 CI、主线 CI 和 Release 工作流均成功；Release 工作流完成原生镜像漏洞扫描、运行时健康检查、权限与资源限制检查、amd64/arm64 构建及稳定标签摘要一致性校验。
- 本地 Trivy 数据库下载和发布后 Docker Hub 完整镜像拉取受当前网络超时影响，未将其记为本地通过；远程 Release 安全门禁与运行契约已经成功完成。
- 当前环境没有可用的 154 真机 SSH 凭据，因此未将本次自动化验收记录为 154 真机验收。

## 发布产物

- 功能提交：`05a2eec`、`c7ae402`。
- 版本准备与标签提交：`85c77230a7fe7fd450dea008e201b1dbf24e6962`。
- 候选分支 CI：[30815952222](https://github.com/kejilion/KPanel/actions/runs/30815952222)。
- 主线 CI：[30816149625](https://github.com/kejilion/KPanel/actions/runs/30816149625)。
- Release 工作流：[30816379812](https://github.com/kejilion/KPanel/actions/runs/30816379812)。
- GitHub Release：[v0.40.3](https://github.com/kejilion/KPanel/releases/tag/v0.40.3)，非草稿、非预发布，包含明确版本更新说明及完整附件。
- `docker.io/kjlion/kejilion-panel:0.40.3` 与 `latest` 均指向 OCI 摘要 `sha256:511774e4cf766b5a07f55d5dc597b602a33cd75e13d4da3d2106913173711254`。
- 镜像包含 `linux/amd64` 和 `linux/arm64`；平台清单摘要分别为 `sha256:b537644cf239591d71bfc7d1637c50435abf5d33e01eab7f9ecee5554b6c6f53` 与 `sha256:87bf2ea63838fe63d6e55af603af02957ea355663afb5299c36ed3371a87c5b4`。
- Release 附件包含 amd64/arm64 Agent、amd64/arm64 轻量节点、部署归档、`SHA256SUMS`、许可证和第三方声明。

## 兼容、风险与回滚

- `packaging/kejilion-app/kpanel.conf` 相对 `v0.40.2` 未变化，因此无需修改或发布 `kejilion/apps`。
- 更新只调整已登录管理界面的前端布局与交互，不新增 API、权限、网络入口、后台任务或持久化状态。
- 回滚点为 `v0.40.2`；回滚只替换 Panel/Agent 版本，不删除用户文件、业务数据、配置或审计记录。
