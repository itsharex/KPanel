# KPanel v0.86.2 发布验收记录

日期：2026-08-19

发布级别：L3（补丁）

## 发布身份

- 发布提交：`8f16d6f87486d50768e667b66154db2093b963d2`
- 注释 Tag：`v0.86.2`（tag object `b12c09c2654eb3466e1ff96795895d81cef0457b`，指向上述提交）
- 发布基线：`v0.86.1`（`0d25048fc28bf990736f2dbcc34235af1b59ec27`）
- L3 bundle：`kpanel-v0.86.2-8f16d6f-l3-v2.bundle`，SHA-256 `2a0b66df35318e627e6d8788f7c523e343f71d7229326c336811d43ce9a0dbe7`

## 变更范围

- 修复桌面文件互传进度条填充使用错误主题变量的问题：桌面工作区和全局传输进度条统一恢复品牌色填充。
- 仅修改 `web/src/styles/desktop.css`、`web/src/styles/main.css`，以及版本/CHANGELOG；不改变传输状态、取消、限流、后端协议或数据。
- 桌面壁纸选择、离开菜单后的延迟切换与过渡动画已经在 v0.86.0 进入主线，本版本只在升级说明中记录，不重复改写。
- 未修改 Go、API、数据库、依赖、端口、Agent 权限、`kejilion.sh`、apps 契约或生产数据。

## 门禁证据

- 本地候选：`npm ci`（0 vulnerabilities）、Web 101 files / 774 tests、i18n 2295 phrases / 20 catalogs、typecheck、production build、governance/version/ecosystem checks、`git diff --check` 均通过。
- 候选 CI：run `32237278954`，success；候选 Dependency freshness：`32237279089`，success。
- 主线 CI：run `32237683206`，success；主线 Dependency freshness：`32237683263`，success。
- Tag Dependency freshness：run `32238145113`，success；Release：run `32238145136`，success。
- 154 隔离 Runner：固定镜像 `sha256:b593c0ffe32e80a6a6ae8fcac38ed916587dbf698a6b6a55fa33887298737148`；`release_gate_runner=pass`，L3 日志 SHA-256 `e56490f8dfa0b7cbea6c8b7b388c9145bcc23352557859d149f87029757ed8f2`。
- L3 覆盖 Go 全量/race/vet、Web 全量、双架构构建、依赖策略、govulncheck、npm audit、Trivy source/image、install safety、managed-script contract 与 app-conf lifecycle；最终 `L3_RC=0`。

## Release 与 OCI

