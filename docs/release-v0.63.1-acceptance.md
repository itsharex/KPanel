# KPanel v0.63.1 发布验收记录

日期：2026-08-11

发布级别：L3

正式提交 / 标签：`6010b0ad281ad0b586d888f7c4f9a5c7d20197af` / `v0.63.1`

上一稳定版本 / 回滚点：`v0.62.2` / `4dfb4d124fcc07a3d2c573e40d4cf693e146dc86` / `sha256:f56a8532df6f50ca255d229a07554fdf48d133ab2e0a1065f828cbb01602bf7f`

## 发布画像

- 概览新增“一键系统调优”：固定 12 项、默认全选且可逐项取消，显示风险说明、逐项进度和断开恢复状态；维护任务逐项调用脚本，首项失败立即停止。
- 新增独立 typed API：`GET/POST /api/v1/system/system-tuning`，没有扩展旧 `/system/actions`；危险写入继续受 Agent、审计、版本和脚本协议边界约束。
- `kejilion.sh` 是唯一业务真源：新增 `KPANEL_SYSTEM_TUNING_PROTOCOL_VERSION="1"`、固定 12 项 `status/apply-item`、共享安全锁和结构化回执；菜单 66 与 KPanel 使用同一实现。
- 脚本修复原第 6 项“仅打印命令但不执行”的问题，改为复用事务化 firewall open-all；LinuxMirrors 与 network-optimize 均固定上游提交和摘要。
- 原始 KPanel 功能提交为 `6dcfcf71c1864a0d213c84114ca188768fd97763`，从最新主线迁入后的等价聚焦提交为 `5e240a11459ee0a58e063b6d960234fd222d929f`，L2 记录提交为 `7a89a77153ac17330649837b79e0771e64bd59cc`。
- `v0.63.0` 已冻结为 `071d4fb5af59426a61e018c465c3e8b2935d7768`，但 Release workflow 在正式产物发布前发现脚本契约校验仍硬编码旧摘要并停止；该 tag 不改写、没有 GitHub Release、没有正式镜像、没有生产部署。
- `v0.63.1` 增加共享脚本契约门禁 `c3d6508372f4fd7e3133820ba567b47ba7240b83`，最终版本冻结提交为 `6010b0ad281ad0b586d888f7c4f9a5c7d20197af`。产品功能相对冻结的 `v0.63.0` 未变。

## 未上线内容审计与排除项

- 本次候选只迁入系统调优的两个 KPanel 聚焦提交、对应 `kejilion.sh` 聚焦提交、发布契约修复和版本元数据；没有整体提交任何脏工作树。
- 旧诊断救援、服务状态、文件管理器应用配置等未完成草稿均保持原状并排除；后续必须分别审计、验收并形成独立候选。
- apps 主线保持 `e7f90760b71cfe69c8b05af40131ab89739eb0f5`。`packaging/kejilion-app/kpanel.conf` 在 `v0.62.2` 与 `v0.63.1` 的 blob 均为 `7289637a42b8209b301772139ff4404d08e196d2`，因此没有制造 apps 空提交。
- `C:\GitHub\kejilion-apps-file-manager\kpanel.conf` 等无关本地差异已保留，未覆盖、未提交、未上线。

## kejilion.sh 真源与门禁

- `kejilion/sh` 主线和功能分支均发布精确提交 `28f89c1b34df4b25e6ef9b144c328fdea75dbac9`。
- GitHub raw `kejilion.sh` SHA-256 为 `0583f7cd5be1f0bb6ec48d92e2cf224bfabfafada5788658bda4414ba9561229`；公开下载复核通过，根/CN 归一化同步，协议 marker 唯一。
- 154 聚焦发布检查通过：根/CN 语法与同步、system tuning smoke、多网络防火墙 smoke。证据 `/root/kpanel-release-evidence/system-tuning-sh-28f89c1/summary.log`，SHA-256 `88c48ec2681956a6766937d309a8f3670f1bbb73ee063443af3c0eeb3869f599`。
- 批量 Shell 测试另有 23 项通过、7 项既有 OpenClaw harness 依赖失败；这些失败与本功能无关，但本记录不将其表述为“全量 Shell 通过”。
- KPanel Dockerfile、文档、Release workflow、`scripts/verify-change.sh` 和镜像运行时现在都通过同一 `scripts/check-managed-script-contract.sh` 派生并核对脚本提交、raw URL、摘要、OCI label 与镜像内实际文件，避免多处硬编码再次漂移。

