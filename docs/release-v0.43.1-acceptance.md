# KPanel v0.43.1 发布验收

## 发布范围

v0.43.1 把 AI 助手从基础对话与结构化调用补齐为轻量运维闭环：修正 Docker 资源查询工具路由，增加会话级手动/安全自动审批，并新增宿主机进程排行、磁盘热点、日志尾读、可恢复文件清理和 Nginx 校验后重载。现有站点、应用、Docker、系统清理和备份迁移业务继续复用同一 Host Operation 与 Agent 校验链路，没有引入 Hermes、Sidecar、通用 Shell、外部工具链或新依赖。

## 版本与制品

- 发布提交：`c567eb6d528237f64272b723a1f808a234e2e497`
- 标签：`v0.43.1`
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.43.1>
- 候选 CI：<https://github.com/kejilion/KPanel/actions/runs/30894072885>
- main CI：<https://github.com/kejilion/KPanel/actions/runs/30895273523>
- Release 工作流：<https://github.com/kejilion/KPanel/actions/runs/30895474861>
- Docker 多架构摘要：`sha256:854d7f542e54b62cd7cc1aeb998b364e1da14d6743a3f2d6fdc86f3a05903ab2`
- linux/amd64：`sha256:474bdc4726cac97ae0717c457edafe88786b0c261fd7f5332b82ecf732f31da9`
- linux/arm64：`sha256:2d48f99a5acd031d7189d033a6098695e658dc9b8734c5205aa7ac485a633fe5`
- `0.43.1` 与 `latest` 指向同一多架构摘要。
- 公开应用市场 `kpanel.conf` Blob：`0a603abfe77beb045c4e7648dd60f5e4a1876e4d`；SHA-256：`b1e23371da402ebfcb61d56a377f77b6b78c44a454e23b99119bab5aaf895d0f`，与仓库发布配置一致。

## 自动化与安全验收

- 前端 i18n、typecheck、生产构建成功；Vitest `38` 个文件、`246` 项测试通过。
- Linux `go test ./...`、`go vet ./...` 通过；AI、Agent、Panel 与 SystemInfo 关键包 `-race` 通过。
- 版本一致性与生态策略检查通过；Trivy 源码、依赖、Secret、Dockerfile 和最终镜像 HIGH/CRITICAL 扫描均为 `0`。
- OpenAI-compatible、Anthropic、Gemini 仍通过同一 Tool Registry 调用结构化工具；本轮无协议、依赖、端口、Compose 或应用市场安装契约变更。
- AI 文件内容额外拒绝密钥、`.env`、SSH 凭据、私钥、进程环境和 KPanel 数据；文件改写拒绝 KPanel、Docker、systemd、SSH、PAM、sudo、cron、系统账号及系统程序目录。Agent 仍二次执行路径、符号链接、保护目录和资源版本校验。
- 进程采样不读取命令行；磁盘分析固定根目录、同文件系统、不跟随符号链接，最多四线程、20 万目录项、8 秒且单实例只运行一个扫描。

## 154 隔离真机工作流

候选 Panel、候选 Agent、本地 Mock Provider 和真实 154 Host/Docker 组成隔离验收环境；没有使用用户提供的 API Key，也没有调用付费模型：

- 自动会话标题、Docker 资源只读路由、文件列表/读取、聊天工具数据过滤通过。
- `manual` 模式在文件写入前阻塞，批准后执行；`auto` 模式对普通文件写入和可恢复回收站操作自动放行；KPanel 保护路径仍拒绝。
- CPU 闭环依次读取系统概览、宿主机进程排行和 Docker 容器指标，并复查；没有任意 PID kill。
- 磁盘闭环分析 `/tmp`，读取测试文件 `resourceVersion`，移入 Agent 回收站并复查；验收目录清理后测试文件与回收站一并移除。
- Nginx 闭环尾读真实错误日志、执行固定 `nginx -t`、安全重载并再次校验，全部成功。
- 迁移闭环盘点真实 Docker 容器，进入手动审批后创建迁移前备份；未提供目标主机时不执行传输。最新完整归档为 `/home/docker/.kpanel-backups/docker-20260804T090858Z-4bf115ad.tar.gz`，大小 `128,998,011` B，SHA-256：`79644e619d31aad8b91f4381687386cef3143cc0a418531c1d511f8e8ffbdc2b`。
- 审批模式在 Panel 重启后保持；审计包含 AI 工具事件且不包含 Provider Key、文件内容或保护文件内容。

