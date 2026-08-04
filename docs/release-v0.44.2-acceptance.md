# KPanel v0.44.2 发布验收

## 发布范围

v0.44.2 修复 AI 附件在 ModSecurity 后被误判为超大非文件 JSON 的问题，将带附件消息改为标准
`multipart/form-data`；同时修复实时工具事件零值时间戳造成的工具卡片集中前置，并为模型未输出说明的
工具批次补充简短可见进度。反向代理或 Provider 的 HTML 错误页不再原样展示。

## 版本与制品

- 修复提交：`5feaf81a46dd87edfe8e248046f8777845fc25df`。
- 发布提交：`c920ba3d17295d383732b242c6b9dfdefcdcfc75`。
- 标签：`v0.44.2`。
- 候选 CI：<https://github.com/kejilion/KPanel/actions/runs/30922316515>。
- main CI：<https://github.com/kejilion/KPanel/actions/runs/30922535896>。
- Release 工作流：<https://github.com/kejilion/KPanel/actions/runs/30922809540>。
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.44.2>。
- Docker 多架构摘要：`sha256:396c854af8b796924bb87db4c3dbf7adf9dc65bb5cf31bf8116a8cd5a3f3b3e8`。
- linux/amd64：`sha256:c543c59a943d58ec8d6e663a2f96d9fbbc7e69842584bea96dbd320b7ea734be`。
- linux/arm64：`sha256:9752f293473fd0ae4149e07124871ae1e7e2cca31bbc02095c4b0fdb0d6f7a1f`。

Release 已公开且不是 prerelease；Agent、Node、部署归档、许可证、第三方声明和 `SHA256SUMS`
共 8 个附件完整。`0.44.2` 与 `latest` 已由发布工作流晋升到相同 OCI index。

## 自动化与协议回归

- 本地 `internal/ai` 全量测试和 Panel AI 定向测试通过；前端 39 个文件共 254 项测试通过。
- 前端 typecheck、生产构建、`go vet ./...`、linux/amd64 与 linux/arm64 CGO-free 构建通过。
- 候选 CI 与 main CI 均通过变更门禁、特权核心 race、`govulncheck`、`npm audit`、Trivy 和应用市场生命周期测试。
- Release 工作流通过源码复验、原生镜像扫描、运行时契约、多架构构建、SBOM/Provenance 和公开发布。
- 在 `kp.kejilion.pro` 使用 700 KiB 请求体复现旧 Base64 JSON 为 `400 text/html`；相同数据改为 multipart
  后越过 WAF 并到达 KPanel 路由，返回测试会话不存在的应用层 404。由此确认无需放宽全站
  `SecRequestBodyNoFilesLimit 524288`。

## 154 上线与运行观察

- 升级前为 `0.44.1/ok`、healthy、Agent active、重启 0、OOM=false；旧镜像摘要为
  `sha256:41d74a39f7dab9bc61f319f3cffd7724cb115f2674b0b5d6bb45f7c88120b45a`。
- 一致性备份：`/root/kpanel-backups/v0.44.2-preupgrade-20260804T151657Z`，目录权限 `0700`、文件权限
  `0600`。SQLite 在线备份 122,880 B，SHA-256 为
  `9b583fa4aeb98aaffc329771fdaf3e03d4cb9a6137d71e046094113b7273f0a0`；全目录归档 20,668,148 B，
  SHA-256 为 `8c7c8b17f83d6ce5889806d32d05f8205f400083f7dfa0d106adbb20840d03ab`。
- 使用 `KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update k app kpanel` 完成原生应用市场更新，实际拉取正式摘要
  `sha256:396c854af8b796924bb87db4c3dbf7adf9dc65bb5cf31bf8116a8cd5a3f3b3e8`。
- 升级后容器与 Agent 均为 `0.44.2`；公网 `https://kp.kejilion.pro/api/v1/health` 连续返回
  `0.44.2/ok`。容器保持 `65532:65532`、只读根文件系统、非特权、`cap-drop ALL`、256 MiB 和 128 PID。
- AI SQLite `integrity_check=ok`，迁移 9 已生效；`ai.db` 与 `ai-secrets.key` 权限均为 `0600`。
- 连续约 5 分钟、每 30 秒一次共 10 次采样全部为 healthy、重启 0、OOM=false；启动期内存约
  73.42–73.82 MiB，随后回落到 10.89–11.02 MiB，CPU 约 0.02%–0.04%，近期错误日志计数为 0。

## 回滚与未验证边界

- 公共镜像回滚点为 `v0.44.1`，摘要
  `sha256:41d74a39f7dab9bc61f319f3cffd7724cb115f2674b0b5d6bb45f7c88120b45a`；现场回滚使用上述成对备份恢复
  `ai.db*`、`ai-secrets.key`、Compose、Agent 和旧镜像。
- 154 当前没有 Provider、模型或会话，因此没有调用真实付费 API 做登录态图片问答；multipart 编解码、实际类型检测、
  WAF 穿透和三协议图像载荷由单元、集成、CI 与公网无密钥协议回归覆盖。
- 独立域名 `kpanel.kejilion.eu.org` 在本次验收结束时仍为 `0.44.1`，不属于本次已授权的 154 更新目标，未越权部署。
