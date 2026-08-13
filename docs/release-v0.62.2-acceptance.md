# KPanel v0.62.2 发布验收记录

日期：2026-08-11

发布级别：L3

正式提交 / 标签：`4dfb4d124fcc07a3d2c573e40d4cf693e146dc86` / `v0.62.2`

上一稳定版本 / 回滚点：`v0.62.1` / `69c22532e3130445b00b360745902da684333995` / `sha256:7486c1f096a5ce315d0a2b461d6c62e53f6d823820b28d1bf3fe258ecd7a10c6`

## 发布画像

- 修复移动端浏览器无法可靠携带认证头下载文件的问题：已认证会话通过受 CSRF 和同源保护的接口申请短期下载票据，浏览器随后使用无 Cookie 的固定下载 URL 发起原生下载。
- 下载票据只保存规范化绝对路径的摘要映射，随机令牌不进入日志；有效期 5 分钟、内存上限 128 个，支持 `GET`、`HEAD`、`Range` 和同一票据重试。
- Agent 与 Panel 文件流补齐 `HEAD` 转发；原有受认证文件内容接口、安全边界和 `kejilion.sh` 真源均未改变。
- 原始业务提交为 `2bdb5fd9febcd4014cca765664e38e56c811d3a7`；从最新主线 `ceb7b749d859e437588c583af3c0346232d47b1a` 迁入后的聚焦提交为 `0b51daa5f3544de9a57b8365e419e706d5013651`；版本冻结提交为 `4dfb4d124fcc07a3d2c573e40d4cf693e146dc86`。
- 无数据库、端口、Compose、应用配置或宿主机服务迁移。

## 未上线内容审计与排除项

- 本次候选只包含移动端认证下载修复及 `0.62.2` 版本元数据，没有整体发布任何脏工作区。
- `kejilion-panel-diagnostics-rescue` 的 3 个前端草稿、`kejilion-panel-overview-service-status` 的 2 个前端草稿和 `kejilion-apps-file-manager/kpanel.conf` 均未达到独立候选条件，已保持原状并排除。
- `kejilion.sh` 主线保持 `e9c3078eb516b05f9df6d2a9294cf3b226ca02bd`，原始脚本 SHA-256 为 `147f624c479931c21b7d92392ff3e3a1a58b19bea4f98741f4ec114ab933546a`；本轮没有脚本业务变更。
- apps 主线保持 `e7f90760b71cfe69c8b05af40131ab89739eb0f5`。KPanel 与 apps 的 `kpanel.conf` blob 均为 `7289637a42b8209b301772139ff4404d08e196d2`，因此没有制造 apps 空提交。

## 自动门禁

- 候选全量验证：Go 测试、Agent/Panel race、`go vet ./...`、Linux amd64/arm64 构建、Web 84 个测试文件和 605 项测试、i18n 2064 条、typecheck、生产构建、ecosystem/version/diff 检查全部通过。
- L2 日志：`/root/kpanel-release-evidence/v0.62.2/l2-verify.log`，SHA-256 `184e40325469c095bad5679a821686edffef2ccbadfaf326a6a464fc3d6b3bce`；race 证据 SHA-256 `fb26a322f14da91d74cba4072f1da3ed4d776791d24dc72d747189b5ec09bc9a`。
- `VERIFY_BASE_REF=origin/main make verify-release` 通过；Trivy Panel/Agent 可达漏洞为 0，密钥扫描通过，应用配置生命周期输出 `app_conf_lifecycle=pass`。
- L3 日志 SHA-256 `ae31f932ae95997b5f4f46a1cd57776c6005ed231490735e99574037a0b5e754`；应用配置生命周期证据 SHA-256 `2e5731ab77ce82c4dc53d45bf14e09b46f522d7cca3da33f7d2fcf0cbc7a0c3f`。
- 候选 CI `31468645231` 成功；产品主线 CI `31468867291` 成功。
- Release run `31469146923` 的源码、安全、构建、镜像扫描、运行时契约、正式附件、多架构镜像、`latest` 晋级和公开 Release 均成功。发布后删除候选分支已实际成功，但即时查询因 GitHub 引用缓存误报失败。
- 上述清理验证已由主线提交 `b1c808c1a47b3877c32a443e8580221cdbdf00c1` 改为带重试的 `git ls-remote --exit-code` 精确判定；该治理修复的主线 CI `31469783046` 成功，不改写已经冻结的 `v0.62.2` 产品产物。

## 隔离真机与浏览器验收

- 154 隔离环境通过真实 Shell → Agent → Panel 文件流闭环：无认证返回 401、缺少 CSRF 返回 403；票据下载返回 200，`Range` 返回 206，`HEAD` 返回 200，同一票据重试返回 200，伪造令牌和追加查询参数返回 404。
- 业务 L2 证据 SHA-256 `1ab560f1082bb0bdd4216dad64808d46e81f2d6b4301f6b3c91293f236cfd10d`。
- 浏览器在 `390x844` 视口使用真实临时文件验证：文件可见，点击“下载”触发浏览器下载事件；文档宽度、视口宽度和滚动宽度均为 390，无页面级横向溢出，控制台错误 0。
- 隔离用户、数据、容器、systemd 单元、隧道和临时文件均已清理；生产在隔离验收期间未变化。

## 发布产物与公开仓库

