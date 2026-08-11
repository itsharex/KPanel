# KPanel v0.61.4 发布验收记录

日期：2026-08-11

发布级别：L3

## 发布范围

本补丁包含两项生产可靠性修复：

- Agent systemd 单元显式保留 `CAP_SYS_PTRACE`，使受限服务中的固定只读 `ss` 适配器能够读取 socket owner 元数据；端口占用接口继续以 `kejilion.sh` 输出为唯一真源，不新增通用进程跟踪能力。
- 应用市场更新在拉取、配置、Agent 和服务切换前，使用当前镜像、当前 KPanel 网络和回环临时端口执行一次真实 Docker 端口发布预检。预检失败时清理临时容器并在切换前中止，不自动重启 Docker。

正式源码、标签和镜像对应提交：

```text
e1b236bf6804ff48f770f3f4e42c467c71a46c61
```

聚焦提交：

```text
f4a8eeb fix(agent): retain socket owner metadata
9059bfc fix(updater): preflight Docker port publishing
4bb6b3b test(updater): preserve lifecycle audit assertions
e1b236b chore: prepare KPanel 0.61.4
```

受管脚本真源未变：

```text
kejilion/sh@d82f043aa95064235b2bfe370e25e141cd75c321
SHA-256 40a9d77aa89d53a4e360026a6d0698622a248d01f059a1c92299dc56068d14f2
```

## v0.61.3 处置

v0.61.3 的公开 Release 与标签保持不可变。该版本在生产功能门禁中发现已安装 Agent systemd 能力集会剥离 socket owner 元数据，真实端口记录 79/79 降级为未知程序，因此未通过生产验收并已安全回滚到 v0.61.2。

回滚证据：

```text
/root/kpanel-release-evidence/v0.61.3/production-rollback-20260811T032817Z/summary.log
SHA-256 03a3c7fc71ec27e7e22a50487593e7c977609c1c82b9fb747c0f5d1ab6c2ea5a
```

v0.61.4 重新执行完整 L2、L3、CI、Release、公开镜像和生产门禁，不沿用 v0.61.3 的生产结论。

## 发布前验证

