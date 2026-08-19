# KPanel v0.86.3 发布验收记录

日期：2026-08-19

发布级别：L3（补丁）

## 发布身份

- 发布提交：`c36adcece47ff9ba82010703906e49f8ef76bfa2`
- 注释 Tag：`v0.86.3`，不可变指向上述提交
- 发布基线：`v0.86.2`（`43e6cbf4ffaff1e4a70b0f98b00df4a54c89fee5`）
- 候选分支：`release/v0.86.3-candidate`

## 变更范围

- 壁纸选择器关闭后等待 `500ms` 再切换，避免弹窗退出与新壁纸加载同帧闪动。
- 新增独立固定的 `.desktop__wallpaper-veil` 遮罩层；遮罩使用 `0.42s` opacity 过渡，旧壁纸和新壁纸继续使用既有淡入淡出/位移过渡。
- 仅涉及桌面壁纸前端组件、样式和回归测试，以及版本与 CHANGELOG；未修改 Go、API、数据库、依赖、端口、Agent、`kejilion.sh` 或 apps 契约。

## 门禁证据

- 本地候选：`npm ci`（0 vulnerabilities）、定向 2 文件/40 测试、Web 101 文件/774 测试、i18n 2295 phrases / 20 catalogs、typecheck、production build、版本/治理检查、`git diff --check` 均通过。
- 候选 CI：run `32242688362`，success；候选 Dependency freshness：`32242688365`，success。
- 主线 CI：run `32243040252`，success；主线 Dependency freshness：`32243040238`，success。
- Release workflow：run `32243475390`，success；Release 为非 draft、非 prerelease。
- 本机 Linux L3 Runner 曾因 Docker Hub 基础镜像拉取超时而未启动；未将其误报为产品失败。候选/主线 CI 的完整 verify（含 race、govulncheck、npm audit、Trivy、双架构构建及 apps lifecycle）均成功，作为本次 L3 发布门禁依据。

## Release 与 OCI

- GitHub Release：[v0.86.3](https://github.com/kejilion/KPanel/releases/tag/v0.86.3)。
- 公开 OCI index：`docker.io/kjlion/kejilion-panel@sha256:f0b9cb6056084a8a7dfcebde80d00eee3ab1228e9368b6794b7125ae7ba892a1`。
- OCI 子清单：linux/amd64 `sha256:eaa2ef02fa344e1b608e22cbee8e61d0782315851f701b71482aca6ace04d9e6`；linux/arm64 `sha256:eaa704518d9983a7d142a00b06e3bef358a0e8128a35017f181b404fec41b16e`。
- `latest` 当前与 `0.86.3` 指向同一 index digest；Release `SHA256SUMS` asset digest 为 `sha256:5ecec52bb053c998e005486bca6d2847f54d2d53f7115e275c8468d54f91f252`。
- 本版没有脚本或 apps 配置变更，不需同步其他仓库。

## 154 生产升级

- 正式目标仅 `arena-154`；`108` 不连接、不测试、不备份、不部署。
- 升级前只读状态：v0.86.2、Panel healthy、Agent active、Agent restart=0；旧镜像为 `sha256:200de84ee2bf4fe98a0d0267668339fb504bf0b41cd74de8efbcfab3d5abe7a0`。
- 停写一致性备份：`/root/kpanel-backups/v0.86.3-preupgrade-arena154-20260819T104800Z`。
  - `kpanel-state.tgz` SHA-256：`fdb2faaa8e3e5c99aea52124abf98b252d4d09c0e59d1d36ef9f026e7b0e9891`
  - `panel-image.tar` SHA-256：`d647df0a5c3d15c9c4011f021f1c502bea779c85161812868e20ac40da5afce1`
- 使用标准更新入口完成升级：`KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update bash /home/docker/kpanel/bin/kejilion.sh app kpanel`；输出 `KPANEL_PROGRESS 100`，退出码 0。
- 升级后健康检查连续 3 次返回 `initialized=true,status=ok,version=0.86.3`；当前容器 digest 为上述 OCI index、状态 healthy；Agent active、restart=0；未发现本次升级引入的 OOM 或重启。
- 本轮未做生产浏览器自动化壁纸截图验收；候选本地视觉复核已完成，生产保留给用户进行最终体验确认。

## 回滚

- 成套回滚点：v0.86.2、旧 OCI `sha256:200de84ee2bf4fe98a0d0267668339fb504bf0b41cd74de8efbcfab3d5abe7a0` 与上述备份中的匹配镜像、Compose、`.env`、Agent 文件和数据。
- 回滚时必须成套恢复镜像、Compose、`.env`、Agent 文件及数据，执行 `systemctl daemon-reload` 后启动并核对 health、healthy、restart/OOM；不得只换镜像或只改模式/配置。
- 本轮未执行回滚；154 当前保持 v0.86.3 healthy。

## 交付节奏与异常

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-08-19T18:21:15+08:00
- 候选冻结时间：2026-08-19T18:21:15+08:00
- 生产完成时间：2026-08-19T18:54:32+08:00
- 提交到生产用时：0.56 小时
- 是否回滚、紧急热修复或重复发布：否
- 若发生失败，发现时间、恢复时间和逃逸门禁：本机 L3 Runner 因 Docker Hub 拉取超时未启动；远端 CI verify 成功，未逃逸到产品门禁
<!-- kpanel-release-metrics:end -->

<!-- kpanel-release-process-metrics:start -->
- 已记录发布流程异常或无效证据拦截次数：1
- 其中生产写操作开始后异常次数：0
<!-- kpanel-release-process-metrics:end -->

## 遗留风险

- 壁纸切换未在生产执行自动化视觉回归；用户应在 154 的实际浏览器中确认关闭弹窗后无闪白/黑屏及切换平滑。
- 本补丁无常驻后端任务，未执行长时间 soak。
- `108` 永久不纳入 KPanel 任何线上测试、灰度、备份或部署。
- 若发现产品问题，必须基于最新 main 创建新的最小补丁，不改写 `v0.86.3` Tag/Release。
