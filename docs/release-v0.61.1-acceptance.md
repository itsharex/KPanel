# KPanel v0.61.1 发布验收记录

日期：2026-08-11

发布级别：L3

## 发布范围

本轮正式上线由 `v0.61.0` 的两组功能和 `v0.61.1` 的生产兼容性补丁组成：

- 概览 → 网络工具：端口占用只读查看、限流自动关机；
- 概览 → 基础系统设置：账户、密码、SSH 公钥、角色和 SSH 登录策略管理；
- `v0.61.1` 修复 `/etc/passwd` 存在协议用户名语法无法表示的历史服务账户时，账户快照 `total` 与实际记录数不一致的问题。

正式源码、标签和发布镜像对应的源码提交为：

```text
6ed9be271707de746995b58b54993f5ca4395e91
```

受管脚本真源为：

```text
kejilion/sh@d82f043aa95064235b2bfe370e25e141cd75c321
SHA-256 40a9d77aa89d53a4e360026a6d0698622a248d01f059a1c92299dc56068d14f2
```

根目录与 `cn/kejilion.sh` 除既有区域参数外保持一致，network-operations protocol v1 和 account-management protocol v1 marker 均唯一。

## 发布过程中的阻断与回滚

`v0.61.0` 首次 Release 被运行镜像契约门禁拦截，原因是 Release 工作流仍断言上一版受管脚本提交和摘要；修复提交 `a1337a9` 更新三处契约值后，候选 CI、主线 CI 和 Release 重跑成功。

`v0.61.0` 首次生产升级后，只读账户状态在生产发行版数据上返回 `503 system_accounts_unavailable`。根因是脚本按 `/etc/passwd` 全部行数生成 `total`，同时按协议用户名语法过滤无法表示的历史服务账户。生产随即回滚到 `v0.60.2` 镜像、Agent 和受管脚本，Panel/Agent 均恢复健康且 0 重启；未恢复数据归档，因此未覆盖升级期间产生的数据。

补丁提交 `d82f043` 改为只统计协议可以表示并实际返回的账户，并加入对应回归测试。已发布的 `v0.61.0` 标签和资产未被重写，最终以语义化补丁版本 `v0.61.1` 发布。

## 发布前验证

- `kejilion.sh` 19 项 Shell 测试全部通过，根/CN 同步通过；真实 root Linux 状态协议在含历史服务用户名时返回 24 条总数和 24 条记录；
- 隔离 Ubuntu 24.04 root Linux 的 Shell → Agent → Panel L2 通过：生产同类账户快照边界、真实 `ss`、限流启用/更新/停用、crontab 保留、关机替身、账户密码/密钥/角色/SSH 策略、Root 安全迁移、版本冲突、锁冲突、回滚、`rollback-failed`、`needs-attention` 和完整原状恢复均通过；
- L2 证据目录为 `/root/kpanel-v0611-l2-20260811/evidence-final`；`summary.log` SHA-256 为 `c45f937523160f7a7984754f02dce6b60f1284b35581a6ad2dc5c13cd330dc5b`，`audit.json` SHA-256 为 `4a4d8777ff3cbf1adafb5b16e4995f7704ea584413bf8da117de2c403e80c607`；
- Linux L3 `make verify-release` 通过，覆盖版本一致性、生态策略、Go 测试、race、vet、依赖审计、漏洞/密钥/配置扫描、Linux 双架构构建、最终镜像扫描和应用生命周期；
- 前端 83 个测试文件、596 项测试通过；i18n 检查通过，共 2011 条文案和 19 个按页加载语言包；TypeScript 与生产构建通过；
- 公开 `v0.61.1` 镜像端到端检查输出 `image_e2e=pass`；
- GitHub Release 中 5 个二进制/部署归档均通过公开 `SHA256SUMS` 校验，部署包 169 个条目可正常读取。

远端门禁：

- 候选 CI：<https://github.com/kejilion/KPanel/actions/runs/31423801577>，成功；
- 主线 CI：<https://github.com/kejilion/KPanel/actions/runs/31424586331>，成功；
- Release：<https://github.com/kejilion/KPanel/actions/runs/31424908328>，成功。

## 发布产物

GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.61.1>

Release 为非 draft、非 prerelease，共 8 个附件。Docker 镜像索引如下：

```text
0.61.1 / latest:
sha256:8e817980a6f05ee27ca987bea809ce426f317645f1ee1b3c63031ba933d385de

linux/amd64:
sha256:a55194da61120b242c7ca55ae394b80ed13668f8a8618b4efe159f461541135c

linux/arm64:
sha256:bb8132fb5000dd30f22c93c1b6805818253093e9d38619fa4113534d1e8facec
```