## 自动门禁

- KPanel 全量验证通过：`go test -count=1 ./...`、核心四包 race、`go vet ./...`、Linux amd64/arm64 Agent 与 Panel 构建、Web 85 个测试文件和 609 项测试、i18n 2109 条、typecheck、生产构建、ecosystem/version/diff 检查。
- `v0.63.1` 在 154 重新执行完整 L3：Trivy Panel/Agent 可达漏洞为 0，密钥扫描通过，镜像构建、运行约束、应用配置生命周期和新增 `managed_script_contract=pass` 均通过。
- L3 日志 `/root/kpanel-release-evidence/v0.63.1/l3-verify-release.log`，SHA-256 `bbf0db8046265d1fbfa060fa40e8957149365ce6ef9bcf4548c43495b061032b`；应用配置生命周期证据 SHA-256 `2e5731ab77ce82c4dc53d45bf14e09b46f522d7cca3da33f7d2fcf0cbc7a0c3f`。
- 候选 CI `31479460859`、主线 CI `31479720048`、Release run `31479983958` 均成功，且三个运行的 `head_sha` 都是正式提交 `6010b0ad...`。
- `v0.63.0` 失败的 Release run `31478136998` 停在“Verify runtime image contract”，发生在 draft Release、镜像推送和生产部署之前；失败候选分支 `release/v0.63.0-candidate` 保留为不可变故障证据。
- 候选 bundle：`C:\GitHub\_release-artifacts\v0.63.1\kpanel-v0.63.1-6010b0a.bundle`，SHA-256 `e99c5b8f7f00f17f798bf5714443f94afd7a22d740717a63ea0d9ec76112d17a`。

## 隔离真机与浏览器验收

- 154 隔离 L2 通过真实 Shell → Agent → Panel：固定 12 项读取、typed 单项选择、正式 systemd Agent 后台任务、running/完成状态、审计、stale version=409，以及注入失败后的首项停止与后续项不执行。
- L2 还原了工具注入、隔离容器、systemd 单元和临时目录；危险的 Swap、SSH 端口和开放所有端口没有在生产宿主执行。L2 证据目录 `/root/kpanel-release-evidence/system-tuning-6dcfcf7`，`SHA256SUMS` 摘要为 `173c1ac2684986760735f5916cc2246243512e482b4b91f2acdc14ba6802b71f`。
- 浏览器隔离验收确认：入口显示 12 项默认全选；清空后为 0/12，单选后为 1/12；Swap、SSH 端口和开放所有端口风险标识存在；390x844 无横向溢出，控制台错误 0，未发送危险 POST。
- 浏览器证据 `/root/kpanel-release-evidence/v0.63.0/browser-summary.log`，SHA-256 `080c6a38c270c0b4d2fb8dcdbae40459a6e3d30e9d2e01325f49526219e86379`。该验收对应产品冻结提交 `071d4fb...`；`v0.63.1` 只改发布校验脚本和版本元数据，UI 业务 diff 未变化，并由最终公开镜像 E2E 与生产健康验证补充覆盖。

## 发布产物与公开仓库