- [GitHub Release v0.62.2](https://github.com/kejilion/KPanel/releases/tag/v0.62.2) 为非 draft、非 prerelease，包含四个 Linux 原生二进制、部署归档、LICENSE、`SHA256SUMS` 和第三方许可清单。
- `docker.io/kjlion/kejilion-panel:0.62.2` 与 `latest` OCI index 均为 `sha256:f56a8532df6f50ca255d229a07554fdf48d133ab2e0a1065f828cbb01602bf7f`。
- `linux/amd64`：`sha256:18a6844420324869ef87e19fb8dd9576bf42987ef8323d09da0ed8e7936fd803`；`linux/arm64`：`sha256:37faf970c1a92b1e7e9b7172e31eabd5e412816d84faa64c1a9c9fa4f335f0d2`。
- 从公开仓库重新拉取后，版本、源码修订、脚本修订、脚本摘要和受限运行契约均通过，输出 `image_e2e=pass`；证据 SHA-256 `d6918030f72af42d7c376ceb7a1f902705a8473b11ff4ccacb39706e81322eb9`。
- 候选 bundle：`C:\GitHub\_release-artifacts\v0.62.2\kpanel-v0.62.2-4dfb4d1.bundle`，SHA-256 `ccb5121c5666ef50c861567693a5d4ca2946a5b2b1a3d59b633af78c8648211a`。

## 生产部署与观察

- 生产目标为 154 主机配置的 `http://154.36.153.9:8080`；`kp.kejilion.pro` 是另一实例，不作为本次生产真源。
- 升级前制作停写一致性备份：`/root/kpanel-backups/v0.62.2-preupgrade-20260811T074205Z`；归档 `/root/kpanel-backups/v0.62.2-preupgrade-20260811T074205Z.tar.gz`，SHA-256 `8f96658b53f67e23aa9d9014d55a32b67e9e6bc82edb78b2ec31274f366ab4e4`。独立解包和逐文件清单校验均通过。
- 使用标准应用市场入口 `KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update /home/docker/kpanel/bin/kejilion.sh app kpanel` 升级成功，未触发回滚；此前出现过的 Docker iptables 链缺失问题本次未复现。
- 升级后 Panel `0.62.2` running/healthy/0 重启、OOM false；Agent `0.62.2` active/running、协议 `v1alpha1`、`NeedDaemonReload=no`。
- 容器镜像、源码修订、脚本修订和脚本 SHA-256 与正式产物一致；宿主安装脚本只含预期的 `permission_granted=true` 本地状态差异，归一化后与固定原始脚本逐字节一致。
- JSON 可解析、SQLite `integrity_check=ok`，稳定身份文件未变化，近端日志未发现 panic、fatal 或 OOM。
- 60 次、2 秒间隔持续采样全部通过：HTTP 始终 200、API 始终 ok、版本始终 `0.62.2`、容器始终 healthy/0 重启、Agent 始终 active。
- 生产证据目录：`/root/kpanel-release-evidence/v0.62.2/production-20260811T074205Z`；升级日志 SHA-256 `7723c8e55d970d0984cc1e4bd3336be255d644e9e5dd0377fc0539ab7c92be4a`，总结 SHA-256 `bda8b793c7d40c453d3a26b2080a9921456a441d2f5ad22bb6490a86cc0911aa`，持续采样 SHA-256 `c900f864ddae47962d556340f2c1498998d3da56491ad2d598e00d9a6b1e482f`。
- 未在生产创建临时账户或文件，也未签发真实下载票据；这是避免生产业务写入的刻意安全边界，写路径已由 154 隔离真机和浏览器闭环覆盖。

## 回滚

- 源码/tag：`v0.62.1` / `69c22532e3130445b00b360745902da684333995`。
- 镜像：`sha256:7486c1f096a5ce315d0a2b461d6c62e53f6d823820b28d1bf3fe258ecd7a10c6`。
- 停止 Panel 与 Agent，将 Compose 镜像恢复到上述不可变摘要；按需从已验证备份恢复 `/home/docker/kpanel`、`/root/apps/kpanel.conf`、systemd 单元和 drop-in，执行 `systemctl daemon-reload` 后启动 Agent 与 Panel。
- 复核版本、健康状态、重启数、脚本摘要、SQLite 完整性和稳定身份文件。只有数据或配置也需要回退时，才在停写状态下恢复相应备份。

## 交付节奏与遗留风险

- 首个业务提交：2026-08-11T14:24:34+08:00；候选冻结：2026-08-11T14:59:10+08:00；正式 Release：2026-08-11T15:33:51+08:00；生产持续采样完成：2026-08-11T15:49:42+08:00。
- 首个业务提交到生产完成约 1.42 小时；候选冻结到生产完成约 0.84 小时。产品发布和生产升级没有发生回滚或热修复。
- 下载票据保存在 Panel 内存中，有效期 5 分钟；Panel 重启会使尚未使用的票据提前失效，用户重新点击下载即可获取新票据。
- 生产未执行真实认证下载写验收；移动端浏览器下载已在 154 隔离真机完成，生产只做无业务写入的健康和一致性验证。
- 排除的旧草稿仍是本地未上线内容，后续必须分别审计、验证并形成独立候选，不能自动随主线推送。
- 本轮复用现有 `release-kpanel` 与项目版本治理流程，没有新增重复工作流；根据真实发布事件完善了候选分支清理验证，并已由主线 CI 验证。
