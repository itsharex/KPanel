# KPanel v0.53.0 上线验收记录

## 发布结论

- 发布时间：2026-08-09（Asia/Shanghai）。
- 生产目标：`arena-154`（`154.36.153.9:8080`）。
- 版本：`v0.53.0`。
- 发布提交：`4c37a0e019c66bf6348d2e9f5e33d19438516f28`。
- 上一稳定版本：`v0.52.0`（`a314dd90e583696deac0a07375524fc7a37ce910`）。
- 结论：发布、生产部署和自动化验收通过。

## 上线范围

### 网站外观与桌面入口

- 从已发现站点的本机 Nginx 首页安全提取并缓存网站名称，在网站列表的域名下方显示。
- 桌面网站名称按“用户重命名、网站名称、域名”顺序回退。
- favicon 不可用时，使用按站点稳定配色的字母图标和网站角标。

### 桌面窗口细节

- 脚本应用图标和名称合并到终端窗口标题栏，移除内容区重复信息栏。
- 进程管理窗口恢复标准页面容器间距。

### AI Docker 契约加固

- 全部 Docker maintenance actions 使用逐动作必填字段契约。
- 修改现有容器、镜像、网络、存储卷、备份或 daemon 配置前读取最新真实状态和资源版本。
- 参数错误、资源冲突和资源不存在按含义反馈并重新规划；精确镜像删除不得由悬空镜像清理替代。

## 提交记录

- 来源 `62f406a`，候选 `5d12a56`：`feat(sites): surface website appearance metadata`。
- 来源 `6435a14`，候选 `2f87652`：`feat(desktop): merge app identity into script terminal chrome`。
- 来源 `7c38ba6`，候选 `334937e`：`fix(desktop): align process manager window spacing`。
- 来源 `840f34d`，候选 `5598e6e`：`fix(ai): enforce Docker maintenance contracts`。
- `4c37a0e`：`chore: prepare KPanel 0.53.0`。

## 上线前验证

- 版本一致性和工作流 YAML 检查通过。
- `npm ci`：252 个包，0 个漏洞。
- TypeScript 类型检查通过。
- 71 个前端测试文件、476 项测试通过。
- i18n 检查通过：1,730 条文案、18 个延迟目录。
- 生产构建通过；主入口 gzip 23.88 KiB，新增与修改页面均低于既有预算。
- Linux `go test ./...` 与 `go vet ./...` 通过。
- 网站外观来源分支 CI 通过：[run 31268033477](https://github.com/kejilion/KPanel/actions/runs/31268033477)。
- 脚本终端来源分支 CI 通过：[run 31268114137](https://github.com/kejilion/KPanel/actions/runs/31268114137)。
- 进程间距来源分支 CI 通过：[run 31268110955](https://github.com/kejilion/KPanel/actions/runs/31268110955)。
- 候选分支 CI 通过：[run 31268815814](https://github.com/kejilion/KPanel/actions/runs/31268815814)。
- 主分支 CI 通过：[run 31268939602](https://github.com/kejilion/KPanel/actions/runs/31268939602)。
- 发布工作流通过：[run 31269072345](https://github.com/kejilion/KPanel/actions/runs/31269072345)。

## 发布产物

- GitHub Release：[v0.53.0](https://github.com/kejilion/KPanel/releases/tag/v0.53.0)，非草稿、非预发布，共 8 个附件。
- OCI 多架构索引及 `latest`：`sha256:6e28e1fc0cdebe939cbedd75da7eb9180e00f558d6489335609c1f87126879d5`。
- linux/amd64：`sha256:8b5fbbf3ad2b388d0a1aeaf6c0f481e51d817c22c0681f690630dc6221b1b842`。
- linux/arm64：`sha256:2df31a00017618154d444ee7bd86987a9109cf67e60ec04012200a3916fa608c`。
- 公共镜像隔离拉取与 E2E 返回 `image_e2e=pass`。
- `apps` 安装/更新契约没有变化，KPanel 打包配置与 `apps/kpanel.conf` blob 均为 `d49383667cea8c3b7294bf40ba1e272370a2cb87`，无需应用市场提交。
- `kejilion.sh` 没有变化，本地与远端 `main` 均为 `dd26cf7eb962f985f94773c15f9b643677b4471c`。

## 生产备份与部署

- 升级前备份：`/root/kpanel-backups/v0.53.0-preupgrade-20260808T172451Z`，目录权限 `0700`。
- SQLite 备份大小 122,880 B，`PRAGMA integrity_check=ok`。
- 应用归档大小 21,586,731 B，全部备份校验和通过。
- 生产更新命令：`KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update k app kpanel`。

## 上线后验证

- 健康接口连续 3 次返回 `ok`，版本均为 `0.53.0`。
- 容器镜像为 `sha256:6e28e1fc0cdebe939cbedd75da7eb9180e00f558d6489335609c1f87126879d5`，状态 `running/healthy`，重启数 0，未发生 OOM。
- 非 root、只读根、CPU/内存/PID 限制、`CapDrop=ALL` 与 `no-new-privileges` 均生效。
- Agent 为 `active/running/enabled`，重启数 0，`NeedDaemonReload=no`；宿主机与镜像 Agent SHA-256 均为 `35127e414bf4c7d72b58b95c3869b6bbf2e58d715c19411b695bbdc17d8b459b`。
- SQLite 完整性检查通过；部署后 10 分钟 Panel 与 Agent 日志没有 `panic`、`fatal` 或 `error`。
- Docker 备份和 Docker 环境真实状态接口在 Agent Unix Socket 上返回有效 JSON。
- 生产发现 3 个站点：1 个符合外观接口条件但首页无有效 `<title>`，2 个不符合抓取条件；外观接口与空名称回退域名逻辑正常，当前生产数据没有可展示的网站名称。
- 2 分钟稳定性观察：60 次健康请求零失败，Panel/Agent 零重启；Agent RSS 23,660 KiB 至 23,844 KiB，FD 7 至 7。
- 外部健康接口返回 `0.53.0`，首页 HTTP 200 且包含标题。

## 回滚方案

1. 将生产镜像固定回 `v0.52.0` 的摘要 `sha256:e800d0d7cc8fffc906f37fe41ac1570a3e1fc6c71ac392eb130a04a452cc86e4`。
2. 重新创建 KPanel 容器并确认健康接口返回 `0.52.0`。
3. 如需恢复数据，先停止 KPanel、再次备份当前状态，再使用 `/root/kpanel-backups/v0.53.0-preupgrade-20260808T172451Z`。
4. 回滚后核验容器健康、Agent 二进制一致性、SQLite 完整性及日志。

## 已知限制

- 当前生产站点没有有效、可展示的首页标题，因此线上仅验证了真实接口和预期回退；标题实际显示已由隔离抓取测试、前端测试和 GitHub CI 覆盖。
- 未执行带登录态的生产 UI 人工点击；布局、窗口标题和回退图标由组件测试、生产构建及来源分支 CI 覆盖。
