# KPanel v0.54.0 上线验收记录

## 发布结论

- 发布时间：2026-08-09（Asia/Shanghai）。
- 生产目标：`arena-154`（`154.36.153.9:8080`）。
- 版本：`v0.54.0`。
- 发布提交：`7a376d691f29646501c0648e025b0685e8cbcfa3`。
- 上一稳定版本：`v0.53.0`（`4c37a0e019c66bf6348d2e9f5e33d19438516f28`）。
- 结论：发布、生产部署和自动化验收通过。

## 上线范围

### 桌面窗口导航

- 桌面窗口标题栏新增统一后退按钮，并支持 `Alt + ←`。
- 文件目录、历史监控、进程管理器和监控缩放使用各自窗口内的独立导航历史。

### 轻量内嵌浏览器

- 桌面新增浏览器入口和居中网址起始页，网站图标与自定义 URL 共用单实例多标签窗口。
- 最多保留 8 个标签，同时只保活 2 个 iframe；窗口进入后台 45 秒后释放 iframe，恢复时只重建当前页面。
- 达到标签上限时不覆盖现有页面；目标网站拒绝内嵌时保留“用系统浏览器打开”。
- 仅允许规范化后的无凭据 HTTP/HTTPS 地址，并继续受同源策略、CSP、`X-Frame-Options` 和 sandbox 约束。

### 概览入口顺序

- 历史监控与进程管理快捷入口调整为与对应数据卡一致的顺序。

## 提交记录

- 来源 `47bb8ed`，候选 `dd6dde3`：`fix(desktop): restore window back navigation`。
- 来源 `cc3aee9`，候选 `4a562eb`：`feat(desktop): add sandboxed website browser`。
- 来源 `22f7ece`，候选 `83ea12a`：`feat(desktop): add lightweight browser tabs`。
- 来源 `d121714`，候选 `1ef83a8`：`fix(overview): reorder monitoring shortcuts`。
- `7a376d6`：`chore: prepare KPanel 0.54.0`。

## 上线前验证

- 版本一致性、生态策略和发布工作流 YAML 检查通过。
- `npm ci`：252 个包，`npm audit` 为 0 个漏洞。
- TypeScript 类型检查通过；73 个前端测试文件、497 项测试通过。
- i18n 检查通过：1,730 条文案、19 个延迟目录。
- 生产构建通过；浏览器页面 JS gzip 3.82 KiB、CSS gzip 2.07 KiB，桌面页面 JS gzip 13.54 KiB。
- Linux `go test ./...`、核心权限包 race 和 `go vet ./...` 通过。
- `govulncheck` 为 0 个可达漏洞；Trivy 源码与最终镜像扫描均为 0 个高危/严重发现。
- Linux amd64/arm64 的 Panel、Agent、Node 和 `kpctl` 构建通过。
- 安装安全、`kejilion.sh` 应用生命周期、最终镜像 E2E 和 256 MiB/1 CPU/128 PID/非 root/只读根运行契约通过。
- 候选分支 CI 通过：[run 31291674593](https://github.com/kejilion/KPanel/actions/runs/31291674593)。
- 主分支 CI 通过：[run 31291794422](https://github.com/kejilion/KPanel/actions/runs/31291794422)。
- 发布工作流通过：[run 31291931750](https://github.com/kejilion/KPanel/actions/runs/31291931750)。

## 发布产物

- GitHub Release：[v0.54.0](https://github.com/kejilion/KPanel/releases/tag/v0.54.0)，非草稿、非预发布，共 8 个附件。
- OCI 多架构索引及 `latest`：`sha256:6441741237a74a916902e0e2328e2ee29ded8bcdc935f073c140cd5bd26d93d5`。
- linux/amd64：`sha256:56b1818f0696ed9de0f9aa2532b05e60311a8fc12444c784505d5180f17de480`。
- linux/arm64：`sha256:03785488d43ad4a38d8dcae47e8c42d8257b3fa9c28be8199ff0e2726b2fc21d`。
- 公共镜像重新拉取与 E2E 返回 `image_e2e=pass`；OCI revision 为发布提交 `7a376d6`。
- `apps` 安装/更新契约没有变化，两端 `kpanel.conf` blob 均为 `d49383667cea8c3b7294bf40ba1e272370a2cb87`，无需应用市场提交。
- `kejilion.sh` 仓库没有变化，本地与远端 `main` 均为 `dd26cf7eb962f985f94773c15f9b643677b4471c`。

## 生产备份与部署

- 升级前备份：`/root/kpanel-backups/v0.54.0-preupgrade-20260809T031900Z`，目录权限 `0700`。
- SQLite 备份大小 122,880 B，应用归档大小 5,963,519 B；全部备份校验和通过。
- 生产更新使用 `KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update k app kpanel` 等价入口，脚本内置自动回滚未触发。
- 升级前镜像索引：`sha256:6e28e1fc0cdebe939cbedd75da7eb9180e00f558d6489335609c1f87126879d5`。

## 上线后验证

- 公网首页返回 HTTP 200；公开健康接口返回 `status=ok`、`version=0.54.0`、协议 `v1alpha1`。
- 容器镜像为 `sha256:6441741237a74a916902e0e2328e2ee29ded8bcdc935f073c140cd5bd26d93d5`，状态 `running/healthy`，重启数 0。
- Agent 为 `active`，重启数 0，`NeedDaemonReload=no`；Panel 到 Agent 的版本与协议健康检查通过。
- 宿主机 Agent 与发布镜像 Agent SHA-256 均为 `f98520cfded0d1d6c0bb13519d059b8070662774d34e19b902f848742c28ed02`。
- 宿主机 `kejilion.sh` 只保留既有 `permission_granted=true` 本机设置，其余内容与镜像内固定脚本一致。
- SQLite `PRAGMA integrity_check=ok`；最近 10 分钟 Panel 与 Agent 日志中没有 `panic`、`fatal` 或 `error`。
- 生产镜像包含 `WebBrowserView-BLk0iwig.js`（11,431 B）；首页 CSP 只新增受限的 `frame-src 'self' http: https:`，其余安全头保持生效。
- 2 分钟稳定性观察：60 次健康请求零失败，Panel/Agent 零重启；Panel 使用约 12.51 MiB/256 MiB，Agent RSS 25,800 KiB。

## 回滚方案

1. 将 `docker.io/kjlion/kejilion-panel:latest` 固定回 `v0.53.0` 索引 `sha256:6e28e1fc0cdebe939cbedd75da7eb9180e00f558d6489335609c1f87126879d5`。
2. 从升级前备份恢复 Agent、Compose、环境文件和脚本，执行 `systemctl daemon-reload` 后重启 Agent 与 Panel。
3. 如需恢复数据，先停止 KPanel、再次备份当前状态，再使用 `/root/kpanel-backups/v0.54.0-preupgrade-20260809T031900Z`。
4. 回滚后核验健康接口版本、容器与 Agent 状态、SQLite 完整性和日志。

## 已知限制

- 目标网站可通过 CSP 或 `X-Frame-Options` 拒绝 iframe；这是浏览器安全边界，不会代理绕过，用户需改用系统浏览器打开。
- 未执行带登录态的生产 UI 人工点击；窗口导航、多标签、上限提示、休眠和安全边界由 497 项组件测试、真实浏览器性能验收、生产构建、候选 CI 与 Release 运行时契约覆盖。
- `https://kp.kejilion.pro/` 当前返回 HTTP 404，不是本次部署目标；本轮未修改其域名路由。生产 IP 入口与健康接口均正常。
