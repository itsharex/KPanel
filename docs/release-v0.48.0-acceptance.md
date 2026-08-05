# KPanel v0.48.0 发布验收

## 发布范围

本版本将未上线内容收敛为 5 组：

1. 历史监控与整体 UI：框选缩放同步浏览器历史，支持最多 5 个容器对比，并优化桌面与手机端布局、间距和滚动边界。
2. 文件管理浏览器历史：成功进入目录时写入规范化路由，浏览器前进、后退和刷新可恢复目录。
3. 文件管理系统目录写入：修复 systemd 257 下 `/etc`、`/var/local` 等目录被错误挂载为只读的问题；Panel 数据目录继续保持只读和 API 禁止访问。
4. 前端大页面性能：概览可选请求并行加载，文件首次列表上限调整为 100，应用、Docker、文件和监控页面减少重复排序、检索与时间解析；历史监控默认范围调整为 6 小时。
5. 项目协作与发布规范：补充跨智能体任务、独立 worktree、单写入者、交付包、发布冻结、冲突恢复、AI Task Issue 和可复用工作流规则。

本次不包含数据库迁移、端口变更、`kejilion.sh` 协议变更或 SSH 端口修改。

## 版本与产物

- 发布提交：`b298c9647b07d50709b753a25beb85af72274027`。
- 标签：`v0.48.0`。
- 候选分支 CI：<https://github.com/kejilion/KPanel/actions/runs/31037520063>，结论为 `success`。
- 主分支 CI：<https://github.com/kejilion/KPanel/actions/runs/31037707927>，结论为 `success`。
- Release 工作流：<https://github.com/kejilion/KPanel/actions/runs/31037969923>，结论为 `success`。
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.48.0>，已公开且不是 prerelease，共 8 个发布资产。
- Docker OCI index：`sha256:144a64a77768e23b7d266d0d66ac944c77b2a968d11dbb5e9d2dd7d1b148a491`；`0.48.0` 与 `latest` 一致。
- linux/amd64：`sha256:1aac6a041404f65bc341e793fadb324fb42da18be05dd64fce0adc2294795847`。
- linux/arm64：`sha256:3b143376347af8f6912c0b14b65bcbd7dd6907f648c28469a87f4c99b1a38914`。
- `kejilion/apps` 同步提交：`1f2740666a55ccbb3749ce83168e073c1ea08431`；`kpanel.conf` 的 Git clean blob 与发布仓库均为 `d49383667cea8c3b7294bf40ba1e272370a2cb87`。

## 自动化与隔离验收

- Windows 集成候选通过 49 个前端测试文件、316 项测试、typecheck、1,653 条国际化短语检查和生产构建；主入口 JS 为 `22.58 KiB gzip`，主 CSS 为 `18.64 KiB gzip`。
- 154 在精确提交 `b298c96` 上完成 L3：Go 全量测试、核心 race、`govulncheck`、npm audit、Trivy 源码与最终镜像扫描、Linux 构建、部署隔离、应用配置生命周期和最终镜像验证全部通过。
- systemd 257 隔离临时服务中，候选 Agent 健康返回 200；通过文件 API 向 `/etc` 与 `/var/local` 写入均返回 201；Panel 数据目录 API 返回 403，命名空间内 `/etc` 为读写、Panel 数据目录为只读。测试文件、二进制和服务均已清理。
- 154 从 Docker Hub 重新拉取公开不可变摘要并执行独立冷启动 E2E，输出 `image_e2e=pass`；测试容器、网络和临时数据均已清理。
- 生产登录页通过 1440×900 与 390×844 浏览器验收；手机端 `scrollWidth=innerWidth=390`，无横向溢出，控制台无错误或警告。没有可复用的已登录浏览器会话，因此认证后的页面未重复执行浏览器点击验收。

## 154 生产上线

- 升级前 Panel 与 Agent 为 `0.47.0`，镜像摘要 `sha256:a2dd6c96eb32163a725a054bd3ddb0b8914b01cadd690f6673a01271e5a02ce2`；Panel healthy、重启 0，Agent active。
- 升级前备份：`/root/kpanel-backups/v0.48.0-preupgrade-20260805T191623Z`，目录权限 0700。SQLite 在线备份 122,880 B，`integrity_check=ok`，SHA-256 为 `5b05d3bfb21d628de4ab2fcba415961099faaf686358dd969705e8db014e3220`；应用归档 20,925,046 B，SHA-256 为 `a6ef1cb35cdc88176ecb6125191a61b6f9ce4da4e56d97fd769545011419e662`。
- 使用 `KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update k app kpanel` 完成标准应用市场更新；应用列表先快进到 `1f27406`，实际拉取镜像摘要与 Release OCI index 完全一致。
- 升级后 Panel 健康接口和 Agent 均报告 `0.48.0`；镜像修订标签为 `b298c96`。Panel healthy、重启 0、OOM=false，Agent active、重启 0。
- Panel 继续使用 `65532:65532`、只读根文件系统、256 MiB、1 CPU、128 PID、`cap-drop ALL` 和 `no-new-privileges`，网络仍为内部网与出口网双网络。
- 生产 Agent 的 6 小时监控查询返回 200；通过文件 API 向 `/etc` 与 `/var/local` 写入均返回 201；Panel 数据目录返回 403，并在服务命名空间内保持只读。测试文件已清理。
- 生产 `ai.db integrity_check=ok`；连续 3 轮 Panel、Agent 和 6 小时监控检查均为 200，Panel/Agent 错误日志计数均为 0。

## 回滚

- 源码与标签回滚点：`v0.47.0` / `668df46ddf439c5398e9bff69731e5eaeef58830`。
- 镜像回滚点：`docker.io/kjlion/kejilion-panel@sha256:a2dd6c96eb32163a725a054bd3ddb0b8914b01cadd690f6673a01271e5a02ce2`。
- 现场回滚应使用升级前应用归档与 SQLite 在线备份成对恢复 `/home/docker/kpanel`，重新标记上述镜像并重启 Agent 与 Compose；不得覆盖其他业务容器、网站、证书或数据库。
- 本版本没有数据结构迁移；直接回滚到 `v0.47.0` 可保留现有数据格式，但 systemd 257 下文件管理会再次受到根文件系统只读限制。