## 轻量验收

- stripped `paneld`：v0.43.0 为 `12,972,194` B，v0.43.1 为 `13,013,154` B，仅增加 `40,960` B。
- 两个并行能力保留 256 MiB 限额；完整 Mock 运维验收峰值为 `140.2 MiB / 256 MiB`，未发生 OOM。
- 多轮工具调用后重启空闲为 `75.23 MiB / 256 MiB`；v0.42.0 同类 Mock 基线为 `72.6 MiB / 256 MiB`，增量约 `2.63 MiB`，低于 25 MiB 目标。
- 生产升级前 v0.43.0 Docker 统计为 `13.65 MiB / 256 MiB`；升级后 v0.43.1 暖机为 `12.91 MiB`，连续观察十分钟为 `13.09 MiB / 256 MiB`。

## 154 上线结果

- 使用公开应用市场 `KJ_APP_ACTION=update` 原生链路完成 v0.43.0 → v0.43.1 更新，输出 `KPanel 更新完成 / Update Complete`。
- Agent：`0.43.1 v1alpha1`，服务 `active/running`、`NRestarts=0`；安装二进制与正式镜像 `/release/kejilion-agent` SHA-256 均为 `268f3604b5e1a3a3636db75ee7e9aa5b515ba88f96918771b3e2212ebfb1a4c6`。
- Panel：`version=0.43.1`、`status=ok`、容器 `healthy`；连续观察十分钟 Panel/Agent 重启数均为 `0`，运行镜像 ID 为正式摘要。
- 配置的公共入口 `http://154.36.153.9:8080` 下 `/`、`/ai` 和健康接口均返回 `200`；未登录 Provider API 返回 `401`，无效 Host 返回 `421`。
- Agent 生产实测进程双排行、`/tmp` 磁盘分析、Nginx 日志尾读和 `nginx -t` 均成功；四项新 capability 全部启用。
- Panel 继续使用 `65532:65532`、只读根文件系统、`privileged=false`、`cap_drop: ALL`、`no-new-privileges`、256 MiB、128 PIDs、内部与出口双网络及最小出口网关 `/32` 信任。
- AI SQLite `integrity_check=ok`、WAL 正常、`approval_mode` 迁移存在；`ai.db` 与 `ai-secrets.key` 权限均为 `0600`。最近 10 分钟 Panel/Agent 日志未发现 panic、fatal 或 error。
- 另一台公开实例 `https://kpanel.kejilion.eu.org` 仅做黑盒复核：健康接口为 `0.43.1/ok`，`/ai` 返回 `200`，未登录 Provider API 返回 `401`；本次没有登录或修改该主机。

## 回滚

- 公开源码与镜像回滚点：`v0.43.0`，镜像摘要 `sha256:8da3c071341faebb8356148876486726e63b1b651c403ad5bd57086c53401574`。
- 154 升级前一致性备份：`/root/kpanel-backups/v0.43.1-preupgrade-20260804T092205Z/kpanel-home.tar.gz`。
- 备份权限 `0600`，大小 `20,592,829` B，SHA-256：`1206ca02048c0748f17db500e780fac06fd6d50710e0486287f5857e6fa917c4`。
- 回滚时恢复旧镜像、Agent、Compose、systemd 单元与完整 KPanel 目录；`ai.db*` 必须与 `ai-secrets.key` 成对保留。

## 已知边界

- Mock Provider 证明协议归一化后的工具调用与宿主机闭环，不等同于某个真实模型的质量或兼容性测试；用户先前在会话中发送的 Key 未用于本轮验收，应轮换后再做独立短期真 API 测试。
- 迁移验收只完成盘点和备份；没有明确目标主机与可信 SSH 配置，因此没有执行跨机传输。
- 154 的可选域名 `gm.kejilion.eu.org` 当前提供自签名证书且连接复核失败；KPanel 配置的公共入口仍为 IP HTTP，升级前后未改动该域名。该 Nginx/TLS 问题需作为独立任务处理。
- 原 `feature/health-center` 脏工作树保持在 `c9a3419`，本次未 stash、reset、提交或修改其文件。
