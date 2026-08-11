# KPanel v0.61.2 发布验收记录

日期：2026-08-11

发布级别：L3

## 发布范围

本补丁修复账户管理弹窗在读取不含 SSH 公钥的账户后消失的问题：Agent 将空公钥集合稳定序列化为 `[]`，Web API 边界同时兼容历史 `null` 响应，加载期间弹窗保持打开。

正式源码、标签和镜像对应提交：

```text
14da1a0a6785d2fcd60b0fc1c4746f285d0aaee7
```

受管脚本真源未变：

```text
kejilion/sh@d82f043aa95064235b2bfe370e25e141cd75c321
SHA-256 40a9d77aa89d53a4e360026a6d0698622a248d01f059a1c92299dc56068d14f2
```

## 发布前验证

- 变更从 `85955f94cd50e40381778bcc6ebd7f755ec8ce83` 迁入独立候选；版本冻结提交为 `14da1a0a6785d2fcd60b0fc1c4746f285d0aaee7`，相对基线 `b6e6083aa08b53448c4d7e36b9cba33db30fbc33` 仅含 10 个预期文件；
- 前端 83 个测试文件、598 项测试通过；i18n 2011 条、TypeScript、生产构建通过；Go `go test -count=1 ./...` 与 `go vet ./...` 通过；
- 154 隔离候选真实读取 28 个宿主账户：`total=records=28`，`sshKeys:null=0`，非数组为 0；Panel API 同样返回 28/28；
- 154 浏览器闭环通过：弹窗加载时保持打开，加载完成仍可见；默认展示 2 张账户卡，显示系统账户后为 28 张；关闭再打开正常，页面错误为 0；
- L2 摘要位于 `/root/kpanel-release-evidence/v0.61.2/l2-evidence/summary.log`，SHA-256 为 `3b5f1ee93bcb0e86c02d460792834c06c7b5f1dbe56c17c0770da9db7850beed`；
- Linux L3 `make verify-release` 通过，覆盖版本一致性、Go/前端全量测试、race、vet、依赖与漏洞审计、源码和最终镜像扫描、双架构构建、镜像契约及应用配置生命周期；
- 候选 CI [31451374694](https://github.com/kejilion/KPanel/actions/runs/31451374694)、主线 CI [31452768765](https://github.com/kejilion/KPanel/actions/runs/31452768765)、Release [31452920327](https://github.com/kejilion/KPanel/actions/runs/31452920327) 均成功，且均对应上述源码提交。

## 发布产物

[GitHub Release v0.61.2](https://github.com/kejilion/KPanel/releases/tag/v0.61.2) 为非 draft、非 prerelease，共 8 个附件。公开 `SHA256SUMS` 中 4 个双架构二进制和部署归档均校验通过；部署归档包含 170 个条目。

Docker 镜像：

```text
0.61.2 / latest:
sha256:f6c750001b3787b76da70e4c9d48abc8ce091b856e80b3f1517ae8cea61a9e9b

linux/amd64:
sha256:1cf5f98c695d4bae5fc80dc13553ae73e3f74d58ebfc6bf06821f1186c2d5696

linux/arm64:
sha256:f7dc9304b913799cd7ff4ee4b7d01a7813ea4eba8e14e408d984364a0e96f9f8
```

154 公开镜像端到端检查输出 `image_e2e=pass`。

`packaging/kejilion-app/kpanel.conf` 与 `kejilion/apps` 主线中的 `kpanel.conf` blob 均为 `d49383667cea8c3b7294bf40ba1e272370a2cb87`；apps 主线为 `1f2740666a55ccbb3749ce83168e073c1ea08431`，本补丁无需 apps 提交。

## 生产部署与验收

升级前生产版本为 `0.61.1`，Panel healthy、Agent active，重启数均为 0。停写一致性备份：

```text
/root/kpanel-backups/v0.61.2-preupgrade-20260811T024358Z
```

归档 SHA-256：

```text
510a3d1384d1370d18fc8099a77d5fbdb165469306229e4dac54a09fa95db04e  kpanel.tar.gz
```

备份包含 94 个文件和 123 个树条目；独立解包后逐项清单一致，备份态 SQLite `quick_check` 通过。旧版恢复健康后，使用标准应用市场更新入口部署：

```text
KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update /home/docker/kpanel/bin/kejilion.sh app kpanel
```

上线后验证：

- 本机与配置的权威入口均返回 `status=ok`、`version=0.61.2`、`protocolVersion=v1alpha1`；
- Panel 为 running、healthy、0 重启，运行镜像索引为 `sha256:f6c750...a9e9b`，OCI 源码提交为 `14da1a0...aaee7`；
- Agent 为 active、0 重启、`NeedDaemonReload=no`，宿主二进制与正式镜像一致，SHA-256 为 `4c650746460f6e464572734e5c8066ef41e9174a2549d0a4cc5c603a1d9ec84b`；
- 受管脚本除 `permission_granted=true` 安装标记外与正式镜像固定脚本一致，两个协议 marker 均唯一；
- 5 项相关 capability 全部 enabled；生产只读账户快照为 28/28，`sshKeys:null=0`、非数组为 0；
- 2 个 SQLite、5 个 JSON、12 个 JSONL 文件（13,668 条记录）校验通过；升级后 Panel/Agent 的 panic、fatal、error 级事件均为 0；
- 约 2 分钟内，本机与权威入口 60/60 次采样成功，账户在第 1、30、60 次均读取成功，Panel/Agent 重启数保持 0。记录位于 `/root/kpanel-release-evidence/v0.61.2/production-health.log`，SHA-256 为 `324b1ee481434f0103d6e41a8c8f532f802be1f99285a4ee7e0d308746ac31bb`。

生产未执行创建/删除账户、修改密码或公钥、调整角色和 SSH 策略、限流启停、关机或重启等危险写操作。

## 回滚点

正常代码回滚目标：

```text
v0.61.1
source 6ed9be271707de746995b58b54993f5ca4395e91
image sha256:8e817980a6f05ee27ca987bea809ce426f317645f1ee1b3c63031ba933d385de
```

本补丁没有数据库、端口、Compose 或脚本协议迁移。普通回滚将 Compose 镜像固定到上述摘要后重建 Panel，并从旧镜像恢复 Agent；受管脚本真源在两个版本间相同。仅当数据或配置也需要回退时，才停止 Panel/Agent、校验本轮停写归档 SHA-256，并恢复 `/root/kpanel-backups/v0.61.2-preupgrade-20260811T024358Z`。

## 遗留风险

- 生产不传输管理员凭据，因此生产浏览器逐按钮操作未执行；真实宿主数据上的浏览器闭环已在 154 隔离候选完成，正式生产则通过相同源码/镜像、真实 Agent 账户读取和持续健康采样验收；
- L2 证据、生产健康记录与备份保留在 root-only 目录，未纳入临时清理。