- [GitHub Release v0.63.1](https://github.com/kejilion/KPanel/releases/tag/v0.63.1) 为非 draft、非 prerelease，包含四个 Linux 原生二进制、部署归档、LICENSE、`SHA256SUMS` 和第三方许可清单，共 8 个附件。
- `docker.io/kjlion/kejilion-panel:0.63.1` 与 `latest` OCI index 均为 `sha256:94519b0bcfc539055a99b6a8ba91f3c691d9096e00dce489eba4ab5f4db050f8`。
- `linux/amd64`：`sha256:65a58abe1322ccb34eafd197a024edb3ba119b730c21df94ba909e1af69b3149`；`linux/arm64`：`sha256:b842336847c2c7a2b2c9cc640550e658cb557d153c52157897a7d14a54b78ee3`。
- 从公开仓库重新拉取后，版本、源码修订、脚本修订、脚本摘要、双架构与受限运行契约均通过，输出 `image_e2e=pass`；pull 日志 SHA-256 `09b0e2a43cde0f5a80235991355044d49507b961d5f9d06961ff37093c2df5c4`，E2E 日志 SHA-256 `d6918030f72af42d7c376ceb7a1f902705a8473b11ff4ccacb39706e81322eb9`。

## 生产部署与观察

- 生产目标为 154 主机配置的 `http://154.36.153.9:8080`；`kp.kejilion.pro` 是另一实例，不作为本次生产真源。
- 升级前制作停写一致性备份：`/root/kpanel-backups/v0.63.1-preupgrade-20260811T100958Z`；归档 `/root/kpanel-backups/v0.63.1-preupgrade-20260811T100958Z.tar.gz`，SHA-256 `8635b3c50e44bc57c791485716d2c232675f4aa2834f74345f7fe8cfdca5e682`；manifest SHA-256 `c0eb2794a43389923cf8ae249afb4269da4c909c0f2a9ac6c4d2b41ec48c2577`。停写复制、SQLite quick check、JSON 解析、逐文件清单和独立解包均通过。
- 使用标准入口 `KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update /home/docker/kpanel/bin/kejilion.sh app kpanel` 升级成功，未触发自动回滚；部署日志 SHA-256 `99f0e54e60ef2a8b0a9d88baa0e38bb12a4e87e30a72360867e2dd2f8908bedf`。
- 升级后本机和公开健康接口均为 `0.63.1` / `ok`；Panel running/healthy/0 重启、OOM false；Agent `0.63.1 v1alpha1`、active/running、`NeedDaemonReload=no`、`ExecMainStatus=0`。
- 容器源码修订、OCI index、amd64 子清单、固定脚本提交与 raw 摘要均与正式产物一致。宿主安装脚本仅含许可、统计与区域本地偏好差异，归一化后与固定 raw 脚本逐字节一致。
- SQLite `quick_check=ok`，Panel JSON 均可解析；`.env`、Agent 配置、Agent token、AI 密钥、集群身份密钥和 apps `kpanel.conf` 与升级前一致；近 10 分钟 Panel/Agent 日志未发现 panic、fatal 或 OOM。
- 60 次、2 秒间隔持续采样全部通过：本机与公开 HTTP 始终 `ok/0.63.1`，容器始终 running/healthy/0 重启/OOM false，Agent 始终 active。采样时间为 `2026-08-11T10:13:44Z` 至 `2026-08-11T10:15:53Z`。
- 生产证据目录：`/root/kpanel-release-evidence/v0.63.1/production-20260811T100958Z`；上线后验证摘要 SHA-256 `74efa4a8d6c85f7d13961130ae88d50aca977a9aeac7e8a0120e253cc62a5b5f`，采样 TSV SHA-256 `5b166401d6d185ce3d10318d8e08f14ae3930c1bce8adc0800ab7ae8ea295221`。
- 生产未运行一键调优、未修改 Swap、SSH、firewall、timezone 或软件源；写路径已由 154 隔离真机覆盖，生产只做无业务写入的一致性和健康验证。

## 回滚

- 源码/tag：`v0.62.2` / `4dfb4d124fcc07a3d2c573e40d4cf693e146dc86`。
- 镜像：`sha256:f56a8532df6f50ca255d229a07554fdf48d133ab2e0a1065f828cbb01602bf7f`。
- 停止 Panel 与 Agent，将 Compose 镜像恢复到上述不可变摘要；按需从已验证备份恢复 `/home/docker/kpanel`、`/root/apps/kpanel.conf`、systemd 单元和 symlink，执行 `systemctl daemon-reload` 后启动 Agent 与 Panel。
- 复核版本、容器健康和重启数、Agent 状态、脚本摘要、SQLite/JSON 完整性和稳定身份文件。只有数据或配置也需要回退时，才在停写状态下恢复相应备份。

## 交付节奏与遗留风险

- 聚焦功能提交：2026-08-11T16:30:07+08:00；最终候选冻结：2026-08-11T17:45:43+08:00；正式 Release 完成：2026-08-11T18:01:16+08:00；生产持续采样完成：2026-08-11T18:15:53+08:00。
- 聚焦功能提交到生产完成约 1 小时 46 分钟；最终候选冻结到生产完成约 30 分钟。生产升级没有发生应用回滚。
- `v0.63.0` 的失败证明现有发布门禁在产物发布前阻断了脚本摘要漂移；`v0.63.1` 已将该规则收敛为单一共享检查，但失败 tag 和候选分支仍刻意保留，不应删除或改写。
- 生产没有执行系统调优写验收；这是刻意安全边界，不代表危险宿主操作已在生产验证。
- 排除的旧草稿仍是本地未上线内容，不能自动随主线推送。
- 本轮复用项目 `release-kpanel` 和版本治理流程，并根据真实故障完善了固定脚本契约检查；没有新增重复工作流。
