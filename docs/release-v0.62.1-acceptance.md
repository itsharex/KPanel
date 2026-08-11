# KPanel v0.62.1 发布验收记录

日期：2026-08-11

发布级别：L3

正式提交 / 标签：`69c22532e3130445b00b360745902da684333995` / `v0.62.1`

上一稳定版本 / 回滚点：`v0.61.4` / `e1b236bf6804ff48f770f3f4e42c467c71a46c61` / `sha256:c250a50cc338333b66a5925bd5f87358040b5e3a5ac7c3c38b138dab6b62d999`

## 发布画像

- 概览“基础系统设置”新增轻量 SSH 防御管理，以 `kejilion.sh` 为唯一真源，覆盖状态、启停/卸载、温和/标准/严格三档、封禁查询与解封、可信 IP/CIDR 和最近 20 条事件。
- 使用独立 typed Agent/Panel API、资源版本、共享写锁、固定动作、危险操作审计和持久化长任务；没有扩展旧通用系统动作接口。
- 同时修复空 `disks` 返回 `null` 导致概览崩溃，以及 SSH 防御 Manager 写动作递归锁死。
- 发布镜像固定 `kejilion/sh@e9c3078eb516b05f9df6d2a9294cf3b226ca02bd`，原始脚本 SHA-256 为 `147f624c479931c21b7d92392ff3e3a1a58b19bea4f98741f4ec114ab933546a`。
- 无数据库、端口或 Compose 迁移。升级本身不启用、停用或卸载 Fail2Ban，也不修改封禁、可信地址或防御策略。
- 风险等级为高：功能可以修改宿主机 SSH 防御状态，因此写路径只在隔离真机验收；生产只做状态读取。

## 精确提交与 v0.62.0 处置

- `e9c3078`：发布 `kejilion.sh` SSH 防御 manager protocol v1。
- `563735a`：迁入最新主线的 KPanel SSH 防御业务实现。
- `1ec2769`：准备 `0.62.0` 元数据。
- `v0.62.0` Release 在运行时镜像契约门禁被拦截：工作流仍核对旧脚本提交和摘要。该标签保持不可变；未公开 GitHub Release、未推送 `0.62.0` 正式镜像、未进入生产。
- `69c2253`：将 Release 镜像契约同步到新脚本提交/摘要，并准备下一可用补丁 `0.62.1`。业务代码相对 `1ec2769` 未变化。

## 自动门禁

- `kejilion.sh`：新 manager smoke、旧 F2B smoke、根/CN 同步、双脚本 `bash -n` 通过；GitHub raw 内容 SHA-256 与固定摘要一致，marker 唯一。
- KPanel：Go 全量测试、相关包 race、`go vet ./...`、Linux amd64/arm64 Agent/Panel 构建通过；Web 84 个测试文件、603 项测试通过；i18n 2048 条、typecheck、生产构建、ecosystem/version/diff 检查通过。
- 精确 `69c2253` 在 154 发布环境重新执行 `VERIFY_BASE_REF=origin/main make verify-release`，覆盖全量测试、race/vet、双架构构建、源码与镜像安全扫描、受限运行时镜像契约和应用配置生命周期；Trivy 可达漏洞与密钥均为 0，输出 `L3 release verification completed` 与 `app_conf_lifecycle=pass`。
- L3 证据：`/root/kpanel-release-evidence/v0.62.1/l3-verify-release.log`，SHA-256 `a042230364ce1a2f38ff057afcb5b6fd4779a3e56e5fa59ca9e38b96f4ae8f1d`。
- 候选 CI：`31461477373`，成功；主线 CI：`31461782998`，成功；Release：`31461958343`，成功。
- 失败的 `v0.62.0` Release run `31461145170` 作为门禁逃逸记录保留，没有人工跳过或覆盖。

## 隔离真机与浏览器验收

- 154 隔离 Ubuntu 24.04 systemd 容器使用正式候选镜像，运行真实 Fail2Ban sshd jail 与 iptables；验证 Shell → Agent → Panel 读取、启用、严格/标准切换、可信 CIDR 增删、真实单 IP 封禁/解封、资源版本冲突、审计和失败回滚。
- 隔离环境卸载完成并清理容器、网络、服务、隧道和临时目录；未触碰生产 Fail2Ban。
- 浏览器验证登录、概览版本、SSH 防御入口、加载期间弹窗稳定、完整内容、桌面与 `390x844` 窄屏；文档宽度分别未超过视口，页面控制台错误 0。
- L2 业务代码提交为 `1ec2769`；`69c2253` 相对它仅变更 Release 契约、版本元数据和本记录，精确 `69c2253` 另由 L3、公开镜像 E2E 与生产只读验收覆盖。
- L2 证据：`/root/kpanel-release-evidence/v0.62.0/l2-evidence/summary.log`，SHA-256 `c8b14f1a77b5338f4bbebd035d7e60b9d0c8d51f529af6ac09e9d0e7ea7914f3`。

## 发布产物与公开仓库