- GitHub Release：[v0.86.2](https://github.com/kejilion/KPanel/releases/tag/v0.86.2)，非 draft、非 prerelease。
- 公开 OCI：`docker.io/kjlion/kejilion-panel@sha256:200de84ee2bf4fe98a0d0267668339fb504bf0b41cd74de8efbcfab3d5abe7a0`。
- OCI 提供 linux/amd64（`sha256:db85de69fc4ee53947c5a1b24f197633e09dce19c232b09398f57190f7be235d`）和 linux/arm64（`sha256:8a33cbc00f4b6af036c693a34c3eee8f230535ff6d4dec79e4250733d71b5fc1`）。
- OCI labels：`org.opencontainers.image.version=0.86.2`、`org.opencontainers.image.revision=8f16d6f87486d50768e667b66154db2093b963d2`；受管脚本 revision `fdb0ac0e1f2b98d27339937e7f8eb0c9299c56a9`、SHA-256 `d8c06ad40c2845a2ee3f1f4c9f0780b7e30d65a58bca91a80cdca5c390222408`。
- 154 从不可变 digest 回拉并执行 `packaging/tests/image-e2e.sh`：`image_e2e=pass`；日志 SHA-256 `d6918030f72af42d7c376ceb7a1f902705a8473b11ff4ccacb39706e81322eb9`。
- Release `SHA256SUMS` asset digest：`sha256:520a5b3c7b7e94b51c2721e1047c5011843f5498d3f331ed18f37a2d9e835936`。本版无 apps/脚本仓库变更，不需同步提交。

## 154 生产升级

- 正式目标仅 `arena-154`；`108` 永久不纳入 KPanel 测试、灰度、备份或部署，本轮未连接。
- 升级前为 v0.86.1，旧镜像 `sha256:534c48bed39b05594656216308a050e0f0d7940e96aa55dbfcaf8f10f0c1aa5f`，Panel healthy、Agent active、restart=0、OOM=false。
- 停写一致性备份：`/root/kpanel-backups/v0.86.2-preupgrade-arena154-20260819T094033Z`；归档、镜像 tar、`.env` 和 apps 配置均完成抽取/hash 校验。关键 SHA-256：目录归档 `a154fa7057e55489c5b9f425cd848cf6407768c09b928ed1a4eb2905b4073018`，旧镜像 tar `e7d5d344e3f86e70d0eba64331687f4342fe6a5daefe2ab37f3c7ab1a4727fbb`，`.env` `0ba468d8031cd5f67c5a7ffb0333c13dabd1c72c5223f7004df0084712a096f4`，apps 配置 `82f06ca32ce827ef8d0c9c72e65eed9180841a23cbc507237072b58a0807ef04`。
- 标准更新入口：`KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update bash /home/docker/kpanel/bin/kejilion.sh app kpanel`；脚本输出 `KPANEL_PROGRESS 100` 和“应用更新完成”，实际新容器已启动。
- 升级后：`/api/v1/health` 连续 3 次返回 `status=ok, version=0.86.2`；容器 digest `sha256:200de84e…`、healthy、restart=0、OOM=false；Agent systemd active，日志为 `version=0.86.2`；apps 配置 SHA-256 仍为 `82f06ca32ce827ef8d0c9c72e65eed9180841a23cbc507237072b58a0807ef04`。

## 回滚

- 成套回滚点：v0.86.1、旧 OCI `sha256:534c48bed39b05594656216308a050e0f0d7940e96aa55dbfcaf8f10f0c1aa5f` 与上述备份目录中的匹配镜像、Compose、`.env`、Agent 文件、数据和 apps 配置。
- 回滚步骤：停止 Agent 与 Compose，恢复匹配版本的镜像/Compose/`.env`/Agent 文件和数据，执行 `systemctl daemon-reload`，启动并核对 `/api/v1/health`、healthy、restart/OOM 及配置哈希；不得只换镜像或只改配置。
- 本轮未执行回滚；154 当前保持 v0.86.2 healthy。

## 交付节奏与异常

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-08-19T17:00:32+08:00
- 候选冻结时间：2026-08-19T17:01:24+08:00
- 生产完成时间：2026-08-19T17:42:12+08:00
- 提交到生产用时：0.69 小时
- 是否回滚、紧急热修复或重复发布：否
- 若发生失败，发现时间、恢复时间和逃逸门禁：不适用
<!-- kpanel-release-metrics:end -->

<!-- kpanel-release-process-metrics:start -->
- 已记录发布流程异常或无效证据拦截次数：3
- 其中生产写操作开始后异常次数：1
<!-- kpanel-release-process-metrics:end -->

异常说明：首次 L3 bundle 未包含业务事实新鲜度所需的历史稳定 tags，按 fail-closed 停止并重新生成 v2 bundle；停写备份脚本最后的 Windows→SSH 换行使 `df` 参数带 CR，服务已恢复且归档校验通过；生产更新外层日志包装的 `PIPESTATUS` 引号错误导致包装返回码误报 1，但应用更新本身完成、Panel/Agent 健康，未触发回滚或重复写入。

## 遗留风险

- 本版只改变进度条前端颜色；壁纸延迟切换属于 v0.86.0 已上线能力，本轮未增加新的壁纸代码或迁移。
- 未进行长时间 soak；本补丁无常驻后端任务，生产已完成三次健康采样。
- `108` 永久不纳入 KPanel 任何线上测试或部署。
- 若后续发现产品行为问题，必须基于最新 main 制作新的最小补丁，不改写 `v0.86.2` Tag/Release。
