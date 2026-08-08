# KPanel v0.52.0 上线验收记录

## 发布结论

- 发布时间：2026-08-08（Asia/Shanghai）
- 生产目标：`arena-154`（`154.36.153.9:8080`）
- 版本：`v0.52.0`
- 发布提交：`a314dd90e583696deac0a07375524fc7a37ce910`
- 上一稳定版本：`v0.51.0`（`c7e074f22b10991448a921a51213599e0114ba96`）
- 结论：发布、部署和自动化验收通过。

## 上线范围

本次仅包含以下两组能力，不包含系统中心、治理规则、应用市场配置或 `kejilion.sh` 变更。

### 桌面体验

- AI 窗口滚动边界收敛。
- 脚本应用在独立终端窗口中运行。
- 已由应用市场承载的网站应用不再重复显示网站快捷图标。

### 进程管理

- 新增轻量进程管理页面和桌面任务栏入口。
- 支持受限进程列表、搜索与排序。
- 仅允许对经 `PID + startTimeTicks` 二次确认的进程发送 `SIGTERM` 或 `SIGKILL`，防止 PID 复用误操作。

## 提交记录

- `95c946f` `fix(desktop): contain AI window scrolling`
- `ee47015` `feat(desktop): open script apps in dedicated terminal`
- `5b83325` `fix(desktop): hide app-backed website icons`
- `f1cb879` `feat(system): add lightweight process manager`
- `8761fe4` `feat(desktop): add taskbar process manager shortcut`
- `a314dd9` `chore: prepare KPanel 0.52.0`

## 上线前验证

- 版本字段一致性检查通过。
- `npm ci` 通过，252 个包，0 个漏洞。
- TypeScript 类型检查通过。
- 70 个测试文件、469 项测试通过。
- i18n 检查通过：1,732 条文案、18 个延迟目录。
- 生产构建通过；主入口 gzip 23.90 KiB，新增页面均低于预算。
- 候选分支 CI 通过：[run 31264236660](https://github.com/kejilion/KPanel/actions/runs/31264236660)。
- 主分支 CI 通过：[run 31264372351](https://github.com/kejilion/KPanel/actions/runs/31264372351)。
- 发布工作流通过：[run 31264520667](https://github.com/kejilion/KPanel/actions/runs/31264520667)。
- 公共镜像隔离启动与健康检查通过。
- `apps` 的 `kpanel.conf` 与 KPanel 打包配置一致，且本次无 `apps` 变更。

## 发布产物

- GitHub Release：[v0.52.0](https://github.com/kejilion/KPanel/releases/tag/v0.52.0)。
- OCI 多架构索引及 `latest`：`sha256:e800d0d7cc8fffc906f37fe41ac1570a3e1fc6c71ac392eb130a04a452cc86e4`。
- linux/amd64：`sha256:0d46067133472e4b29ee0d3f485012cba494d2d3f492078f98fef069d566e8da`。
- linux/arm64：`sha256:4f7e0fd3d9d93421519342baf62d96e36cf4dedc873037140f0bd344db3605bd`。

## 生产备份与部署

- 升级前备份：`/root/kpanel-backups/v0.52.0-preupgrade-20260808T153555Z`，目录权限 `0700`。
- SQLite 备份完整性检查通过；应用数据归档和元数据校验通过。
- 生产更新命令：`KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update k app kpanel`。
- 生产容器运行镜像：`sha256:e800d0d7cc8fffc906f37fe41ac1570a3e1fc6c71ac392eb130a04a452cc86e4`。

## 上线后验证

- 健康接口连续 3 次返回 `ok`，版本均为 `0.52.0`。
- 容器状态为 `running/healthy`，重启数为 0，未发生 OOM。
- 非 root 用户、只读根文件系统、CPU/内存/PID 限制、`CapDrop=ALL` 和 `no-new-privileges` 均生效。
- Agent 服务为 `active/running/enabled`，重启数为 0，`NeedDaemonReload=no`。
- 宿主机 Agent 与镜像内 Agent 的 SHA-256 一致：`a6d29ade9806b055587c22c4026b1c31224ccf64351e6415bf4e6ebbe23648c4`。
- SQLite `PRAGMA integrity_check` 返回 `ok`。
- 部署后 10 分钟 Panel 与 Agent 日志中未发现 `panic`、`fatal` 或 `error`。
- 外部健康接口返回 `0.52.0`，首页 HTTP 200 且包含标题。
- 进程列表真机检查通过：返回 229 个进程中的受限 32 项，未暴露命令行、环境变量等敏感字段。
- 进程操作真机检查通过：错误的 `startTimeTicks` 返回 409 且未终止进程；正确身份仅终止专用 `sleep` 测试进程。
- 2 分钟稳定性抽样通过：60 次进程接口请求零失败、Agent 零重启；RSS 从 23,848 KiB 降至 23,512 KiB，FD 从 7 变为 9，均在容差内。

## 回滚方案

1. 将生产镜像固定回 `v0.51.0` 对应镜像摘要 `sha256:9006758325cb941eeaed90f668a48bf03cf4ac6da288dd8723face887986aba4`。
2. 重新创建 KPanel 容器并确认健康接口返回 `0.51.0`。
3. 如需恢复数据，使用升级前备份 `/root/kpanel-backups/v0.52.0-preupgrade-20260808T153555Z`；恢复前先停止 KPanel 并再次备份当前数据。
4. 回滚后核验容器健康、Agent 版本一致性、SQLite 完整性及近 10 分钟日志。

## 已知限制

- 当前自动化环境未执行带登录态的生产 UI 人工点击与 30 分钟页面驻留；核心接口、隔离写操作、外部访问和 2 分钟接口稳定性均已通过。
- 本地 Windows 环境未安装 Go 且 Docker daemon 不可用；Go、race、安全扫描、Linux 构建与镜像验证由已通过的 GitHub Actions 承担。