- [GitHub Release v0.62.1](https://github.com/kejilion/KPanel/releases/tag/v0.62.1) 为非 draft、非 prerelease，共 8 个附件；公开 `SHA256SUMS` 全部通过，部署归档可正常解包。
- `docker.io/kjlion/kejilion-panel:0.62.1` 与 `latest` OCI index 均为 `sha256:7486c1f096a5ce315d0a2b461d6c62e53f6d823820b28d1bf3fe258ecd7a10c6`。
- `linux/amd64`：`sha256:1334b98e0e1d995918f7f306bf03ca53cf8756b0cc7dac5cd262bb731b381bf8`；`linux/arm64`：`sha256:248ad22d38873604f823ceecbbb6305d1545bc944d3067036ce701835705e127`。
- 从公开仓库重新拉取后，版本、源码修订、脚本修订、脚本摘要和受限运行契约均通过，输出 `image_e2e=pass`。
- 公开产物证据：`/root/kpanel-release-evidence/v0.62.1/public-artifacts/summary.log`，SHA-256 `0cb058d0c04c60e1db4c414b4e2243af3bed5064973f935c8d1387f89e554746`。
- `packaging/kejilion-app/kpanel.conf`、本地 `kejilion/apps` 和 `origin/main` 的 blob 均为 `7289637a42b8209b301772139ff4404d08e196d2`；安装/更新契约未变化，无需 apps 空提交。apps 主线保持 `e7f90760b71cfe69c8b05af40131ab89739eb0f5`。

## 生产部署与观察

- 生产升级前为 `v0.61.4`，Panel running/healthy/0 重启，Agent active，Fail2Ban inactive。
- 停写一致性备份：`/root/kpanel-backups/v0.62.1-preupgrade-20260811T054041Z`；归档 `/root/kpanel-backups/v0.62.1-preupgrade-20260811T054041Z.tar.gz`，SHA-256 `d75716289c60affe8d36a4e5df6fab80c7cf624b54d96ade44018a04add78bc9`。原目录与独立解包副本均通过逐文件清单校验。
- 使用标准应用市场入口 `KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update /home/docker/kpanel/bin/kejilion.sh app kpanel` 升级，输出 `KPanel 更新完成 / Update Complete`。
- 升级后 Panel `0.62.1` running/healthy/0 重启，Agent `0.62.1` active/ok、协议 `v1alpha1`、`NeedDaemonReload=no`；源码修订、OCI index、脚本修订和脚本 SHA-256 与正式产物一致。
- 生产只读 SSH 防御状态为 disabled/not running；Fail2Ban 在升级前后均为 inactive，配置文件清单未变化。未执行启停/卸载、档位切换、封禁/解封或可信地址写操作。
- JSON 可解析、SQLite `integrity_check=ok`，身份密钥未变化，Panel/Agent 近端日志 panic/fatal 为 0。
- 60 次、2 秒间隔持续采样全部通过：Panel 始终 ok/healthy/0 重启，Agent 始终 active/ok，版本始终 `0.62.1`，Fail2Ban 60/60 inactive，SSH 防御 60/60 disabled，临时容器残留 0。
- 生产首轮证据：`/root/kpanel-release-evidence/v0.62.1/production-20260811T054309Z/summary.log`，SHA-256 `d0aca304a70338d3d39566a21e9bbcb367d74037fab27b018644ac8fca39b830`；采样摘要 SHA-256 `8bb936ddd3f2dd42f593bf42e1de25a1e3cdb1d3aaf30505afc08ab03bd2d3fd`。

## 回滚

- 源码/tag：`v0.61.4` / `e1b236bf6804ff48f770f3f4e42c467c71a46c61`。
- 镜像：`sha256:c250a50cc338333b66a5925bd5f87358040b5e3a5ac7c3c38b138dab6b62d999`。
- 先将 Compose 镜像恢复到上述不可变摘要，并从备份恢复 v0.61.4 Agent、`kejilion.sh`、systemd 单元和配置；执行 `systemctl daemon-reload` 后启动 Agent 与 Panel，复核版本、健康、重启数和数据完整性。
- 只有数据或配置也需要回退时，才在停写后从已验证备份恢复 `data`、密钥和配置。
- 回滚 KPanel/Agent 不会撤销管理员曾主动执行的 Fail2Ban 主机状态；本次生产没有此类写入，因此不存在额外 SSH 防御状态回退。

## 交付节奏与遗留风险

- 首个业务提交：2026-08-11T12:29:30+08:00；最终补丁候选冻结：2026-08-11T13:23:08+08:00；生产持续采样完成：2026-08-11T13:48:36+08:00。
- 首个业务提交到生产完成约 1.32 小时；最终 `v0.62.1` 候选冻结到生产完成约 0.43 小时。
- 本轮发生一次重复发布：`v0.62.0` 被发布工作流的旧脚本摘要契约拦截，逃逸门禁是“运行时镜像契约”；通过新补丁版本修复，未移动旧 tag，未手工公开失败 Release。
- 生产未验证 SSH 防御写操作、真实封禁或卸载，这是刻意的安全边界；这些写路径已由同业务代码候选的隔离真机闭环覆盖。
- 验收文档与 root-only 证据/备份保留；本轮没有新增重复工作流，复用并修正现有 release-kpanel 门禁。