`packaging/kejilion-app/kpanel.conf` 与 `kejilion/apps` 当前真源 blob 均为 `d49383667cea8c3b7294bf40ba1e272370a2cb87`，无需 apps 提交；`kejilion/apps` 主线为 `1f2740666a55ccbb3749ce83168e073c1ea08431`。

## 生产部署与验收

最终部署前生产版本为回滚后的 `0.60.2`，Panel healthy、0 重启，Agent active、0 重启、`NeedDaemonReload=no`。通过标准应用市场更新入口部署：

```text
KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update /home/docker/kpanel/bin/kejilion.sh app kpanel
```

最终停写一致性备份：

```text
/root/kpanel-backups/v0.61.1-preupgrade-20260810T194549Z
```

归档校验：

```text
1201bfa8160d9522f31d7d3bff4f80344104cc4442414673dafb13c4778b47bd  kpanel.tar.gz
```

归档已独立解包，与停写源目录 93 个文件和 123 个树条目逐项一致；随后旧版恢复健康，再执行正式更新。前一次 `v0.61.0` 尝试前的备份 `/root/kpanel-backups/v0.61.0-preupgrade-20260810T190501Z` 及其 SHA-256 `97d54c5f1b32a28790a6b6af7e2aee82db5604535bd87346c5a72db5273468f8` 同样保留。

上线后验证：

- 本机与配置的权威外部入口均返回 `status=ok`、`version=0.61.1`、`protocolVersion=v1alpha1`；
- Panel 为 running、healthy、0 重启，镜像索引为 `sha256:8e817980...385de`，OCI 源码提交为 `6ed9be2`；
- Agent 为 active、0 重启、`NeedDaemonReload=no`；宿主二进制与正式镜像内 Agent 的 SHA-256 均为 `a64d220ffe08d8e32699b98713d5ff9e8cb15070b25584022b09ca9622a3e226`；
- 受管 `kejilion.sh` 仅继承 `permission_granted=true` 安装标记，其他内容与镜像内固定脚本一致，两个协议 marker 唯一；
- 五项新 capability 全部 enabled；端口占用、限流状态和账户状态三个生产只读接口均通过，账户快照为 28 条总数和 28 条记录；
- 2 个 SQLite 数据库 `PRAGMA quick_check` 返回 `ok`，5 个 JSON 文件和 13242 条 JSONL 记录均可解析；
- 更新后 Panel/Agent 日志中 `panic`、`fatal` 和 error 级事件均为 0；
- 约 2 分钟内，本机与权威外部入口 60/60 次健康采样成功，版本、Panel/Agent 重启数始终稳定；账户快照在第 1、30、60 次均通过。采样记录 SHA-256 为 `c1a7a1830942c5dbaee5c81a1da8058da93c35dcd9a0754a34b80874ca336180`。

生产未执行限流启停、真实关机/重启、创建或删除测试账户、修改密码/公钥/角色、调整 SSH 策略或禁用 Root；这些危险写操作只在隔离 L2 环境验收。

## 回滚点

源码回滚：`v0.60.2`（`e1501e00747d9b7f89edb2cdac943f12f22f3178`）。

镜像回滚：

```text
docker.io/kjlion/kejilion-panel@sha256:508d5eefdb43e05b8e8a42b18d6758eb99387116de978e6b2da82c8a782bf982
```

脚本回滚：`kejilion/sh@a01067d63676b36c6067275d03a4827a7cb142bd`，SHA-256 `d9d9aa70de2c440f557c4db433bf0435ced813e46cb1acbb5c96f8ff601d8181`。

本版本没有数据库、端口或 Compose 迁移。普通回滚将 Compose 镜像固定到上述摘要后重建 Panel，并从旧镜像恢复 Agent 和受管脚本；只有数据或配置也需要回退时，才停止 Panel/Agent、验证备份 SHA-256 并恢复最终停写归档。回滚 Panel/Agent 不会撤销管理员之后主动执行的账户、SSH、限流脚本或 crontab 变更，这些宿主状态须按审计与恢复快照单独处理。

## 遗留风险

- 未使用生产管理员凭据执行浏览器逐按钮验收，避免读取或传输生产凭据；界面和完整写入链路由组件测试、全量测试、L3、公开镜像 E2E 和隔离 L2 覆盖；
- 账户列表对协议用户名语法无法表示的历史账户会有意过滤；本补丁保证总数、截断状态与实际返回记录保持一致，但这类账户仍不能通过面板修改；
- 两份生产备份和 L2 证据均保留在 root-only 目录，未纳入本轮临时清理。
