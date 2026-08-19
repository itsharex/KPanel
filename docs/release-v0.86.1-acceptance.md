# KPanel v0.86.1 发布验收记录

日期：2026-08-19

发布级别：L3（补丁）

## 发布身份

- 发布提交：`0d25048fc28bf990736f2dbcc34235af1b59ec27`
- 注释 Tag：`v0.86.1`（tag object `14b18beda3d83a8fa1406392adcc05234d2187bf`，指向上述提交）
- 发布基线：`v0.86.0`，生产回滚镜像 `sha256:d27e5fb6ec0d56cc6a862754bc2d21a4fbfd5608e79b2561304b670fb0f24ed8`
- Bundle：`kpanel-v0.86.1-0d25048.bundle`，SHA-256 `3b63bf28e045e2d341de7a319c7fd3104dc0b3a8139a5b3b899815c8b96279c0`

## 变更范围

- AI 助手正文、输入框、权限模式、模型选择、思考强度的字号和层级可读性调整；紧凑标签与代码信息保持不低于 12px。
- 窄屏主要操作触控高度与辅助说明换行优化；浅色/深色主题、键盘操作和业务交互保持不变。
- 仅修改 `web/src/styles/main.css`、`web/src/styles/ai-typography.test.ts`，以及版本/CHANGELOG；未改变 Go、API、数据库、依赖、端口、Agent 权限、`kejilion.sh` 或应用市场契约。
- 无数据迁移；回滚到 v0.86.0 只恢复原有 AI 字号和布局，不修改用户数据或会话。

## 门禁证据

- 候选 CI：run `32230969296`，success；候选 Dependency freshness：`32230969202`，success。
- 主线 CI：run `32231322111`，success；主线 Dependency freshness：`32231322173`，success。
- Tag Dependency freshness：run `32231692152`，success；Release：run `32231692143`，success。
- 154 Runner 完整 `release_gate_runner=pass`，L3 日志 SHA-256 `2990db385717416f6588234d0e9743ea309ae10bcc00260c4078769fecb6dc85`；Go 全量、Web 101 files/774 tests、race/vet、govulncheck、npm audit、Trivy source/image、双架构构建、install safety、managed-script contract、app-conf lifecycle 均通过。
- Web：i18n 2295 phrases、typecheck、production build、git diff --check 通过；AI 定向 28/28 通过。
- 本地精确候选浏览器复核：桌面与 390px 视口无横向溢出；截图 `ai-390.png` 保存在受限 release artifacts。预览 manifest 固定源提交 `0d25048...`。

## Release 与 OCI

- GitHub Release：[v0.86.1](https://github.com/kejilion/KPanel/releases/tag/v0.86.1)，非 draft、非 prerelease。
- 公开 OCI：`docker.io/kjlion/kejilion-panel@sha256:534c48bed39b05594656216308a050e0f0d7940e96aa55dbfcaf8f10f0c1aa5f`。
- OCI 同时提供 linux/amd64 与 linux/arm64；镜像 `org.opencontainers.image.version=0.86.1`、`org.opencontainers.image.revision=0d25048fc28bf990736f2dbcc34235af1b59ec27`。
- 154 从不可变 digest 回拉并运行 `packaging/tests/image-e2e.sh`：`image_e2e=pass`；日志 SHA-256 `d6918030f72af42d7c376ceb7a1f902705a8473b11ff4ccacb39706e81322eb9`。
- Release 资产 `SHA256SUMS` 已由 GitHub Release 生成并核对；本版本为前端补丁，`kejilion/apps/kpanel.conf` 无需提交同步。

## 154 生产升级

- 目标仅 `arena-154`；`prod-108` 永久不纳入 KPanel 测试、灰度、备份或部署，本轮未连接。
- 升级前：Panel v0.86.0 healthy，Agent active，restart=0、OOM=false；旧镜像为 `sha256:d27e5fb6ec0d56cc6a862754bc2d21a4fbfd5608e79b2561304b670fb0f24ed8`。
- 停写一致性备份：`/root/kpanel-backups/v0.86.1-preupgrade-arena154-20260819T082528Z`；包含旧镜像、Compose、`.env`、`agent.env`、Agent unit、脚本、面板数据及 apps 配置。归档恢复抽取校验通过，`SHA256SUMS` SHA-256 `64a4a8fa0bd8fb83f1689c612fa80cf84b1c9b23cf37ffaab36e1d21f5c56b9a`。
- 标准更新入口：`KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update bash /home/docker/kpanel/bin/kejilion.sh app kpanel`，退出码 0；未执行业务数据、SSH、防火墙、账户或 Docker 项目写操作。
- 升级后：Panel `0.86.1`，`/api/v1/health` 连续 3 次 `status=ok`；容器 digest `sha256:534c48...`、healthy、restart=0、OOM=false；Agent systemd active，日志报告 `version=0.86.1`；近 5 分钟无 panic/fatal/OOM。
- 升级前后 `/root/apps/kpanel.conf` SHA-256 均为 `82f06ca32ce827ef8d0c9c72e65eed9180841a23cbc507237072b58a0807ef04`；配置与数据保持原位。

## 回滚

- 成套回滚点：v0.86.0 镜像 digest、上述备份目录中的镜像/Compose/`.env`/Agent 文件和数据。
- 回滚步骤：停止 Agent 与 Compose，恢复匹配版本镜像、Compose、`.env`、Agent 文件和数据，`systemctl daemon-reload`，启动 Agent/Compose，核对 `/api/v1/health`、healthy、restart/OOM 与配置哈希；不得只换镜像或只改 mode/config。
- 本轮未执行回滚；生产保持 v0.86.1 healthy。

## 交付节奏与异常

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-08-19T15:46:20+08:00
- 候选冻结时间：2026-08-19T15:46:20+08:00
- 生产完成时间：2026-08-19T16:27:19+08:00
- 提交到生产用时：0.68 小时
- 是否回滚、紧急热修复或重复发布：否
- 若发生失败，发现时间、恢复时间和逃逸门禁：不适用
<!-- kpanel-release-metrics:end -->

<!-- kpanel-release-process-metrics:start -->
- 已记录发布流程异常或无效证据拦截次数：1
- 其中生产写操作开始后异常次数：0
<!-- kpanel-release-process-metrics:end -->

异常说明：首次备份脚本对 stopped/exited 状态判断过严；未进入新版本写入，已恢复基线并重新完成备份校验。

## 遗留风险

- 本版不改变 AI 后端能力；字号与布局已由自动化和本地精确候选浏览器复核，生产只做健康与版本核对。
- 108 永久不纳入 KPanel 任何线上测试或部署。
- 若后续发现产品行为问题，必须基于最新 main 制作新的最小补丁，不改写 v0.86.1 Tag/Release。
