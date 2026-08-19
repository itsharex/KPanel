# KPanel v0.87.0 发布验收记录

日期：2026-08-20

发布级别：L3（兼容功能）

## 发布身份

- 发布提交：`15261a3ee1fa7af45d62134719f21f28f6554a9`
- 注释 Tag：`v0.87.0`，不可变指向上述提交
- 集成基线：`origin/main@ad20461992ff6e9a0ffd83a42dbc607c05cbfaec`
- 稳定回滚 Tag：`v0.86.3`（`c36adcece47ff9ba82010703906e49f8ef76bfa2`）
- 候选分支：`release/v0.87.0-candidate`

## 变更范围

- 诊断页增加一键跑分结果摘要：从受管脚本的真实结果中提取维度、耗时、状态和性能指标；保留终端原始输出和降级路径，不改变脚本执行边界。
- 桌面增加可配置小插件网格、服务状态卡片，以及统一的小插件/快捷方式管理和持久化；时钟布局在窄屏下保持可用。
- 诊断摘要解析器修复协议标记保留和磁盘速率单位换算，避免 `KPANEL_RESULT` 被错误覆盖或 `kbps` 被显示成错误单位。
- 未修改数据库迁移、外部端口、`kejilion.sh`、Agent/Node 协议、apps 仓库契约或生产数据结构；没有破坏性迁移。

## 门禁证据

- 本地候选：`npm ci`（0 vulnerabilities）、Web 105 文件/791 测试、i18n 2382 phrases / 20 catalogs、typecheck、production build、版本一致性、治理一致性、`git diff --check` 均通过。
- 本机未具备 Go runtime，且 Windows 工作区的 WSL Docker socket 权限不足，未将本机 Go/L3 结果误报为通过；完整 Go、竞态、govulncheck、npm audit、Trivy、双架构构建及 apps lifecycle 由远端门禁完成。
- 候选 CI：run `32272556227`，success；Dependency freshness：run `32271603664`，success。
- 主线 CI：run `32272799100`，success；Tag Dependency freshness：run `32273100928`，success。
- 主线曾在发布前发现诊断摘要的两项真实回归（磁盘速率单位、protocol parser 标记），由 `15261a3` 修复后候选和主线门禁重新通过；未进入生产。

## Release 与 OCI

- GitHub Release：[v0.87.0](https://github.com/kejilion/KPanel/releases/tag/v0.87.0)，非 draft、非 prerelease。
- Release workflow：run `32273100915`，success。
- 公开 OCI index：`docker.io/kjlion/kejilion-panel@sha256:9ff54a33fb9f325bc205c06c5416678394e7e1a5d63a452c22f8ffcac6100fb2`。
- OCI 子清单：linux/amd64 `sha256:02fa744817951236b17621de39980cdd66f7ca9b21ce08ad262aec0594b3b876`；linux/arm64 `sha256:503e542de86743d8aed210676c1e1dcfc6ef4a2bc36bc51aecba94830e3b976`。
- `latest` 与 `0.87.0` 指向同一 index digest；Release `SHA256SUMS` asset digest 为 `sha256:4cae73f81796123952a69c050087e67bc5058c1044a92936acee3b5efd7dbd7b`。
- 公开镜像 E2E 在 arena-154 隔离端口 `18096` 以该 immutable digest 回拉通过；测试脚本、容器和网络已清理。首次外层 SSH 包装因退出码转义产生非产品异常，随后使用同一镜像干净重跑并以退出码 0 通过。
- 本版未改变 `packaging/kejilion-app/kpanel.conf`；apps 仓库无须提交同步，远端 apps main 保持 clean。

## 154 生产升级

- 正式目标仅 `arena-154`；`108` 不连接、不测试、不备份、不部署。
- 升级前状态：v0.86.3、Panel healthy、Agent active、Agent restart=0；旧 OCI 为 `sha256:f0b9cb6056084a8a7dfcebde80d00eee3ab1228e9368b6794b7125ae7ba892a1`。
- 停写一致性备份：`/root/kpanel-backups/v0.87.0-preupgrade-arena154-20260819T160851Z`。
  - `kpanel-state.tgz` SHA-256：`35a64f9e694ad0aa5a004b04d44b3f283268cc87559d4487ce9fa7824c87b4a1`
  - `panel-image.tar` SHA-256：`ebaea1c656ff1089a6c35a697dbedd11628109f2e97441d66a04735fd1c39dfe`
  - 备份后 `/` 可用空间：8.9 GiB（99 GiB 总量、91% 使用率）。
- 使用标准更新入口完成升级：`KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update bash /home/docker/kpanel/bin/kejilion.sh app kpanel`，退出码 0，进度到 `KPANEL_PROGRESS 100`。
- 升级后容器 image digest 为上述 `v0.87.0` OCI index，容器 health 为 `healthy`；Panel health 连续 3 次返回 `initialized=true,status=ok,version=0.87.0`（2026-08-19T16:10:10Z、16:10:12Z、16:10:14Z）；Agent 为 `active`、restart=0；未发现 OOM 或重启。
- 本轮没有在生产执行浏览器自动化截图；诊断与桌面功能保留给用户在实际浏览器中做最终体验确认。

## 回滚

- 成套回滚点：`v0.86.3`、旧 OCI `sha256:f0b9cb6056084a8a7dfcebde80d00eee3ab1228e9368b6794b7125ae7ba892a1` 与上述备份中的匹配镜像、Compose、`.env`、Agent 文件和数据。
- 回滚时必须成套恢复镜像、Compose、`.env`、Agent 文件及数据，执行 `systemctl daemon-reload` 后启动并核对 Panel health、容器 healthy、Agent active、restart/OOM；不得只换镜像或只改配置。
- 本轮未执行回滚；154 当前保持 v0.87.0 healthy。

## 交付节奏与异常

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-08-19T23:29:33+08:00
- 候选冻结时间：2026-08-19T23:51:28+08:00
- 生产完成时间：2026-08-20T00:12:18+08:00
- 提交到生产用时：0.71 小时
- 是否回滚、紧急热修复或重复发布：是（发布前门禁修复，未进入生产）
- 若发生失败，发现时间、恢复时间和逃逸门禁：发现时间: 2026-08-19T23:48:00+08:00;恢复时间: 2026-08-19T23:51:28+08:00;逃逸门禁: 未逃逸: 主线 CI 在生产前阻断诊断摘要回归，修复后候选与主线门禁重新通过
<!-- kpanel-release-metrics:end -->

<!-- kpanel-release-process-metrics:start -->
- 已记录发布流程异常或无效证据拦截次数：2
- 其中生产写操作开始后异常次数：0
<!-- kpanel-release-process-metrics:end -->

## 遗留风险

- 诊断摘要依赖受管脚本实际输出；未知字段继续显示终端原始结果，不应据摘要推断未执行项目成功。
- 桌面小插件的用户偏好持久化已覆盖本地重载；本轮未做生产浏览器自动化截图，用户应在 154 实际环境确认布局与小插件交互。
- 首次本机 L3 因环境缺少 Go/Docker 权限未运行；远端 Release L3/安全/双架构门禁已通过。
- `108` 永久不纳入 KPanel 任何线上测试、灰度、备份或部署。
- 若发现产品问题，必须基于最新 main 创建新的最小补丁，不改写 `v0.87.0` Tag/Release。