- 应用配置生命周期测试通过，覆盖安装、更新、回滚、systemd 能力精确断言，以及 Docker 端口发布预检失败时不调用 systemd、不切换登记文件/二进制并清理临时容器。
- 生产同环境的只读 Docker 端口发布预检通过，未修改生产版本，证据位于 `/root/kpanel-release-evidence/v0.61.4/docker-port-preflight/summary.log`，SHA-256 为 `e346aaab57c397d96e56ec38554acdfe8a017e8e5b4fa36434a84b18aaea2c11`。
- 154 隔离 L2 使用正式候选镜像和完整 systemd 安全指令启动 Agent。`CAP_SYS_PTRACE` 同时存在于 bounding/ambient 集合；真实记录 79/79 识别成功，未知程序为 0，识别出 6 类程序；nginx 50 条原始记录归并为 8 个唯一监听端点。
- 隔离 Panel 返回 80 条真实记录，80/80 识别成功。浏览器验证程序分组、nginx 搜索、重复端口合并、技术详情和 390×844 窄屏无横向溢出，页面错误为 0。目标主机未安装 `avahi-daemon` 与 `dhcpcd`，因此没有伪造这两类生产数据。
- L2 摘要位于 `/root/kpanel-release-evidence/v0.61.4/l2-evidence/summary.log`，SHA-256 为 `8b90a298276d9c78c15a35ec438971b9c5abefeb2007f08d3f42a442e5a949d0`。
- Linux L3 `VERIFY_BASE_REF=origin/main make verify-release` 通过，覆盖 12 个变更文件、Go/前端全量测试、race、vet、i18n、构建、依赖与漏洞审计、源码/镜像扫描、双架构构建、镜像契约和应用配置生命周期。前端共 83 个测试文件、599 项测试，i18n 共 2026 条、19 个目录；可达漏洞为 0。
- L3 日志位于 `/root/kpanel-release-evidence/v0.61.4/l3-verify-release.log`，SHA-256 为 `6f840c0baf5482d5d99791bbce1674e86ff0a4c655b887c06e3a86696aa5a059`。
- 候选 CI [31456161671](https://github.com/kejilion/KPanel/actions/runs/31456161671)、主线 CI [31456880119](https://github.com/kejilion/KPanel/actions/runs/31456880119) 与 Release [31457072155](https://github.com/kejilion/KPanel/actions/runs/31457072155) 均成功。

## 发布产物

[GitHub Release v0.61.4](https://github.com/kejilion/KPanel/releases/tag/v0.61.4) 为非 draft、非 prerelease，共 8 个附件。公开 `SHA256SUMS` 校验通过，部署归档包含 171 个条目；公开镜像端到端检查输出 `image_e2e=pass`，日志 SHA-256 为 `d6918030f72af42d7c376ceb7a1f902705a8473b11ff4ccacb39706e81322eb9`。

Docker 镜像：

```text
0.61.4 / latest:
sha256:c250a50cc338333b66a5925bd5f87358040b5e3a5ac7c3c38b138dab6b62d999

linux/amd64:
sha256:be8789a4b9fd4c66c5be06d0c9973a89f031d6d4f5d74cb4de8ae2e49c45d721

linux/arm64:
sha256:ea2b7abc2ab97f8588c7fd4852b7aec6fb5bb6213a99ed7487c07c173e581f16
```

`packaging/kejilion-app/kpanel.conf` 与 `kejilion/apps` 主线配置 blob 均为 `7289637a42b8209b301772139ff4404d08e196d2`；apps 主线提交为 `e7f90760b71cfe69c8b05af40131ab89739eb0f5`，Linux 文件 SHA-256 为 `87064543b4c8303f23d057e2a7612686551b5521ef232915cf2e39fe520be5fc`。

## 生产部署与验收

升级前生产明确运行 v0.61.2 不可变镜像，Panel 为 running/healthy/0，Agent 为 active/0。停写一致性备份：

```text
/root/kpanel-backups/v0.61.4-preupgrade-20260811T041014Z
```

压缩归档：

```text
/root/kpanel-backups/v0.61.4-preupgrade-20260811T041014Z.tar.gz
SHA-256 caad0afce260e6e022d9366b040f4830255e70925585f58d2343bb71f38b6865
```

备份清单 SHA-256 为 `2a5f9f479a5918d49c3d7578b7b075031662d92d6969149173d1ddd5d2dd10b4`；原目录和独立解包副本均逐文件校验通过。旧版恢复健康后，通过标准应用市场入口升级。

上线后验证：

- Panel 为 running/healthy/0，版本为 `0.61.4`，OCI 源码提交为 `e1b236b...a46c61`，镜像索引与正式摘要完全一致。
- Agent 为 active/0，`NeedDaemonReload=no`，实际进程的 bounding/ambient 能力集中均存在 `CAP_SYS_PTRACE`。
- 真实 Agent 端口接口返回 80 条记录，未知程序为 0，共识别 6 类程序；应用更新预检临时容器残留为 0。
- Panel/Agent 健康 JSON、Panel 状态 JSON 和 SQLite `integrity_check` 全部通过；应用密钥与集群节点身份文件和升级前备份一致。
- 首轮生产摘要位于 `/root/kpanel-release-evidence/v0.61.4/production-20260811T041243Z/summary.log`，SHA-256 为 `f3a6807868e56d8095eae701d0b63628e0c64b0a2a95202f4745d7781e8288cc`。
- 约两分钟内 60/60 次采样成功：Panel/Agent 始终健康且重启数为 0，版本始终为 `0.61.4`；采样后未知程序仍为 0，临时容器为 0，panic/fatal 为 0。持续健康摘要 SHA-256 为 `ce56060ed97e298570f28af165309c41ae08298d37e2b2e9136cd9e5dabf106a`。

生产未执行关机、重启 Docker、修改 SSH 策略、创建/删除账户、修改密码或公钥、调整角色、启停限流等危险写操作。

## 回滚点

首选代码与镜像回滚目标：

```text
v0.61.2
source 14da1a0a6785d2fcd60b0fc1c4746f285d0aaee7
image sha256:f6c750001b3787b76da70e4c9d48abc8ce091b856e80b3f1517ae8cea61a9e9b
```

普通回滚需将 Compose 固定到上述摘要，从 v0.61.2 镜像恢复 Agent，并恢复备份中的 v0.61.2 systemd 单元后执行 `daemon-reload`。只有数据或配置也需回退时，才停止 Panel/Agent、校验本轮停写归档 SHA-256，并恢复 `/root/kpanel-backups/v0.61.4-preupgrade-20260811T041014Z`。

## 遗留风险

- Docker 端口发布预检解决的是“更新前发现并安全中止”，不会自行修复宿主机缺失的 `DOCKER` iptables 链，也不会自动重启 Docker。若预检失败，仍需先修复宿主 Docker/iptables 状态再重试更新。
- 生产不传输管理员凭据，因此未执行生产浏览器登录与按钮级写操作；真实数据的浏览器闭环已在同一正式候选的 154 隔离环境完成，生产则通过相同源码/镜像、真实 Agent 读取、数据完整性和持续健康采样验收。
- L2、生产证据与备份保留在 root-only 目录，未纳入临时清理。
